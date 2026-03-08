package events

import (
	"log/slog"
	"sync"
)

const DefaultMaxEvents = 1024
const MaxBatchSize = 128

type MatrixEvent struct {
	Seq     uint64
	ID      string
	RoomID  string
	Sender  string
	Type    string
	Content any
}

type MatrixEventBus struct {
	mu   sync.Mutex
	cond *sync.Cond

	buffer []MatrixEvent
	size   uint64

	start uint64
	count uint64

	nextSeq uint64

	// reusable batch buffer (micro-optimization)
	batch []MatrixEvent
}

func NewMatrixEventBus(size int) *MatrixEventBus {

	if size <= 0 {
		size = DefaultMaxEvents
	}

	b := &MatrixEventBus{
		buffer:  make([]MatrixEvent, size),
		size:    uint64(size),
		nextSeq: 1,
		batch:   make([]MatrixEvent, MaxBatchSize),
	}

	b.cond = sync.NewCond(&b.mu)

	return b
}

func (b *MatrixEventBus) Publish(e MatrixEvent) uint64 {

	b.mu.Lock()
	defer b.mu.Unlock()

	e.Seq = b.nextSeq
	b.nextSeq++

	idx := (b.start + b.count) % b.size
	b.buffer[idx] = e

	if b.count < b.size {
		b.count++
	} else {
		b.start = (b.start + 1) % b.size
	}

	b.cond.Broadcast()

	slog.Debug("matrix event published",
		"seq", e.Seq,
		"room_id", e.RoomID,
		"sender", e.Sender,
	)

	return e.Seq
}

func (b *MatrixEventBus) getEventsAfterLocked(cursor uint64) ([]MatrixEvent, uint64, bool) {

	if b.count == 0 {
		return nil, cursor, false
	}

	oldest := b.buffer[b.start%b.size].Seq
	newest := b.buffer[(b.start+b.count-1)%b.size].Seq

	if cursor < oldest-1 {
		return nil, oldest, true
	}

	if cursor >= newest {
		return nil, cursor, false
	}

	batchCount := 0
	var lastSeq uint64

	for i := uint64(0); i < b.count && batchCount < MaxBatchSize; i++ {

		idx := (b.start + i) % b.size
		ev := b.buffer[idx]

		if ev.Seq > cursor {

			b.batch[batchCount] = ev
			batchCount++
			lastSeq = ev.Seq
		}
	}

	if batchCount == 0 {
		return nil, cursor, false
	}

	return b.batch[:batchCount], lastSeq, false
}

func (b *MatrixEventBus) GetEventsAfter(cursor uint64) ([]MatrixEvent, uint64, bool) {

	b.mu.Lock()
	defer b.mu.Unlock()

	return b.getEventsAfterLocked(cursor)
}

func (b *MatrixEventBus) WaitForEvents(cursor uint64) ([]MatrixEvent, uint64, bool) {

	b.mu.Lock()
	defer b.mu.Unlock()

	for {

		events, next, reset := b.getEventsAfterLocked(cursor)

		if reset || len(events) > 0 {
			return events, next, reset
		}

		b.cond.Wait()
	}
}

func (b *MatrixEventBus) Status() map[string]interface{} {
	b.mu.Lock()
	defer b.mu.Unlock()

	return map[string]interface{}{
		"size":       b.size,
		"count":      b.count,
		"start_seq":  b.buffer[b.start%b.size].Seq,
		"next_seq":   b.nextSeq,
		"oldest_seq": b.buffer[(b.start+b.count-1)%b.size].Seq,
		"next_event": b.nextSeq,
	}
}
