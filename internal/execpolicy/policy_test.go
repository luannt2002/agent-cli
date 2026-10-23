package execpolicy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

const userSpec = "zai:glm-5.3 zai:glm-5.3-flash main:qwen3.8 " +
	"claude_code:claude-opus-5-5,claude_code:claude-sonnet-5-5,claude_code:claude-haiku-5-5 codex:gpt-6.1-sol"

// Mặc định (không đặt gì): model chính không giới hạn như trước, nhưng giao
// việc sang CLI agent khác provider bị tắt dù CLI có trên máy và có key.
func TestDefaultPolicyDeniesDelegation(t *testing.T) {
	t.Parallel()
	p, err := FromEnv(env(map[string]string{"ANTHROPIC_API_KEY": "x", "OPENAI_API_KEY": "y"}))
	require.NoError(t, err)
	require.False(t, p.Restricted())
	require.True(t, p.AllowsModel("zai", "glm-9"))
	require.Nil(t, p.DelegationModels(BackendClaudeCode))
	require.Nil(t, p.DelegationModels(BackendCodex))
	var d *Denied
	require.ErrorAs(t, p.CheckDelegation(BackendClaudeCode, "claude-opus-5-5"), &d)
	require.Equal(t, CodeCrossProviderNotAllowed, d.Code)
}

// Danh sách của người dùng: model chính chỉ GLM 5.3/5.3-flash và qwen3.8;
// model mới, model cũ, provider khác đều bị chặn (allowlist, không denylist).
func TestAllowlistIsAuthoritativeForMainModels(t *testing.T) {
	t.Parallel()
	p, err := FromEnv(env(map[string]string{AllowedModelsEnv: userSpec}))
	require.NoError(t, err)
	require.True(t, p.Restricted())
	require.True(t, p.AllowsModel("zai", "glm-5.3-flash"))
	require.False(t, p.AllowsModel("ZAI", "glm-5.3"), "provider IDs are exact, like the config")
	require.False(t, p.AllowsModel("claude_code", "claude-opus-5-5"), "a main provider named like a backend borrows nothing")
	require.True(t, p.AllowsModel("main", "qwen3.8"))
	for _, c := range [][2]string{{"zai", "glm-5.4"}, {"zai", "glm-4.6"}, {"main", "qwen3.9"}, {"openai", "gpt-6.1-sol"}, {"anthropic", "claude-opus-5-5"}} {
		require.Falsef(t, p.AllowsModel(c[0], c[1]), "%s:%s", c[0], c[1])
		var d *Denied
		require.ErrorAs(t, p.CheckModel(c[0], c[1]), &d)
		require.Equal(t, CodeModelNotAllowed, d.Code)
		require.Contains(t, d.Allowed, "main:qwen3.8")
	}
}

// Bật giao việc: chỉ model liệt kê của từng backend; Fable, Astra, model cũ,
// model lạ đều bị từ chối; backend không liệt kê thì không có.
func TestDelegationModelAllowlist(t *testing.T) {
	t.Parallel()
	p, err := FromEnv(env(map[string]string{AllowedModelsEnv: userSpec, DelegationEnv: "on"}))
	require.NoError(t, err)
	require.Equal(t, []string{"claude-opus-5-5", "claude-sonnet-5-5", "claude-haiku-5-5"}, p.DelegationModels(BackendClaudeCode))
	require.Equal(t, []string{"gpt-6.1-sol"}, p.DelegationModels(BackendCodex))
	require.NoError(t, p.CheckDelegation(BackendClaudeCode, "claude-haiku-5-5"))
	for _, c := range [][2]string{
		{BackendClaudeCode, "claude-fable-5-1"},
		{BackendClaudeCode, "claude-opus-4-1"},
		{BackendClaudeCode, "claude-opus-6-0"},
		{BackendCodex, "gpt-6-astra"},
		{BackendCodex, "gpt-6-luna"},
	} {
		var d *Denied
		require.ErrorAsf(t, p.CheckDelegation(c[0], c[1]), &d, "%s:%s", c[0], c[1])
		require.Equal(t, CodeModelNotAllowed, d.Code)
	}

	only, err := FromEnv(env(map[string]string{AllowedModelsEnv: "zai:glm-5.3", DelegationEnv: "on"}))
	require.NoError(t, err)
	require.Nil(t, only.DelegationModels(BackendCodex), "a backend with no listed model stays off")
	var d *Denied
	require.ErrorAs(t, only.CheckDelegation(BackendCodex, "gpt-6.1-sol"), &d)
	require.Equal(t, CodeExternalAgentNotAllowed, d.Code)
}

