package ringbuffer

import (
	"errors"
	"sync"
)

var (
	ErrIsFull       = errors.New("ringbuffer is full")
	ErrIsEmpty      = errors.New("ringbuffer is empty")
	ErrAccuqireLock = errors.New("no lock to accquire")
)

// RingBuffer is a circular buffer that implement io.ReaderWriter interface.
type RingBuffer[T any] struct {
	buf       []T
	size      int
	r         int // next position to read
	w         int // next position to write
	isFull    bool
	mu        sync.Mutex
	coverable bool
}

// New returns a new RingBuffer whose buffer has the given size.
func New[T any](size int, coverable bool) *RingBuffer[T] {
	return &RingBuffer[T]{
		buf:       make([]T, size),
		size:      size,
		coverable: coverable,
	}
}

// ReadByte reads and returns the next byte from the input or ErrIsEmpty.
func (r *RingBuffer[T]) Read() (t T, err error) {
	r.mu.Lock()
	if r.w == r.r && !r.isFull {
		r.mu.Unlock()
		return t, ErrIsEmpty
	}
	t = r.buf[r.r]
	r.r++
	if r.r == r.size {
		r.r = 0
	}

	r.isFull = false
	r.mu.Unlock()
	return t, err
}
func (r *RingBuffer[T]) Peek() (t T, err error) {
	r.mu.Lock()
	if r.w == r.r && !r.isFull {
		r.mu.Unlock()
		return t, ErrIsEmpty
	}

	v := r.buf[r.r]
	r.mu.Unlock()
	return v, nil
}

// WriteByte writes one byte into buffer, and returns ErrIsFull if buffer is full.
func (r *RingBuffer[T]) Write(t T) error {
	r.mu.Lock()
	err := r.write(t)
	r.mu.Unlock()
	return err
}

// TryWriteByte writes one byte into buffer without blocking.
// If it has not succeeded to accquire the lock, it return ErrAccuqireLock.
func (r *RingBuffer[T]) TryWrite(t T) error {
	ok := r.mu.TryLock()
	if !ok {
		return ErrAccuqireLock
	}

	err := r.write(t)
	r.mu.Unlock()
	return err
}

func (r *RingBuffer[T]) write(t T) error {
	if r.w == r.r && r.isFull {
		if !r.coverable {
			return ErrIsFull
		}
		defer func() {
			r.r = r.w
		}()
	}
	r.buf[r.w] = t
	r.w++

	if r.w == r.size {
		r.w = 0
	}
	if r.w == r.r {
		r.isFull = true
	}

	return nil
}

// Length return the length of available read bytes.
func (r *RingBuffer[T]) Length() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.w == r.r {
		if r.isFull {
			return r.size
		}
		return 0
	}

	if r.w > r.r {
		return r.w - r.r
	}

	return r.size - r.r + r.w
}

// Capacity returns the size of the underlying buffer.
func (r *RingBuffer[T]) Capacity() int {
	return r.size
}

// Free returns the length of available bytes to write.
func (r *RingBuffer[T]) Free() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.w == r.r {
		if r.isFull {
			return 0
		}
		return r.size
	}

	if r.w < r.r {
		return r.r - r.w
	}

	return r.size - r.w + r.r
}

// All returns all available read bytes. It does not move the read pointer and only copy the available data.
func (r *RingBuffer[T]) All() []T {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.w == r.r {
		if r.isFull {
			buf := make([]T, r.size)
			copy(buf, r.buf[r.r:])
			copy(buf[r.size-r.r:], r.buf[:r.w])
			return buf
		}
		return nil
	}

	if r.w > r.r {
		buf := make([]T, r.w-r.r)
		copy(buf, r.buf[r.r:r.w])
		return buf
	}

	n := r.size - r.r + r.w
	buf := make([]T, n)

	if r.r+n < r.size {
		copy(buf, r.buf[r.r:r.r+n])
	} else {
		c1 := r.size - r.r
		copy(buf, r.buf[r.r:r.size])
		c2 := n - c1
		copy(buf[c1:], r.buf[0:c2])
	}

	return buf
}

// IsFull returns this ringbuffer is full.
func (r *RingBuffer[T]) IsFull() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.isFull
}

// IsEmpty returns this ringbuffer is empty.
func (r *RingBuffer[T]) IsEmpty() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return !r.isFull && r.w == r.r
}

// Reset the read pointer and writer pointer to zero.
func (r *RingBuffer[T]) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.r = 0
	r.w = 0
	r.isFull = false
}
