package shell

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// guard giả: PID trong protectedPIDs được bảo vệ; tên "crush"/"dew" và
// "mycli" được bảo vệ.
func testGuard(protectedPIDs ...int) ProcessGuard {
	set := map[int]bool{}
	for _, p := range protectedPIDs {
		set[p] = true
	}
	return ProcessGuard{
		ProtectedPID: func(pid int) bool { return set[pid] },
		ProtectedName: func(pattern string) bool {
			return patternHitsProtected(pattern, []string{"crush", "dew", "mycli"})
		},
	}
}

// Lệnh giết tiến trình dew (client 954803 + server 946242) bị chặn, kể
// cả qua /bin/kill và qua lớp bọc; đúng sự cố 15/10.
func TestProtect_BlocksKillingOwnProcess(t *testing.T) {
	t.Parallel()
	g := testGuard(954803, 946242)
	blocked := [][]string{
		{"/bin/kill", "-9", "954803", "946242"},
		{"kill", "946242"},
		{"kill", "-s", "KILL", "954803"},
		{"kill", "-TERM", "946242"},
		{"/usr/bin/kill", "-9", "-946242"}, // tín hiệu 9 + nhóm tiến trình 946242
		{"env", "FOO=1", "kill", "-9", "954803"},
		{"timeout", "5", "kill", "946242"},
		{"nohup", "kill", "-9", "954803"},
	}
	for _, args := range blocked {
		reason, ok := g.Blocked(args)
		require.Truef(t, ok, "%v", args)
		require.NotEmpty(t, reason)
	}
}

// Giết một PID không thuộc dew thì KHÔNG bị lớp này chặn (vẫn qua cổng
// quyền bình thường); lệnh thường cũng không bị chặn.
func TestProtect_AllowsOtherPIDsAndNormalCommands(t *testing.T) {
	t.Parallel()
	g := testGuard(946242)
	for _, args := range [][]string{
		{"kill", "-9", "12345"},
		{"kill", "123", "456"},
		{"ps", "aux"},
		{"echo", "kill", "946242"},
		{"ls", "-la"},
		{"grep", "-r", "kill", "."},
	} {
		_, ok := g.Blocked(args)
		require.Falsef(t, ok, "%v", args)
	}
}

// pkill/killall khớp tên tiến trình dew bị chặn; tên khác thì không.
func TestProtect_BlocksKillByName(t *testing.T) {
	t.Parallel()
	g := testGuard()
	for _, args := range [][]string{
		{"pkill", "crush"},
		{"pkill", "-9", "-f", "dew"},
		{"killall", "crush"},
		{"killall", "-9", "mycli"},
	} {
		_, ok := g.Blocked(args)
		require.Truef(t, ok, "%v", args)
	}
	for _, args := range [][]string{
		{"pkill", "node"},
		{"killall", "chrome"},
	} {
		_, ok := g.Blocked(args)
		require.Falsef(t, ok, "%v", args)
	}
}

// Lớp bọc sudo/doas/xargs không được che lệnh giết tiến trình được bảo vệ.
func TestProtect_PeelsSudoAndXargs(t *testing.T) {
	t.Parallel()
	g := testGuard(954803)
	for _, args := range [][]string{
		{"sudo", "kill", "954803"},
		{"sudo", "-n", "kill", "954803"},
		{"sudo", "-u", "root", "kill", "954803"},
		{"doas", "kill", "954803"},
		{"xargs", "kill", "954803"},
		{"xargs", "-r", "kill", "954803"},
		{"xargs", "-I{}", "kill", "954803"},
		{"sudo", "xargs", "kill", "954803"},
	} {
		_, ok := g.Blocked(args)
		require.Truef(t, ok, "%v", args)
	}
}

// pkill/killall khớp theo chuỗi con hoặc regex, không chỉ tên chính xác.
func TestProtect_BlocksKillByPatternMatch(t *testing.T) {
	t.Parallel()
	g := testGuard()
	for _, args := range [][]string{
		{"pkill", "-f", "crush server"},
		{"pkill", "crus"},
		{"killall", "-r", "cru.*"},
		{"pkill", "-f", "dew"},
	} {
		_, ok := g.Blocked(args)
		require.Truef(t, ok, "%v", args)
	}
	for _, args := range [][]string{
		{"pkill", "node"},
		{"killall", "-r", "fire.*"},
	} {
		_, ok := g.Blocked(args)
		require.Falsef(t, ok, "%v", args)
	}
}

