package eventbus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/core/concurrency"
	"github.com/fluxorio/fluxor/pkg/core/eventloop"
	"github.com/fluxorio/fluxor/pkg/core/failfast"
	"github.com/nats-io/nats.go"
)

// ClusterNATSConfig configures the NATS-backed (cluster) EventBus.
type ClusterNATSConfig struct {
	// URL is the NATS server URL, e.g. "nats://127.0.0.1:4222".
	URL string

	// Prefix is prepended to all subjects. Default: "fluxor".
	Prefix string

	// Name is an optional NATS connection name.
	Name string

	// RequestTimeout is the default timeout used by Request when timeout==0.
	RequestTimeout time.Duration

	// ExecutorConfig controls bounded handler execution.
	// If zero, defaults are used.
	ExecutorConfig concurrency.ExecutorConfig

	// UseEventLoopGroup enables EventLoopGroup for CPU-based routing
	// Default: false (backward compatible)
	UseEventLoopGroup bool

	// EventLoopConfig configures EventLoopGroup (only used if UseEventLoopGroup is true)
	EventLoopConfig eventloop.EventLoopConfig
}

// NewClusterEventBusNATS creates a clustered EventBus backed by NATS.
//
// Address mapping:
// - Publish: <prefix>.pub.<address>
// - Send:    <prefix>.send.<address> (queue group: same subject)
// - Request: <prefix>.req.<address>  (queue group: same subject)
func NewClusterEventBusNATS(ctx context.Context, gocmd core.GoCMD, cfg ClusterNATSConfig) (core.EventBus, error) {
	if ctx == nil {
		return nil, fmt.Errorf("ctx cannot be nil")
	}
	if gocmd == nil {
		return nil, fmt.Errorf("gocmd cannot be nil")
	}

	url := cfg.URL
	if url == "" {
		url = nats.DefaultURL
	}
	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "fluxor"
	}
	reqTimeout := cfg.RequestTimeout
	if reqTimeout <= 0 {
		reqTimeout = 5 * time.Second
	}

	execCfg := cfg.ExecutorConfig
	if execCfg.Workers == 0 && execCfg.QueueSize == 0 {
		execCfg = concurrency.DefaultExecutorConfig()
		execCfg.Workers = 16
		execCfg.QueueSize = 4096
	}

	nc, err := nats.Connect(url, func(o *nats.Options) error {
		if cfg.Name != "" {
			o.Name = cfg.Name
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	executor := concurrency.NewExecutor(ctx, execCfg)

	eb := &clusterNATSEventBus{
		ctx:            ctx,
		gocmd:          gocmd,
		nc:             nc,
		prefix:         prefix,
		requestTimeout: reqTimeout,
		executor:       executor,
		logger:         core.NewDefaultLogger(),
	}

	// Create EventLoopGroup if enabled
	if cfg.UseEventLoopGroup {
		loopConfig := cfg.EventLoopConfig
		if loopConfig.NumLoops == 0 && loopConfig.QueueSize == 0 {
			loopConfig = eventloop.DefaultEventLoopConfig()
		}
		loopGroup, err := eventloop.NewEventLoopGroup(ctx, loopConfig)
		if err != nil {
			eb.logger.Error(fmt.Sprintf("Failed to create EventLoopGroup: %v, falling back to Executor-only", err))
		} else {
			eb.loopGroup = loopGroup
		}
	}

	return eb, nil
}

type clusterNATSEventBus struct {
	ctx   context.Context
	gocmd core.GoCMD

	nc *nats.Conn

	prefix         string
	requestTimeout time.Duration

	executor  concurrency.Executor
	loopGroup *eventloop.EventLoopGroup // EventLoopGroup for CPU-based routing (optional)
	logger    core.Logger
}

func (eb *clusterNATSEventBus) PublishWithContext(ctx context.Context, address string, body interface{}) error {
	if err := core.ValidateAddress(address); err != nil {
		return err
	}
	if err := core.ValidateBody(body); err != nil {
		return err
	}
	if ctx == nil {
		ctx = eb.ctx // Fallback to EventBus context
	}

	data, err := encodeBody(body)
	if err != nil {
		return err
	}

	msg := &nats.Msg{
		Subject: eb.subjectPub(address),
		Data:    data,
		Header:  nats.Header{},
	}
	// Extract FloxID and RequestID from provided context
	if floxid := core.GetFloxID(ctx); floxid != "" {
		msg.Header.Set("X-Flox-ID", floxid)
	}
	if rid := core.GetRequestID(ctx); rid != "" {
		msg.Header.Set("X-Request-ID", rid)
	}

	return eb.nc.PublishMsg(msg)
}

func (eb *clusterNATSEventBus) Publish(address string, body interface{}) error {
	if err := core.ValidateAddress(address); err != nil {
		return err
	}
	if err := core.ValidateBody(body); err != nil {
		return err
	}

	data, err := encodeBody(body)
	if err != nil {
		return err
	}

	msg := &nats.Msg{
		Subject: eb.subjectPub(address),
		Data:    data,
		Header:  nats.Header{},
	}
	if rid := core.GetRequestID(eb.ctx); rid != "" {
		msg.Header.Set("X-Request-ID", rid)
	}

	return eb.nc.PublishMsg(msg)
}

func (eb *clusterNATSEventBus) Send(address string, body interface{}) error {
	if err := core.ValidateAddress(address); err != nil {
		return err
	}
	if err := core.ValidateBody(body); err != nil {
		return err
	}

	data, err := encodeBody(body)
	if err != nil {
		return err
	}

	msg := &nats.Msg{
		Subject: eb.subjectSend(address),
		Data:    data,
		Header:  nats.Header{},
	}
	if rid := core.GetRequestID(eb.ctx); rid != "" {
		msg.Header.Set("X-Request-ID", rid)
	}

	return eb.nc.PublishMsg(msg)
}

func (eb *clusterNATSEventBus) SendWithContext(ctx context.Context, address string, body interface{}, timeout time.Duration) (core.Message, error) {
	// Use Request method which already handles headers
	// We'll update Request to use context if provided, but for now delegate to Request
	// Note: This is a simplified implementation - full request-reply with context requires correlation ID tracking
	return eb.Request(address, body, timeout)
}

func (eb *clusterNATSEventBus) Request(address string, body interface{}, timeout time.Duration) (core.Message, error) {
	if err := core.ValidateAddress(address); err != nil {
		return nil, err
	}
	if err := core.ValidateBody(body); err != nil {
		return nil, err
	}
	if err := core.ValidateTimeout(timeout); err != nil {
		return nil, err
	}

	data, err := encodeBody(body)
	if err != nil {
		return nil, err
	}

	if timeout <= 0 {
		timeout = eb.requestTimeout
	}

	msg := &nats.Msg{
		Subject: eb.subjectReq(address),
		Data:    data,
		Header:  nats.Header{},
	}
	if rid := core.GetRequestID(eb.ctx); rid != "" {
		msg.Header.Set("X-Request-ID", rid)
	}

	resp, err := eb.nc.RequestMsg(msg, timeout)
	if err != nil {
		return nil, err
	}

	h := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			h[k] = v[0]
		}
	}

	return &clusterNATSMessage{
		body:         resp.Data,
		headers:      h,
		replySubject: "",
		eb:           eb,
	}, nil
}

