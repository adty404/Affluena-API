package async

import (
	"context"
	"log"
	"runtime/debug"
)

// SafeGo runs a function in a new goroutine and recovers from panics.
// It ensures that background tasks do not crash the entire application.
func SafeGo(ctx context.Context, name string, fn func(context.Context)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in async task [%s]: %v\nStack trace:\n%s", name, r, string(debug.Stack()))
			}
		}()
		// We use the provided context. If the caller passes the request context,
		// it might be canceled when the request ends. Callers should be careful
		// to pass context.Background() if they want the task to outlive the request.
		fn(ctx)
	}()
}
