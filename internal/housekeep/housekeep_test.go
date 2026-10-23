package housekeep

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeCache dựng một GOCACHE giả: README của Go cùng các tệp quản lý mà trim không được đụng.
func fakeCache(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range map[string]string{
		"README":         goCacheREADME + "\nRun \"go clean -cache\" if the directory is getting too large.\n",
		"trim.txt":       "1760000000",
		"testexpire.txt": "1\n",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644))
	}
	return dir
}

// putEntry ghi mục số n (kind 'a' hoặc 'd') size byte, dùng lần cuối age trước. asDir: mục
// `-d` là thư mục chứa một tệp (cache executable của Go).
func putEntry(t *testing.T, dir string, n int, kind byte, size int, age time.Duration, asDir bool) string {
	t.Helper()
	sum := sha256.Sum256([]byte(strconv.Itoa(n)))
	name := hex.EncodeToString(sum[:]) + "-" + string(kind)
	p := filepath.Join(dir, name[:2], name)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	body := []byte(strings.Repeat("x", size))
	if asDir {
		require.NoError(t, os.Mkdir(p, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(p, "tool"), body, 0o755))
	} else {
		require.NoError(t, os.WriteFile(p, body, 0o644))
	}
	when := time.Now().Add(-age)
	require.NoError(t, os.Chtimes(p, when, when))
	return p
}

const day = 24 * time.Hour

// Xoá mục dùng lâu nhất trước tới khi ≤ trần; mục `-d` là thư mục bị xoá cả cây; tệp lạ,
// symlink, thư mục con không phải hai chữ hex và tệp quản lý của Go không bị đụng.
func TestTrimTo_RemovesOldestFirstUntilUnderTheCap(t *testing.T) {
	dir := fakeCache(t)
	oldest := putEntry(t, dir, 1, 'd', 100, 10*day, false)
	second := putEntry(t, dir, 2, 'a', 100, 6*day, false)
	exe := putEntry(t, dir, 3, 'd', 100, 4*day, true)
	kept1 := putEntry(t, dir, 4, 'a', 100, 2*day, false)
	kept2 := putEntry(t, dir, 5, 'd', 100, 12*time.Hour, false)

	outside := t.TempDir()
	target := filepath.Join(outside, "precious")
	require.NoError(t, os.WriteFile(target, []byte(strings.Repeat("p", 1000)), 0o644))
	sum := sha256.Sum256([]byte("link"))
	linkName := hex.EncodeToString(sum[:]) + "-d"
	link := filepath.Join(dir, linkName[:2], linkName)
	require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
	require.NoError(t, os.Symlink(target, link))
	// Thư mục con là symlink trỏ ra ngoài, chứa một tên mục hợp lệ.
	outEntry := filepath.Join(outside, hex.EncodeToString(sum[:])+"-a")
	require.NoError(t, os.WriteFile(outEntry, []byte("o"), 0o644))
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "ee")))
	foreign := filepath.Join(dir, filepath.Base(filepath.Dir(oldest)), "notes.txt")
	require.NoError(t, os.WriteFile(foreign, []byte("mine"), 0o644))
	fuzz := filepath.Join(dir, "fuzz", hex.EncodeToString(sum[:])+"-a")
	require.NoError(t, os.MkdirAll(filepath.Dir(fuzz), 0o755))
	require.NoError(t, os.WriteFile(fuzz, []byte("f"), 0o644))
	longAgo := time.Now().Add(-30 * day)
	for _, p := range []string{foreign, fuzz, filepath.Join(dir, "README"), filepath.Join(dir, "trim.txt")} {
		require.NoError(t, os.Chtimes(p, longAgo, longAgo))
	}

	c, err := ScanGoCache(dir)
	require.NoError(t, err)
	require.Equal(t, int64(500), c.Bytes, "only the five entries count")
	require.Equal(t, 5, c.Entries)

	r := c.TrimTo(250, false)
	require.Equal(t, TrimResult{Removed: 3, Freed: 300}, r)
	for _, gone := range []string{oldest, second, exe} {
		require.NoFileExists(t, gone)
		require.NoDirExists(t, gone)
	}
	for _, stays := range []string{
		kept1, kept2, foreign, fuzz, link, target, outEntry,
		filepath.Join(dir, "README"), filepath.Join(dir, "trim.txt"), filepath.Join(dir, "testexpire.txt"),
	} {
		_, err := os.Lstat(stays)
		require.NoErrorf(t, err, "%s must stay", stays)
	}
	after, err := ScanGoCache(dir)
	require.NoError(t, err)
	require.Equal(t, int64(200), after.Bytes)
}

