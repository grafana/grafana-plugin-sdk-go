package licensing

import "time"

func fixedTestTime() {
	timeNow = func() time.Time {
		return time.Date(2019, 10, 11, 17, 30, 40, 0, time.UTC)
	}
}

func restoreTime() {
	timeNow = time.Now
}
