//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"
)

var client EventBusClient

func main() {
	// Export functions to JavaScript
	js.Global().Set("FluxorEventBus", js.ValueOf(map[string]interface{}{
		"connect":     js.FuncOf(connect),
		"publish":     js.FuncOf(publish),
		"send":        js.FuncOf(send),
		"request":     js.FuncOf(request),
		"subscribe":   js.FuncOf(subscribe),
		"unsubscribe": js.FuncOf(unsubscribe),
		"close":       js.FuncOf(closeClient),
	}))

	// Keep the program running
	select {}
}

// connect establishes WebSocket connection
func connect(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return js.ValueOf(map[string]interface{}{
			"error": "wsURL required",
		})
	}

	wsURL := args[0].String()

	// Client will be created in JavaScript bindings
	// This is just a placeholder for WASM exports
	// The actual WebSocket connection is handled by bindings.js
	client = nil

	// Connect WebSocket in JavaScript
	// This will be handled by bindings.js
	return js.ValueOf(map[string]interface{}{
		"success": true,
	})
}

// publish publishes a message
func publish(this js.Value, args []js.Value) interface{} {
	if client == nil {
		return js.ValueOf(map[string]interface{}{
			"error": "not connected",
		})
	}

	if len(args) < 2 {
		return js.ValueOf(map[string]interface{}{
			"error": "address and body required",
		})
	}

	address := args[0].String()
	body := args[1]

	// Convert JS value to Go interface
	var bodyInterface interface{}
	if err := convertJSValue(body, &bodyInterface); err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("invalid body: %v", err),
		})
	}

	err := client.Publish(address, bodyInterface)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": err.Error(),
		})
	}

	return js.ValueOf(map[string]interface{}{
		"success": true,
	})
}

// send sends a point-to-point message
func send(this js.Value, args []js.Value) interface{} {
	if client == nil {
		return js.ValueOf(map[string]interface{}{
			"error": "not connected",
		})
	}

	if len(args) < 2 {
		return js.ValueOf(map[string]interface{}{
			"error": "address and body required",
		})
	}

	address := args[0].String()
	body := args[1]

	var bodyInterface interface{}
	if err := convertJSValue(body, &bodyInterface); err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("invalid body: %v", err),
		})
	}

	err := client.Send(address, bodyInterface)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": err.Error(),
		})
	}

	return js.ValueOf(map[string]interface{}{
		"success": true,
	})
}

// request sends a request and waits for reply
func request(this js.Value, args []js.Value) interface{} {
	if client == nil {
		return js.ValueOf(map[string]interface{}{
			"error": "not connected",
		})
	}

	if len(args) < 2 {
		return js.ValueOf(map[string]interface{}{
			"error": "address and body required",
		})
	}

	address := args[0].String()
	body := args[1]
	timeout := 5 * time.Second

	if len(args) >= 3 {
		if timeoutMs := args[2].Int(); timeoutMs > 0 {
			timeout = time.Duration(timeoutMs) * time.Millisecond
		}
	}

	var bodyInterface interface{}
	if err := convertJSValue(body, &bodyInterface); err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("invalid body: %v", err),
		})
	}

	// Request is handled by JavaScript bindings
	// Return a promise that will be resolved by bindings.js
	return js.Undefined()
}

// subscribe subscribes to an address
func subscribe(this js.Value, args []js.Value) interface{} {
	if client == nil {
		return js.ValueOf(map[string]interface{}{
			"error": "not connected",
		})
	}

	if len(args) < 2 {
		return js.ValueOf(map[string]interface{}{
			"error": "address and handler required",
		})
	}

	address := args[0].String()
	handler := args[1]

	if !handler.Type().Equal(js.TypeFunction) {
		return js.ValueOf(map[string]interface{}{
			"error": "handler must be a function",
		})
	}

	// Subscription is handled by JavaScript bindings
	// This is just a placeholder

	return js.ValueOf(map[string]interface{}{
		"success": true,
	})
}

// unsubscribe unsubscribes from an address
func unsubscribe(this js.Value, args []js.Value) interface{} {
	if client == nil {
		return js.ValueOf(map[string]interface{}{
			"error": "not connected",
		})
	}

	if len(args) < 1 {
		return js.ValueOf(map[string]interface{}{
			"error": "address required",
		})
	}

	// Unsubscribe is handled by JavaScript bindings
	address := args[0].String()
	var err error
	err = nil

	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": err.Error(),
		})
	}

	return js.ValueOf(map[string]interface{}{
		"success": true,
	})
}

// closeClient closes the client
func closeClient(this js.Value, args []js.Value) interface{} {
	if client == nil {
		return js.ValueOf(map[string]interface{}{
			"error": "not connected",
		})
	}

	err := client.Close()
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": err.Error(),
		})
	}

	return js.ValueOf(map[string]interface{}{
		"success": true,
	})
}

// convertJSValue converts JS value to Go interface
func convertJSValue(jsVal js.Value, result interface{}) error {
	if jsVal.Type() == js.TypeObject {
		jsonStr := js.Global().Get("JSON").Call("stringify", jsVal).String()
		return json.Unmarshal([]byte(jsonStr), result)
	}

	// Handle primitive types
	switch jsVal.Type() {
	case js.TypeString:
		*result.(*interface{}) = jsVal.String()
	case js.TypeNumber:
		*result.(*interface{}) = jsVal.Float()
	case js.TypeBoolean:
		*result.(*interface{}) = jsVal.Bool()
	case js.TypeNull, js.TypeUndefined:
		*result.(*interface{}) = nil
	default:
		return fmt.Errorf("unsupported JS type: %v", jsVal.Type())
	}

	return nil
}

// convertToJSValue converts Go value to JS value
func convertToJSValue(val interface{}) js.Value {
	jsonBytes, err := json.Marshal(val)
	if err != nil {
		return js.Undefined()
	}

	jsonStr := string(jsonBytes)
	return js.Global().Get("JSON").Call("parse", jsonStr)
}
