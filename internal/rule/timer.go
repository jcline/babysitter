package rule

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru"
)

type TimerConfig struct {
	Ranges []*TimeRange
	cache  *lru.TwoQueueCache
}

func NewTimerConfig() (*TimerConfig, error) {
	return &TimerConfig{}, nil
}

type TimeRange struct {
	// Start, End setup the range for exact and inexact comparisons, the
	// only difference will be that we ignore the date component for inexact
	// comparisons
	Start, End time.Time
	// startNs, endNs are the pre-calculated result of
	//   start.Sub(start.Truncate(24 * time.Hour)
	// Doing this makes things a tad more fragile, but sped up the
	// computation by 2.6x
	startNs, endNs int64
	// Days is the list of days the inexact comparison should validate
	// against
	Days []time.Weekday
	// dayBitmask is a bitmask representation of the Days array so that we
	// don't have to loop to calculate whether a given time is on the same
	// day
	dayBitmask int
	Exact      bool
}

func (tr *TimeRange) UnmarshalJSON(data []byte) error {
	type Intermediate struct {
		Start time.Time `json:start`
		End   time.Time `json:end`
		Days  *[]string `json:days`
	}

	var i Intermediate
	err := json.Unmarshal(data, &i)
	if err != nil {
		return err
	}

	tr.Start = i.Start
	tr.End = i.End
	var seen int
	if i.Days != nil {
		for _, d := range *i.Days {
			var day time.Weekday
			switch strings.ToLower(d) {
			case "m", "mon", "monday":
				day = time.Monday
			case "tu", "tue", "tuesday":
				day = time.Tuesday
			case "w", "wed", "wednesday":
				day = time.Wednesday
			case "th", "thu", "thursday":
				day = time.Thursday
			case "f", "fri", "friday":
				day = time.Friday
			case "sa", "sat", "saturday":
				day = time.Saturday
			case "su", "sun", "sunday":
				day = time.Sunday
			default:
				return fmt.Errorf("invalid weekday %s", d)
			}
			if seen & 1 << day {
				return fmt.Errorf("cannot have duplicates in list %s", d)
			}
			seen |= 1 << day
		}
	}
	return nil
}

func (tr *TimeRange) computeBitmask() {
	var bitmask int
	for _, d := range tr.Days {
		bitmask = bitmask | 1<<d
	}
	tr.dayBitmask = bitmask
}

func (tr *TimeRange) preComputeInexact() {
	tr.computeBitmask()
	tr.startNs = tr.Start.Sub(tr.Start.Truncate(24 * time.Hour)).Nanoseconds()
	tr.endNs = tr.End.Sub(tr.End.Truncate(24 * time.Hour)).Nanoseconds()
}

//Within determines if a given time (t) is within a defined time range (tr)
//The matching works like: start <= t && start < end
func (tr *TimeRange) Within(t time.Time) (bool, error) {
	//fmt.Printf("%s %s\n", tr, t)
	if tr.Exact {
		return t.Equal(tr.Start) ||
			(t.After(tr.Start) && t.Before(tr.End)), nil
	} else {
		weekday := 1 << t.Weekday()
		found := (tr.dayBitmask & weekday) > 0

		// The idea here is to calculate the number of nanoseconds since
		// the day began, then we can setup a standard range comparison
		// without having to fiddle with hours/minutes/nanoseconds
		candidate := t.Sub(t.Truncate(24 * time.Hour))

		if candidate.Nanoseconds() >= tr.startNs &&
			candidate.Nanoseconds() < tr.endNs {
			return found, nil
		}

		return false, nil
	}

	return false, nil
}

func (tr *TimeRange) String() string {
	var b strings.Builder
	if tr.Exact {
		b.WriteString("[")
		b.WriteString(tr.Start.String())
		b.WriteString(", ")
		b.WriteString(tr.End.String())
		b.WriteString(")")
	} else {
		for i, d := range tr.Days {
			b.WriteString(d.String())
			if i < len(tr.Days)-1 {
				b.WriteString(", ")
			}
		}
		b.WriteString(" [")
		b.WriteString(
			fmt.Sprintf("%d:%d.%d (%dns)",
				tr.Start.Hour(),
				tr.Start.Minute(),
				tr.Start.Nanosecond(),
				tr.startNs,
			))
		b.WriteString(", ")
		b.WriteString(
			fmt.Sprintf("%d:%d.%d (%dns)",
				tr.End.Hour(),
				tr.End.Minute(),
				tr.End.Nanosecond(),
				tr.endNs,
			))
		b.WriteString(")")
	}

	return b.String()
}

func NewTimeRangeExact(start, end time.Time) *TimeRange {
	return &TimeRange{
		Exact: true,
		Start: start,
		End:   end,
	}
}

func NewTimeRangeInexact(days []time.Weekday, start, end time.Time) *TimeRange {
	tr := &TimeRange{
		Exact: false,
		Days:  days,
		Start: start,
		End:   end,
	}
	tr.preComputeInexact()
	return tr
}
