package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/housekeep"
	"github.com/charmbracelet/crush/internal/lock"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

// Biến của bước dọn, số GB nguyên (1 GB = 10^9 byte), đọc như DEW_ROUTINE_SLOTS.
const (
	goCacheMaxEnv = "DEW_GOCACHE_MAX" // cache build Go lớn hơn thì trim; mặc định 20, 0 = không trần
	minFreeEnv    = "DEW_MIN_FREE_GB" // `routine run` không chạy khi đĩa còn ít hơn; mặc định 10, 0 = không kiểm
	gb            = 1_000_000_000
)

// goCacheTrimEvery: lượt routine quét cache Go tối đa mỗi giờ một lần (đo 21/10: quét 50
// nghìn mục mất 0,4 giây, cache thật khoảng 200 nghìn mục).
const goCacheTrimEvery = time.Hour

// currentUID là uid của người dùng; test thay để giả tệp của người khác.
var currentUID = os.Getuid

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Show what can be deleted to free disk space; delete only what is safe (--apply)",
	Long: `List the caches and leftovers that builds and dew runs pile up, in three classes:

  SAFE   deleted by --apply: the Go build cache, down to 80% of DEW_GOCACHE_MAX GB
         (default 20, 0 = no cap) when it is over, oldest entries first; and
         worktrees of finished routine runs that hold no work (as "routine gc")
  ASK    shown with its size and the exact command; you decide: npm, pnpm and pip
         caches, Docker images and build cache
  NEVER  shown with the reason, never deleted: the Go module cache (offline builds
         need it), Docker volumes (data), files of another user in routine worktrees

Without --apply nothing is deleted.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		limit, err := envGB(goCacheMaxEnv, 20)
		if err != nil {
			return &usageError{err}
		}
		minFree, err := envGB(minFreeEnv, 10)
		if err != nil {
			return &usageError{err}
		}
		o := cleanOpts{GoCacheMax: limit, MinFree: minFree, DataDir: routineDataDir(), Run: housekeep.Output}
		o.Apply, _ = cmd.Flags().GetBool("apply")
		o.JSON, _ = cmd.Flags().GetBool("json")
		o.Home, _ = os.UserHomeDir()
		return runClean(cmd.Context(), o, cmd.OutOrStdout())
	},
}

func init() {
	cleanCmd.Flags().Bool("apply", false, "Delete the SAFE rows: Go build cache over its cap, routine worktrees that hold no work")
	cleanCmd.Flags().Bool("json", false, "Print the result as JSON")
	rootCmd.AddCommand(cleanCmd)
}

// envGB đọc biến key (số GB nguyên) như routineSlots: biến export, không có thì tệp env
// của dew; rỗng thì def. Trả byte.
func envGB(key string, def int64) (int64, error) {
	v, err := execpolicy.FileEnv(key)
	if err != nil {
		return 0, err
	}
	if v == "" {
		return def * gb, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < 0 || n > math.MaxInt64/gb {
		return 0, fmt.Errorf("invalid %s=%q: use a whole number of GB (0 = off)", key, v)
	}
	return n * gb, nil
}

// scanGoCache tìm và quét cache build Go; (nil, nil) khi không có (không có go, GOCACHE=off,
// chưa có thư mục hay README).
func scanGoCache(ctx context.Context, run housekeep.Runner) (*housekeep.GoCache, error) {
	dir := housekeep.GoCacheDir(ctx, run)
	if dir == "" {
		return nil, nil
	}
	c, err := housekeep.ScanGoCache(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return c, err
}

// trimGoCache trim về 80% của limit (khoảng đệm: lượt sau không phải trim lại ngay) và mô tả kết quả.
func trimGoCache(c *housekeep.GoCache, limit int64) (housekeep.TrimResult, string) {
	r := c.TrimTo(limit*8/10, false)
	s := fmt.Sprintf("removed %d old entries (%s), now %s", r.Removed, humanize.Bytes(uint64(r.Freed)), humanize.Bytes(uint64(c.Bytes-r.Freed)))
	if r.Failed > 0 {
		s += fmt.Sprintf("; %d could not be removed (owned by another user?)", r.Failed)
	}
	return r, s
}

// routineHousekeep là bước dọn đầu mỗi lượt routine (sau khi dọn lượt cũ): trim cache build
// Go vượt trần, rồi từ chối chạy (75) khi đĩa chứa dữ liệu dew còn ít hơn o.MinFree.
func routineHousekeep(ctx context.Context, o routineOpts, d routineDeps, now func() time.Time, w io.Writer) error {
	if o.GoCacheMax > 0 {
		autoTrimGoCache(ctx, o.DataDir, o.GoCacheMax, now, w)
	}
	if o.MinFree <= 0 {
		return nil
	}
	freeBytes := d.FreeBytes
	if freeBytes == nil {
		freeBytes = housekeep.FreeBytes
	}
	free, err := freeBytes(o.DataDir)
	if err != nil { // không đo được thì vẫn chạy: chặn mọi routine vì một filesystem lạ là tệ hơn
		fmt.Fprintf(w, "[routine] warning: cannot measure free disk space: %v\n", err)
		return nil
	}
	if free < uint64(o.MinFree) {
		return &exitCodeError{code: exitCodeBusy, err: fmt.Errorf("only %s free on the disk holding %s (%s=%d): see what you can delete with `dew clean` (0 = no check)",
			humanize.Bytes(free), o.DataDir, minFreeEnv, o.MinFree/gb)}
	}
	return nil
}

// autoTrimGoCache trim cache build Go khi nó vượt limit: tối đa mỗi goCacheTrimEvery (dấu thời
// gian trong dataDir, chung cho mọi routine) và chỉ khi giữ được khoá trim (lượt khác đang
// trim thì bỏ qua).
func autoTrimGoCache(ctx context.Context, dataDir string, limit int64, now func() time.Time, w io.Writer) {
	// Tên bắt đầu bằng chấm: không trùng thư mục của một routine nào (tên routine không bắt đầu bằng chấm).
	release, err := lock.TryFile(filepath.Join(dataDir, ".gocache-trim.lock"))
	if err != nil {
		return
	}
	defer release()
	stamp := filepath.Join(dataDir, ".gocache-trim.last")
	if raw, err := os.ReadFile(stamp); err == nil {
		if last, err := time.Parse(time.RFC3339, strings.TrimSpace(string(raw))); err == nil {
			if d := now().Sub(last); d >= 0 && d < goCacheTrimEvery {
				return
			}
		}
	}
	// Ghi dấu trước khi quét: quét lỗi cũng không bị quét lại ở mỗi lượt.
	_ = os.WriteFile(stamp, []byte(now().UTC().Format(time.RFC3339)), 0o600)
	start := time.Now()
	c, err := scanGoCache(ctx, housekeep.Output)
	switch {
	case err != nil:
		fmt.Fprintf(w, "[routine] warning: not trimming the Go build cache: %v\n", err)
	case c != nil && c.Bytes > limit:
		_, s := trimGoCache(c, limit)
		fmt.Fprintf(w, "[routine] Go build cache %s was over %s (%s): %s in %v\n",
			humanize.Bytes(uint64(c.Bytes)), goCacheMaxEnv, humanize.Bytes(uint64(limit)), s, time.Since(start).Round(time.Millisecond))
	}
}

// cleanOpts là đầu vào `dew clean`, tiêm được để test.
type cleanOpts struct {
	Apply, JSON bool
	GoCacheMax  int64  // byte; 0 = không trần
	DataDir     string // thư mục dữ liệu routine
	Home        string
	Run         housekeep.Runner             // lệnh chỉ đọc đo cache của công cụ khác
	MinFree     int64                        // byte; đĩa còn ít hơn thì gợi ý thêm `go clean -cache`
	FreeBytes   func(string) (uint64, error) // nil: housekeep.FreeBytes
}

// cleanReport là kết quả của `dew clean --json`.
type cleanReport struct {
	Items       []housekeep.Item      `json:"items"`
	RoutineGC   string                `json:"routine_gc"` // các dòng của `routine gc` (chạy thử khi không --apply)
	GoCacheTrim *housekeep.TrimResult `json:"go_cache_trim,omitempty"`
	FreeBytes   int64                 `json:"free_bytes"` // chỗ trống của đĩa chứa DataDir; -1: không đo được
}

// runClean kê các mục theo hạng. --apply chỉ làm hạng SAFE: trim cache build Go và gỡ worktree
// routine như `routine gc` (không --include-ignored). Lệnh của hạng ASK chỉ được in.
func runClean(ctx context.Context, o cleanOpts, w io.Writer) error {
	var rep cleanReport
	c, err := scanGoCache(ctx, o.Run)
	switch {
	case err != nil:
		rep.Items = append(rep.Items, housekeep.Item{Class: housekeep.Safe, Name: "Go build cache", Bytes: -1, Effect: "not touched", Error: err.Error()})
	case c != nil:
		rep.Items = append(rep.Items, goCacheItem(c, o.GoCacheMax))
	}

	var gcOut bytes.Buffer
	st, gcErr := routineGCStats(ctx, o.DataDir, nil, gcOpts{DryRun: !o.Apply}, &gcOut)
	rep.RoutineGC = gcOut.String()
	rep.Items = append(rep.Items, housekeep.Item{Class: housekeep.Safe, Name: "routine worktrees", Path: o.DataDir, Bytes: st.Freed,
		Effect:  fmt.Sprintf("%d worktree(s) of finished runs that hold no work (reports and session data stay); %d kept, see below", st.Removed, st.Kept),
		Command: "dew routine gc"})
	wts, _ := filepath.Glob(filepath.Join(o.DataDir, "*", "wt", "*"))
	for _, wt := range wts {
		if hint := housekeep.FindStuck(wt, currentUID()).Hint(wt); hint != "" {
			rep.Items = append(rep.Items, housekeep.Item{Class: housekeep.Never, Name: "undeletable files", Path: wt, Bytes: -1, Effect: hint})
		}
	}
	// Đĩa thiếu mà cache Go còn dưới trần: --apply không trim gì nhưng `routine run` vẫn thoát 75.
	freeBytes := o.FreeBytes
	if freeBytes == nil {
		freeBytes = housekeep.FreeBytes
	}
	freeDir := existingDir(o.DataDir)
	free, freeErr := freeBytes(freeDir)
	rep.FreeBytes = int64(free)
	if freeErr != nil {
		rep.FreeBytes = -1
	}
	low := freeErr == nil && o.MinFree > 0 && free < uint64(o.MinFree)
	if low && c != nil && (o.GoCacheMax == 0 || c.Bytes <= o.GoCacheMax) {
		rep.Items = append(rep.Items, housekeep.Item{Class: housekeep.Ask, Name: "Go build cache", Path: c.Dir, Bytes: c.Bytes,
			Effect:  "the disk is low and the cache is under its cap: deleting all of it frees the most, and every package builds again once",
			Command: "go clean -cache"})
	}
	rep.Items = append(rep.Items, housekeep.Inventory(ctx, o.Home, o.Run)...)

	var trimLine string
	if o.Apply && c != nil && o.GoCacheMax > 0 && c.Bytes > o.GoCacheMax {
		r, s := trimGoCache(c, o.GoCacheMax)
		rep.GoCacheTrim, trimLine = &r, s
	}

	if o.JSON {
		return errors.Join(json.NewEncoder(w).Encode(rep), gcErr)
	}
	fmt.Fprintf(w, "%-5s  %-18s  %9s  %s\n", "CLASS", "ITEM", "SIZE", "PATH")
	for _, it := range rep.Items {
		size := "?"
		if it.Bytes >= 0 {
			size = humanize.Bytes(uint64(it.Bytes))
		}
		fmt.Fprintf(w, "%-5s  %-18s  %9s  %s\n", it.Class, it.Name, size, it.Path)
		for _, line := range []string{it.Error, it.Effect} {
			if line != "" {
				fmt.Fprintf(w, "       %s\n", line)
			}
		}
		if it.Command != "" {
			fmt.Fprintf(w, "       run: %s\n", it.Command)
		}
	}
	label := "routine gc --dry-run"
	if o.Apply {
		label = "routine gc"
	}
	fmt.Fprintf(w, "\n%s:\n%s", label, rep.RoutineGC)
	switch {
	case freeErr != nil:
		fmt.Fprintf(w, "\nfree space on the disk holding %s: cannot measure: %v\n", freeDir, freeErr)
	case low:
		fmt.Fprintf(w, "\nfree space on the disk holding %s: %s, below %s=%d: routine runs refuse to start; free space with the ASK commands above\n",
			freeDir, humanize.Bytes(free), minFreeEnv, o.MinFree/gb)
	default:
		fmt.Fprintf(w, "\nfree space on the disk holding %s: %s\n", freeDir, humanize.Bytes(free))
	}
	switch {
	case !o.Apply:
		fmt.Fprintln(w, "\nNothing was deleted: dew clean --apply deletes only the SAFE rows.")
	case trimLine != "":
		fmt.Fprintf(w, "Go build cache: %s\n", trimLine)
	}
	return gcErr
}

// existingDir là dir, hoặc thư mục cha gần nhất có trên đĩa (máy chưa chạy routine nào).
func existingDir(dir string) string {
	for {
		if _, err := os.Stat(dir); err == nil || filepath.Dir(dir) == dir {
			return dir
		}
		dir = filepath.Dir(dir)
	}
}

// goCacheItem là dòng SAFE của cache build Go: dung lượng theo tuổi và trần.
func goCacheItem(c *housekeep.GoCache, limit int64) housekeep.Item {
	h := func(n int64) string { return humanize.Bytes(uint64(n)) }
	it := housekeep.Item{Class: housekeep.Safe, Name: "Go build cache", Path: c.Dir, Bytes: c.Bytes,
		Effect: fmt.Sprintf("by last use: <1 day %s, 1-3 days %s, 3-7 days %s, older %s; Go rebuilds what it needs, so the next builds are slower",
			h(c.Age[0]), h(c.Age[1]), h(c.Age[2]), h(c.Age[3]))}
	switch {
	case limit == 0:
		it.Effect += "; no cap (" + goCacheMaxEnv + "=0)"
	case c.Bytes > limit:
		it.Effect += fmt.Sprintf("; over %s (%s): --apply trims it to %s, oldest first", goCacheMaxEnv, h(limit), h(limit*8/10))
		it.Command = "dew clean --apply"
	default:
		it.Effect += fmt.Sprintf("; under %s (%s)", goCacheMaxEnv, h(limit))
	}
	return it
}