// Alias của CLI (tự trỏ sang model mới nhất) không được làm ranh giới; giá
// trị sai cú pháp chặn lúc khởi động.
func TestParseRejectsAliasesAndGarbage(t *testing.T) {
	t.Parallel()
	for _, spec := range []string{
		"claude_code:opus", "claude_code:fable", "codex:latest", "glm-5.3", "zai:", ":glm",
		"claude_code:sonnet[1m]", "claude_code:claude-3-5-haiku-latest", "codex:gpt-6-latest",
	} {
		_, err := Parse(spec)
		require.Errorf(t, err, "%q", spec)
	}
	_, err := FromEnv(env(map[string]string{DelegationEnv: "maybe"}))
	require.Error(t, err)
	var d *Denied
	require.False(t, errors.As(err, &d))
}

// Lệnh bash gọi thẳng CLI agent bị nhận ra, kể cả qua đường dẫn, launcher và
// npx; lệnh thường thì không.
func TestExternalAgentCommand(t *testing.T) {
	t.Parallel()
	hits := [][]string{
		{"claude", "-p", "hi"},
		{"/home/u/.local/bin/claude", "--dangerously-skip-permissions"},
		{"codex", "exec", "x"},
		{"env", "FOO=1", "-u", "BAR", "codex", "exec"},
		{"timeout", "30", "claude", "-p", "x"},
		{"nohup", "gemini", "-p", "x"},
		{"sudo", "-E", "opencode", "run"},
		{"sudo", "-u", "bob", "claude", "-p", "x"},
		{"nice", "-n", "5", "codex", "exec"},
		{"timeout", "-k", "5", "60", "claude"},
		{"npx", "-y", "@anthropic-ai/claude-code", "-p", "x"},
		{"bunx", "@openai/codex", "exec"},
		{"npx", "claude", "-p", "x"},
		{"npm", "exec", "--", "claude", "-p", "x"},
		{"pnpm", "dlx", "codex"},
		{"/home/u/.nvm/versions/node/bin/claude.exe", "-p"},
		{"./node_modules/.bin/claude-code"},
		{"env", "-S", "claude -p hi"},
		{"doas", "-u", "bob", "claude"},
		{"env", "doas", "claude"},
		{"xargs", "-n", "1", "codex"},
		{"find", ".", "-exec", "claude", "-p", "{}", ";"},
		{"bash", "-c", "cd x && claude -p hi"},
		{"sh", "-c", "A=1 codex exec x"},
		{"dew", "run", "--yolo", "x"},
		{"agent-cli", "run", "--yolo", "x"},
		{"crush", "server", "--host", "unix:///tmp/s"},
		{"dew-tmux", "new", "x", "--model", "a/b"},
		{"/home/u/.local/bin/dew-tmux", "/w/p"},
		{"./scripts/dew-tmux.sh", "ls"},
		{"agent-cli-tmux", "rm", "x"},
	}
	for _, a := range hits {
		_, ok := ExternalAgentCommand(a)
		require.Truef(t, ok, "%q", a)
	}
	misses := [][]string{
		{"git", "status"},
		{"go", "test", "./..."},
		{"dew", "jobs"},
		{"agent-cli", "jobs"},
		{"crush", "--version"},
		{"goose", "-dir", "migrations", "postgres", "up"},
		{"dew", "ps", "--all"},
		{"bash", "-c", "echo claude"},
		{"find", ".", "-name", "*.go"},
		{"npm", "exec", "prettier"},
		{"npx", "prettier", "--write", "."},
		{"echo", "claude"},
		{"timeout", "5", "make", "test"},
		{"env", "-u", "claude", "ls"},
		{"sudo", "-u", "codex", "id"},
	}
	for _, a := range misses {
		_, ok := ExternalAgentCommand(a)
		require.Falsef(t, ok, "%q", a)
	}
}

