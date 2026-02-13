# SDTW Message Queue - Implementation Tasks

> **Spec:** [spec.md](spec.md)
> **Created:** 2026-02-12
> **Status:** Ready for Implementation

> **Dependencies:**
> - SDTW Adapter Implementation Plan v2.0
> - ArmorClaw Phase 1 (Production-Ready)

---

## Summary

**Total Tasks:** 7 major tasks
**Estimated Effort:** 3-4 weeks
**Priority:** P0 (Foundational - blocks other SDTW work)

**Technologies:**
- SQLite (modernc.org/sqlite)
- Go 1.24+
- Prometheus metrics

---

## Task Breakdown

### Phase 1: Core Queue Implementation (Week 1)

- [ ] **1.1** Create queue package structure
  - File: `bridge/internal/queue/queue.go`
  - Define core types (Message, QueueStatus, EnqueueResult, etc.)
  - Define QueueConfig struct
  - Estimate: 2-3 hours

- [ ] **1.2** Implement database schema initialization
  - File: `bridge/internal/queue/schema.sql`
  - CREATE TABLE statements for queue_items and dead_letter_queue
  - CREATE INDEX statements for performance
  - CREATE TRIGGER statements for auto-expiration and DLQ movement
  - Estimate: 2-3 hours

- [ ] **1.3** Implement MessageQueue struct with SQLite backend
  - File: `bridge/internal/queue/queue.go`
  - NewMessageQueue() constructor with WAL mode configuration
  - SQLite connection pooling setup
  - PRAGMA optimization settings (cache_size, mmap_size, synchronous)
  - Estimate: 4-6 hours

### Phase 2: Queue Operations (Week 1-2)

- [ ] **2.1** Implement Enqueue operation
  - File: `bridge/internal/queue/queue.go`
  - Validate and assign defaults to Message
  - Serialize attachments/metadata to JSONBLOB
  - Insert with proper error handling
  - Return EnqueueResult with position/depth
  - Estimate: 3-4 hours

- [ ] **2.2** Implement Dequeue operation
  - File: `bridge/internal/queue/queue.go`
  - SELECT with FOR UPDATE SKIP LOCKED for concurrency
  - Mark message as inflight atomically
  - Deserialize JSONBLOB fields
  - Return DequeueResult with remaining depth
  - Estimate: 4-6 hours

- [ ] **2.3** Implement Ack/Nack operations
  - File: `bridge/internal/queue/queue.go`
  - Ack: Mark as 'acked', delete from queue
  - Nack: Increment attempts, calculate next retry with backoff
  - Auto-move to DLQ on max attempts exceeded
  - Estimate: 3-4 hours

- [ ] **2.4** Implement BackoffStrategy with jitter
  - File: `bridge/internal/queue/backoff.go`
  - Exponential backoff calculation
  - Jitter addition (10% of delay)
  - Max delay cap enforcement
  - Estimate: 1-2 hours

### Phase 3: Dead Letter Queue (Week 2)

- [ ] **3.1** Implement DLQ operations
  - File: `bridge/internal/queue/dlq.go`
  - moveToDLQ(): Transfer failed messages with error categorization
  - GetDLQMessages(): Query with platform filter and pagination
  - MarkDLQReviewed(): Update reviewed flag
  - RetryDLQMessage(): Re-enqueue to main queue
  - Estimate: 4-6 hours

- [ ] **3.2** Implement cleanup loop
  - File: `bridge/internal/queue/cleanup.go`
  - Background goroutine for periodic cleanup
  - Delete reviewed DLQ items older than 90 days
  - Delete expired messages from main queue
  - Estimate: 2-3 hours

### Phase 4: Batch Operations (Week 2)

- [ ] **4.1** Implement batch dequeue
  - File: `bridge/internal/queue/batch.go`
  - DequeueBatch(): Retrieve multiple messages in single transaction
  - Bulk mark as inflight
  - Maintain priority ordering
  - Estimate: 3-4 hours

- [ ] **4.2** Implement Peek operation
  - File: `bridge/internal/queue/queue.go`
  - Peek(): Return next message without state change
  - For monitoring and inspection purposes
  - Estimate: 1-2 hours

### Phase 5: Metrics & Monitoring (Week 2)

- [ ] **5.1** Implement QueueMetrics collector
  - File: `bridge/internal/queue/metrics.go`
  - Track: enqueued, dequeued, acked, retried, dlq counts
  - Thread-safe counter updates
  - ToPrometheusMetrics(): Export in Prometheus text format
  - Estimate: 2-3 hours

- [ ] **5.2** Implement Stats() operation
  - File: `bridge/internal/queue/stats.go`
  - Aggregate queue statistics (total, pending, inflight, failed)
  - Calculate wait time metrics (avg, p95)
  - Return QueueStats struct
  - Estimate: 2-3 hours

### Phase 6: Lifecycle Management (Week 2-3)

- [ ] **6.1** Implement Get operation
  - File: `bridge/internal/queue/queue.go`
  - Get(): Retrieve specific message by ID
  - Full deserialization of JSONBLOB fields
  - Estimate: 1-2 hours

- [ ] **6.2** Implement Shutdown operation
  - File: `bridge/internal/queue/queue.go`
  - Graceful shutdown with inflight draining
  - Wait up to 30 seconds for empty queue
  - Close database connection properly
  - Estimate: 2-3 hours

