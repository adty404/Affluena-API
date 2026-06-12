package caldate

import "time"

func AddMonthsClamped(current time.Time, months int) time.Time {
	loc := current.Location()
	targetFirst := time.Date(current.Year(), current.Month()+time.Month(months), 1, current.Hour(), current.Minute(), current.Second(), current.Nanosecond(), loc)
	lastDay := time.Date(targetFirst.Year(), targetFirst.Month()+1, 0, current.Hour(), current.Minute(), current.Second(), current.Nanosecond(), loc).Day()

	day := current.Day()
	if day > lastDay {
		day = lastDay
	}

	return time.Date(targetFirst.Year(), targetFirst.Month(), day, current.Hour(), current.Minute(), current.Second(), current.Nanosecond(), loc)
}
