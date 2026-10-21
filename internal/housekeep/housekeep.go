// Package housekeep đo và dọn rác do việc dew chạy để lại (cache build Go), và kê
// các cache khác kèm lệnh dọn để người dùng tự quyết. Chỉ xoá cache build Go; mọi
// thứ khác chỉ đo bằng lệnh đọc.
package housekeep

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

// goCacheREADME là dòng đầu README mà Go ghi vào GOCACHE (cmd/go/internal/cache/default.go).
const goCacheREADME = "This directory holds cached build artifacts from the Go build system."

// ErrNotGoCache: thư mục không có README của Go, nên không xoá gì trong đó.
var ErrNotGoCache = errors.New("not a Go build cache (its README does not start with Go's line)")

// Hạng của một mục: dew tự xoá được (SAFE), chỉ báo kèm lệnh (ASK), không bao giờ xoá (NEVER).
const (
	Safe  = "SAFE"
	Ask   = "ASK"
	Never = "NEVER"
)

// Item là một dòng của bảng dọn.
type Item struct {
	Class   string `json:"class"`
	Name    string `json:"name"`
	Path    string `json:"path,omitempty"`
	Bytes   int64  `json:"bytes"`  // -1: không đo được
	Effect  string `json:"effect"` // xoá thì được và mất gì
	Command string `json:"command,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Runner chạy một lệnh chỉ đọc và trả stdout.
type Runner func(ctx context.Context, name string, args ...string) (string, error)

// Output là Runner thật: hạn 5 giây (pnpm/pip/docker có thể treo hay gọi mạng), chạy ở
// thư mục tạm để go.mod của thư mục hiện tại không kéo một toolchain khác về. WaitDelay làm
// hạn đó thành trần cứng khi một tiến trình con (daemon) còn giữ stdout.
func Output(ctx context.Context, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, name, args...)
	c.Dir = os.TempDir()
	c.WaitDelay = time.Second
	out, err := c.Output()
	return string(out), err
}

// GoCacheDir trả GOCACHE: biến môi trường, không có thì `go env GOCACHE`. Rỗng khi không
// có go, GOCACHE=off, hay lỗi.
func GoCacheDir(ctx context.Context, run Runner) string {
	dir := os.Getenv("GOCACHE")
	if dir == "" {
		out, err := run(ctx, "go", "env", "GOCACHE")
		if err != nil {
			return ""
		}
		dir = strings.TrimSpace(out)
	}
	if dir == "off" {
		return ""
	}
	return dir
}

var (
	cacheSubdir = regexp.MustCompile(`^[0-9a-f]{2}$`)
	cacheEntry  = regexp.MustCompile(`^[0-9a-f]{64}-[ad]$`)
)

// GoCache là một cache build Go đã quét.
type GoCache struct {
	Dir     string
	Bytes   int64 // tổng kích thước các mục
	Entries int
	// Age là số byte theo lần dùng cuối (mtime): <1 ngày, 1-3, 3-7, >7 ngày.
	Age     [4]int64
	entries []goEntry
}

type goEntry struct {
	path  string
	size  int64
	mtime time.Time
	dir   bool
}

// ScanGoCache quét dir, chỉ khi dòng đầu README đúng chuỗi của Go. Chỉ tính các mục
// `<64 hex>-a|d` trong thư mục con hai chữ hex, không theo symlink.
// shortcut: kích thước là kích thước tệp, không phải khối trên đĩa (du lớn hơn khoảng 4 KiB
// mỗi mục nhỏ); đổi sang Stat_t.Blocks khi trần cần khớp du.
func ScanGoCache(dir string) (*GoCache, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "README"))
	if err != nil {
		return nil, err
	}
	if first, _, _ := strings.Cut(string(raw), "\n"); first != goCacheREADME {
		return nil, fmt.Errorf("%s: %w", dir, ErrNotGoCache)
	}
	subs, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	c := &GoCache{Dir: dir}
	now := time.Now()
	for _, s := range subs {
		if !s.IsDir() || !cacheSubdir.MatchString(s.Name()) { // IsDir của ReadDir không theo symlink
			continue
		}
		names, _ := os.ReadDir(filepath.Join(dir, s.Name()))
		for _, n := range names {
			if !cacheEntry.MatchString(n.Name()) || (!n.IsDir() && !n.Type().IsRegular()) {
				continue
			}
			info, err := n.Info()
			if err != nil {
				continue
			}
			e := goEntry{path: filepath.Join(dir, s.Name(), n.Name()), size: info.Size(), mtime: info.ModTime(), dir: n.IsDir()}
			if e.dir {
				e.size = DirSize(e.path)
			}
			c.entries = append(c.entries, e)
			c.Bytes += e.size
			c.Age[ageBucket(now.Sub(e.mtime))] += e.size
		}
	}
	c.Entries = len(c.entries)
	return c, nil
}

func ageBucket(d time.Duration) int {
	switch {
	case d < 24*time.Hour:
		return 0
	case d < 3*24*time.Hour:
		return 1
	case d < 7*24*time.Hour:
		return 2
	}
	return 3
}

// TrimResult đếm một lượt trim.
type TrimResult struct {
	Removed int   `json:"removed"` // mục đã xoá (sẽ xoá, khi chạy thử)
	Freed   int64 `json:"freed"`   // byte
	Failed  int   `json:"failed"`  // mục không xoá được (thường do quyền): đếm, không dừng
}

// busyWindow: một build giữ đường dẫn tệp cache đã tra và đọc nó sau, còn Go chỉ cập nhật mtime
// khi mục cũ hơn mtimeInterval (1 giờ): mục vừa được tra có thể trông cũ 59 phút. Gấp đôi
// khoảng đó thì mục mà một build tra trong giờ qua không bao giờ bị xoá.
const busyWindow = 2 * time.Hour

// TrimTo xoá mục dùng lâu nhất trước tới khi cache ≤ limit byte; dryRun chỉ đếm. Trim không làm
// hỏng cache (Go coi mục mất là miss và dựng lại). Không xoá mục dùng trong busyWindow, và Lstat
// lại từng mục ngay trước khi xoá để bỏ qua mục được dùng sau lúc quét; còn lại khoảng hở giữa
// Lstat và Remove (micro giây): build tra đúng mục đó lúc ấy có thể hỏng một lần.
// shortcut: không nhận ctx, lượt trim lớn không huỷ được giữa chừng (đo: 25 nghìn mục 0,5 s);
// thêm ctx khi cache thật làm lượt trim kéo dài.
func (c *GoCache) TrimTo(limit int64, dryRun bool) TrimResult {
	var r TrimResult
	sort.Slice(c.entries, func(i, j int) bool { return c.entries[i].mtime.Before(c.entries[j].mtime) })
	left := c.Bytes
	busy := time.Now().Add(-busyWindow)
	for _, e := range c.entries {
		if left <= limit || e.mtime.After(busy) {
			break
		}
		if !dryRun {
			if info, err := os.Lstat(e.path); err == nil && info.ModTime().After(e.mtime) {
				continue // một build vừa dùng nó
			}
			var err error
			if e.dir {
				err = os.RemoveAll(e.path)
			} else {
				err = os.Remove(e.path)
			}
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				r.Failed++
				continue
			}
		}
		r.Removed++
		r.Freed += e.size
		left -= e.size
	}
	return r
}

// DirSize là tổng kích thước tệp thường trong dir (không theo symlink).
func DirSize(dir string) int64 {
	var n int64
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				n += info.Size()
			}
		}
		return nil
	})
	return n
}

// Inventory đo các cache ASK và NEVER bằng lệnh chỉ đọc. Công cụ không có (hay đường
// dẫn không tồn tại) thì bỏ dòng; lỗi khác thành dòng có Error.
// shortcut: đo bằng cách đi cả cây, không giới hạn thời gian (kho pnpm lớn mất vài giây);
// thêm hạn khi `dew clean` chậm thật.
func Inventory(ctx context.Context, home string, run Runner) []Item {
	var items []Item
	dirItem := func(it Item, path string, err error) {
		path = strings.TrimSpace(path)
		switch {
		case errors.Is(err, exec.ErrNotFound):
			return
		case err != nil:
			it.Bytes, it.Error = -1, "cannot measure: "+err.Error()
		case path == "":
			return
		default:
			if _, err := os.Stat(path); err != nil {
				return
			}
			it.Path, it.Bytes = path, DirSize(path)
		}
		items = append(items, it)
	}
	if home != "" {
		dirItem(Item{Class: Ask, Name: "npm cache", Effect: "npm downloads packages again", Command: "npm cache clean --force"},
			filepath.Join(home, ".npm"), nil)
	}
	p, err := run(ctx, "pnpm", "store", "path")
	dirItem(Item{Class: Ask, Name: "pnpm store", Effect: "pnpm downloads packages no project uses again", Command: "pnpm store prune"}, p, err)
	p, err = run(ctx, "pip", "cache", "dir")
	dirItem(Item{Class: Ask, Name: "pip cache", Effect: "pip downloads and builds wheels again", Command: "pip cache purge"}, p, err)
	p, err = run(ctx, "go", "env", "GOMODCACHE")
	dirItem(Item{Class: Never, Name: "Go module cache",
		Effect: "offline builds (GOPROXY=off, -mod=readonly) need it, and it cannot be downloaded again without network"}, p, err)

	out, err := run(ctx, "docker", "system", "df", "--format", "{{.Type}}\t{{.Size}}\t{{.Reclaimable}}")
	switch {
	case errors.Is(err, exec.ErrNotFound):
	case err != nil:
		items = append(items, Item{Class: Ask, Name: "Docker", Bytes: -1, Effect: "images and build cache", Error: "cannot measure: " + err.Error()})
	default:
		sc := bufio.NewScanner(strings.NewReader(out))
		for sc.Scan() {
			f := strings.Split(sc.Text(), "\t")
			if len(f) < 3 {
				continue
			}
			reclaim, _, _ := strings.Cut(f[2], " ")
			switch f[0] {
			case "Images":
				items = append(items, Item{Class: Ask, Name: "Docker images", Bytes: parseSize(reclaim),
					Effect: "images no container uses; pulled or built again when needed", Command: "docker image prune -a"})
			case "Build Cache":
				items = append(items, Item{Class: Ask, Name: "Docker build cache", Bytes: parseSize(reclaim),
					Effect: "the next docker builds start without cache", Command: "docker builder prune"})
			case "Local Volumes":
				items = append(items, Item{Class: Never, Name: "Docker volumes", Bytes: parseSize(f[1]),
					Effect: "volumes hold data (databases, uploads) that cannot be rebuilt"})
			}
		}
	}
	return items
}

// parseSize đọc kích thước kiểu docker ("2.5GB", "0B"); -1 khi không đọc được.
func parseSize(s string) int64 {
	n, err := humanize.ParseBytes(s)
	if err != nil {
		return -1
	}
	return int64(n)
}

// Stuck đếm thứ trong một thư mục mà người dùng không xoá được: mục của user khác (build trong
// Docker chạy root) và thư mục của chính mình không có quyền ghi (module cache Go là 0555).
type Stuck struct {
	Foreign       int
	FirstForeign  string // mục đầu theo thứ tự duyệt: thư mục cao nhất của nhánh đó
	Owner         int    // chủ của FirstForeign
	ReadOnly      int
	FirstReadOnly string
}

// FindStuck duyệt dir (không tính dir, không theo symlink). Windows không có uid: luôn rỗng.
func FindStuck(dir string, uid int) Stuck {
	var s Stuck
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == dir {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		switch o, ok := fileUID(info); {
		case !ok:
		case o != uid:
			if s.Foreign == 0 {
				s.FirstForeign, s.Owner = p, o
			}
			s.Foreign++
		case d.IsDir() && info.Mode().Perm()&0o200 == 0:
			if s.ReadOnly == 0 {
				s.FirstReadOnly = p
			}
			s.ReadOnly++
		}
		return nil
	})
	return s
}

// Hint nói thứ gì trong dir không xoá được, của ai và cách xử lý (trả quyền, không xoá thứ chưa
// xem); rỗng khi không có gì.
func (s Stuck) Hint(dir string) string {
	var parts []string
	if s.Foreign > 0 {
		who := "uid " + strconv.Itoa(s.Owner)
		if u, err := user.LookupId(strconv.Itoa(s.Owner)); err == nil {
			who = u.Username + " (" + who + ")"
		}
		parts = append(parts, fmt.Sprintf(`%d file(s) owned by %s, e.g. %s: take them back with sudo chown -R "$(id -u):$(id -g)" %s, `+
			`and run Docker builds as docker run --user "$(id -u):$(id -g)" so they write files you own`, s.Foreign, who, s.FirstForeign, dir))
	}
	if s.ReadOnly > 0 {
		parts = append(parts, fmt.Sprintf("%d read-only dir(s), e.g. %s (Go makes its module cache read-only): make them writable with chmod -R u+w %s",
			s.ReadOnly, s.FirstReadOnly, dir))
	}
	if len(parts) == 0 {
		return ""
	}
	return "you cannot delete " + strings.Join(parts, "; ") + "; then run dew routine gc"
}
