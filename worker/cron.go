package worker

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const maxCronIterations = 50000

// Schedule is a parsed 5-field cron expression: minute, hour, day-of-month,
// month, day-of-week (0 = Sunday, 7 accepted as Sunday). Fields support "*",
// exact values, lists ("a,b"), ranges ("a-b"), and steps ("*/n", "a-b/n").
// Month and weekday names are not supported.
type Schedule struct {
	minute  [60]bool
	hour    [24]bool
	dom     [32]bool
	mon     [13]bool
	dow     [8]bool
	domStar bool
	dowStar bool
	spec    string
}

// ParseCron compiles a 5-field cron expression into a Schedule.
func ParseCron(spec string) (*Schedule, error) {
	fields := strings.Fields(spec)
	if len(fields) != 5 {
		return nil, fmt.Errorf("worker: cron %q: want 5 fields, got %d", spec, len(fields))
	}
	s := &Schedule{spec: spec}
	if err := parseField(fields[0], 0, 59, s.minute[:]); err != nil {
		return nil, fmt.Errorf("worker: cron %q: minute: %w", spec, err)
	}
	if err := parseField(fields[1], 0, 23, s.hour[:]); err != nil {
		return nil, fmt.Errorf("worker: cron %q: hour: %w", spec, err)
	}
	if err := parseField(fields[2], 1, 31, s.dom[:]); err != nil {
		return nil, fmt.Errorf("worker: cron %q: day-of-month: %w", spec, err)
	}
	if err := parseField(fields[3], 1, 12, s.mon[:]); err != nil {
		return nil, fmt.Errorf("worker: cron %q: month: %w", spec, err)
	}
	if err := parseDOW(fields[4], s.dow[:]); err != nil {
		return nil, fmt.Errorf("worker: cron %q: day-of-week: %w", spec, err)
	}
	s.domStar = strings.HasPrefix(fields[2], "*")
	s.dowStar = strings.HasPrefix(fields[4], "*")
	return s, nil
}

// String returns the original specification.
func (s *Schedule) String() string { return s.spec }

// NextFire returns the first matching instant strictly after t, in t's
// location, or the zero Time when no match exists within its bounded search
// horizon (unsatisfiable expressions such as Feb 30).
func (s *Schedule) NextFire(after time.Time) time.Time {
	t := after.Add(time.Minute)
	t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, after.Location())
	for i := 0; i < maxCronIterations; i++ {
		if !s.mon[t.Month()] {
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
			continue
		}
		if !s.dayMatches(t) {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
			continue
		}
		if !s.hour[t.Hour()] {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, t.Location())
			continue
		}
		if s.minute[t.Minute()] {
			return t
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}
}

func (s *Schedule) dayMatches(t time.Time) bool {
	domOK := s.dom[t.Day()]
	dowOK := s.dow[int(t.Weekday())]
	switch {
	case s.domStar && s.dowStar:
		return true
	case s.domStar:
		return dowOK
	case s.dowStar:
		return domOK
	default:
		return domOK || dowOK
	}
}

func parseField(field string, min, max int, set []bool) error {
	mark := func(v int) error {
		if v < min || v > max {
			return fmt.Errorf("value %d out of range [%d,%d]", v, min, max)
		}
		set[v] = true
		return nil
	}
	for _, term := range strings.Split(field, ",") {
		base, step := term, 1
		if i := strings.Index(term, "/"); i >= 0 {
			base = term[:i]
			n, err := strconv.Atoi(term[i+1:])
			if err != nil || n < 1 {
				return fmt.Errorf("bad step in %q", term)
			}
			step = n
		}
		lo, hi := min, max
		switch {
		case base == "*":
		case strings.Contains(base, "-"):
			bounds := strings.SplitN(base, "-", 2)
			a, errA := strconv.Atoi(bounds[0])
			b, errB := strconv.Atoi(bounds[1])
			if errA != nil || errB != nil || a > b {
				return fmt.Errorf("bad range in %q", term)
			}
			lo, hi = a, b
		default:
			v, err := strconv.Atoi(base)
			if err != nil {
				return fmt.Errorf("bad value in %q", term)
			}
			lo, hi = v, v
		}
		for v := lo; v <= hi; v += step {
			if err := mark(v); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseDOW(field string, set []bool) error {
	if err := parseField(field, 0, 7, set); err != nil {
		return err
	}
	set[0] = set[0] || set[7]
	set[7] = false
	return nil
}
