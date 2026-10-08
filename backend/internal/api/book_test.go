package api

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/luk224/pomofarm/backend/internal/service"
)

// addPomodoro stores a finished Pomodoro directly (as the app would have left it), ending at `end` (UTC).
func addPomodoro(t *testing.T, e *env, end string, plannedS int, tag, status string, pausedS int) {
	t.Helper()
	var tagID any
	if tag != "" {
		dbExec(t, e, `INSERT OR IGNORE INTO tags (player_id, name) VALUES (1, ?)`, tag)
		var id int64
		if err := e.db.QueryRow(`SELECT id FROM tags WHERE player_id = 1 AND name = ?`, tag).Scan(&id); err != nil {
			t.Fatal(err)
		}
		tagID = id
	}
	endT, err := time.Parse(time.RFC3339, end)
	if err != nil {
		t.Fatal(err)
	}
	start := endT.Add(-time.Duration(plannedS+pausedS) * time.Second)
	dbExec(t, e, `INSERT INTO pomodoros (player_id, plot_id, plant_type, tag_id, planned_s, started_at, paused_total_s, ended_at, reward_focus, status)
		VALUES (1, NULL, 'daisy', ?, ?, ?, ?, ?, 1, ?)`, tagID, plannedS, start.UTC().Format("2006-01-02T15:04:05.000000Z"), pausedS, endT.UTC().Format("2006-01-02T15:04:05.000000Z"), status)
}

func getBook(t *testing.T, e *env, query string) (int, service.BookMonth) {
	t.Helper()
	code, body := e.do("GET", "/api/book"+query, nil)
	var b service.BookMonth
	if code == 200 {
		if err := json.Unmarshal(body, &b); err != nil {
			t.Fatalf("book json: %v\n%s", err, body)
		}
	}
	return code, b
}

func TestBookTotalsByTagAndWeekday(t *testing.T) {
	e := newEnv(t, 1)
	// 2026-10-05 is a Monday, 2026-10-07 a Wednesday
	addPomodoro(t, e, "2026-10-05T10:00:00Z", 1500, "tesis", "completed", 0)
	addPomodoro(t, e, "2026-10-05T15:00:00Z", 2700, "tesis", "completed", 300) // a pause is not focus time
	addPomodoro(t, e, "2026-10-07T09:00:00Z", 600, "", "completed", 0)
	addPomodoro(t, e, "2026-10-07T11:00:00Z", 1500, "ingles", "completed", 0)
	addPomodoro(t, e, "2026-10-07T12:00:00Z", 1500, "ingles", "cancelled", 0) // cancelled: no focus
	addPomodoro(t, e, "2026-09-30T12:00:00Z", 900, "tesis", "completed", 0)   // another month
	code, b := getBook(t, e, "?month=2026-10")
	if code != 200 {
		t.Fatalf("status %d", code)
	}
	if b.Pomodoros != 4 || b.Seconds != 1500+2700+600+1500 {
		t.Fatalf("totals: %d Pomodoros, %d s", b.Pomodoros, b.Seconds)
	}
	if len(b.Tags) != 3 || b.Tags[0].Name != "tesis" || b.Tags[0].Pomodoros != 2 || b.Tags[0].Seconds != 4200 || b.Tags[1].Name != "ingles" || b.Tags[2].Name != "" || b.Tags[2].Seconds != 600 {
		t.Fatalf("tags (most time first): %+v", b.Tags)
	}
	if len(b.Weekdays) != 7 || b.Weekdays[0].Pomodoros != 2 || b.Weekdays[0].Seconds != 4200 || b.Weekdays[2].Pomodoros != 2 || b.Weekdays[1].Pomodoros != 0 || b.Weekdays[6].Pomodoros != 0 {
		t.Fatalf("weekdays (Monday first): %+v", b.Weekdays)
	}
	if len(b.Days) != 2 || b.Days[0].Date != "2026-10-05" || b.Days[1].Date != "2026-10-07" {
		t.Fatalf("days: %+v", b.Days)
	}
	_, sep := getBook(t, e, "?month=2026-09")
	if sep.Pomodoros != 1 || sep.Seconds != 900 {
		t.Fatalf("September: %+v", sep)
	}
	_, none := getBook(t, e, "?month=2020-01")
	if none.Pomodoros != 0 || none.Seconds != 0 || len(none.Tags) != 0 || len(none.Days) != 0 {
		t.Fatalf("an empty month must be empty: %+v", none)
	}
}

