package retry

import "time"

type Config struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}
