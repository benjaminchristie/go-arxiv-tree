package ratelimiter

import (
	"sync"
	"time"
)

var enabled bool
var lock sync.Mutex
var ticker *time.Ticker

func init() {
	enabled = false
}

func Enable() <-chan time.Time {
	lock.Lock()
	defer lock.Unlock()
	ticker = time.NewTicker(3 * time.Second)
	enabled = true
	return ticker.C
}

func IsEnabled() bool {
	return enabled
}

func WaitIfEnabled() {
	if enabled {
		<-ticker.C
	}
}

func Wait() {
	if !enabled {
		Enable()
	}
	_, ok := <-ticker.C
	go func() {
		if !ok {
			ticker.Reset(3 * time.Second)
		}
	}()
}

func Reset(d time.Duration) <-chan time.Time {
	lock.Lock()
	defer lock.Unlock()
	ticker.Reset(d)
	return ticker.C
}
