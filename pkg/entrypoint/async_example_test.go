package entrypoint_test

import (
	"context"
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/entrypoint"
)

// ExampleFutureT_Await demonstrates async/await-style syntax
func ExampleFutureT_Await() {
	// Create a type-safe future
	promise := entrypoint.NewPromiseT[string]()

	// Simulate async operation
	go func() {
		time.Sleep(50 * time.Millisecond)
		promise.Complete("Hello, World!")
	}()

	// Await the result (async/await style)
	ctx := context.Background()
	result, err := promise.Await(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println(result)
	// Output: Hello, World!
}

// ExampleThen demonstrates Node.js Promise.then() style chaining
func ExampleThen() {
	promise := entrypoint.NewPromiseT[int]()

	// Complete with initial value
	go func() {
		time.Sleep(10 * time.Millisecond)
		promise.Complete(10)
	}()

	ctx := context.Background()

	// Chain transformations (Promise.then() style)
	result := entrypoint.Then(promise, func(n int) (string, error) {
		return fmt.Sprintf("Number: %d", n*2), nil
	})

	final, err := result.Await(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println(final)
	// Output: Number: 20
}

// ExampleCatch demonstrates Node.js Promise.catch() style error handling
func ExampleCatch() {
	promise := entrypoint.NewPromiseT[string]()

	// Simulate error
	go func() {
		time.Sleep(10 * time.Millisecond)
		promise.Fail(fmt.Errorf("operation failed"))
	}()

	ctx := context.Background()

	// Catch and recover from error
	recovered := entrypoint.Catch(promise, func(err error) (string, error) {
		return "Recovered: " + err.Error(), nil
	})

	result, err := recovered.Await(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println(result)
	// Output: Recovered: operation failed
}

// ExampleRequestAsync demonstrates Vert.x request-reply with async/await
func ExampleRequestAsync() {
	// Setup EventBus
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	eventBus := gocmd.EventBus()

	// Register handler
	consumer := eventBus.Consumer("service.address")
	consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
		// Echo the request
		return msg.Reply(msg.Body())
	})

	// Request with async/await style
	requestData := map[string]interface{}{"name": "Fluxor"}
	future := entrypoint.RequestAsync[map[string]interface{}](
		eventBus,
		ctx,
		"service.address",
		requestData,
		5*time.Second,
	)

	result, err := future.Await(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Received: %v\n", result["name"])
	// Output: Received: Fluxor
}

// ExampleAll demonstrates Promise.all() style - wait for all futures
func ExampleAll() {
	// Create multiple promises
	p1 := entrypoint.NewPromiseT[int]()
	p2 := entrypoint.NewPromiseT[int]()
	p3 := entrypoint.NewPromiseT[int]()

	// Complete them asynchronously
	go func() {
		time.Sleep(10 * time.Millisecond)
		p1.Complete(1)
	}()
	go func() {
		time.Sleep(20 * time.Millisecond)
		p2.Complete(2)
	}()
	go func() {
		time.Sleep(30 * time.Millisecond)
		p3.Complete(3)
	}()

	// Wait for all to complete
	ctx := context.Background()
	all := entrypoint.All(ctx, p1, p2, p3)

	results, err := all.Await(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Results: %v\n", results)
	// Output: Results: [1 2 3]
}

// ExampleRace demonstrates Promise.race() style - first to complete wins
func ExampleRace() {
	// Create multiple promises
	p1 := entrypoint.NewPromiseT[string]()
	p2 := entrypoint.NewPromiseT[string]()
	p3 := entrypoint.NewPromiseT[string]()

	// Complete them at different times
	go func() {
		time.Sleep(100 * time.Millisecond)
		p1.Complete("First")
	}()
	go func() {
		time.Sleep(50 * time.Millisecond) // This will win
		p2.Complete("Second")
	}()
	go func() {
		time.Sleep(150 * time.Millisecond)
		p3.Complete("Third")
	}()

	// Race - first to complete wins
	ctx := context.Background()
	race := entrypoint.Race(ctx, p1, p2, p3)

	result, err := race.Await(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println(result)
	// Output: Second
}

// ExampleRequestAsync_combined demonstrates combining Vert.x and Node.js patterns
func ExampleRequestAsync_combined() {
	// Vert.x style: EventBus request-reply
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	eventBus := gocmd.EventBus()

	// Register service handler
	consumer := eventBus.Consumer("user.service")
	consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
		// Process and return user data
		userData := map[string]interface{}{
			"id":   123,
			"name": "John Doe",
		}
		return msg.Reply(userData)
	})

	// Node.js style: async/await with type safety
	userFuture := entrypoint.RequestAsync[map[string]interface{}](
		eventBus,
		ctx,
		"user.service",
		map[string]interface{}{"id": 123},
		5*time.Second,
	)

	// Chain transformation (Promise.then() style) - can be done before or after await
	nameFuture := entrypoint.Then(userFuture, func(data map[string]interface{}) (string, error) {
		if name, ok := data["name"].(string); ok {
			return "Hello, " + name, nil
		}
		return "", fmt.Errorf("name not found")
	})

	greeting, err := nameFuture.Await(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println(greeting)
	// Output: Hello, John Doe
}