func TestBookUsesThePlayersTimeZone(t *testing.T) {
	e := newEnv(t, 1)
	// 23:30 UTC on Sunday 4 Oct is already Monday 5 Oct in Madrid (UTC+2); and 22:30 UTC on 30 Sep is 1 Oct there.
	addPomodoro(t, e, "2026-10-04T23:30:00Z", 1500, "a", "completed", 0)
	addPomodoro(t, e, "2026-09-30T22:30:00Z", 1500, "a", "completed", 0)
	_, utc := getBook(t, e, "?month=2026-10&tz=UTC")
	_, mad := getBook(t, e, "?month=2026-10&tz=Europe/Madrid")
	if utc.Pomodoros != 1 || utc.Weekdays[6].Pomodoros != 1 {
		t.Fatalf("UTC: %+v", utc)
	}
	if mad.Pomodoros != 2 || mad.Weekdays[0].Pomodoros != 1 || mad.TimeZone != "Europe/Madrid" {
		t.Fatalf("Madrid: %+v", mad)
	}
	_, tok := getBook(t, e, "?month=2026-10&tz=Pacific/Auckland")
	if tok.TimeZone != "Pacific/Auckland" {
		t.Fatalf("time zones must work without a system tz database: %+v", tok)
	}
}

func TestBookRejectsBadInput(t *testing.T) {
	e := newEnv(t, 1)
	for _, q := range []string{"?month=2026-13", "?month=oct", "?month=2026-1", "?month=../x", "?tz=Mars/Base", "?tz=" + strings.Repeat("a", 200), "?month=2026-10&tz=%00"} {
		if code, _ := getBook(t, e, q); code != 400 {
			t.Errorf("%s answered %d, want 400", q, code)
		}
	}
	if code, b := getBook(t, e, ""); code != 200 || b.Month == "" || len(b.Months) == 0 {
		t.Fatalf("no query: %d %+v", code, b)
	}
}

func TestBookMonthPickerListsMonthsWithData(t *testing.T) {
	e := newEnv(t, 1)
	addPomodoro(t, e, "2026-03-10T10:00:00Z", 1500, "", "completed", 0)
	addPomodoro(t, e, "2026-05-10T10:00:00Z", 1500, "", "completed", 0)
	_, b := getBook(t, e, "?month=2026-05&tz=UTC")
	cur := e.clock.T.UTC().Format("2006-01")
	want := map[string]bool{"2026-03": true, "2026-05": true, cur: true}
	got := map[string]bool{}
	for _, m := range b.Months {
		got[m] = true
	}
	if len(got) != len(want) {
		t.Fatalf("months %v, want %v", b.Months, want)
	}
	for m := range want {
		if !got[m] {
			t.Fatalf("months %v missing %s", b.Months, m)
		}
	}
	for i := 1; i < len(b.Months); i++ {
		if b.Months[i-1] >= b.Months[i] {
			t.Fatalf("months not sorted: %v", b.Months)
		}
	}
}

// A Pomodoro finished through the real flow lands in the book.
func TestBookCountsAPomodoroFinishedThroughTheAPI(t *testing.T) {
	e := newEnv(t, 1)
	e.expect(201, "POST", "/api/pomodoros", service.PlantRequest{PlotID: 1, PlantType: "daisy", Tag: "lectura"})
	e.clock.Advance(11 * time.Minute)
	_, b := getBook(t, e, "?tz=UTC")
	if b.Pomodoros != 1 || b.Seconds != 600 || len(b.Tags) != 1 || b.Tags[0].Name != "lectura" {
		t.Fatalf("book after a finished Pomodoro: %+v", b)
	}
}

func csvOf(t *testing.T, e *env, query string) ([][]string, []byte) {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/book/export.csv"+query, nil)
	resp, err := e.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("csv status %d: %s", resp.StatusCode, buf.String())
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") || !strings.Contains(resp.Header.Get("Content-Disposition"), "attachment") {
		t.Fatalf("headers: %q %q", ct, resp.Header.Get("Content-Disposition"))
	}
	raw := buf.Bytes()
	if !bytes.HasPrefix(raw, []byte("\xEF\xBB\xBF")) {
		t.Fatal("the CSV must start with a UTF-8 byte-order mark so Excel reads accents")
	}
	recs, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(raw, []byte("\xEF\xBB\xBF")))).ReadAll()
	if err != nil {
		t.Fatalf("the CSV does not parse: %v\n%s", err, raw)
	}
	return recs, raw
}

