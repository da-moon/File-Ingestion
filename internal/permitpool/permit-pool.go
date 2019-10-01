package permitpool

// PermitPool -
type PermitPool interface {
	Acquire()
	Release()
}

// New -
func New(opts ...option) PermitPool {
	result := &permitPool{}
	for _, opt := range opts {
		opt(result)
	}
	// Default number of parallel operations
	if result.permits < 1 {
		result.permits = 128
	}
	result.sem = make(chan int, result.permits)
	return result
}

// Acquire returns when a permit has been acquired
func (c *permitPool) Acquire() {
	c.sem <- 1
}

// Release returns a permit to the pool
func (c *permitPool) Release() {
	<-c.sem
}
