package chanx

import "github.com/stonever/chanx/ringbuffer"

// RingChan is a bounded chan with ringbuffer.
// In is used to write without blocking, which supports multiple writers.
// and Out is used to read, which supports multiple readers.
// You can close the in channel if you want.
type RingChan[T any] struct {
	In     chan<- T                  // channel for write
	Out    <-chan T                  // channel for read
	buffer *ringbuffer.RingBuffer[T] // buffer
}

// Len returns len of In plus len of Out plus len of buffer.
// It is not accurate and only for your evaluating approximate number of elements in this chan,
// see https://github.com/smallnest/chanx/issues/7.
func (c RingChan[T]) Len() int {
	return len(c.In) + c.buffer.Length() + len(c.Out)
}

// NewRingChan is like NewUnboundedChan but you can set initial capacity for In, Out, Buffer.
func NewRingChan[T any](size int) *RingChan[T] {
	in := make(chan T, 0)
	out := make(chan T, 0)
	ch := RingChan[T]{In: in, Out: out, buffer: ringbuffer.New[T](size, true)}

	go processRingChan(in, out, ch.buffer)

	return &ch
}

func processRingChan[T any](in, out chan T, buffer *ringbuffer.RingBuffer[T]) {
	defer close(out)
loop:
	for {
		val, ok := <-in
		if !ok { // in is closed
			break loop
		}

		// make sure values' order
		// buffer has some values
		if buffer.Length() > 0 {
			err := buffer.Write(val)
			if err != nil {
				panic(err)
			}
		} else {
			// out is not full and buffer is empty
			// put value in out instead of buffer
			select {
			case out <- val:
				continue
			default:
			}

			// out is full
			err := buffer.Write(val)
			if err != nil {
				panic(err)
			}
		}

		for !buffer.IsEmpty() {
			peek, err := buffer.Peek()
			if err != nil {
				panic(err)
			}
			select {
			case val, ok := <-in:
				if !ok { // in is closed
					break loop
				}
				err := buffer.Write(val)
				if err != nil {
					panic(err)
				}
			case out <- peek:
				// there get length not safe
				_, err := buffer.Read()
				if err != nil {
					panic(err)
				}
				if buffer.IsEmpty() { // after burst
					buffer.Reset()
				}
			}
		}
	}
	// after in is closed, there can be reached
	// drain
	for !buffer.IsEmpty() {
		t, err := buffer.Read()
		if err != nil {
			panic(err)
		}
		out <- t
	}
	buffer.Reset()
}