func (eb *clusterNATSEventBus) Consumer(address string) core.Consumer {
	// Fail-fast: keep contract consistent with in-memory EventBus.
	// Invalid address is a programmer error and should be caught in dev.
	if err := core.ValidateAddress(address); err != nil {
		failfast.Err(err)
	}
	// Create consumer object. Handler() will create subscriptions.
	return newClusterNATSConsumer(address, eb)
}

func (eb *clusterNATSEventBus) Close() error {
	// Drain executor and NATS.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = eb.executor.Shutdown(ctx)
	_ = eb.nc.Drain()
	eb.nc.Close()
	return nil
}

func (eb *clusterNATSEventBus) subjectPub(address string) string {
	return eb.prefix + ".pub." + address
}
func (eb *clusterNATSEventBus) subjectSend(address string) string {
	return eb.prefix + ".send." + address
}
func (eb *clusterNATSEventBus) subjectReq(address string) string {
	return eb.prefix + ".req." + address
}

type clusterNATSConsumer struct {
	address string
	eb      *clusterNATSEventBus

	mu         sync.Mutex
	handler    core.MessageHandler
	subs       []*nats.Subscription
	completion chan struct{}
	registered bool
}

func newClusterNATSConsumer(address string, eb *clusterNATSEventBus) *clusterNATSConsumer {
	return &clusterNATSConsumer{
		address:    address,
		eb:         eb,
		completion: make(chan struct{}),
	}
}

func (c *clusterNATSConsumer) Handler(handler core.MessageHandler) core.Consumer {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handler = handler
	if c.registered {
		return c
	}
	c.registered = true

	// Subscribe to publish subject (fanout)
	pubSub, err := c.eb.nc.Subscribe(c.eb.subjectPub(c.address), c.onMsg(""))
	if err == nil {
		c.subs = append(c.subs, pubSub)
	}

	// Subscribe to send subject (queue group for point-to-point)
	sendSubject := c.eb.subjectSend(c.address)
	sendSub, err := c.eb.nc.QueueSubscribe(sendSubject, sendSubject, c.onMsg(""))
	if err == nil {
		c.subs = append(c.subs, sendSub)
	}

	// Subscribe to request subject (queue group; replies are handled via msg.Reply)
	reqSubject := c.eb.subjectReq(c.address)
	reqSub, err := c.eb.nc.QueueSubscribe(reqSubject, reqSubject, c.onMsg(""))
	if err == nil {
		c.subs = append(c.subs, reqSub)
	}

	return c
}