// Chạy thử chỉ đếm, không xoá gì.
func TestTrimTo_DryRunRemovesNothing(t *testing.T) {
	dir := fakeCache(t)
	a := putEntry(t, dir, 1, 'a', 100, 3*day, false)
	b := putEntry(t, dir, 2, 'd', 100, 2*day, true)
	c, err := ScanGoCache(dir)
	require.NoError(t, err)
	require.Equal(t, TrimResult{Removed: 2, Freed: 200}, c.TrimTo(0, true))
	require.FileExists(t, a)
	require.DirExists(t, b)
}

// Go chỉ cập nhật mtime khi mục cũ hơn 1 giờ, nên mục mà một build vừa tra (và sẽ đọc sau)
// có thể trông cũ 59 phút: không xoá mục dùng trong 2 giờ qua, kể cả khi cache vẫn trên trần.
func TestTrimTo_KeepsEntriesUsedInTheLastTwoHours(t *testing.T) {
	dir := fakeCache(t)
	old := putEntry(t, dir, 1, 'a', 100, 3*time.Hour, false)
	fresh := putEntry(t, dir, 2, 'd', 100, 90*time.Minute, false)
	c, err := ScanGoCache(dir)
	require.NoError(t, err)
	require.Equal(t, TrimResult{Removed: 1, Freed: 100}, c.TrimTo(0, false))
	require.NoFileExists(t, old)
	require.FileExists(t, fresh)
}

// Một build dùng mục SAU lúc quét (Go đặt mtime về hiện tại): Lstat lại ngay trước khi xoá
// thấy mtime mới hơn và bỏ qua mục đó (`dew clean --apply` chạy GC, đo kho giữa lúc quét và lúc xoá).
func TestTrimTo_SkipsAnEntryUsedAfterTheScan(t *testing.T) {
	dir := fakeCache(t)
	used := putEntry(t, dir, 1, 'a', 100, 5*day, false)
	other := putEntry(t, dir, 2, 'a', 100, 4*day, false)
	kept := putEntry(t, dir, 3, 'a', 100, 3*day, false)
	c, err := ScanGoCache(dir)
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, os.Chtimes(used, now, now))
	require.Equal(t, TrimResult{Removed: 2, Freed: 200}, c.TrimTo(100, false))
	require.FileExists(t, used)
	require.NoFileExists(t, other)
	require.NoFileExists(t, kept, "the next oldest goes instead")
}

// Hạn 5 giây là trần cứng: tiến trình con để lại một tiến trình giữ stdout thì vẫn trả về sau
// tối đa WaitDelay, không chờ tới khi tiến trình đó thoát.
func TestOutput_ReturnsEvenIfAChildKeepsStdoutOpen(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	start := time.Now()
	out, _ := Output(t.Context(), "sh", "-c", "sleep 20 & echo started")
	require.Less(t, time.Since(start), 5*time.Second)
	require.Equal(t, "started\n", out)
}

// Chỉ nhận thư mục có README mà dòng đầu đúng chuỗi của Go; không thì không quét, không xoá.
func TestScanGoCache_RefusesADirWithoutGosREADME(t *testing.T) {
	dir := t.TempDir()
	entry := putEntry(t, dir, 1, 'a', 10, 30*day, false)
	_, err := ScanGoCache(dir)
	require.Error(t, err, "no README")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "README"), []byte("my notes\n"+goCacheREADME+"\n"), 0o644))
	_, err = ScanGoCache(dir)
	require.ErrorIs(t, err, ErrNotGoCache)
	require.FileExists(t, entry)
}

// Hồ sơ tuổi theo lần dùng cuối: <1 ngày, 1-3, 3-7, >7.
func TestScanGoCache_AgeProfile(t *testing.T) {
	dir := fakeCache(t)
	putEntry(t, dir, 1, 'a', 1, 12*time.Hour, false)
	putEntry(t, dir, 2, 'a', 20, 2*day, false)
	putEntry(t, dir, 3, 'd', 300, 5*day, true)
	putEntry(t, dir, 4, 'd', 4000, 10*day, false)
	c, err := ScanGoCache(dir)
	require.NoError(t, err)
	require.Equal(t, [4]int64{1, 20, 300, 4000}, c.Age)
}

