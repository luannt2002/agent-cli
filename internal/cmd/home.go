package cmd

import (
	"cmp"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/crush/internal/client"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/gitutil"
	homedir "github.com/charmbracelet/crush/internal/home"
	"github.com/charmbracelet/crush/internal/projects"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/charmbracelet/crush/internal/server"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/home"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	// homeExitCode is the exit status of a workspace TUI that asks to go
	// back to Home (Alt+W). Home shows again; any other status ends
	// dew.
	homeExitCode = 10
	// homeChildEnv marks a workspace TUI started by Home: on Alt+W it exits
	// with homeExitCode instead of starting a new Home.
	homeChildEnv = "AGENT_CLI_HOME_CHILD"
	// launcherEnv holds the path of the dew launcher, which sets it.
	// Workspaces are started through it so each gets the launcher's
	// per-project --host.
	launcherEnv = "AGENT_CLI_LAUNCHER"
	// homeStatusProbes caps the per-session "waiting?" requests to one
	// server; a busy session past the cap shows as running.
	homeStatusProbes = 20
)

func stdioIsTerminal() bool {
	return term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd())
}

// shouldRunHome reports whether a root invocation shows Home: no argument,
// stdin and stdout on a terminal, and no flag but --debug. Any other flag
// (--cwd, --session, --continue, --yolo, --data-dir, --host...) targets a
// workspace directly. Subcommands never reach the root command.
func shouldRunHome(flags *pflag.FlagSet, args []string, tty bool) bool {
	if len(args) > 0 || !tty {
		return false
	}
	direct := false
	flags.Visit(func(f *pflag.Flag) { direct = direct || f.Name != "debug" })
	return !direct
}

// runHome shows Home and runs what the user picks, until they quit Home or
// a workspace exits with anything but homeExitCode.
func runHome(cmd *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	exe, err := homeLauncher()
	if err != nil {
		return err
	}
	debug, _ := cmd.Flags().GetBool("debug")
	sty := common.LoadThemeStyles("")
	code, err := homeLoop(
		func() (home.Choice, error) { return home.Run(cmd.Context(), homeSource{}, &sty, cwd) },
		func(c home.Choice) (int, error) { return runChild(homeChildCmd(exe, c, debug)) },
	)
	if err == nil && code != 0 {
		os.Exit(code) // The workspace TUI already reported why.
	}
	return err
}

// homeLoop picks with Home and runs each choice, showing Home again while the
// workspace exits with homeExitCode. It returns the status dew ends
// with.
func homeLoop(pick func() (home.Choice, error), run func(home.Choice) (int, error)) (int, error) {
	for {
		c, err := pick()
		if err != nil || c.Dir == "" {
			return 0, err
		}
		code, err := run(c)
		if err != nil || code != homeExitCode {
			return code, err
		}
	}
}

// homeLauncher is what Home starts workspaces with: the launcher when
// dew was started through it, else this binary.
func homeLauncher() (string, error) {
	// Env gốc: `.env` của repo (main nạp sau ảnh chụp) không được chọn chương trình Home chạy.
	if exe := execpolicy.OriginalEnv(launcherEnv); exe != "" {
		return exec.LookPath(exe)
	}
	return os.Executable()
}

// homeChildCmd runs the workspace TUI for c on this terminal.
func homeChildCmd(exe string, c home.Choice, debug bool) *exec.Cmd {
	args := []string{"--cwd", c.Dir}
	if c.SessionID != "" {
		args = append(args, "--session", c.SessionID)
	}
	if debug {
		args = append(args, "--debug")
	}
	// Not tied to a context: the TUI owns the terminal and its signals.
	cmd := exec.Command(exe, args...) //nolint:noctx
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = append(homeEnv(), homeChildEnv+"=1")
	return cmd
}

// homeEnv là env cho tiến trình dew mà Home dựng (workspace, Home khi quay
// lại): env GỐC, không biến nào của `.env` mà main đã nạp. Con chụp env này làm env
// gốc của nó (biến tin cậy, launcher, và env mà CLI con giao việc nhận); nó chạy cùng
// cwd nên tự nạp lại đúng `.env` đó.
func homeEnv() []string { return execpolicy.OriginalEnviron() }

// runChild runs c and returns its exit status; death by a signal counts as
// a failure.
func runChild(c *exec.Cmd) (int, error) {
	err := c.Run()
	if ee := (*exec.ExitError)(nil); errors.As(err, &ee) {
		return max(ee.ExitCode(), 1), nil
	}
	if err != nil {
		return 1, err
	}
	return 0, nil
}

// execHome replaces this process with Home, from dir (where dew was
// started; the TUI changed into its workspace).
func execHome(dir string) error {
	if dir != "" {
		_ = os.Chdir(dir) // First: a relative launcher path is relative to it.
	}
	exe, err := homeLauncher()
	if err != nil {
		return err
	}
	return syscall.Exec(exe, []string{exe}, homeEnv())
}

