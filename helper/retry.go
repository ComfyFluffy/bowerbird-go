package helper

import (
	"time"
)

func DefaultBackoff(min, max time.Duration, tries int) time.Duration {
	if sleep := (1 << tries) * min; sleep < max && sleep != 0 {
		return sleep
	}
	return max
}

type Retryer struct {
	WaitMin, WaitMax time.Duration
	TriesMax         int
}

func (r *Retryer) Retry(f func() error, end func(error) bool) {
	tries := 0
	for tries < r.TriesMax {
		tries++
		err := f()
		if end(err) {
			return
		}
		time.Sleep(DefaultBackoff(r.WaitMin, r.WaitMax, tries))
	}
}