// Chính sách đọc từ tệp env của dew (chỉ hai khoá, bỏ qua dòng khác kể
// cả khoá API) và từ env đã export (thắng tệp).
func TestLoadReadsEnvFileAndProcessEnv(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	envFile := filepath.Join(dir, "env")
	require.NoError(t, os.WriteFile(envFile, []byte(`# comment
DEW_API_KEY='secret-must-not-matter'
export DEW_ALLOWED_MODELS="zai:glm-5.3 claude_code:claude-opus-5-5"  # trailing comment
DEW_DELEGATION=on   # bật giao việc
`), 0o600))

	p, err := load(env(map[string]string{"DEW_ENV_FILE": envFile}))
	require.NoError(t, err)
	require.True(t, p.AllowsModel("zai", "glm-5.3"))
	require.False(t, p.AllowsModel("zai", "glm-4.6"))
	require.Equal(t, []string{"claude-opus-5-5"}, p.DelegationModels(BackendClaudeCode))

	// Env đã export thắng tệp.
	p, err = load(env(map[string]string{"DEW_ENV_FILE": envFile, DelegationEnv: "off", AllowedModelsEnv: "zai:glm-4.6"}))
	require.NoError(t, err)
	require.True(t, p.AllowsModel("zai", "glm-4.6"))
	require.Nil(t, p.DelegationModels(BackendClaudeCode))

	// Không có tệp: mặc định (không giới hạn model chính, giao việc tắt).
	p, err = load(env(map[string]string{"HOME": dir}))
	require.NoError(t, err)
	require.False(t, p.Restricted())
	require.Nil(t, p.DelegationModels(BackendCodex))

	file, err := readEnvFile(envFile)
	require.NoError(t, err)
	require.Len(t, file, 2, "only the two policy keys are kept")
}

// Biến tin cậy mang tên cũ AGENT_CLI_* (trước khi đổi tên dew) không được bỏ
// qua lặng lẽ: env gốc hay tệp env còn tên cũ thì báo lỗi nêu cả hai tên.
func TestLegacyTrustedNamesRefused(t *testing.T) {
	t.Parallel()
	require.NoError(t, legacyTrusted(env(map[string]string{AllowedModelsEnv: "zai:glm-5.3", "AGENT_CLI_API_KEY": "x"})))
	for _, k := range []string{"AGENT_CLI_ALLOWED_MODELS", "AGENT_CLI_CLAUDE_PERMISSION_MODE", "AGENT_CLI_CODEX_SANDBOX", "AGENT_CLI_ENV_FILE"} {
		err := legacyTrusted(env(map[string]string{k: "v"}))
		require.ErrorContains(t, err, k)
		require.ErrorContains(t, err, "DEW_"+strings.TrimPrefix(k, "AGENT_CLI_"))
	}

	envFile := filepath.Join(t.TempDir(), "env")
	require.NoError(t, os.WriteFile(envFile, []byte("# AGENT_CLI_DELEGATION=on (comment: ignored)\nexport AGENT_CLI_ALLOWED_MODELS=zai:glm-5.3\n"), 0o600))
	_, err := readEnvFile(envFile)
	require.ErrorContains(t, err, envFile+":2: AGENT_CLI_ALLOWED_MODELS is the old name of DEW_ALLOWED_MODELS")
}