// Mục không xoá được (thư mục con chỉ đọc, như tệp do Docker chạy root để lại) được đếm,
// không làm hỏng lượt trim; các mục khác vẫn bị xoá.
func TestTrimTo_CountsEntriesItCannotRemove(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs POSIX permissions and a non-root user")
	}
	dir := fakeCache(t)
	stuck := putEntry(t, dir, 1, 'a', 100, 9*day, false)
	sub := filepath.Dir(stuck)
	require.NoError(t, os.Chmod(sub, 0o555))
	t.Cleanup(func() { _ = os.Chmod(sub, 0o755) })
	var others []string
	for n := 2; len(others) < 2; n++ {
		p := putEntry(t, dir, n, 'd', 100, 8*day, false)
		if filepath.Dir(p) == sub {
			require.NoError(t, os.Remove(p))
			continue
		}
		others = append(others, p)
	}
	c, err := ScanGoCache(dir)
	require.NoError(t, err)
	r := c.TrimTo(0, false)
	require.Equal(t, 1, r.Failed)
	require.Equal(t, 2, r.Removed)
	require.Equal(t, int64(200), r.Freed)
	require.FileExists(t, stuck)
	for _, p := range others {
		require.NoFileExists(t, p)
	}
}

// Đo thời gian quét + trim trên cache giả ~50 nghìn mục (cache thật đo 21/10: khoảng 200 nghìn).
func TestTrimTo_TimingOnFiftyThousandEntries(t *testing.T) {
	if testing.Short() {
		t.Skip("creates 50k files")
	}
	dir := fakeCache(t)
	const n = 50_000
	old := time.Now().Add(-3 * day)
	for i := range 256 {
		require.NoError(t, os.Mkdir(filepath.Join(dir, hex.EncodeToString([]byte{byte(i)})), 0o755))
	}
	for i := range n {
		sum := sha256.Sum256([]byte(strconv.Itoa(i)))
		name := hex.EncodeToString(sum[:]) + "-a"
		p := filepath.Join(dir, name[:2], name)
		require.NoError(t, os.WriteFile(p, []byte("x"), 0o644))
		when := old.Add(time.Duration(i) * time.Second)
		require.NoError(t, os.Chtimes(p, when, when))
	}
	start := time.Now()
	c, err := ScanGoCache(dir)
	require.NoError(t, err)
	scanned := time.Since(start)
	require.Equal(t, n, c.Entries)
	start = time.Now()
	r := c.TrimTo(n/2, false)
	trimmed := time.Since(start)
	require.Equal(t, n/2, r.Removed)
	t.Logf("TIMING %d entries: scan %v, trim of %d entries %v", n, scanned, r.Removed, trimmed)
}

func TestGoCacheDir(t *testing.T) {
	calls := 0
	run := func(_ context.Context, name string, args ...string) (string, error) {
		calls++
		require.Equal(t, "go env GOCACHE", name+" "+strings.Join(args, " "))
		return "/from/go/env\n", nil
	}
	t.Setenv("GOCACHE", "/from/env")
	require.Equal(t, "/from/env", GoCacheDir(t.Context(), run))
	require.Zero(t, calls, "the variable wins without running go")
	t.Setenv("GOCACHE", "off")
	require.Empty(t, GoCacheDir(t.Context(), run))
	t.Setenv("GOCACHE", "")
	require.Equal(t, "/from/go/env", GoCacheDir(t.Context(), run))
	missing := func(context.Context, string, ...string) (string, error) {
		return "", &exec.Error{Name: "go", Err: exec.ErrNotFound}
	}
	require.Empty(t, GoCacheDir(t.Context(), missing), "no go: skip")
}

