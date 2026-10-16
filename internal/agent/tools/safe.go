package tools

import (
	"runtime"
	"slices"
	"strings"
)

// safeCommands skip the permission prompt. Never list a command that runs
// another command (env, nice, nohup, time, timeout): the prefix match would
// auto-approve whatever it wraps. Nor one that can print the environment or
// config values (printenv, ps with the e modifier, git config): those hold
// API keys and remote URLs with tokens.
var safeCommands = []string{
	// Bash builtins and core utils
	"cal",
	"date",
	"df",
	"du",
	"echo",
	"free",
	"groups",
	"hostname",
	"id",
	"ls",
	"pwd",
	"set",
	"top",
	"type",
	"uname",
	"unset",
	"uptime",
	"whatis",
	"whereis",
	"which",
	"whoami",

	// Git
	"git blame",
	"git branch",
	"git describe",
	"git diff",
	"git grep",
	"git log",
	"git ls-files",
	"git ls-remote",
	"git remote",
	"git rev-parse",
	"git shortlog",
	"git show",
	"git status",
	"git tag",
}

var chainingMetacharacters = []string{
	";",
	"|",
	"&", // Also covers && and &>.
	"$(",
	"`",
	">",
	"<",
	"\n",
	"\r",
}

// containsCommandChaining reports whether s contains shell metacharacters
// that enable command chaining or substitution.
func containsCommandChaining(s string) bool {
	return slices.ContainsFunc(chainingMetacharacters, func(c string) bool {
		return strings.Contains(s, c)
	})
}

// argLeavesWorkspace reports whether an argument can point a read-only command
// outside the working directory or expand a secret: an absolute or ~ path, a
// ".." segment, or $VAR, also as the value of --flag=value. Such a command
// prompts, like the view/ls/glob/grep tools do for paths outside the project.
// shortcut: a relative path through a symlink that leaves the project is not
// seen here; upgrade by resolving the path with outsideWorkingDir.
func argLeavesWorkspace(arg string) bool {
	if strings.Contains(arg, "$") {
		return true
	}
	if strings.HasPrefix(arg, "-") {
		// Giá trị dính liền cờ ngắn (-f/etc/x) hay sau "=" đều có thể là đường dẫn.
		if strings.ContainsAny(arg, "/~") {
			return true
		}
		if _, value, ok := strings.Cut(arg, "="); ok {
			arg = value
		}
	}
	return strings.HasPrefix(arg, "/") || strings.HasPrefix(arg, "~") ||
		arg == ".." || strings.HasPrefix(arg, "../") || strings.HasSuffix(arg, "/..") || strings.Contains(arg, "/../")
}

// commandLeavesWorkspace applies argLeavesWorkspace the way the shell would see
// the words: quotes and backslashes are dropped first (the shell joins a dot,
// an empty quoted string and ./etc into ../etc, and turns a backslash-escaped
// slash into a plain one), and brace expansion
// ({/etc,}) is refused outright since it can build any path.
// shortcut: dropping quotes also splits "a b" into two words, which can only
// add a prompt. This is a deny-list on syntax, not proof that no safe command
// can reach outside; the real fix is an allow-list of flags per command.
func commandLeavesWorkspace(cmd string) bool {
	flat := strings.NewReplacer("'", "", `"`, "", `\`, "").Replace(cmd)
	return strings.ContainsAny(flat, "{}") || slices.ContainsFunc(strings.Fields(flat), argLeavesWorkspace)
}

// isSafeCommand reports whether cmd may skip the permission prompt: no
// chaining, no argument that leaves the workspace, and it starts with a
// safeCommands entry (whole word or word-flag).
func isSafeCommand(cmd string) bool {
	if containsCommandChaining(cmd) || commandLeavesWorkspace(cmd) {
		return false
	}
	lower := strings.ToLower(cmd)
	for _, safe := range safeCommands {
		if !strings.HasPrefix(lower, safe) {
			continue
		}
		// "git <verb>-<x>" is an alias or a git-<verb>-<x> program, not the
		// verb: after a git verb only a flag entry (git config --get) may go
		// on with "-".
		gitVerb := strings.HasPrefix(safe, "git ") && !strings.Contains(safe, " -")
		if n := len(safe); n != len(lower) && lower[n] != ' ' && (lower[n] != '-' || gitVerb) {
			continue
		}
		return !strings.HasPrefix(safe, "git ") ||
			isReadOnlyGit(safe, lower) && !slices.ContainsFunc(strings.Fields(cmd), gitArgRunsProgram)
	}
	return false
}

// gitProgramOptions name a program for git to run (pager, upload-pack) or
// let it run a configured diff/textconv driver; --no-index makes diff/grep
// compare files outside the repository. Top-level -c, --config-env,
// --exec-path and GIT_* env prefixes need no entry: they come before the
// verb, so the safeCommands prefix never matches them.
var gitProgramOptions = []string{"--open-files-in-pager", "--upload-pack", "--exec", "--ext-diff", "--textconv", "--no-index"}

// gitArgRunsProgram reports whether a git argument can make git run another
// program: grep's -O[<pager>], also bundled (-iO), or a gitProgramOptions
// entry or any prefix of one (git accepts unique abbreviations, e.g.
// --open=true). Fails closed: --text, a prefix of --textconv, prompts too.
func gitArgRunsProgram(arg string) bool {
	if !strings.HasPrefix(arg, "--") {
		return strings.HasPrefix(arg, "-") && strings.Contains(arg, "O")
	}
	name, _, _ := strings.Cut(strings.ToLower(arg), "=")
	return len(name) > 2 && slices.ContainsFunc(gitProgramOptions, func(o string) bool { return strings.HasPrefix(o, name) })
}

// gitReadOnlyArgs: for git verbs that also mutate, the only auto-approved
// argument lists (every arg must be listed; anything else prompts).
var gitReadOnlyArgs = map[string][]string{
	"git branch": {"-a", "-r", "--list", "-v"},
	"git tag":    {"-l", "--list"},
	"git remote": {"-v"},
}

// isReadOnlyGit vets a git command whose verb matched safe. It fails closed:
// "--output" writes a file (git rejects abbreviations of it), and quoting,
// expansion or globs could build that word, so they prompt too.
func isReadOnlyGit(safe, lower string) bool {
	if strings.ContainsAny(lower, `'"\${*?[`) || strings.Contains(lower, "--output") {
		return false
	}
	allowed, restricted := gitReadOnlyArgs[safe]
	if !restricted {
		return true
	}
	fields := strings.Fields(lower)
	if len(fields) < 2 || fields[0]+" "+fields[1] != safe {
		return false
	}
	args := fields[2:]
	if safe == "git remote" && len(args) > 0 && args[0] == "show" {
		return true
	}
	return !slices.ContainsFunc(args, func(a string) bool { return !slices.Contains(allowed, a) })
}

func init() {
	if runtime.GOOS == "windows" {
		safeCommands = append(
			safeCommands,
			// Windows-specific commands
			"ipconfig",
			"nslookup",
			"ping",
			"systeminfo",
			"tasklist",
			"where",
		)
	}
}
