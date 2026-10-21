package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/loop"
	"github.com/stretchr/testify/require"
)

// writeStatsRun ghi một lượt chạy giả: journal thật (loop.FileJournal) trong thư mục lượt và
// report.json. finished rỗng = lượt chưa đóng: như báo cáo RUNNING thật (executeRoutine), nó
// KHÔNG có trường journal, chỉ báo cáo cuối mới có.
func writeStatsRun(t *testing.T, dataDir, name string, started, finished time.Time, events []loop.Event) string {
	t.Helper()
	dir := filepath.Join(dataDir, name, "runs", started.UTC().Format("20060102-150405"))
	journal := filepath.Join(dir, name+".jsonl")
	j, err := loop.OpenFileJournal(journal)
	require.NoError(t, err)
	require.NoError(t, j.Append(loop.Event{Time: started, Kind: "start", Note: name}))
	for _, e := range events {
		require.NoError(t, j.Append(e))
	}
	require.NoError(t, j.Close())
	rep := routineReport{Routine: name, Started: started, Finished: finished, Reason: "GREEN", Journal: journal}
	if finished.IsZero() {
		rep.Reason, rep.Journal = "RUNNING", ""
	}
	require.NoError(t, writeReport(dir, rep))
	return journal
}

func req(at time.Time, attempt, retry int, ttft *int64, totalMS int64, status string) loop.Event {
	return loop.Event{Time: at, Kind: "llm_request", Attempt: attempt, Retry: retry, TTFTMS: ttft, TotalMS: totalMS, Status: status}
}

func ptr(v int64) *int64 { return &v }

// 20 request TTFT 100..2000ms (tổng = 3×TTFT) cộng 1 request hết hạn mức không có byte nào
// (50ms). Phân vị theo hạng gần nhất: TTFT p50 = phần tử thứ 10 = 1000, p95 = thứ 19 = 1900;
// tổng (21 giá trị) p50 = thứ 11 = 3000, p95 = thứ 20 = 5700. Số đúng tuyệt đối: dữ liệu là
// số nguyên ms, không có sai số đo.
func TestRoutineStats_PercentilesCountsAndTokens(t *testing.T) {
	t.Parallel()
	data := t.TempDir()
	t0 := time.Date(2026, 10, 21, 8, 0, 0, 0, time.UTC)
	var run1, run2 []loop.Event
	for i := int64(1); i <= 10; i++ {
		e := req(t0.Add(time.Duration(i)*time.Second), 1, 0, ptr(i*100), i*300, "ok")
		e.PromptTokens, e.CompletionTokens = 1000, 10
		run1 = append(run1, e)
	}
	run1[4].Status = loop.ClassStreamStall
	quota := req(t0.Add(20*time.Second), 1, 0, nil, 50, loop.ClassQuota)
	quota.HTTPStatus = 429
	run1 = append(run1, quota)
	t1 := t0.Add(time.Hour)
	for i := int64(11); i <= 20; i++ {
		retry := 0
		switch {
		case i == 12 || i == 13:
			retry = 1 // hai request trong CÙNG lần chạy lại thứ nhất: một lần chạy lại
		case i == 14:
			retry = 2
		}
		e := req(t1.Add(time.Duration(i)*time.Second), 1, retry, ptr(i*100), i*300, "ok")
		e.PromptTokens, e.CompletionTokens = 2000, 20
		run2 = append(run2, e)
	}
	run2[0].Status = "server_error"
	writeStatsRun(t, data, "a", t0, t0.Add(30*time.Minute), run1)
	writeStatsRun(t, data, "a", t1, t1.Add(30*time.Minute), run2)

	stats := routineStats(data, "", 0, t1.Add(time.Hour), nil)
	require.Len(t, stats, 1)
	s := stats[0]
	require.Equal(t, "a", s.Routine)
	require.Equal(t, 2, s.Runs)
	require.Equal(t, 21, s.Requests)
	require.Equal(t, int64(1000), s.TTFTP50MS)
	require.Equal(t, int64(1900), s.TTFTP95MS)
	require.Equal(t, int64(3000), s.TotalP50MS)
	require.Equal(t, int64(5700), s.TotalP95MS)
	require.Equal(t, 1, s.Stalls)
	require.Equal(t, 1, s.Quota)
	require.Equal(t, 2, s.Retries, "lần chạy lại 1 và 2 của dew, không phải số request")
	require.Equal(t, map[string]int{"ok": 18, loop.ClassStreamStall: 1, loop.ClassQuota: 1, "server_error": 1}, s.Statuses)
	require.Equal(t, int64(10*1000+10*2000), s.PromptTokens)
	require.Equal(t, int64(10*10+10*20), s.CompletionTokens)
	require.Empty(t, s.NoData)

	var text bytes.Buffer
	require.NoError(t, printRoutineStats(&text, stats, false))
	out := text.String()
	t.Logf("routine stats:\n%s", out)
	require.Contains(t, out, "TTFT p50/p95")
	require.Contains(t, out, "1s/1.9s", "TTFT in theo giây/ms đúng đơn vị")
	require.Contains(t, out, "3s/5.7s")
	require.Contains(t, out, "server_error=1")

	var js bytes.Buffer
	require.NoError(t, printRoutineStats(&js, stats, true))
	var back []routineStat
	require.NoError(t, json.Unmarshal(js.Bytes(), &back))
	require.Equal(t, stats, back)
	require.Contains(t, js.String(), `"ttft_p50_ms":1000`)
}

