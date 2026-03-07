package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

type ctxKey struct{}

var traceCtxKey = ctxKey{}

type Trace struct {
	ID        string
	ParentID  string
	StartTime time.Time
	EndTime   time.Time
	Name      string
	Attrs     map[string]interface{}
}

func New(name string) *Trace {
	return &Trace{
		ID:        generateID(),
		StartTime: time.Now(),
		Name:      name,
		Attrs:     make(map[string]interface{}),
	}
}

func (t *Trace) End() {
	t.EndTime = time.Now()
}

func (t *Trace) Duration() time.Duration {
	if t.EndTime.IsZero() {
		return time.Since(t.StartTime)
	}
	return t.EndTime.Sub(t.StartTime)
}

func (t *Trace) SetAttr(key string, value interface{}) *Trace {
	t.Attrs[key] = value
	return t
}

func (t *Trace) SetParent(parentID string) *Trace {
	t.ParentID = parentID
	return t
}

func (t *Trace) Child(name string) *Trace {
	return &Trace{
		ID:        generateID(),
		ParentID:  t.ID,
		StartTime: time.Now(),
		Name:      name,
		Attrs:     make(map[string]interface{}),
	}
}

func (t *Trace) Context() context.Context {
	return context.WithValue(context.Background(), traceCtxKey, t)
}

func FromContext(ctx context.Context) *Trace {
	if t, ok := ctx.Value(traceCtxKey).(*Trace); ok {
		return t
	}
	return nil
}

func ContextWithTrace(ctx context.Context, t *Trace) context.Context {
	return context.WithValue(ctx, traceCtxKey, t)
}

func Start(ctx context.Context, name string) (context.Context, *Trace) {
	parent := FromContext(ctx)
	t := New(name)
	if parent != nil {
		t.SetParent(parent.ID)
	}
	return ContextWithTrace(ctx, t), t
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func Span(ctx context.Context, name string, fn func(ctx context.Context) error) error {
	ctx, t := Start(ctx, name)
	defer t.End()
	return fn(ctx)
}

func SpanResult[T any](ctx context.Context, name string, fn func(ctx context.Context) (T, error)) (T, error) {
	ctx, t := Start(ctx, name)
	defer t.End()
	return fn(ctx)
}

type Event struct {
	Name      string
	Timestamp time.Time
	Attrs     map[string]interface{}
}

func (t *Trace) AddEvent(name string, attrs ...interface{}) {
	event := Event{
		Name:      name,
		Timestamp: time.Now(),
		Attrs:     make(map[string]interface{}),
	}
	for i := 0; i < len(attrs); i += 2 {
		if i+1 < len(attrs) {
			if key, ok := attrs[i].(string); ok {
				event.Attrs[key] = attrs[i+1]
			}
		}
	}
}