- [ ] **6.3** Implement Requeue operation
  - File: `bridge/internal/queue/queue.go`
  - Requeue(): Move message back to pending state
  - Reset next_retry to now
  - Estimate: 1 hour

### Phase 7: Testing & Documentation (Week 3-4)

- [ ] **7.1** Write comprehensive unit tests
  - File: `bridge/internal/queue/queue_test.go`
  - TestEnqueue: Verify message insertion and depth
  - TestDequeue: Verify FIFO ordering and inflight marking
  - TestRetryWithBackoff: Verify exponential backoff calculation
  - TestDeadLetterQueue: Verify DLQ movement after max retries
  - TestPriorityOrdering: Verify high-priority messages first
  - TestBatchDequeue: Verify bulk operations
  - TestStatistics: Verify metrics accuracy
  - TestConcurrency: 100 goroutines * 10 messages each
  - Estimate: 6-8 hours

- [ ] **7.2** Write integration tests
  - File: `bridge/internal/queue/integration_test.go`
  - Test database persistence across restarts
  - Test WAL recovery after unclean shutdown
  - Test concurrent access (multiple queue instances)
  - Test with file-based database (not :memory:)
  - Estimate: 4-6 hours

- [ ] **7.3** Write benchmarks
  - File: `bridge/internal/queue/benchmark_test.go`
  - BenchmarkEnqueue: Single-threaded throughput
  - BenchmarkDequeue: Single-threaded throughput
  - BenchmarkConcurrentEnqueue: Multi-goroutine throughput
  - BenchmarkConcurrentDequeue: Multi-goroutine throughput
  - Estimate: 2-3 hours

- [ ] **7.4** Update RPC API documentation
  - File: `docs/reference/rpc-api.md`
  - Document queue-related RPC methods (if exposed)
  - Document queue status in health check responses
  - Estimate: 1-2 hours

- [ ] **7.5** Create configuration guide
  - File: `docs/guides/sdtw-queue-configuration.md`
  - Database path configuration
  - Performance tuning options (pool_size, cache_size, wal_mode)
  - Backup and restore procedures
  - Estimate: 2-3 hours

---

## Dependencies

| Task | Depends On | Blocks |
|-------|------------|---------|
| 1.1 | None | Phase 1 tasks |
| 1.2 | 1.1 complete | 1.3 |
| 1.3 | 1.2 complete | Phase 2 tasks |
| 2.1 | 1.3 complete | 2.2, 2.3, 2.4 |
| 2.2 | 2.1 complete | 2.3 |
| 2.3 | 2.2, 2.4 complete | 2.1 (retries) |
| 2.4 | None | 2.3 |
| 3.1 | 2.3 complete | Phase 4 tasks |
| 3.2 | 3.1 complete | None |
| 4.1 | 2.2 complete | Phase 5 tasks |
| 4.2 | 2.2 complete | None |
| 5.1 | All previous complete | 5.2 |
| 5.2 | All previous complete | Phase 6 tasks |
| 6.1 | 2.2 complete | None |
| 6.2 | All queue ops complete | None |
| 6.3 | 6.1 complete | None |
| 7.1 | All implementation complete | 7.2 |
| 7.2 | 7.1 complete | 7.3 |
| 7.3 | 7.1 complete | 7.4, 7.5 |
| 7.4 | 5.1, 5.2 complete | None |
| 7.5 | 6.2 complete | None |

---

## Implementation Order

**Sequential (must follow order):**
1. Phase 1 (all tasks) - Core queue structure
2. Phase 2 (all tasks) - Basic queue operations
3. Phase 3 (all tasks) - Dead letter queue
4. Phase 4 (all tasks) - Batch operations
5. Phase 5 (all tasks) - Metrics
6. Phase 6 (all tasks) - Lifecycle
7. Phase 7 (all tasks) - Testing & docs

**Can proceed in parallel:**
- Unit tests (7.1) can be written alongside implementation
- Documentation (7.4, 7.5) can be written in parallel with integration tests

---

## Risk Assessment

| Risk | Impact | Mitigation |
|-------|--------|------------|
| SQLite WAL file corruption | High | Regular backups, PRAGMA integrity_check |
| Database lock contention | Medium | Connection pooling, FOR UPDATE SKIP LOCKED |
| Memory leak from goroutines | Medium | Proper context cancellation, graceful shutdown |
| Unbounded queue growth | High | Soft limits with alerts, TTL on messages |
| Concurrent write conflicts | Low | Transaction isolation, retry logic |

---

## Success Criteria

### Functional
- [ ] All 20+ unit tests passing
- [ ] Integration tests validate persistence
- [ ] Concurrency tests handle 1000+ operations
- [ ] Dead letter queue functions correctly

### Performance
- [ ] Enqueue p95 < 50ms
- [ ] Dequeue p95 < 50ms
- [ ] Batch dequeue improves throughput 5x
- [ ] Memory footprint < 100MB per instance

### Reliability
- [ ] Database survives unclean shutdown (WAL recovery)
- [ ] No message loss in tests
- [ ] Graceful shutdown drains inflight messages
- [ ] Prometheus metrics export correctly

---

**Total Estimated Effort:** ~80-120 hours (3-4 weeks for 1 engineer)

**Next Step:** Implement Task 1.1 - Create queue package structure

---

**Dependencies for SDTW Adapters:**
- This queue package is foundational for all SDTW adapters
- Adapter interface cannot be implemented without queue backend
- Policy engine depends on queue for reliable message delivery
