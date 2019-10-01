package permitpool

import (
	"sync"
)

// option - options setter method
type option func(*permitPool)

// permitPool - main
type permitPool struct {
	stateLock sync.Mutex
	sem       chan int
	permits   int
}

// WithPermits - sets number of permits in the permit pool
func WithPermits(arg int) option {
	return func(e *permitPool) {
		e.stateLock.Lock()
		defer e.stateLock.Unlock()
		e.permits = arg
	}
}
