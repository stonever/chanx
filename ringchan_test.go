package chanx

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestMakeRingChan(t *testing.T) {
	size := 10
	ch := NewRingChan[int](size)
	assert.Equal(t, 0, ch.Len())

	for i := 0; i < 12; i++ {
		ch.In <- i
	}
	assert.Equal(t, size, ch.Len())

	for i := 0; i < 10; i++ {
		select {
		case out := <-ch.Out:
			assert.Equal(t, i+2, out)
		}
	}
	time.Sleep(time.Millisecond)
	assert.Equal(t, 0, ch.Len())

	t.Log("OK")

}
func TestRWRingChan(t *testing.T) {
	size := 10
	ch := NewRingChan[int](size)
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