// Không đặt DEW_ENV_FILE mà tệp env tên cũ ~/.config/agent-cli/env còn đó (máy chưa
// chuyển, hoặc trình cài thấy cả hai thư mục nên không gộp): allowlist trong đó sẽ bị bỏ
// lặng lẽ, nên dừng và nêu đường dẫn. Chọn DEW_ENV_FILE tường minh thì không xét.
func TestLoadRefusesLeftoverOldEnvFile(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	old := filepath.Join(home, ".config", "agent-cli", "env")
	require.NoError(t, os.MkdirAll(filepath.Dir(old), 0o700))
	require.NoError(t, os.WriteFile(old, []byte("DEW_ALLOWED_MODELS=zai:glm-5.3\n"), 0o600))

	_, err := load(env(map[string]string{"HOME": home}))
	require.ErrorContains(t, err, old)
	require.ErrorContains(t, err, filepath.Join(home, ".config", "dew", "env"))

	newFile := filepath.Join(t.TempDir(), "env")
	require.NoError(t, os.WriteFile(newFile, []byte("DEW_DELEGATION=on\n"), 0o600))
	_, err = load(env(map[string]string{"HOME": home, EnvFileEnv: newFile}))
	require.NoError(t, err)
}

// ~/.config/agent-cli là symlink tương thích trỏ tới ~/.config/dew: hai đường là MỘT tệp, không
// từ chối (lời nhắn "xoá tệp cũ" sẽ xoá chính tệp env mới). Hai tệp thật khác nhau thì vẫn từ chối,
// và lời nhắn chỉ lối thoát DEW_ENV_FILE và thư mục cũ, không bảo xoá tệp.
func TestLegacyEnvFileSymlinkToNewIsFine(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	cfg := filepath.Join(home, ".config")
	require.NoError(t, os.MkdirAll(filepath.Join(cfg, "dew"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(cfg, "dew", "env"), []byte("DEW_DELEGATION=on\n"), 0o600))
	require.NoError(t, os.Symlink("dew", filepath.Join(cfg, "agent-cli")))
	_, err := load(env(map[string]string{"HOME": home}))
	require.NoError(t, err)

	home2 := t.TempDir()
	for _, d := range []string{"dew", "agent-cli"} {
		require.NoError(t, os.MkdirAll(filepath.Join(home2, ".config", d), 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(home2, ".config", d, "env"), []byte("# "+d+"\n"), 0o600))
	}
	_, err = load(env(map[string]string{"HOME": home2}))
	require.ErrorContains(t, err, EnvFileEnv+"=")
	require.ErrorContains(t, err, filepath.Join(home2, ".config", "agent-cli")+" is not a symlink")
	require.NotContains(t, err.Error(), "delete the old one")
}

func TestShellValue(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		`"a b"`: "a b", `'a b' # c`: "a b", `on   # x`: "on", ` off `: "off", `"unterminated`: "unterminated",
	} {
		require.Equal(t, want, shellValue(in), in)
	}
}

// Tệp env viết bằng dạng shell khác (declare -x, readonly, KEY+=, nhiều lệnh
// một dòng) thì báo lỗi, không lặng lẽ bỏ ranh giới.
// Khoá không có tiền tố DEW_ không có tên cũ: dòng nào cũng "chứa" tên rỗng, nên kiểm
// tên cũ phải bỏ qua chứ không báo mọi dòng là tên cũ.
func TestReadEnvKeysWithoutTheDEWPrefix(t *testing.T) {
	t.Parallel()
	f := filepath.Join(t.TempDir(), "env")
	require.NoError(t, os.WriteFile(f, []byte("# comment\nOTHER=1\nROUTINE_SLOTS=4\n"), 0o600))
	got, err := readEnvKeys(f, "ROUTINE_SLOTS")
	require.NoError(t, err)
	require.Equal(t, "4", got["ROUTINE_SLOTS"])
}

func TestEnvFileUnusualSpellingsFailClosed(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for i, line := range []string{
		"declare -x DEW_ALLOWED_MODELS=zai:glm-5.3",
		"typeset -x DEW_ALLOWED_MODELS=zai:glm-5.3",
		"readonly DEW_DELEGATION=on",
		"X=1; DEW_ALLOWED_MODELS=zai:glm-5.3",
		"DEW_ALLOWED_MODELS+=zai:glm-5.3",
	} {
		f := filepath.Join(dir, fmt.Sprintf("env%d", i))
		require.NoError(t, os.WriteFile(f, []byte(line+"\n"), 0o600))
		_, err := readEnvFile(f)
		require.Errorf(t, err, "%q", line)
	}
	ok := filepath.Join(dir, "ok")
	require.NoError(t, os.WriteFile(ok, []byte("export\tDEW_DELEGATION=on\r\n# DEW_ALLOWED_MODELS=commented\n"), 0o600))
	got, err := readEnvFile(ok)
	require.NoError(t, err)
	require.Equal(t, "on", strings.TrimSpace(got[DelegationEnv]))
}

