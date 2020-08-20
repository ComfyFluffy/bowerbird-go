package helper

import (
	"io"
	"sync"
)

// LazyReadSeeker builds the ReadSeeker until it is being readed or seeked.
type LazyReadSeeker struct {
	builder func() (io.ReadSeeker, error)
	r       io.ReadSeeker
	err     error
	once    sync.Once
}

func (lrs *LazyReadSeeker) Read(p []byte) (int, error) {
	lrs.once.Do(func() {
		lrs.r, lrs.err = lrs.builder()
	})
	if lrs.err != nil {
		return 0, lrs.err
	}
	return lrs.r.Read(p)
}

// Seek calls the builded ReadSeeker's Seek method.
func (lrs *LazyReadSeeker) Seek(offset int64, whence int) (int64, error) {
	lrs.once.Do(func() {
		lrs.r, lrs.err = lrs.builder()
	})
	if lrs.err != nil {
		return 0, lrs.err
	}
	return lrs.r.Seek(offset, whence)
}

// NewLazyReadSeeker returns a new LazyReadSeeker with builder func.
func NewLazyReadSeeker(builder func() (io.ReadSeeker, error)) *LazyReadSeeker {
	return &LazyReadSeeker{
		builder: builder,
	}
}
