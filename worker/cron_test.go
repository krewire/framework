// Tests for KWF-L5H2F

package worker_test

import (
	"context"
	"testing"
	"time"

	"github.com/krewire/framework/worker"
)

func TestFRK_SVC_060_Cron_NextFire_TableDriven(t *testing.T) {
	utc := time.UTC
	cases := []struct {
		name  string
		spec  string
		after time.Time
		want  time.Time
	}{
		{
			name:  "WildcardAdvancesOneMinute",
			spec:  "* * * * *",
			after: time.Date(2026, 8, 24, 10, 30, 15, 0, utc),
			want:  time.Date(2026, 8, 24, 10, 31, 0, 0, utc),
		},
		{
			name:  "StepEveryFifteenMinutes",
			spec:  "*/15 * * * *",
			after: time.Date(2026, 8, 24, 10, 30, 0, 0, utc),
			want:  time.Date(2026, 8, 24, 10, 45, 0, 0, utc),
		},
		{
			name:  "StepCarriesIntoNextHour",
			spec:  "*/15 * * * *",
			after: time.Date(2026, 8, 24, 10, 46, 0, 0, utc),
			want:  time.Date(2026, 8, 24, 11, 0, 0, 0, utc),
		},
		{
			name:  "HourRangeAndDOWRangeSkipWeekend",
			spec:  "0 9-17 * * 1-5",
			after: time.Date(2026, 8, 21, 18, 0, 5, 0, utc),
			want:  time.Date(2026, 8, 24, 9, 0, 0, 0, utc),
		},
		{
			name:  "MinuteListSameHour",
			spec:  "0,30 12 * * *",
			after: time.Date(2026, 8, 24, 12, 0, 0, 0, utc),
			want:  time.Date(2026, 8, 24, 12, 30, 0, 0, utc),
		},
		{
			name:  "DOMListJumpsToNextMonth",
			spec:  "30 4 1,15 * *",
			after: time.Date(2026, 8, 24, 0, 0, 0, 0, utc),
			want:  time.Date(2026, 9, 1, 4, 30, 0, 0, utc),
		},
		{
			name:  "DOWSevenIsSunday",
			spec:  "0 0 * * 7",
			after: time.Date(2026, 8, 23, 0, 0, 0, 0, utc),
			want:  time.Date(2026, 8, 30, 0, 0, 0, 0, utc),
		},
		{
			name:  "RangeWithStep",
			spec:  "0-20/10 * * * *",
			after: time.Date(2026, 8, 24, 5, 12, 0, 0, utc),
			want:  time.Date(2026, 8, 24, 5, 20, 0, 0, utc),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := worker.ParseCron(tc.spec)
			if err != nil {
				t.Fatalf("ParseCron(%q): %v", tc.spec, err)
			}
			got := s.NextFire(tc.after)
			if !got.Equal(tc.want) {
				t.Fatalf("NextFire(%s) = %s, want %s", tc.after, got, tc.want)
			}
			if !got.After(tc.after) {
				t.Fatalf("NextFire must be strictly after the given instant")
			}
		})
	}
}

func TestFRK_SVC_060_Cron_NextFire_UnsatisfiableReturnsZero(t *testing.T) {
	s, err := worker.ParseCron("0 0 30 2 *")
	if err != nil {
		t.Fatalf("ParseCron: %v", err)
	}
	if got := s.NextFire(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); !got.IsZero() {
		t.Fatalf("Feb 30 schedule returned %s, want zero time", got)
	}
}

func TestFRK_SVC_060_Cron_ParseRejectsInvalid(t *testing.T) {
	for _, spec := range []string{
		"",
		"* * * *",
		"* * * * * *",
		"60 * * * *",
		"* 24 * * *",
		"* * 0 1 *",
		"* * * 13 *",
		"bad * * * *",
		"*/0 * * * *",
		"10-5 * * * *",
	} {
		if _, err := worker.ParseCron(spec); err == nil {
			t.Errorf("ParseCron(%q) accepted invalid spec", spec)
		}
	}
}

func TestFRK_SVC_060_EnqueueValidatesCron(t *testing.T) {
	q := worker.NewInMemoryQueue()
	if _, err := q.Enqueue(context.Background(), noopJob(), worker.Options{Cron: "* not a cron"}); err == nil {
		t.Fatal("Enqueue accepted invalid cron spec")
	}
}