// "Song song" của một request = số lượt chạy KHÁC (mọi routine) có khoảng [started, finished]
// giao khoảng của request. Lượt chưa đóng (báo cáo RUNNING không có trường journal) vẫn được đọc
// journal trong thư mục lượt và lấy giờ sửa journal làm điểm cuối. Routine không có dòng
// llm_request nói rõ không có dữ liệu thay vì in số 0.
func TestRoutineStats_ConcurrentOtherRunsAndOldJournals(t *testing.T) {
	t.Parallel()
	data := t.TempDir()
	t0 := time.Date(2026, 10, 21, 8, 0, 0, 0, time.UTC)
	m := func(min int) time.Time { return t0.Add(time.Duration(min) * time.Minute) }
	var reqs []loop.Event
	for _, at := range []int{1, 12, 16, 30, 55} { // khác song song: 0, 1 (b), 2 (b, c), 1 (c), 0
		reqs = append(reqs, req(m(at), 1, 0, ptr(100), 1000, "ok"))
	}
	writeStatsRun(t, data, "a", m(0), m(60), reqs)
	writeStatsRun(t, data, "b", m(10), m(20), nil)
	cj := writeStatsRun(t, data, "c", m(15), time.Time{}, []loop.Event{req(m(40), 1, 0, ptr(200), 1000, "ok")})
	require.NoError(t, os.Chtimes(cj, m(50), m(50)))

	stats := routineStats(data, "", 0, m(120), nil)
	require.Len(t, stats, 3)
	require.Equal(t, "a", stats[0].Routine)
	require.Equal(t, 1, stats[0].ConcurrentP50)
	require.Equal(t, 2, stats[0].ConcurrentMax)
	b, c := stats[1], stats[2]
	require.Equal(t, 1, b.Runs)
	require.Zero(t, b.Requests)
	require.Contains(t, b.NoData, "no request data (before v0.3.1")
	require.Equal(t, 1, c.Runs)
	require.Equal(t, 1, c.Requests, "request của lượt RUNNING vẫn được đếm")
	require.Empty(t, c.NoData)
	require.Equal(t, 1, c.ConcurrentMax, "request lúc 40 phút: chỉ a đang chạy")

	var text bytes.Buffer
	require.NoError(t, printRoutineStats(&text, stats, false))
	t.Logf("routine stats:\n%s", text.String())
	require.Contains(t, text.String(), "1/2", "song song p50/max")
	require.Contains(t, text.String(), "no request data (before v0.3.1")

	only := routineStats(data, "b", 0, m(120), nil)
	require.Len(t, only, 1)
	require.Equal(t, "b", only[0].Routine)
}

// --since chỉ lấy lượt bắt đầu trong khoảng đó; journal đọc không được thì cảnh báo, không im.
func TestRoutineStats_SinceAndUnreadableJournal(t *testing.T) {
	t.Parallel()
	data := t.TempDir()
	t0 := time.Date(2026, 10, 21, 8, 0, 0, 0, time.UTC)
	writeStatsRun(t, data, "a", t0, t0.Add(10*time.Minute), []loop.Event{req(t0.Add(time.Minute), 1, 0, ptr(100), 200, "ok")})
	j2 := writeStatsRun(t, data, "a", t0.Add(2*time.Hour), t0.Add(2*time.Hour+10*time.Minute),
		[]loop.Event{req(t0.Add(2*time.Hour+time.Minute), 1, 0, ptr(700), 900, "ok")})

	stats := routineStats(data, "", time.Hour, t0.Add(2*time.Hour+30*time.Minute), nil)
	require.Len(t, stats, 1)
	require.Equal(t, 1, stats[0].Runs)
	require.Equal(t, int64(700), stats[0].TTFTP50MS)

	require.NoError(t, os.Remove(j2))
	var warn bytes.Buffer
	stats = routineStats(data, "", time.Hour, t0.Add(2*time.Hour+30*time.Minute), &warn)
	require.Equal(t, 1, stats[0].Runs)
	require.Zero(t, stats[0].Requests)
	require.True(t, strings.Contains(warn.String(), "journal"), warn.String())
}

func TestRoutineStats_IsARoutineSubcommand(t *testing.T) {
	t.Parallel()
	c, _, err := routineCmd.Find([]string{"stats"})
	require.NoError(t, err)
	require.Same(t, routineStatsCmd, c)
	require.NotNil(t, c.Flags().Lookup("since"))
	require.NotNil(t, c.Flags().Lookup("json"))
}
