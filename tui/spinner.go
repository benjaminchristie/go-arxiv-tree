package tui

import (
	"time"
)

type Spinner struct {
	Elems []string
	Len   int
	Idx   int
}

func MakeSpinner() *Spinner {
	e := []string{"/", "-", "\\", "|", "/", "-", "\\", "|"}
	s := &Spinner{
		Elems: e,
		Len:   len(e),
		Idx:   0,
	}
	return s
}

func (s *Spinner) Spin() string {
	s.Idx = (s.Idx + 1) % s.Len
	return s.Elems[s.Idx]
}

// should be called as a goroutine
func (s *Spinner) Timer(t time.Duration, c chan string, stop chan bool) {
	timer := time.NewTimer(t)
	for {
		select {
		case _, ok:=<-timer.C:
			if !ok {
				close(c)
				return
			}
			c <- s.Spin()
			timer.Reset(t)
		case <-stop:
			r := timer.Stop()
			if r {
				<-timer.C // drains the channel, just in case
			}
			close(c)
			return
		}
	}
}
