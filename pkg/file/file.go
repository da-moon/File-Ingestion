package file

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	// Storage "github.com/bifrostcloud/bifrost/internal/yggdrasil/engine"
	permitpool "github.com/damoonazarpazhooh/File-Ingestion/internal/permitpool"
	"github.com/kardianos/osext"
	"github.com/mitchellh/colorstring"
	"github.com/palantir/stacktrace"
)

// New - constructs a new file physical Storage using the given directory
// to store data on disk
// TODO : permit pool options
func New(opts ...Option) *Storage {
	result := &Storage{
		logOps: false,
		logCh:  make(chan string),
	}
	for _, opt := range opts {
		opt(result)
	}
	result.permitPool = permitpool.New(
		permitpool.WithPermits(1),
	)
	return result
}

// Init -
func (b *Storage) Init() error {
	errCh := make(chan error)
	defer func() {
		if b.initialized && b.logOps {
			logs := fmt.Sprintf("[yellow][INFO] Storage: Initialized at %s\n", b.path)
			colorstring.Println(logs)
		}
	}()
	b.stateLock.Lock()
	defer b.stateLock.Unlock()
	go b.Do(func() {
		var err error
		// If path is not given , then creates a temproray dir for ops.
		b.logCh <- fmt.Sprintf("[yellow][INFO] Storage : Started Initialization ... ")
		if len(b.path) == 0 {
			b.logCh <- fmt.Sprintf("[yellow][INFO] Storage : path is not given. creating a temproray dir for ops ... ")
			var selfPath string
			selfPath, err = osext.ExecutableFolder()
			if err != nil {
				err = stacktrace.Propagate(err, "[FATAL] Storage :Error getting executable path")
				errCh <- (err)
				return
			}
			b.path, err = ioutil.TempDir(selfPath, "tmp")
			if err != nil {
				err = stacktrace.Propagate(err, "[FATAL] Storage :could not create temporary directory (%s)", selfPath+"tmp")
				errCh <- (err)
				return
			}
		}
		errCh <- nil
		return
	})

	for {

		select {
		case logs := <-b.logCh:
			{
				if b.logOps {
					colorstring.Println(logs)
				}
			}
		case err := <-errCh:
			{
				if err != nil {
					return err
				}
				b.initialized = true
				return nil
			}
		}
	}
}

// Put -
func (b *Storage) Put(ctx context.Context, entry *Entry) error {
	var err error
	if !b.initialized {
		err = stacktrace.NewError("[ERROR] Storage :was not initialized")
		return err
	}
	b.permitPool.Acquire()
	defer b.permitPool.Release()

	b.stateLock.Lock()
	defer b.stateLock.Unlock()
	if b.logOps {
		start := time.Now()
		defer func() {
			duration := fmt.Sprintf("[bold][yellow][INFO] Storage: Put operation took (%v) to complete", time.Now().Sub(start))
			colorstring.Println(duration)
		}()

	}

	errCh := make(chan error)
	go func() {
		errCh <- b.PutInternal(ctx, entry)
	}()
	for {
		select {
		case logs := <-b.logCh:
			{
				if b.logOps {
					colorstring.Println(logs)
				}

			}
		case err := <-errCh:
			{
				if err != nil {

					return err
				}
				return nil
			}
		}
	}
}

// Get -
func (b *Storage) Get(ctx context.Context, k string) (*Entry, error) {
	if !b.initialized {
		err := stacktrace.NewError("[ERROR] Storage :was not initialized")
		return nil, err
	}
	b.permitPool.Acquire()
	defer b.permitPool.Release()

	b.stateLock.RLock()
	defer b.stateLock.RUnlock()
	if b.logOps {
		start := time.Now()
		defer func() {
			duration := fmt.Sprintf("[bold][yellow][INFO] Storage: Get operation took (%v) to complete", time.Now().Sub(start))
			colorstring.Println(duration)
		}()
	}
	errCh := make(chan error)
	entryCh := make(chan *Entry)

	go func() {
		entry, err := b.GetInternal(ctx, k)
		entryCh <- entry
		errCh <- err
	}()
	for {
		select {
		case ent := <-entryCh:
			{
				return ent, nil
			}
		case logs := <-b.logCh:
			{
				if b.logOps {
					colorstring.Println(logs)
				}
			}
		case err := <-errCh:
			{
				return nil, err

			}

		case <-ctx.Done():
			err := stacktrace.Propagate(ctx.Err(), "[FATAL] Storage: Get operation error ")
			return nil, err
		}
	}
}

// Delete -
func (b *Storage) Delete(ctx context.Context, path string) error {
	if !b.initialized {
		err := stacktrace.NewError("[ERROR] Storage :was not initialized")
		return err
	}
	b.permitPool.Acquire()
	defer b.permitPool.Release()
	b.stateLock.Lock()
	defer b.stateLock.Unlock()
	if b.logOps {
		start := time.Now()
		defer func() {
			duration := fmt.Sprintf("[bold][yellow][INFO] Storage: Delete operation took (%v) to complete", time.Now().Sub(start))
			log.Println(duration)
		}()
	}
	errCh := make(chan error)
	go func() {
		errCh <- b.DeleteInternal(ctx, path)
	}()

	for {
		select {
		case err := <-errCh:
			{
				if err != nil {
					return err
				}
				return nil
			}
		case logs := <-b.logCh:
			{
				if b.logOps {
					colorstring.Println(logs)
				}
			}
		case <-ctx.Done():
			{

				return stacktrace.NewError("[ERROR] Storage: Delete operation timeout ")
			}
		}
	}
}

// List -
func (b *Storage) List(ctx context.Context, prefix string) ([]string, error) {
	if !b.initialized {
		err := stacktrace.NewError("[ERROR] Storage :was not initialized")
		return nil, err

	}
	b.permitPool.Acquire()
	defer b.permitPool.Release()

	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	if b.logOps {
		start := time.Now()
		defer func() {
			duration := fmt.Sprintf("[INFO] Storage: List operation took (%v) to complete", time.Now().Sub(start))
			log.Println(duration)
		}()
	}
	outCh := make(chan []string)
	errCh := make(chan error)
	go func() {
		out, err := b.ListInternal(ctx, prefix)
		outCh <- out
		errCh <- err
	}()
	for {
		select {
		case out := <-outCh:
			{
				return out, nil
			}
		case err := <-errCh:
			{
				err = stacktrace.Propagate(err, "[ERROR] Storage: List operation error ")
				return nil, err
			}
		case logs := <-b.logCh:
			{
				if b.logOps {
					colorstring.Println(logs)
				}
			}
		case <-ctx.Done():
			err := stacktrace.NewError("[ERROR] Storage: List operation timeout")
			return nil, err
		}
	}
}
