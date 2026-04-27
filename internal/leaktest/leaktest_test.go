package leaktest

import (
	"sync"
	"testing"
	"time"
)

func TestAssertStable_NoLeakAfterWork(t *testing.T) {
	base := Baseline()
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(5 * time.Millisecond)
		}()
	}
	wg.Wait()
	AssertStable(t, base, 16, 3*time.Second)
}
