// Package messaging provides a message bus abstraction (KWF-L5H2F FRK-SVC-050/051/052).
// Publisher sends messages; Subscriber receives them; Stream provides
// consumer groups with at-least-once delivery. Backends (NATS JetStream, Kafka)
// implement these interfaces.
package messaging

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Message is a bus message (FRK-SVC-050).
type Message struct {
	Subject string
	Data    []byte
	Meta    map[string]string
}

// Handler processes a message (FRK-SVC-050). Returning an error triggers
// redelivery (FRK-SVC-052).
type Handler func(ctx context.Context, msg Message) error

// Subscription is an active subscription handle (FRK-SVC-050).
type Subscription interface {
	Unsubscribe() error
}

// Publisher sends messages to a subject (FRK-SVC-050).
type Publisher interface {
	Publish(ctx context.Context, subject string, data []byte) error
}

// Subscriber receives messages from a subject (FRK-SVC-050).
type Subscriber interface {
	Subscribe(ctx context.Context, subject string, handler Handler) (Subscription, error)
}

// Stream provides consumer-group semantics (FRK-SVC-050).
type Stream interface {
	Publisher
	Subscriber
	// QueueSubscribe binds a named consumer group with at-least-once delivery.
	QueueSubscribe(ctx context.Context, subject, group string, handler Handler) (Subscription, error)
}

// ErrNoMessages signals no message is available now (FRK-SVC-050).
var ErrNoMessages = errors.New("messaging: no messages")

// MemoryStream is an in-process backend for tests and local dev (FRK-SVC-051).
type MemoryStream struct {
	mu         sync.RWMutex
	subs       map[string][]memorySub
	queueSubs  map[string]map[string][]memorySub
	subCounter int
}

type memorySub struct {
	id      string
	subject string
	handler Handler
	msgCh   chan Message
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewMemoryStream creates an empty in-memory stream.
func NewMemoryStream() *MemoryStream {
	return &MemoryStream{
		subs:      map[string][]memorySub{},
		queueSubs: map[string]map[string][]memorySub{},
	}
}

// Publish delivers a message to all matching subscribers (FRK-SVC-050).
func (s *MemoryStream) Publish(ctx context.Context, subject string, data []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msg := Message{Subject: subject, Data: data}
	for _, sub := range s.subs[subject] {
		select {
		case sub.msgCh <- msg:
		case <-sub.ctx.Done():
		}
	}
	for _, group := range s.queueSubs[subject] {
		// round-robin: pick first available consumer in group
		if len(group) > 0 {
			select {
			case group[0].msgCh <- msg:
			case <-group[0].ctx.Done():
			}
		}
	}
	return nil
}

// Subscribe registers a handler for a subject (FRK-SVC-050).
func (s *MemoryStream) Subscribe(ctx context.Context, subject string, handler Handler) (Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subCounter++
	id := fmt.Sprintf("sub-%d", s.subCounter)
	subCtx, cancel := context.WithCancel(ctx)
	sub := memorySub{
		id:      id,
		subject: subject,
		handler: handler,
		msgCh:   make(chan Message, 64),
		ctx:     subCtx,
		cancel:  cancel,
	}
	s.subs[subject] = append(s.subs[subject], sub)
	go s.drain(sub)
	return memorySubscription{s: s, id: id, subject: subject}, nil
}

// QueueSubscribe binds a consumer group (FRK-SVC-050).
func (s *MemoryStream) QueueSubscribe(ctx context.Context, subject, group string, handler Handler) (Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subCounter++
	id := fmt.Sprintf("qsub-%d", s.subCounter)
	subCtx, cancel := context.WithCancel(ctx)
	sub := memorySub{
		id:      id,
		subject: subject,
		handler: handler,
		msgCh:   make(chan Message, 64),
		ctx:     subCtx,
		cancel:  cancel,
	}
	if s.queueSubs[subject] == nil {
		s.queueSubs[subject] = map[string][]memorySub{}
	}
	s.queueSubs[subject][group] = append(s.queueSubs[subject][group], sub)
	go s.drain(sub)
	return memorySubscription{s: s, id: id, subject: subject, group: group}, nil
}

func (s *MemoryStream) drain(sub memorySub) {
	for {
		select {
		case msg := <-sub.msgCh:
			if err := sub.handler(sub.ctx, msg); err != nil {
				// redelivery: requeue on error (FRK-SVC-052)
				select {
				case sub.msgCh <- msg:
				case <-sub.ctx.Done():
					return
				}
			}
		case <-sub.ctx.Done():
			return
		}
	}
}

func (s *MemoryStream) remove(id, subject, group string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if group == "" {
		var kept []memorySub
		for _, sub := range s.subs[subject] {
			if sub.id != id {
				kept = append(kept, sub)
			}
		}
		s.subs[subject] = kept
		return
	}
	if groups, ok := s.queueSubs[subject]; ok {
		var kept []memorySub
		for _, sub := range groups[group] {
			if sub.id != id {
				kept = append(kept, sub)
			}
		}
		groups[group] = kept
	}
}

type memorySubscription struct {
	s       *MemoryStream
	id      string
	subject string
	group   string
}

func (sub memorySubscription) Unsubscribe() error {
	sub.s.remove(sub.id, sub.subject, sub.group)
	return nil
}
