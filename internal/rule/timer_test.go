package rule

import (
	"testing"
	"time"
)

func Test_TimeRange_Within(t *testing.T) {
	tc := []struct {
		name    string
		tr      *TimeRange
		t       time.Time
		success bool
		err     error
	}{
		{
			name:    "empty timerange fails",
			tr:      &TimeRange{},
			t:       time.Now(),
			success: false,
			err:     nil,
		},
		{
			name: "any time passes",
			tr: NewTimeRangeExact(
				time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC),
			),
			t:       time.Now(),
			success: true,
			err:     nil,
		},
		{
			name: "any time passes, no date",
			tr: NewTimeRangeInexact(
				[]time.Weekday{
					time.Sunday,
					time.Monday,
					time.Tuesday,
					time.Wednesday,
					time.Thursday,
					time.Friday,
					time.Saturday,
				},
				time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC),
				time.Date(0, 0, 0, 23, 59, 59, 999999999, time.UTC),
			),
			t:       time.Now(),
			success: true,
			err:     nil,
		},
		{
			name: "monday does not pass",
			tr: NewTimeRangeInexact(
				[]time.Weekday{
					time.Sunday,
					time.Tuesday,
					time.Wednesday,
					time.Thursday,
					time.Friday,
					time.Saturday,
				},
				time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC),
				time.Date(0, 0, 0, 23, 59, 59, 999999999, time.UTC),
			),
			// january 5th, 1970 was a monday, cal 1970
			t:       time.Date(1970, 1, 5, 12, 0, 0, 0, time.UTC),
			success: false,
			err:     nil,
		},
		{
			name: "saturday at midnight does pass",
			tr: NewTimeRangeInexact(
				[]time.Weekday{
					time.Sunday,
					time.Saturday,
				},
				time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC),
				time.Date(0, 0, 0, 23, 59, 59, 999999999, time.UTC),
			),
			// january 3th, 1970 was a saturday, cal 1970
			t:       time.Date(1970, 1, 3, 0, 0, 0, 0, time.UTC),
			success: true,
			err:     nil,
		},
		{
			name: "monday at midnight doesn't pass",
			tr: NewTimeRangeInexact(
				[]time.Weekday{
					time.Sunday,
				},
				time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC),
				time.Date(0, 0, 0, 23, 59, 59, 999999999, time.UTC),
			),
			// january 5th, 1970 was a monday, cal 1970
			t:       time.Date(1970, 1, 5, 0, 0, 0, 0, time.UTC),
			success: false,
			err:     nil,
		},
	}

	for _, test := range tc {
		r, err := test.tr.Within(test.t)
		if r != test.success {
			t.Fatalf("%s got %v expected %v", test.name, r, test.success)
		}

		if err != test.err {
			t.Fatalf("%s got %v exepcted %v", test.name, err, test.err)
		}
	}

}

func Benchmark_TimeRange_Within_False(b *testing.B) {
	tc := struct {
		name    string
		tr      *TimeRange
		t       time.Time
		success bool
		err     error
	}{
		tr: &TimeRange{},
		t:  time.Now(),
	}

	for i := 0; i < b.N; i++ {
		_, _ = tc.tr.Within(tc.t)
	}

}

func Benchmark_TimeRange_Within_True(b *testing.B) {
	tc := struct {
		name    string
		tr      *TimeRange
		t       time.Time
		success bool
		err     error
	}{
		tr: NewTimeRangeExact(
			time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC),
		),
		t: time.Now(),
	}

	for i := 0; i < b.N; i++ {
		_, _ = tc.tr.Within(tc.t)
	}
}

func Benchmark_TimeRange_Within_No_Date_True(b *testing.B) {
	tc := struct {
		name    string
		tr      *TimeRange
		t       time.Time
		success bool
		err     error
	}{
		tr: NewTimeRangeInexact(
			[]time.Weekday{
				time.Sunday,
				time.Monday,
				time.Tuesday,
				time.Wednesday,
				time.Thursday,
				time.Friday,
				time.Saturday,
			},
			time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC),
			time.Date(0, 0, 0, 23, 59, 59, 999999999, time.UTC),
		),
		t: time.Now(),
	}

	for i := 0; i < b.N; i++ {
		_, _ = tc.tr.Within(tc.t)
	}
}