// Ảnh chụp env tin cậy: giá trị .env/cấu hình dự án đặt sau khi chụp không
// tới chính sách, không tới server con; biến khác giữ nguyên.
func TestChildEnvUsesTheSnapshot(t *testing.T) {
	// Không parallel: đổi ảnh chụp cấp gói.
	prev := snapshot
	t.Cleanup(func() { snapshot = prev })
	snapshot = map[string]string{AllowedModelsEnv: "zai:glm-5.3", "HOME": "/home/u"}

	env := ChildEnv([]string{
		"PATH=/usr/bin",
		AllowedModelsEnv + "=claude_code:claude-fable-5-1",
		DelegationEnv + "=on",
		EnvFileEnv + "=/repo/policy.env",
		"HOME=/repo",
	})
	require.ElementsMatch(t, []string{"PATH=/usr/bin", AllowedModelsEnv + "=zai:glm-5.3", "HOME=/home/u"}, env)
	require.Equal(t, "zai:glm-5.3", TrustedEnv(AllowedModelsEnv))
	require.Empty(t, TrustedEnv(DelegationEnv), "set after the snapshot: ignored")
}

// Env gốc cho tiến trình con: biến thêm sau ảnh chụp (`.env`) không có, biến bị đè
// (`env` của cấu hình dự án) về giá trị gốc, biến tiến trình đã gỡ không quay lại.
func TestOriginalEnvironDropsLaterChanges(t *testing.T) {
	// Không parallel: đổi ảnh chụp cấp gói và env tiến trình.
	prevSnap, prevOrig := snapshot, original
	t.Cleanup(func() { snapshot, original = prevSnap, prevOrig })
	snapshot = map[string]string{}
	original = map[string]string{"P02B_KEPT": "orig", "P02B_OVERRIDDEN": "orig", "P02B_REMOVED": "1"}
	t.Setenv("P02B_KEPT", "orig")
	t.Setenv("P02B_OVERRIDDEN", "changed")
	t.Setenv("P02B_ADDED", "dotenv")

	var got []string
	for _, kv := range OriginalEnviron() {
		if strings.HasPrefix(kv, "P02B_") {
			got = append(got, kv)
		}
	}
	require.ElementsMatch(t, []string{"P02B_KEPT=orig", "P02B_OVERRIDDEN=orig"}, got)
}

// Inputs là chính sách tin cậy client gửi server để so: đủ mọi biến tin cậy,
// allowlist/giao việc đã gộp tệp env như Current, và `.env` của repo (nạp sau
// ảnh chụp) không đổi được nó.
func TestInputsUseSnapshotAndEnvFile(t *testing.T) {
	// Không parallel: đổi ảnh chụp cấp gói và env tiến trình.
	prev := snapshot
	t.Cleanup(func() { snapshot = prev })
	dir := t.TempDir()
	envFile := filepath.Join(dir, "env")
	require.NoError(t, os.WriteFile(envFile, []byte("DEW_API_KEY='secret'\nDEW_ALLOWED_MODELS=zai:glm-5.3\nDEW_DELEGATION=on\n"), 0o600))
	snapshot = map[string]string{EnvFileEnv: envFile, "DEW_CODEX_SANDBOX": "read-only"}
	// `.env` của repo nạp sau ảnh chụp: không được chọn chính sách.
	t.Setenv(AllowedModelsEnv, "pp:m-pp")
	t.Setenv(DelegationEnv, "off")

	l, err := loadAll(TrustedEnv)
	require.NoError(t, err)
	require.Len(t, l.inputs, len(trustedKeys), "every trusted key, set or not")
	require.Contains(t, l.inputs, AllowedModelsEnv+"=zai:glm-5.3", "from the env file, not the repo's .env")
	require.Contains(t, l.inputs, DelegationEnv+"=on")
	require.Contains(t, l.inputs, "DEW_CODEX_SANDBOX=read-only")
	require.Contains(t, l.inputs, "HOME=", "unset keys are sent empty")
	require.NotContains(t, strings.Join(l.inputs, "\n"), "secret", "only trusted keys, never other env file lines")
	require.True(t, l.policy.AllowsModel("zai", "glm-5.3"))

	// Biến export trong env gốc thắng tệp, như Current.
	snapshot[AllowedModelsEnv] = "qwen:m-qwen"
	l, err = loadAll(TrustedEnv)
	require.NoError(t, err)
	require.Contains(t, l.inputs, AllowedModelsEnv+"=qwen:m-qwen")
}