func TestBookCSVMatchesTheStoredPomodoros(t *testing.T) {
	e := newEnv(t, 1)
	addPomodoro(t, e, "2026-10-05T10:00:00Z", 1500, "tesis", "completed", 0)
	addPomodoro(t, e, "2026-10-05T15:00:00Z", 2700, `a "quoted", tag`+"\nwith newline", "completed", 300)
	addPomodoro(t, e, "2026-10-06T09:00:00Z", 600, "", "cancelled", 0)
	addPomodoro(t, e, "2026-10-07T09:00:00Z", 1500, "=HYPERLINK(\"http://x\")", "completed", 0)
	addPomodoro(t, e, "2026-10-07T10:00:00Z", 1500, "+1+1", "completed", 0)
	addPomodoro(t, e, "2026-10-07T11:00:00Z", 1500, "ñandú · café", "completed", 0)
	dbExec(t, e, `INSERT INTO pomodoros (player_id, plot_id, plant_type, planned_s, started_at, status) VALUES (1, NULL, 'oak', 3600, '2026-10-08T08:00:00.000000Z', 'running')`) // unfinished: not exported

	recs, _ := csvOf(t, e, "?tz=Europe/Madrid")
	var stored int
	e.db.QueryRow(`SELECT COUNT(*) FROM pomodoros WHERE status IN ('completed','cancelled')`).Scan(&stored)
	if len(recs) != stored+1 || stored != 6 {
		t.Fatalf("%d CSV rows for %d stored Pomodoros", len(recs)-1, stored)
	}
	if strings.Join(recs[0], ",") != "id,inicio,fin,etiqueta,cultivo,minutos,pausa_minutos,gotas,estado" {
		t.Fatalf("header: %v", recs[0])
	}
	r1 := recs[1]
	if r1[1] != "2026-10-05 11:35:00" || r1[2] != "2026-10-05 12:00:00" || r1[3] != "tesis" || r1[5] != "25.0" || r1[8] != "completed" {
		t.Fatalf("first row (Madrid time): %v", r1)
	}
	r2 := recs[2]
	if r2[3] != "a \"quoted\", tag\nwith newline" || r2[5] != "45.0" || r2[6] != "5.0" {
		t.Fatalf("quoting and pause: %v", r2)
	}
	if recs[3][8] != "cancelled" || recs[3][3] != "" {
		t.Fatalf("cancelled row: %v", recs[3])
	}
	if !strings.HasPrefix(recs[4][3], "'=") || !strings.HasPrefix(recs[5][3], "'+") {
		t.Fatalf("a tag that looks like a formula must be neutralised: %q %q", recs[4][3], recs[5][3])
	}
	if recs[6][3] != "ñandú · café" {
		t.Fatalf("accents: %q", recs[6][3])
	}
	// the totals of the file equal the book's
	var minutes float64
	for _, r := range recs[1:] {
		if r[8] == "completed" {
			var m float64
			if _, err := fmtSscan(r[5], &m); err != nil {
				t.Fatal(err)
			}
			minutes += m
		}
	}
	_, b := getBook(t, e, "?month=2026-10&tz=Europe/Madrid")
	if int64(minutes*60+0.5) != b.Seconds {
		t.Fatalf("CSV sums to %.1f min, the book says %d s", minutes, b.Seconds)
	}
}

func TestBookCSVRejectsABadTimeZoneAndIsEmptyWithoutData(t *testing.T) {
	e := newEnv(t, 1)
	recs, _ := csvOf(t, e, "")
	if len(recs) != 1 {
		t.Fatalf("an empty game exports only the header, got %d rows", len(recs))
	}
	req := httptest.NewRequest("GET", "/api/book/export.csv?tz=Mars/Base", nil)
	resp, _ := e.app.Test(req, -1)
	if resp.StatusCode != 400 {
		t.Fatalf("bad tz answered %d", resp.StatusCode)
	}
}

func fmtSscan(s string, v *float64) (int, error) { return fmt.Sscanf(s, "%g", v) }