func (c *clusterNATSConsumer) Completion() <-chan struct{} { return c.completion }

func (c *clusterNATSConsumer) Unregister() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, s := range c.subs {
		_ = s.Unsubscribe()
	}
	c.subs = nil

	select {
	case <-c.completion:
	default:
		close(c.completion)
	}
	return nil
}

func (c *clusterNATSConsumer) onMsg(_ string) nats.MsgHandler {
	return func(nm *nats.Msg) {
		// If EventLoopGroup is enabled, dispatch through it
		if c.eb.loopGroup != nil {
			// Extract headers for key extraction
			headers := make(map[string]string)
			for k, v := range nm.Header {
				if len(v) > 0 {
					headers[k] = v[0]
				}
			}

			// Create event for EventLoopGroup
			event := &eventloop.Event{
				Key:     "", // Will be extracted by dispatcher from headers
				Address: c.address,
				Body:    nm.Data,
				Headers: headers,
				Handler: func(ctx context.Context, ev *eventloop.Event) error {
					// Forward to handler via executor
					task := concurrency.NewNamedTask(
						"cluster-nats-consumer."+c.address,
						func(ctx context.Context) error {
							return c.handleMsg(nm)
						},
					)
					return c.eb.executor.Submit(task)
				},
			}

			// Dispatch to EventLoopGroup
			if err := c.eb.loopGroup.Dispatch(c.eb.ctx, event); err != nil {
				c.eb.logger.Info(fmt.Sprintf("cluster consumer overloaded for %s: %v", c.address, err))
			}
			return
		}

		// Original flow: direct executor submission
		// Bounded execution through executor; drop if queue is full.
		task := concurrency.NewNamedTask(
			"cluster-nats-consumer."+c.address,
			func(ctx context.Context) error {
				return c.handleMsg(nm)
			},
		)
		if err := c.eb.executor.Submit(task); err != nil {
			c.eb.logger.Info(fmt.Sprintf("cluster consumer overloaded for %s: %v", c.address, err))
		}
	}
}

func (c *clusterNATSConsumer) handleMsg(nm *nats.Msg) error {
	c.mu.Lock()
	h := c.handler
	c.mu.Unlock()
	if h == nil {
		return nil
	}

	// Build context and propagate request ID if present.
	base := c.eb.ctx
	if rid := nm.Header.Get("X-Request-ID"); rid != "" {
		base = core.WithRequestID(base, rid)
	}
	fctx := core.NewFluxorContext(base, c.eb.gocmd)

	headers := make(map[string]string)
	for k, v := range nm.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	msg := &clusterNATSMessage{
		body:         nm.Data,
		headers:      headers,
		replySubject: nm.Reply,
		eb:           c.eb,
	}

	return h(fctx, msg)
}

type clusterNATSMessage struct {
	mu           sync.RWMutex
	body         interface{}
	headers      map[string]string
	replySubject string
	eb           *clusterNATSEventBus
}

func (m *clusterNATSMessage) Body() interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.body
}

func (m *clusterNATSMessage) Headers() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]string, len(m.headers))
	for k, v := range m.headers {
		out[k] = v
	}
	return out
}

func (m *clusterNATSMessage) ReplyAddress() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.replySubject
}

func (m *clusterNATSMessage) Reply(body interface{}) error {
	if m.replySubject == "" {
		return core.ErrNoReplyAddress
	}

	data, err := encodeBody(body)
	if err != nil {
		return err
	}

	reply := &nats.Msg{
		Subject: m.replySubject,
		Data:    data,
		Header:  nats.Header{},
	}
	if rid := core.GetRequestID(m.eb.ctx); rid != "" {
		reply.Header.Set("X-Request-ID", rid)
	}

	return m.eb.nc.PublishMsg(reply)
}

func (m *clusterNATSMessage) DecodeBody(v interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, ok := m.body.([]byte)
	if !ok {
		return fmt.Errorf("body is not []byte, got %T", m.body)
	}
	return core.JSONDecode(data, v)
}

func (m *clusterNATSMessage) Fail(failureCode int, message string) error {
	return m.Reply(map[string]interface{}{
		"failureCode": failureCode,
		"message":     message,
	})
}

func encodeBody(body interface{}) ([]byte, error) {
	if bodyBytes, ok := body.([]byte); ok {
		return bodyBytes, nil
	}
	return core.JSONEncode(body)
}
