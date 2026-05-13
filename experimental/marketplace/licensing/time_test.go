package licensing

import (
	"testing"
	"time"
)

func fixedTestTime(tb testing.TB, fixedTime time.Time) {
	tb.Helper()
	timeNow = func() time.Time {
		return fixedTime
	}
	tb.Cleanup(func() {
		timeNow = time.Now
	})
}