// workspaceKey is the 8 hex digits the launcher names a workspace's server
// socket after: agent-cli-<key>.sock, key = sha1 of the workspace root.
func workspaceKey(path string) string {
	sum := sha1.Sum([]byte(path))
	return hex.EncodeToString(sum[:])[:8]
}

// socketKey returns the workspace key of a launcher socket host, or "".
func socketKey(host string) string {
	name := filepath.Base(host)
	if !strings.HasPrefix(name, "agent-cli-") || !strings.HasSuffix(name, ".sock") {
		return ""
	}
	return strings.TrimSuffix(strings.TrimPrefix(name, "agent-cli-"), ".sock")
}

// homeSource is Home's view of this machine: the workspace registry and the
// per-project servers, found the way `ps` finds them. It never starts a
// server, and opens a workspace database only to list that one workspace.
type homeSource struct{}

var _ home.Source = homeSource{}

func (homeSource) Workspaces(ctx context.Context) ([]home.Workspace, error) {
	list, err := projects.List()
	if err != nil {
		return nil, err
	}
	var out []home.Workspace
	for _, p := range list {
		// Home bỏ chờ sau hạn của nó: dừng luôn ở đây thay vì Stat/đọc
		// branch tiếp (một mount treo có thể chặn từng lời gọi).
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !p.OnThisMachine() {
			continue
		}
		_, statErr := os.Stat(p.Path)
		w := toHomeWorkspace(p)
		w.Missing = errors.Is(statErr, fs.ErrNotExist)
		if !w.Missing {
			w.Branch = gitutil.CurrentBranch(p.Path)
		}
		out = append(out, w)
	}
	return out, nil
}

// toHomeWorkspace đọc mục sổ cho Home. Home mở workspace không kèm -D, nên
// data dir là cái TUI sẽ mở: .crush ở gốc dự án.
// shortcut: data dir tên khác .crush (options.data_directory) thì tin sổ, không
// đọc lại cấu hình; mục mà lần mở cuối dùng -D tên khác cũng rơi vào đây.
func toHomeWorkspace(p projects.Project) home.Workspace {
	dataDir := p.DataDir
	if dataDir == "" || filepath.Base(dataDir) == ".crush" {
		dataDir = filepath.Join(p.Path, ".crush")
	}
	return home.Workspace{
		Path: p.Path, DataDir: dataDir, LastAccessed: p.LastAccessed, Pinned: p.Pinned,
		Legacy: config.LegacyDataDirs(p.Path, p.Path, dataDir, p.DataDirs()),
	}
}

func (homeSource) Probe(ctx context.Context, paths []string) map[string]home.Live {
	return probeLive(ctx, discoverPSHosts(), paths)
}

func (homeSource) Sessions(ctx context.Context, ws home.Workspace) ([]home.Session, error) {
	return listHomeSessions(ctx, discoverPSHosts(), ws)
}

// Add registers an existing directory (its workspace root) and returns it.
// An already registered workspace is returned as is, keeping its data dir.
func (homeSource) Add(_ context.Context, dir string) (home.Workspace, error) {
	abs, err := filepath.Abs(homedir.Long(dir))
	if err != nil {
		return home.Workspace{}, err
	}
	if fi, err := os.Stat(abs); err != nil || !fi.IsDir() {
		return home.Workspace{}, fmt.Errorf("not an existing folder: %s", dir)
	}
	root, err := projects.NormalizeWorkspace(abs)
	if err != nil {
		return home.Workspace{}, err
	}
	list, err := projects.List()
	if err != nil {
		return home.Workspace{}, err
	}
	if i := slices.IndexFunc(list, func(p projects.Project) bool { return p.Path == root && p.OnThisMachine() }); i >= 0 {
		return toHomeWorkspace(list[i]), nil
	}
	// The TUI registers the data dir its config resolves once it starts.
	dataDir := filepath.Join(root, ".crush")
	if err := projects.Register(root, dataDir); err != nil {
		return home.Workspace{}, err
	}
	return home.Workspace{Path: root, DataDir: dataDir, LastAccessed: time.Now()}, nil
}

func (homeSource) Pin(path string, pinned bool) error { return projects.Pin(path, pinned) }

func (homeSource) Remove(path string) error { return projects.Remove(path) }

// stateRank orders what several sockets of one workspace can report.
var stateRank = map[home.ServerState]int{home.ServerOffline: 0, home.ServerUnknown: 1, home.ServerLive: 2}

// probeLive asks, concurrently, the server of each path (found by the socket
// name the launcher derives from the path) for its run counts. A path with
// no socket, or whose socket refuses connections, is offline; one that does
// not answer within ctx is unknown.
func probeLive(ctx context.Context, hosts, paths []string) map[string]home.Live {
	byKey := map[string][]string{}
	for _, h := range hosts {
		if k := socketKey(h); k != "" {
			byKey[k] = append(byKey[k], h)
		}
	}
	out := make(map[string]home.Live, len(paths))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, p := range paths {
		for _, h := range byKey[workspaceKey(p)] {
			wg.Go(func() {
				l := probeHost(ctx, h)
				mu.Lock()
				defer mu.Unlock()
				if stateRank[l.State] >= stateRank[out[p].State] {
					out[p] = l
				}
			})
		}
	}
	wg.Wait()
	return out
}

