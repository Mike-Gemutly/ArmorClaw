package cache

import (
	"container/list"
	"sync"
	"time"
)

type entry struct {
	key       string
	value     interface{}
	expiresAt time.Time
	element   *list.Element
}

type LRU struct {
	mu          sync.RWMutex
	items       map[string]*entry
	eviction    *list.List
	maxSize     int
	defaultTTL  time.Duration
	onEvict     func(key string, value interface{})
}

type LRUConfig struct {
	MaxSize    int
	DefaultTTL time.Duration
	OnEvict    func(key string, value interface{})
}

func NewLRU(cfg LRUConfig) *LRU {
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 1000
	}
	if cfg.DefaultTTL <= 0 {
		cfg.DefaultTTL = 5 * time.Minute
	}
	return &LRU{
		items:      make(map[string]*entry),
		eviction:   list.New(),
		maxSize:    cfg.MaxSize,
		defaultTTL: cfg.DefaultTTL,
		onEvict:    cfg.OnEvict,
	}
}

func (l *LRU) Get(key string) (interface{}, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	ent, exists := l.items[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(ent.expiresAt) {
		l.removeEntryLocked(ent)
		return nil, false
	}

	l.eviction.MoveToFront(ent.element)
	return ent.value, true
}

func (l *LRU) Set(key string, value interface{}) {
	l.SetWithTTL(key, value, l.defaultTTL)
}

func (l *LRU) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if ent, exists := l.items[key]; exists {
		ent.value = value
		ent.expiresAt = time.Now().Add(ttl)
		l.eviction.MoveToFront(ent.element)
		return
	}

	if len(l.items) >= l.maxSize {
		l.evictOldestLocked()
	}

	ent := &entry{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	ent.element = l.eviction.PushFront(ent)
	l.items[key] = ent
}

func (l *LRU) Delete(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if ent, exists := l.items[key]; exists {
		l.removeEntryLocked(ent)
	}
}

func (l *LRU) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, ent := range l.items {
		if l.onEvict != nil {
			l.onEvict(ent.key, ent.value)
		}
	}
	l.items = make(map[string]*entry)
	l.eviction.Init()
}

func (l *LRU) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.items)
}

func (l *LRU) PurgeExpired() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	count := 0
	for key, ent := range l.items {
		if now.After(ent.expiresAt) {
			l.removeEntryLocked(ent)
			delete(l.items, key)
			count++
		}
	}
	return count
}

func (l *LRU) evictOldestLocked() {
	oldest := l.eviction.Back()
	if oldest == nil {
		return
	}
	ent := oldest.Value.(*entry)
	l.removeEntryLocked(ent)
}

func (l *LRU) removeEntryLocked(ent *entry) {
	if l.onEvict != nil {
		l.onEvict(ent.key, ent.value)
	}
	delete(l.items, ent.key)
	l.eviction.Remove(ent.element)
}

func (l *LRU) GetOrCompute(key string, compute func() (interface{}, error)) (interface{}, error) {
	return l.GetOrComputeWithTTL(key, compute, l.defaultTTL)
}

func (l *LRU) GetOrComputeWithTTL(key string, compute func() (interface{}, error), ttl time.Duration) (interface{}, error) {
	if val, ok := l.Get(key); ok {
		return val, nil
	}

	val, err := compute()
	if err != nil {
		return nil, err
	}

	l.SetWithTTL(key, val, ttl)
	return val, nil
}
