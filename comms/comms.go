package comms

import (
	"errors"
	"sync"
)

type Comm struct {
	PublicChan  chan interface{}
	privateChan chan interface{}
	Enabled     bool
	last        interface{}
	lock        *sync.Mutex
	Callback    func(interface{}) interface{}
}

func MakeComm(i int, cb ...func(interface{}) interface{}) *Comm {
	var private, public chan interface{}
	var lock sync.Mutex
	if i == 0 {
		private = make(chan interface{})
		public = make(chan interface{})
	} else {
		private = make(chan interface{}, i)
		public = make(chan interface{}, i)
	}
	c := &Comm{
		PublicChan:  public,
		privateChan: private,
		Enabled:     true,
		last:        nil,
		lock:        &lock,
	}
	if len(cb) != 0 {
		c.Callback = cb[0]
	} else {
		c.Callback = func(i interface{}) interface{} { return i }
	}

	go func() {
		for {
			c.Middleware()
		}
	}()
	return c
}

func (c *Comm) Middleware() bool {
	if c.Enabled {
		l, ok := <-c.privateChan
		if !ok {
			return false
		}
		newL := c.Callback(l)
		c.last = newL
		c.PublicChan <- newL
		return true
	}
	return false
}

func (c *Comm) Enable() {
	c.Enabled = true
	go func() {
		var b bool
		for {
			b = c.Middleware()
			if !b {
				return
			}
		}
	}()
}

func (c *Comm) Disable() {
	c.Enabled = false
}

func (c *Comm) Send(v interface{}) error {
	if c.Enabled {
		c.lock.Lock()
		defer c.lock.Unlock()
		c.privateChan <- v
		c.last = v
		return nil
	}
	return errors.New("Channel not enabled")
}

func (c *Comm) GetLast() interface{} {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.last
}