func probeHost(ctx context.Context, host string) home.Live {
	c, err := homeClient(ctx, host)
	if err != nil {
		// Nothing listening is a dead server; anything else may be a busy
		// one, so it is unknown, never idle.
		if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, fs.ErrNotExist) {
			return home.Live{}
		}
		return home.Live{State: home.ServerUnknown}
	}
	sessions, _, err := liveSessions(ctx, c)
	if err != nil {
		return home.Live{State: home.ServerUnknown}
	}
	l := home.Live{State: home.ServerLive}
	for _, s := range sessions {
		switch s.Status {
		case home.StatusRunning:
			l.Running++
		case home.StatusWaiting:
			l.Waiting++
		}
	}
	return l
}

// homeClient returns a client for host once it answered Health.
func homeClient(ctx context.Context, host string) (*client.Client, error) {
	u, err := server.ParseHostURL(host)
	if err != nil {
		return nil, err
	}
	c, err := client.NewClient("", u.Scheme, u.Host)
	if err != nil {
		return nil, err
	}
	return c, c.Health(ctx)
}

// liveSessions lists the top-level sessions of every workspace on a server,
// newest first, with their run state and the workspace path to open them in.
// Only busy sessions are asked whether they wait for an answer. It also
// returns the server's workspaces.
func liveSessions(ctx context.Context, src psSource) ([]home.Session, []proto.Workspace, error) {
	workspaces, err := src.ListWorkspaces(ctx)
	if err != nil {
		return nil, nil, err
	}
	var out []home.Session
	seen := map[string]bool{}
	probes := 0
	for _, ws := range workspaces {
		list, err := src.ListSessions(ctx, ws.ID)
		if err != nil {
			return nil, nil, err
		}
		for _, s := range list {
			if seen[s.ID] {
				continue
			}
			seen[s.ID] = true
			st := home.StatusIdle
			if s.IsBusy {
				st = home.StatusRunning
				if probes < homeStatusProbes {
					probes++
					if info, err := src.GetAgentSessionInfo(ctx, ws.ID, s.ID); err == nil && info.PendingPrompt {
						st = home.StatusWaiting
					}
				}
			}
			out = append(out, home.Session{ID: s.ID, Title: s.Title, UpdatedAt: time.Unix(s.UpdatedAt, 0), Status: st, Dir: ws.Path})
		}
	}
	slices.SortStableFunc(out, func(a, b home.Session) int { return b.UpdatedAt.Compare(a.UpdatedAt) })
	return out, workspaces, nil
}

// listHomeSessions lists ws's sessions, newest first and at most
// SessionLimit: through the live server when it holds ws's data dir open,
// else straight from the database. It never starts a server for that.
func listHomeSessions(ctx context.Context, hosts []string, ws home.Workspace) ([]home.Session, error) {
	var out []home.Session
	served := map[string]bool{} // Data dirs read through a live server.
	for _, h := range hosts {
		if socketKey(h) != workspaceKey(ws.Path) {
			continue
		}
		c, err := homeClient(ctx, h)
		if err != nil {
			continue
		}
		// shortcut: every workspace on a per-project server is taken as this
		// one; the launcher only routes this root there.
		list, wss, err := liveSessions(ctx, c)
		if err != nil {
			continue
		}
		out = append(out, list...)
		for _, w := range wss {
			served[w.DataDir] = true
		}
	}
	var dbErr error
	if dataDir := cmp.Or(ws.DataDir, filepath.Join(ws.Path, ".crush")); !served[dataDir] {
		var list []home.Session
		list, dbErr = dbSessions(ctx, filepath.Join(dataDir, "crush.db"), home.SessionLimit)
		out = append(out, list...)
	}
	// A session found twice (a copied database) is listed once: its newest
	// copy, the live one on a tie.
	slices.SortStableFunc(out, func(a, b home.Session) int { return b.UpdatedAt.Compare(a.UpdatedAt) })
	seen := map[string]bool{}
	out = slices.DeleteFunc(out, func(s home.Session) bool {
		dup := seen[s.ID]
		seen[s.ID] = true
		return dup
	})
	return out[:min(len(out), home.SessionLimit)], dbErr
}

// dbSessions reads the newest top-level sessions straight from a workspace
// database: read-only and unmigrated, so it selects only columns every
// schema version has. Statuses are unknown without a server.
func dbSessions(ctx context.Context, dbPath string, limit int) ([]home.Session, error) {
	if _, err := os.Stat(dbPath); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	conn, err := db.ConnectReadOnly(ctx, dbPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	rows, err := conn.QueryContext(ctx,
		`SELECT id, title, updated_at FROM sessions WHERE parent_session_id IS NULL ORDER BY updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []home.Session
	for rows.Next() {
		var s home.Session
		var updated int64
		if err := rows.Scan(&s.ID, &s.Title, &updated); err != nil {
			return nil, err
		}
		s.UpdatedAt = time.Unix(updated, 0)
		out = append(out, s)
	}
	return out, rows.Err()
}