// Trong test, Inputs kín như Current (mọi biến rỗng), để client và server
// cùng tiến trình test luôn khớp nhau.
func TestInputsHermeticInTests(t *testing.T) {
	t.Parallel()
	in, err := Inputs()
	require.NoError(t, err)
	require.Len(t, in, len(trustedKeys))
	for _, kv := range in {
		require.True(t, strings.HasSuffix(kv, "="), kv)
	}
	// Người gọi sửa slice trả về không được đổi chính sách đã nạp.
	in[0] = "TAMPERED=1"
	again, err := Inputs()
	require.NoError(t, err)
	require.NotContains(t, again, "TAMPERED=1")
}

func TestExternalAgentCommandInterpreterLiterals(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"node", "/tmp/codex.js", "exec", "hi"},
		{"node", "-e", `require('child_process').execFileSync('codex', ['exec', 'hi'])`},
		{"node", "--eval=require('child_process').spawnSync('/opt/bin/codex', ['exec'])"},
		{"python3.12", "-c", `import subprocess; subprocess.run(['codex', 'exec', 'hi'])`},
		{"python", "-c", `import os; os.system('claude -p hi')`},
		{"env", "X=1", "node", "-e", `require('child_process').execSync('codex exec hi')`},
		{"ruby", "-e", `system('claude', '-p', 'hi')`},
		{"perl", "-e", `exec 'codex', 'exec', 'hi'`},
		{"php", "-r", `shell_exec('codex exec hi');`},
		{"bash", "-lc", "codex exec hi"},
	} {
		name, blocked := ExternalAgentCommand(args)
		require.Truef(t, blocked, "must block %q, got %q", args, name)
	}
	for _, args := range [][]string{
		{"node", "-e", `console.log('codex')`},
		{"python3", "-c", `print('claude')`},
		{"node", "-e", `require('child_process').execSync('git status')`},
		{"python3", "-c", `import subprocess; subprocess.run(['go', 'test', './...'])`},
		{"node", "build.js"},
	} {
		_, blocked := ExternalAgentCommand(args)
		require.Falsef(t, blocked, "must allow %q", args)
	}
}