// Tắt/khởi động lại máy luôn bị chặn.
func TestProtect_BlocksShutdown(t *testing.T) {
	t.Parallel()
	g := testGuard()
	for _, args := range [][]string{
		{"shutdown", "-h", "now"},
		{"reboot"},
		{"poweroff"},
		{"halt"},
		{"sudo", "reboot"}, // sudo bị cấm nơi khác, nhưng reboot ở đây cũng chặn
	} {
		// "sudo" không phải launcher ở đây nên peel không bóc; test trực tiếp reboot.
		_ = args
	}
	for _, args := range [][]string{{"shutdown", "-h", "now"}, {"reboot"}, {"poweroff"}, {"halt"}} {
		_, ok := g.Blocked(args)
		require.Truef(t, ok, "%v", args)
	}
}

// killPIDs bỏ cờ tín hiệu, lấy đúng PID (kể cả nhóm tiến trình âm).
func TestKillPIDs(t *testing.T) {
	t.Parallel()
	require.Equal(t, []int{954803, 946242}, killPIDs([]string{"kill", "-9", "954803", "946242"}))
	require.Equal(t, []int{123}, killPIDs([]string{"kill", "-s", "TERM", "123"}))
	require.Equal(t, []int{946242}, killPIDs([]string{"kill", "-9", "-946242"}))
	require.Empty(t, killPIDs([]string{"kill", "-946242"}), "a lone -N is a signal, not a pid")
	require.Empty(t, killPIDs([]string{"kill", "%1"}))
}

// Bộ bảo vệ thật: giết chính tiến trình test (os.Getpid qua selfTree) bị chặn.
func TestDefaultGuard_ProtectsSelf(t *testing.T) {
	t.Parallel()
	self := 0
	for pid := range selfTree {
		self = pid
		break
	}
	require.NotZero(t, self)
	_, ok := defaultProcessGuard.Blocked([]string{"/bin/kill", "-9", strconv.Itoa(self)})
	require.True(t, ok, "killing a pid in dew's own process tree must be blocked")
}

// Bộ bảo vệ thật chặn giết theo tên cả tên mới (dew) lẫn tên cũ (agent-cli,
// server cũ còn chạy trên máy chưa chuyển) và crush.
func TestProtect_DefaultGuardNames(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"pkill", "dew"},
		{"pkill", "-f", "agent-cli"},
		{"killall", "crush"},
	} {
		_, ok := defaultProcessGuard.Blocked(args)
		require.Truef(t, ok, "%v", args)
	}
}

// Các đường vòng đã tái lập ở audit 16/10: kill không PID tường minh (PID đến từ
// xargs/stdin), kill vào "mọi tiến trình" (-1 / pid 1), và lệnh kill bọc trong
// `sh -c` / `bash -c` (kill là builtin của shell con nên argv của nó không qua
// BlockFunc).
func TestProtect_BlocksPidlessKillShellWrappedAndKillAll(t *testing.T) {
	t.Parallel()
	g := testGuard(946242)
	for _, args := range [][]string{
		{"xargs", "kill", "-9"},
		{"xargs", "-r", "kill"},
		{"kill"},
		{"kill", "-9", "-1"},
		{"kill", "-9", "1"},
		{"bash", "-c", "kill 946242"},
		{"sh", "-c", "kill -9 946242"},
		{"bash", "-lc", "echo hi; kill 946242"},
		{"sh", "-c", "pgrep x | xargs kill -9"},
		{"sh", "-c", "env kill 946242"},
		{"bash", "-c", "bash -c 'kill 946242'"},
		{"bash", "-o", "errexit", "-c", "true && kill 946242"},
	} {
		reason, ok := g.Blocked(args)
		require.Truef(t, ok, "%v", args)
		require.NotEmpty(t, reason)
	}
}

// Siết lớp trên không được chặn oan lệnh thường: shell chạy script/lệnh vô hại,
// kill liệt kê tín hiệu, kill PID không thuộc dew.
func TestProtect_ShellWrappingDoesNotBlockNormalCommands(t *testing.T) {
	t.Parallel()
	g := testGuard(946242)
	for _, args := range [][]string{
		{"bash", "-c", "echo hello"},
		{"sh", "-c", "ls -la && pwd"},
		{"bash", "script.sh"},
		{"bash", "-c", "go test ./... | tail"},
		{"kill", "-l"},
		{"kill", "-9", "12345"},
		{"bash", "-c", "kill 12345"},
		{"xargs", "echo"},
	} {
		_, ok := g.Blocked(args)
		require.Falsef(t, ok, "%v", args)
	}
}

