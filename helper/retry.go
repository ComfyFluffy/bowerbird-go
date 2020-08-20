package helper

import (
	"net/http"
	"time"
)

// DefaultBackoff returns min*2**tries or max as the retry time
func DefaultBackoff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	if sleep := (1 << attemptNum) * min; sleep < max && sleep != 0 {
		return sleep
	}
	return max
}
