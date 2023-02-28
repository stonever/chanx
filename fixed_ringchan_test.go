package chanx

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"sync"
	"testing"
	"time"
)

func TestMakeFixedRingChan(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	size := 10
	ch := NewFixedRingChan[int](ctx, size)
	assert.Equal(t, 0, ch.Len())

	for i := 0; i < 100; i++ {
		ch.In <- i
	}
	assert.Equal(t, size, ch.Len())

	for i := 0; i < 10; i++ {
		select {
		case out := <-ch.Out:
			assert.Equal(t, i+90, out)
		}
	}
	time.Sleep(time.Millisecond)
	assert.Equal(t, 0, ch.Len())

	t.Log("OK")

}
func TestRWRingChan(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	size := 10
	ch := NewFixedRingChan[int](ctx, size)
	assert.Equal(t, 0, ch.Len())
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		for i := 0; i < 1000; i++ {
			time.Sleep(time.Millisecond)
			ch.In <- i
		}
		close(ch.In)
		wg.Done()
	}()
	go func() {
		var last int
		time.Sleep(time.Second * 1)

		for v := range ch.Out {
			fmt.Println(v)
			last = v
		}
		assert.Equal(t, last, 999)
		wg.Done()

	}()
	wg.Wait()
	t.Log("OK")
}