// Kho mục ASK/NEVER chỉ chạy lệnh đọc; công cụ không có thì bỏ dòng, lỗi khác thì ghi lỗi.
func TestInventory_MeasuresOnlyWithReadOnlyCommands(t *testing.T) {
	home := t.TempDir()
	write := func(p string, size int) {
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(strings.Repeat("x", size)), 0o644))
	}
	write(filepath.Join(home, ".npm", "_cacache", "a"), 10)
	pnpm := filepath.Join(t.TempDir(), "store")
	write(filepath.Join(pnpm, "v3", "f"), 20)
	mod := filepath.Join(t.TempDir(), "mod")
	write(filepath.Join(mod, "cache", "download", "z"), 30)

	var calls []string
	run := func(_ context.Context, name string, args ...string) (string, error) {
		cmd := name + " " + strings.Join(args, " ")
		calls = append(calls, cmd)
		switch cmd {
		case "pnpm store path":
			return pnpm + "\n", nil
		case "pip cache dir":
			return "", &exec.Error{Name: "pip", Err: exec.ErrNotFound}
		case "go env GOMODCACHE":
			return mod + "\n", nil
		case "docker system df --format {{.Type}}\t{{.Size}}\t{{.Reclaimable}}":
			return "Images\t5GB\t2.5GB (50%)\nContainers\t10MB\t0B (0%)\nLocal Volumes\t1.2GB\t0B (0%)\nBuild Cache\t3GB\t3GB\n", nil
		}
		return "", errors.New("unexpected command " + cmd)
	}
	items := Inventory(t.Context(), home, run)
	byName := map[string]Item{}
	for _, it := range items {
		byName[it.Name] = it
	}
	require.Equal(t, Item{
		Class: Ask, Name: "npm cache", Path: filepath.Join(home, ".npm"), Bytes: 10,
		Effect: byName["npm cache"].Effect, Command: "npm cache clean --force",
	}, byName["npm cache"])
	require.Equal(t, int64(20), byName["pnpm store"].Bytes)
	require.Equal(t, "pnpm store prune", byName["pnpm store"].Command)
	require.NotContains(t, byName, "pip cache", "no pip: no row")
	mc := byName["Go module cache"]
	require.Equal(t, Never, mc.Class)
	require.Equal(t, int64(30), mc.Bytes)
	require.Empty(t, mc.Command)
	require.Contains(t, mc.Effect, "offline")
	require.Equal(t, Item{
		Class: Ask, Name: "Docker images", Bytes: 2_500_000_000,
		Effect: byName["Docker images"].Effect, Command: "docker image prune -a",
	}, byName["Docker images"])
	require.Equal(t, int64(3_000_000_000), byName["Docker build cache"].Bytes)
	require.Equal(t, "docker builder prune", byName["Docker build cache"].Command)
	vol := byName["Docker volumes"]
	require.Equal(t, Never, vol.Class)
	require.Equal(t, int64(1_200_000_000), vol.Bytes)
	require.Empty(t, vol.Command)
	for _, it := range items {
		require.NotEmpty(t, it.Effect, it.Name)
	}
	require.ElementsMatch(t, []string{
		"pnpm store path", "pip cache dir", "go env GOMODCACHE",
		"docker system df --format {{.Type}}\t{{.Size}}\t{{.Reclaimable}}",
	}, calls, "read-only commands only")

	broken := func(_ context.Context, name string, args ...string) (string, error) {
		if name == "docker" {
			return "", errors.New("Cannot connect to the Docker daemon")
		}
		return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
	}
	items = Inventory(t.Context(), t.TempDir(), broken)
	require.Len(t, items, 1)
	require.Equal(t, "Docker", items[0].Name)
	require.Equal(t, int64(-1), items[0].Bytes)
	require.Contains(t, items[0].Error, "Cannot connect")
}

// Thứ người dùng không xoá được: mục của uid khác (mục đầu theo thứ tự duyệt và chủ của nó), và
// thư mục của chính mình không có quyền ghi (module cache Go là 0555). Lời gợi ý nêu cách xử lý.
func TestFindStuck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no file owners on Windows")
	}
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "target", "debug"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "target", "debug", "app"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "z.txt"), []byte("x"), 0o644))
	require.Empty(t, FindStuck(dir, os.Getuid()).Hint(dir), "all mine and writable")

	s := FindStuck(dir, os.Getuid()+1)
	require.Equal(t, Stuck{Foreign: 4, FirstForeign: filepath.Join(dir, "target"), Owner: os.Getuid()}, s,
		"target, target/debug, target/debug/app, z.txt")
	s.Owner = 0
	for _, want := range []string{
		"4 file(s)", "uid 0", s.FirstForeign, `sudo chown -R "$(id -u):$(id -g)" ` + dir,
		`docker run --user "$(id -u):$(id -g)"`, "dew routine gc",
	} {
		require.Contains(t, s.Hint(dir), want)
	}

	mod := filepath.Join(dir, "gopath", "pkg", "mod", "x@v1")
	require.NoError(t, os.MkdirAll(mod, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(mod, "go.mod"), []byte("module x"), 0o444))
	require.NoError(t, os.Chmod(mod, 0o555))
	t.Cleanup(func() { _ = os.Chmod(mod, 0o755) })
	s = FindStuck(dir, os.Getuid())
	require.Equal(t, Stuck{ReadOnly: 1, FirstReadOnly: mod}, s, "a read-only file in a writable dir can be deleted")
	for _, want := range []string{"1 read-only dir", mod, "chmod -R u+w " + dir, "dew routine gc"} {
		require.Contains(t, s.Hint(dir), want)
	}
}

func TestFreeBytes(t *testing.T) {
	free, err := FreeBytes(t.TempDir())
	if errors.Is(err, errors.ErrUnsupported) {
		t.Skip("no statfs on " + runtime.GOOS)
	}
	require.NoError(t, err)
	require.Positive(t, free)
}
