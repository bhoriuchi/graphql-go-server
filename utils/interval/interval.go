package interval

import (
	"sync"
	"time"
)

// implements a javascript like interval
type Interval struct {
	ticker  *time.Ticker
	done    chan interface{}
	cleared bool
	mx      sync.Mutex
}

// rest resets the ticker which can be used to
// start the ticker over instead of canceling and recreating a new one
func (i *Interval) Reset(timeout time.Duration) {
	i.ticker.Reset(timeout)
}

// clears the interval by setting cleared
func (i *Interval) Clear() {
	i.mx.Lock()
	defer i.mx.Unlock()

	i.ticker.Stop()
	if i.cleared {
		return
	}

	i.done <- nil
	i.cleared = true
}

// setInterval imitates the built-in javascript function
func SetInterval(handler func(i *Interval), timeout time.Duration) *Interval {
	ticker := time.NewTicker(timeout)
	done := make(chan interface{})

	i := &Interval{
		ticker:  ticker,
		done:    done,
		cleared: false,
	}

	go func() {
		for {
			select {
			case <-done:
				ticker.Stop()

				i.mx.Lock()
				i.cleared = true
				i.mx.Unlock()

				return

			case <-ticker.C:
				i.mx.Lock()

				if i.cleared {
					i.mx.Unlock()
					ticker.Stop()
					return
				}

				handler(i)
				i.mx.Unlock()
			}
		}
	}()

	return i
}

// clearInterval imitates the builtin javascript function
func ClearInterval(i *Interval) {
	i.Clear()
}

func SetTimeout(handler func(), timeout time.Duration) *Interval {
	return SetInterval(func(i *Interval) {
		i.Clear()
		handler()
	}, timeout)
}

// clearTimeout imitates the builtin javascript function
func ClearTimeout(i *Interval) {
	i.Clear()
}
