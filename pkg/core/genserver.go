package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CallHandler handles a synchronous call. Returns new state, reply, and optional error.
type CallHandler func(ctx FluxorContext, state interface{}, msg interface{}) (newState interface{}, reply interface{}, err error)

// CastHandler handles an asynchronous cast. Returns new state (reply is fire-and-forget).
type CastHandler func(ctx FluxorContext, state interface{}, msg interface{}) (newState interface{})

// GenServerVerticle is a verticle that maintains state and handles synchronous calls and asynchronous casts
// (GenServer-like behavior). All handlers run on the verticle's event loop; state is updated sequentially.
type GenServerVerticle struct {
	*BaseVerticle
	name         string
	initState    interface{}
	callHandlers map[string]CallHandler
	castHandlers map[string]CastHandler
	mu           sync.Mutex
	state        interface{}
}

// NewGenServerVerticle creates a GenServer verticle with the given name and initial state.
func NewGenServerVerticle(name string, initState interface{}) *GenServerVerticle {
	return &GenServerVerticle{
		BaseVerticle:  NewBaseVerticle(name),
		name:          name,
		initState:     initState,
		state:         initState,
		callHandlers:  make(map[string]CallHandler),
		castHandlers:  make(map[string]CastHandler),
	}
}

// RegisterCall registers a handler for synchronous calls to the given address.
func (gs *GenServerVerticle) RegisterCall(address string, handler CallHandler) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.callHandlers[address] = handler
}

// RegisterCast registers a handler for asynchronous casts to the given address.
func (gs *GenServerVerticle) RegisterCast(address string, handler CastHandler) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.castHandlers[address] = handler
}

// callResult holds the result of a Call for channel delivery.
type callResult struct {
	reply interface{}
	err   error
}

// Call sends a synchronous message to the given address and waits for a reply or timeout.
// Must be called from outside the verticle (e.g. from another verticle or main).
// The handler runs on the GenServer's event loop.
func (gs *GenServerVerticle) Call(ctx FluxorContext, address string, msg interface{}, timeout time.Duration) (interface{}, error) {
	if gs.eventLoop == nil {
		return nil, &EventBusError{Code: "NOT_STARTED", Message: "GenServer not started"}
	}
	replyCh := make(chan callResult, 1)
	task := &genserverCallTask{
		gs:      gs,
		address: address,
		msg:     msg,
		replyCh: replyCh,
	}
	if err := gs.eventLoop.Submit(task); err != nil {
		return nil, err
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case res := <-replyCh:
		return res.reply, res.err
	case <-timer.C:
		return nil, &EventBusError{Code: "CALL_TIMEOUT", Message: fmt.Sprintf("Call %s timed out after %v", address, timeout)}
	case <-ctx.Context().Done():
		return nil, ctx.Context().Err()
	}
}

// Cast sends an asynchronous message to the given address (fire-and-forget).
func (gs *GenServerVerticle) Cast(ctx FluxorContext, address string, msg interface{}) error {
	if gs.eventLoop == nil {
		return &EventBusError{Code: "NOT_STARTED", Message: "GenServer not started"}
	}
	task := &genserverCastTask{
		gs:      gs,
		address: address,
		msg:     msg,
	}
	return gs.eventLoop.Submit(task)
}

// State returns the current state (for use inside handlers only; not safe to call from outside the event loop).
func (gs *GenServerVerticle) State() interface{} {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return gs.state
}

type genserverCallTask struct {
	gs      *GenServerVerticle
	address string
	msg     interface{}
	replyCh chan callResult
}

func (t *genserverCallTask) Name() string { return "genserver.call." + t.address }

func (t *genserverCallTask) Execute(ctx context.Context) error {
	gs := t.gs
	gs.mu.Lock()
	handler := gs.callHandlers[t.address]
	state := gs.state
	fluxorCtx := gs.ctx
	gs.mu.Unlock()

	if handler == nil {
		t.replyCh <- callResult{nil, &EventBusError{Code: "NO_HANDLER", Message: "no call handler for " + t.address}}
		return nil
	}
	newState, reply, err := handler(fluxorCtx, state, t.msg)
	gs.mu.Lock()
	if newState != nil {
		gs.state = newState
	}
	gs.mu.Unlock()
	t.replyCh <- callResult{reply, err}
	return nil
}

type genserverCastTask struct {
	gs      *GenServerVerticle
	address string
	msg     interface{}
}

func (t *genserverCastTask) Name() string { return "genserver.cast." + t.address }

func (t *genserverCastTask) Execute(ctx context.Context) error {
	gs := t.gs
	gs.mu.Lock()
	handler := gs.castHandlers[t.address]
	state := gs.state
	fluxorCtx := gs.ctx
	gs.mu.Unlock()

	if handler != nil {
		newState := handler(fluxorCtx, state, t.msg)
		gs.mu.Lock()
		if newState != nil {
			gs.state = newState
		}
		gs.mu.Unlock()
	}
	return nil
}
