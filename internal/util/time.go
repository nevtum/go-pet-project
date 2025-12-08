package util

import "time"

type Timestamp func() time.Time

func SequencedTime(t time.Time) func() time.Time {
	n := 0

	return func() time.Time {
		newTime := t.Add(time.Duration(n) * time.Nanosecond)
		n++
		return newTime
	}
}