// Script của `bash -c`/`sh -c` được phân tích như shell thật (mvdan), nên dấu nháy lồng
// không làm lọt: trước đây tách bằng strings.Fields, `bash -c "bash -c \"claude -p hi\""`
// cho ra từ `"claude` và qua được. Không phân tích được thì chặn (fail closed).
//
// Vế ngược giữ hợp đồng cũ: chặn theo VỊ TRÍ LỆNH, không theo chữ. "claude" là đối số
// (`echo claude`, `git log --grep codex`) không chạy gì, chặn theo chữ sẽ chặn nhầm cả
// grep/log/commit message nhắc tên công cụ.
func TestExternalAgentCommandParsesQuotedNesting(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"bash", "-c", `bash -c "claude -p hi"`},
		{"bash", "-c", `bash -c 'codex exec x'`},
		{"sh", "-c", `bash -c "sh -c 'claude -p hi'"`},
		{"bash", "-c", `sh -c 'bash -c "codex exec x"'`},
		{"bash", "-c", "bash -c \"bash -c 'gemini -p x'\""},
		{"env", "FOO=1", "bash", "-c", `bash -c "claude -p hi"`},
		{"sh", "-c", `cd sub && "claude" -p hi`},
		{"bash", "-c", `'cl'"aude" -p hi`},
		{"bash", "-c", `x=$(claude -p hi); echo "$x"`},
		{"bash", "-c", "echo `codex exec x`"},
		{"bash", "-c", `f() { claude -p hi; }; f`},
		{"sh", "-c", `FOO=1 timeout 60 codex exec x`},
		// script không phân tích được: bộ tách theo chữ (bản 5337a9f7) vẫn soi
		{"bash", "-c", `claude -p 'unterminated`},
		// script lồng là từ động (`"$1"`): không biết nội dung thì chặn
		{"sh", "-c", `bash -c "$1"`, "_", "claude -p hi"},
		// literal code không phân tích được như shell vẫn được soi theo từ
		{"python3", "-c", `import os; os.system("claude -p 'it''s")`},
	} {
		name, ok := ExternalAgentCommand(args)
		require.Truef(t, ok, "%q must be blocked", args)
		require.NotEmptyf(t, name, "%q", args)
	}
	for _, args := range [][]string{
		{"bash", "-c", "echo claude"},
		{"bash", "-c", `echo "claude -p hi"`},
		{"bash", "-c", `bash -c "echo claude"`},
		{"sh", "-c", `git log --grep 'codex exec'`},
		{"bash", "-c", `go test ./... | tail -5`},
		{"sh", "-c", `cd "$DIR" && make test`},
		{"bash", "-c", `printf '%s\n' "it's claude's turn"`},
		// literal code không phải shell (nháy lẻ) không làm chặn nhầm
		{"python3", "-c", `import subprocess; subprocess.run(['make']); print("don't")`},
		// script không phân tích được mà bộ tách theo chữ không thấy agent: không chặn
		{"bash", "-c", `bash -c "echo 'unterminated"`},
		{"sh", "-c", "if then fi ((("},
	} {
		_, ok := ExternalAgentCommand(args)
		require.Falsef(t, ok, "%q must pass", args)
	}
}

// Soát độc lập 22/10 (KHONG-DAT): các ca dưới đây hoặc bị chặn ở 5337a9f7 rồi lọt sau
// khi đổi sang mvdan (literal bắt đầu bằng toán tử, here-doc, mảng), hoặc là lối lách
// rẻ (nháy kép có `\"`, `--`, `builtin exec`, `eval`). Chặn khi bộ tách theo chữ của
// 5337a9f7 HOẶC bộ phân tích mvdan thấy agent ở vị trí lệnh.
func TestExternalAgentCommandReviewFindings(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		// #1: literal code nối chuỗi, phần sau bắt đầu bằng toán tử shell
		{"python3", "-c", `import subprocess; subprocess.run("git diff" + " | claude -p review", shell=True)`},
		{"python3", "-c", `import os; os.system("make" + "; claude -p hi")`},
		{"node", "-e", `require("child_process").execSync("npm test" + " && codex exec fix")`},
		{"python3", "-c", `import os; os.system("A=1 claude -p '" + p + "'")`},
		// #2: here-doc vào shell, mảng
		{"sh", "-c", "bash <<EOF\nclaude -p hi\nEOF"},
		{"bash", "-c", "sh -c 'bash <<EOF\nclaude -p hi\nEOF'"},
		{"bash", "-c", `a=(claude -p hi); "${a[@]}"`},
		// #3: `\"` trong nháy kép được shell bỏ dấu `\`
		{"sh", "-c", `bash -c "\"claude\" -p hi"`},
		{"sh", "-c", `bash -c "cl\aude -p hi"`},
		// #6: tuỳ chọn của shell trước -c (có tuỳ chọn ăn giá trị), tuỳ chọn sau -c
		{"bash", "-o", "pipefail", "-c", "claude -p hi"},
		{"bash", "-c", "-x", "codex exec x"},
		// #7: `--` sau -c, builtin exec, eval với đối số tĩnh
		{"bash", "-c", "--", "claude -p hi"},
		{"builtin", "exec", "claude", "-p", "hi"},
		{"bash", "-c", "eval claude -p x"},
		{"bash", "-c", `eval "codex exec x"`},
		// soát lại 22/10 (A): $'…' không có dấu `\` là chuỗi tĩnh
		{"bash", "-c", "$'claude' -p hi"},
		// (B): tuỳ chọn dài lạ ăn giá trị làm shellScriptArg dừng sớm; quét kiểu 5337a9f7 vẫn bắt
		{"zsh", "--emulate", "sh", "-c", "claude -p hi"},
		{"ksh", "-R", "x", "-c", "claude -p hi"},
		{"bash", "./x.sh", "-c", "claude"}, // hợp đồng cũ: chặn
		// (C): eval có phần động vẫn được bộ tách theo chữ soi phần tĩnh
		{"bash", "-c", `eval claude "$X"`},
		// (E): `-` như `--`; cụm tuỳ chọn có o ăn giá trị ở arg kế
		{"bash", "-c", "-", "claude -p hi"},
		{"bash", "-co", "pipefail", "claude -p hi"},
	} {
		name, ok := ExternalAgentCommand(args)
		assert.Truef(t, ok, "%q must be blocked", args)
		assert.NotEmptyf(t, name, "%q", args)
	}
	for _, args := range [][]string{
		// #6: -c sau tên script mà chuỗi không có agent: không phân tích được thôi thì không chặn
		{"bash", "./x.sh", "-c", "it's"},
		// #6: zsh mà mvdan không phân tích được: rơi về bộ tách theo chữ, không chặn
		{"zsh", "-c", "print -l ${(f)x}"},
		{"bash", "-c", `echo "\"claude\" -p hi"`},
		// (C): eval của đầu ra lệnh khác (idiom thường gặp) không bị chặn chỉ vì động
		{"bash", "-lc", `eval "$(conda shell.bash hook)" && conda activate x && pytest`},
		{"bash", "-c", `eval "$(ssh-agent -s)" && ssh-add k`},
		{"sh", "-c", "eval $(opam env) && dune build"},
	} {
		_, ok := ExternalAgentCommand(args)
		assert.Falsef(t, ok, "%q must pass", args)
	}
}

