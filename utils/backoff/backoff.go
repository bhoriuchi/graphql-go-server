package backoff

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

type Backoff struct {
	mx       sync.Mutex
	min      time.Duration
	max      time.Duration
	jitter   float64
	factor   float64
	attempts float64
}

type Options struct {
	Min    time.Duration
	Max    time.Duration
	Jitter float64
	Factor float64
}

func NewBackoff(opts *Options) *Backoff {
	if opts == nil {
		opts = &Options{}
	}

	min := 100 * time.Millisecond
	if opts.Min > 0 {
		min = opts.Min
	}

	max := 10000 * time.Millisecond
	if opts.Max > 0 {
		max = opts.Max
	}

	if max < min {
		max = min
	}

	var factor float64 = 2
	if opts.Factor > 1 {
		factor = opts.Factor
	}

	var jitter float64 = 0
	if opts.Jitter > 0 && opts.Jitter <= 1 {
		jitter = opts.Jitter
	}

	b := &Backoff{
		min:      min,
		max:      max,
		factor:   factor,
		jitter:   jitter,
		attempts: 0,
	}

	return b
}

func (b *Backoff) Attempts() float64 {
	b.mx.Lock()
	defer b.mx.Unlock()
	return b.attempts
}

func (b *Backoff) Duration() time.Duration {
	b.mx.Lock()
	defer b.mx.Unlock()
	b.attempts = b.attempts + 1
	ms := float64(b.min.Milliseconds()) * math.Pow(b.factor, b.attempts)

	if b.jitter > 0 {
		r := rand.Float64()
		deviation := math.Floor(r * b.jitter * ms)
		tmp := int(math.Floor(r*10)) & 1
		if tmp == 0 {
			ms = ms - deviation
		} else {
			ms = ms + deviation
		}
	}

	d := time.Duration(math.Min(ms, float64(b.max.Milliseconds())))
	return d * time.Millisecond
}

func (b *Backoff) Reset() {
	b.mx.Lock()
	defer b.mx.Unlock()
	b.attempts = 0
}
