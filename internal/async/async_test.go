package async

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSafeGo_Success(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	done := false
	SafeGo(context.Background(), "test_success", func(ctx context.Context) {
		defer wg.Done()
		done = true
	})

	wg.Wait()
	if !done {
		t.Errorf("Expected async function to run successfully")
	}
}

func TestSafeGo_Panic(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	SafeGo(context.Background(), "test_panic", func(ctx context.Context) {
		defer wg.Done()
		panic("simulated panic")
	})

	// Wait a bit to ensure the panic doesn't crash the test process
	time.Sleep(100 * time.Millisecond)
	wg.Wait()
	// If the test process is still alive, it means the panic was recovered successfully.
}
