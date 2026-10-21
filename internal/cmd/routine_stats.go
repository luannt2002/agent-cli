package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/crush/internal/loop"
	"github.com/spf13/cobra"
)

var routineStatsCmd = &cobra.Command{
	Use:   "stats [NAME]",
	Short: "Show LLM request latency, errors and tokens per routine, from the run journals",
	Long: `Read the journals of the routine runs and print, per routine: runs, LLM requests,
p50/p95 time to the first streamed part (TTFT) and total request time, stalled
streams, dew retries, quota errors, other error classes, prompt/completion tokens,
and how many OTHER routine runs were running during each request (p50/max).

Percentiles are nearest-rank: p50 of 1..10 is 5. A request's time span is its
journal time (when it was sent) plus its total time; a run's span is its report's
started..finished. A run still running (or dead but not yet marked ABANDONED) is
read from the journal in its run directory and ends at that journal's last write,
so a live run that has not written for a while counts as shorter than it is.

Every streamed request of a routine turn is journaled as an "llm_request" event:
TTFT, total time, tokens (the prompt count includes cached tokens), status (ok,
cancelled, or the error class: stream_stall, quota, rate_limit, server_error,
network, timeout, other) and the dew retry index. A retry that fantasy makes by
itself inside a step is a separate request with the same retry index. Journals
written before v0.3.1, or with request_timeout 0, have no request data.`,
	Example: `
# Every routine, all runs
dew routine stats

# One routine, runs started in the last 24 hours, as JSON
dew routine stats nightly-lint --since 24h --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		since, _ := cmd.Flags().GetDuration("since")
		asJSON, _ := cmd.Flags().GetBool("json")
		name := ""
		if len(args) == 1 {
			name = args[0]
		}
		stats := routineStats(routineDataDir(), name, since, time.Now(), cmd.ErrOrStderr())
		if len(stats) == 0 && !asJSON {
			fmt.Fprintf(cmd.OutOrStdout(), "no routine runs in %s\n", routineDataDir())
			return nil
		}
		return printRoutineStats(cmd.OutOrStdout(), stats, asJSON)
	},
}

func init() {
	routineStatsCmd.Flags().Duration("since", 0, "Only runs started within this long, e.g. 24h (0 = every run)")
	routineStatsCmd.Flags().Bool("json", false, "Print a JSON array instead of a table")
	routineStatsCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return &usageError{err} })
	routineCmd.AddCommand(routineStatsCmd)
}

// noRequestData là lời cho routine có lượt chạy mà journal không có dòng llm_request nào.
const noRequestData = "no request data (before v0.3.1, or request_timeout 0)"

// routineStat là tổng hợp số đo request LLM của một routine.
type routineStat struct {
	Routine          string         `json:"routine"`
	Runs             int            `json:"runs"`
	Requests         int            `json:"requests"`
	NoData           string         `json:"no_data,omitempty"`
	TTFTP50MS        int64          `json:"ttft_p50_ms"`
	TTFTP95MS        int64          `json:"ttft_p95_ms"`
	TotalP50MS       int64          `json:"total_p50_ms"`
	TotalP95MS       int64          `json:"total_p95_ms"`
	Stalls           int            `json:"stalls"`
	Retries          int            `json:"retries"` // lần chạy lại của dew có ít nhất một request
	Quota            int            `json:"quota"`
	Statuses         map[string]int `json:"statuses,omitempty"`
	PromptTokens     int64          `json:"prompt_tokens"`
	CompletionTokens int64          `json:"completion_tokens"`
	ConcurrentP50    int            `json:"concurrent_p50"`
	ConcurrentMax    int            `json:"concurrent_max"`
}

// runSpan là khoảng thời gian một lượt chạy routine.
type runSpan struct {
	dir        string
	start, end time.Time
}

// routineStats đọc mọi report.json dưới dataDir (lượt chạy của name, hoặc mọi routine khi name
// rỗng, bắt đầu trong since nếu since > 0) và journal của chúng. warn nhận lời cảnh báo khi một
// journal đọc không được (nil = bỏ).
func routineStats(dataDir, name string, since time.Duration, now time.Time, warn io.Writer) []routineStat {
	if warn == nil {
		warn = io.Discard
	}
	reports, _ := filepath.Glob(filepath.Join(dataDir, "*", "runs", "*", "report.json"))
	var spans []runSpan
	type run struct {
		routine string
		span    runSpan
		journal string
	}
	var runs []run
	for _, p := range reports {
		dir := filepath.Dir(p)
		rep, err := readReport(dir)
		if err != nil || rep.Started.IsZero() {
			continue
		}
		journal := rep.Journal
		if journal == "" {
			// Báo cáo RUNNING chưa có trường journal (chỉ báo cáo cuối có): tìm như abandonRun.
			if js, _ := filepath.Glob(filepath.Join(dir, "*.jsonl")); len(js) > 0 {
				journal = js[len(js)-1]
			}
		}
		end := rep.Finished
		if end.IsZero() {
			// shortcut: lượt chưa đóng (đang chạy, hoặc chết mà chưa được ghi lại thành ABANDONED)
			// kết thúc ở lần ghi journal cuối, nên lượt đang chạy mà lâu chưa ghi bị tính ngắn hơn
			// thật; lượt chưa có journal coi như dài 0. Khi cần chính xác hơn, kiểm tiến trình sống
			// (như `ps`) và lấy giờ hiện tại cho lượt còn sống.
			end = rep.Started
			if fi, err := os.Stat(journal); err == nil {
				end = fi.ModTime()
			}
		}
		span := runSpan{dir: dir, start: rep.Started, end: end}
		spans = append(spans, span)
		routine := filepath.Base(filepath.Dir(filepath.Dir(dir)))
		if name != "" && routine != name || since > 0 && rep.Started.Before(now.Add(-since)) {
			continue
		}
		runs = append(runs, run{routine: routine, span: span, journal: journal})
	}

	type acc struct {
		stat              routineStat
		ttft, total, conc []int64
		retries           map[string]bool
	}
	byName := map[string]*acc{}
	for _, r := range runs {
		a := byName[r.routine]
		if a == nil {
			a = &acc{stat: routineStat{Routine: r.routine}, retries: map[string]bool{}}
			byName[r.routine] = a
		}
		a.stat.Runs++
		if r.journal == "" {
			continue // lượt hỏng trước khi mở journal (ERROR), hoặc RUNNING mà chưa có journal
		}
		events, err := loop.ReadJournal(r.journal)
		if err != nil {
			fmt.Fprintf(warn, "warning: %s: cannot read the journal: %v\n", r.span.dir, err)
			continue
		}
		for _, e := range events {
			if e.Kind != "llm_request" {
				continue
			}
			s := &a.stat
			s.Requests++
			if e.TTFTMS != nil {
				a.ttft = append(a.ttft, *e.TTFTMS)
			}
			a.total = append(a.total, e.TotalMS)
			if s.Statuses == nil {
				s.Statuses = map[string]int{}
			}
			s.Statuses[e.Status]++
			if e.Retry > 0 {
				a.retries[fmt.Sprintf("%s|%d|%d", r.span.dir, e.Attempt, e.Retry)] = true
			}
			s.PromptTokens += e.PromptTokens
			s.CompletionTokens += e.CompletionTokens
			end := e.Time.Add(time.Duration(e.TotalMS) * time.Millisecond)
			others := 0
			for _, o := range spans {
				if o.dir != r.span.dir && o.start.Before(end) && e.Time.Before(o.end) {
					others++
				}
			}
			a.conc = append(a.conc, int64(others))
		}
	}

	out := make([]routineStat, 0, len(byName))
	for _, n := range slices.Sorted(maps.Keys(byName)) {
		a := byName[n]
		s := a.stat
		if s.Requests == 0 {
			s.NoData = noRequestData
		}
		s.TTFTP50MS, s.TTFTP95MS = percentile(a.ttft, 50), percentile(a.ttft, 95)
		s.TotalP50MS, s.TotalP95MS = percentile(a.total, 50), percentile(a.total, 95)
		s.ConcurrentP50, s.ConcurrentMax = int(percentile(a.conc, 50)), int(percentile(a.conc, 100))
		s.Stalls, s.Quota = s.Statuses[loop.ClassStreamStall], s.Statuses[loop.ClassQuota]
		s.Retries = len(a.retries)
		out = append(out, s)
	}
	return out
}

// percentile theo hạng gần nhất: phần tử thứ ceil(p/100·n) của dãy đã sắp; 0 khi rỗng.
func percentile(v []int64, p int) int64 {
	if len(v) == 0 {
		return 0
	}
	s := slices.Sorted(slices.Values(v))
	return s[max((p*len(s)+99)/100, 1)-1]
}

// fmtMS in một số ms thành thời lượng dễ đọc (1.9s, 300ms).
func fmtMS(ms int64) string {
	return (time.Duration(ms) * time.Millisecond).Round(10 * time.Millisecond).String()
}

func printRoutineStats(w io.Writer, stats []routineStat, asJSON bool) error {
	if asJSON {
		return json.NewEncoder(w).Encode(stats)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ROUTINE\tRUNS\tREQUESTS\tTTFT p50/p95\tTOTAL p50/p95\tSTALL\tRETRY\tQUOTA\tOTHER ERRORS\tTOKENS in/out\tCONCURRENT p50/max")
	for _, s := range stats {
		if s.Requests == 0 {
			fmt.Fprintf(tw, "%s\t%d\t0\t%s\n", s.Routine, s.Runs, s.NoData)
			continue
		}
		var other []string
		for _, st := range slices.Sorted(maps.Keys(s.Statuses)) {
			if st != "ok" && st != loop.ClassStreamStall && st != loop.ClassQuota {
				other = append(other, fmt.Sprintf("%s=%d", st, s.Statuses[st]))
			}
		}
		if len(other) == 0 {
			other = []string{"-"}
		}
		fmt.Fprintf(tw, "%s\t%d\t%d\t%s/%s\t%s/%s\t%d\t%d\t%d\t%s\t%d/%d\t%d/%d\n",
			s.Routine, s.Runs, s.Requests, fmtMS(s.TTFTP50MS), fmtMS(s.TTFTP95MS), fmtMS(s.TotalP50MS), fmtMS(s.TotalP95MS),
			s.Stalls, s.Retries, s.Quota, strings.Join(other, ","), s.PromptTokens, s.CompletionTokens, s.ConcurrentP50, s.ConcurrentMax)
	}
	return tw.Flush()
}