// nested bọc inner trong n lớp wrapper; mỗi lớp là một mức lồng của
// externalAgentCommand, nên với `node --` thì n là đúng độ sâu mà inner được soi.
func nested(n int, wrapper []string, inner ...string) []string {
	args := inner
	for range n {
		args = append(slices.Clone(wrapper), args...)
	}
	return args
}

// Lồng quá sâu để soi thì CHẶN (fail closed), không cho qua: trước đây `depth > 4`
// trả "không phải agent", nên agent nằm ở lớp thứ 5 chạy được (audit 16/10 C-10).
// Tới 4 lớp thì giữ nguyên: agent bị bắt, lệnh thường qua.
func TestExternalAgentCommandFailsClosedWhenNestedTooDeep(t *testing.T) {
	t.Parallel()
	node := []string{"node", "--"}
	for depth := 0; depth <= 4; depth++ {
		_, ok := ExternalAgentCommand(nested(depth, node, "claude", "-p", "hi"))
		require.Truef(t, ok, "depth %d: agent caught", depth)
		_, ok = ExternalAgentCommand(nested(depth, node, "ls"))
		require.Falsef(t, ok, "depth %d: ordinary command passes", depth)
	}
	for _, args := range [][]string{
		nested(5, node, "claude", "-p", "hi"),
		nested(6, node, "codex", "exec", "x"),
		nested(5, node, "ls"),
		nested(6, node, "ls"),
		// bash -c ngoài cùng: bash là lớp 0, chuỗi -c tách thành lớp 1, bốn lớp node nữa tới lớp 5/6.
		{"bash", "-c", "node -- node -- node -- node -- claude -p hi"},
		{"bash", "-c", "node -- node -- node -- node -- node -- claude -p hi"},
		// find -exec soi lại mọi -exec ở lớp nông nên agent vẫn bị bắt; lệnh thường lồng 5 lớp thì nay bị chặn.
		nested(5, []string{"find", ".", "-exec"}, "ls"),
	} {
		name, ok := ExternalAgentCommand(args)
		require.Truef(t, ok, "%q: too deep to inspect must be blocked", args)
		require.NotEmpty(t, name)
	}
}
