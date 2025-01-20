package ws

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func ShouldBeClosedByTimeout(t *testing.T, ch chan struct{}) {
	timer := time.NewTimer(time.Second)
	select {
	case _, ok := <-ch:
		assert.False(t, ok, "channel should be closed")
	case <-timer.C:
		assert.Fail(t, "channel should not be blocked this long!")
	}
}

func getWSURLFromHTTPURL(url string) string {
	return strings.Replace(url, "http://", "ws://", 1)
}

// waitOrTimeout waits for fn to finish or times out.
// fn must close the done channel when it is done.
func waitOrTimeout(timeout time.Duration, fn func()) bool {
	done := make(chan struct{})
	go func() {
		fn()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}

}

func waitAllOrTimeout(t *testing.T, timeout time.Duration, fns ...struct {
	message string
	fn      func()
}) {
	doneChan := make(chan int, len(fns))
	done := make([]bool, len(fns))

	// Start goroutines
	for i, fn := range fns {
		go func(i int, fn struct {
			message string
			fn      func()
		}) {
			fn.fn()
			doneChan <- i
		}(i, fn) // Pass `i` and `fn` to avoid closure issues
	}

	exited := 0
outer:
	for {
		select {
		case i := <-doneChan:
			exited++
			done[i] = true
			if exited == len(fns) {
				break outer
			}
		case <-time.After(timeout):
			break outer
		}
	}

	// If all goroutines completed
	if exited == len(fns) {
		return
	}

	// Handle timeout
	for i := 0; i < len(fns); i++ {
		if !done[i] { // Assert for incomplete tasks
			assert.Fail(t, fns[i].message)
		}
	}
	assert.FailNow(t, "timeout")
}
