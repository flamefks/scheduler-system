package utils

import "time"

func BackoffDuration(attempt int, baseDelay time.Duration, maxDelay time.Duration) time.Duration {
	base := baseDelay * time.Millisecond
	if attempt <= 0 {
		return base
	}

	delay := base
	for i := 0; i < attempt; i++ {
		delay *= 2
		if delay > maxDelay*time.Millisecond {
			delay = maxDelay * time.Millisecond
			break
		}
	}
	return delay
}
