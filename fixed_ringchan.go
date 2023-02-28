package chanx

import (
	"context"
)

// FixedRingChan is a bounded chan with ringbuffer.
// In is used to write without blocking, which supports multiple writers.
// and Out is used to read, which supports multiple readers.
type FixedRingChan[T any] struct {
	In     chan<- T            // channel for write
	Out    <-chan T            // channel for read
	buffer *FixedRingBuffer[T] // buffer
}

// Len returns len of In plus len of Out plus len of buffer.
// It is not accurate and only for your evaluating approximate number of elements in this chan,
// see https://github.com/smallnest/chanx/issues/7.
func (c FixedRingChan[T]) Len() int {
	return len(c.In) + c.buffer.Length() + len(c.Out)
}

// NewFixedRingChan is like NewUnboundedChan but you can set initial capacity for In, Out, Buffer.
func NewFixedRingChan[T any](ctx context.Context, size int) *FixedRingChan[T] {
	in := make(chan T, 0)
	out := make(chan T, 0)
	ch := &FixedRingChan[T]{In: in, Out: out, buffer: NewFixedRingBuffer[T](size, true)}

	go processFixedRingChan(ctx, in, out, ch)

	return ch
}

func processFixedRingChan[T any](ctx context.Context, in, out chan T, ch *FixedRingChan[T]) {
	defer close(out)
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	drain := func() {
		for !ch.buffer.IsEmpty() {
			val, err := ch.buffer.Read()
			if err != nil {
				panic(err)
			}
			out <- val
		}

		ch.buffer.Reset()
	}
	for {
		select {
		case <-cctx.Done():
			return
		case val, ok := <-in:
			if !ok { // in is closed
				drain()
				return
			}

			// Write to chan first, write to buffer if can't
			// buffer has some values
			if !ch.buffer.IsEmpty() {
				ch.buffer.Write(val)
			} else {
				// out is not full
				select {
				case out <- val:
					continue
				default:
				}

				// out is full
				ch.buffer.Write(val)
			}

			for !ch.buffer.IsEmpty() {
				peek, err := ch.buffer.Peek()
				if err != nil {
					panic(err)
				}
				select {
				case <-cctx.Done():
					return
				case val, ok := <-in:
					if !ok { // in is closed
						drain()
						return
					}
					ch.buffer.Write(val)
				case out <- peek:
					_, err := ch.buffer.Read()
					if err != nil {
						panic(err)
					}
				}
			}
		}
	}
}