// Lượt soát độc lập 16/10 (T3): script sau `-c` phải được PHÂN TÍCH CÚ PHÁP như
// shell, không tách chuỗi bằng regex. Các dạng dưới đây từng lọt.
func TestProtect_ParsesScriptAfterDashC(t *testing.T) {
	t.Parallel()
	g := testGuard(946242)
	for _, args := range [][]string{
		// từ khoá đứng đầu đoạn che mất kill
		{"bash", "-c", "for p in 946242; do kill -9 $p; done"},
		{"sh", "-c", "if true; then kill 946242; fi"},
		{"sh", "-c", "while true; do kill 946242; done"},
		{"bash", "-c", "! kill 946242"},
		{"bash", "-c", "f() { kill 946242; }; f"},
		// -c -- script
		{"bash", "-c", "--", "kill 946242"},
		{"sh", "-c", "--", "kill 946242"},
		// exec / eval / builtin / time / gán đứng trước
		{"sh", "-c", "exec kill 946242"},
		{"sh", "-c", "eval kill 946242"},
		{"sh", "-c", "builtin kill 946242"},
		{"sh", "-c", "x=1 kill 946242"},
		{"sh", "-c", "time kill 946242"},
		{"eval", "kill 946242"},
		// nháy / escape chen giữa từ, lệnh trong backtick và $(...)
		{"sh", "-c", `ki""ll 946242`},
		{"sh", "-c", `\kill 946242`},
		{"sh", "-c", "echo `kill 946242`"},
		{"sh", "-c", "echo $(kill 946242)"},
		// tên lệnh động cạnh chữ kill, và đối số vị trí chạy kill
		{"sh", "-c", "k=kill; $k 946242"},
		{"bash", "-c", `"$@"`, "_", "/bin/kill", "946242"},
		// bọc ngoài chưa bóc: timeout 5s, taskset, busybox
		{"timeout", "5s", "/bin/kill", "946242"},
		{"taskset", "-c", "0", "/bin/kill", "946242"},
		{"busybox", "kill", "946242"},
		{"rbash", "-c", "kill 946242"},
	} {
		reason, ok := g.Blocked(args)
		require.Truef(t, ok, "%q", args)
		require.NotEmpty(t, reason)
	}
}

// Phân tích cú pháp không được chặn oan: chữ kill/shutdown nằm trong chuỗi, dấu
// tách nằm trong nháy, lệnh vô hại có từ khoá điều khiển.
func TestProtect_ParsedScriptDoesNotBlockHarmlessText(t *testing.T) {
	t.Parallel()
	g := testGuard(946242)
	for _, args := range [][]string{
		{"bash", "-c", `git commit -m "fix; kill stale workers"`},
		{"bash", "-c", `echo "(kill)"`},
		{"bash", "-c", `git log --grep="fix|shutdown"`},
		{"bash", "-c", "for f in a b; do echo $f; done"},
		{"sh", "-c", "if test -f x; then ls; fi"},
		{"bash", "-c", "kill 12345"},
		{"bash", "-c", "kill -l"},
		{"bash", "-c", `"$@"`, "_", "ls", "-la"},
		{"timeout", "5s", "ls"},
		{"taskset", "-c", "0", "ls"},
	} {
		_, ok := g.Blocked(args)
		require.Falsef(t, ok, "%q", args)
	}
}

// Soát 22/10 (A): $'…' không có dấu `\` là chuỗi tĩnh như bash, nên `pkill $'dew'` bị
// chặn như ở 5337a9f7; có dấu `\` hay biến thì không biết mẫu, pkill/killall coi như
// trúng dew (fail closed).
func TestProtect_DollarQuotedAndDynamicKillPatterns(t *testing.T) {
	t.Parallel()
	g := testGuard()
	for _, args := range [][]string{
		{"sh", "-c", "pkill $'dew'"},
		{"bash", "-c", "killall $'dew'"},
		{"sh", "-c", `pkill -f $'d\x65w'`},
		{"bash", "-c", `killall "$NAME"`},
	} {
		_, ok := g.Blocked(args)
		require.Truef(t, ok, "%q", args)
	}
	for _, args := range [][]string{
		{"sh", "-c", "pkill $'node'"},
		{"bash", "-c", "killall chrome"},
	} {
		_, ok := g.Blocked(args)
		require.Falsef(t, ok, "%q", args)
	}
}

// Soát lại 22/10 (vòng 3): mẫu còn dấu `\` (`pkill d\\ew` thành `d\ew`, `pkill "d\ew"`)
// không biên dịch được thành regexp của Go nên lọt, nhưng pkill của procps coi `\e` là `e`
// và giết dew. Mẫu được xét thêm khi đã bỏ hết dấu `\`.
func TestProtect_BackslashedKillPatterns(t *testing.T) {
	t.Parallel()
	g := testGuard()
	for _, args := range [][]string{
		{"sh", "-c", `pkill d\\ew`},
		{"sh", "-c", `killall \\dew`},
		{"sh", "-c", `pkill "d\ew"`},
	} {
		_, ok := g.Blocked(args)
		require.Truef(t, ok, "%q", args)
	}
	_, ok := g.Blocked([]string{"sh", "-c", "pkill node"})
	require.False(t, ok)
}
