package main

import (
	"sync"
	"time"
)

// rateGate spaces calls at least `every` apart (Nominatim allows 1 req/s).
type rateGate struct {
	mu    sync.Mutex
	every time.Duration
	next  time.Time
}
