# SDTW Message Queue Specification

> **Spec:** Persistent Message Queue for SDTW Adapters
> **Created:** 2026-02-12
> **Status:** Planning Complete - Ready for Implementation
> **Type:** Foundational Component

---

## Overview

Implement a production-grade, persistent message queue using SQLite with WAL mode to support reliable message delivery for SDTW (Slack, Discord, Teams, WhatsApp) adapters. This queue provides at-least-once delivery guarantees, retry logic with exponential backoff and jitter, dead letter queue handling, and support for high-throughput concurrent operations while maintaining ACID guarantees.

This component is **foundational** - all SDTW adapters depend on the queue for reliable message delivery, and subsequent components (circuit breaker, policy engine) build upon this infrastructure.

---

## User Stories

### Story 1: Basic Message Queuing

**As** an SDTW adapter, I want to reliably queue outbound messages for delivery to external platforms.

**Acceptance Criteria:**
- Messages can be enqueued with unique IDs
- Queue persists across bridge restarts
- Messages are dequeued in priority order (high → normal → low)
- Dequeue operations are thread-safe

**Workflow:**
1. Adapter receives message from Matrix room
2. Policy engine approves the send operation
3. Adapter enqueues message with target platform metadata
4. Queue worker processes next available message
5. On success: Message marked as acknowledged
6. On failure: Message scheduled for retry with backoff

---

### Story 2: Retry with Backoff

**As** an SDTW adapter, I want failed message sends to be automatically retried with increasing delays.

**Acceptance Criteria:**
- Failed sends are automatically requeued
- Retry delay increases exponentially (1s → 2s → 4s → 8s)
- Jitter (10%) prevents thundering herd
- Max retry limit is enforced
- Messages moved to DLQ after exhausting retries

**Workflow:**
1. Message send fails (timeout, rate limit, network error)
2. Adapter calls Nack(messageID, error)
3. Queue increments attempt counter
4. Backoff strategy calculates next retry time
5. Message reappears in queue after calculated delay
6. After max attempts: Moved to dead letter queue

---

### Story 3: Dead Letter Queue

**As** a system administrator, I want to review and retry messages that failed permanently.

**Acceptance Criteria:**
- Permanently failed messages are moved to DLQ
- DLQ preserves full message context and error details
- Admin can list DLQ messages by platform
- Failed messages can be retried manually
- Reviewed messages are cleaned up after 90 days

**Workflow:**
1. Message fails after max retry attempts
2. Trigger moves message to dead_letter_queue table
3. Error is categorized (timeout, rate_limit, auth_failed, etc.)
4. Admin views DLQ via `/sdtw dlq list` command
5. Admin can retry specific message: `/sdtw dlq retry <id>`
6. Successful retry moves message back to main queue
7. Old reviewed messages are auto-cleaned

---

### Story 4: Batch Processing

**As** a high-throughput adapter, I want to dequeue multiple messages at once for efficiency.

**Acceptance Criteria:**
- Batch dequeue retrieves multiple messages atomically
- All messages marked as inflight in transaction
- Priority ordering maintained within batch
- Configurable batch size (default: 10)

**Workflow:**
1. Queue worker calls DequeueBatch(10)
2. Transaction selects up to 10 pending messages
3. All messages marked inflight with FOR UPDATE SKIP LOCKED
4. Batch returned to adapter for bulk processing
5. Adapter processes each message and calls Ack/Nack

---

### Story 5: Monitoring & Metrics

**As** an operations engineer, I want real-time visibility into queue health and performance.

**Acceptance Criteria:**
- Prometheus metrics exported for queue operations
- Statistics API returns current depth, wait times
- Alerts fire on high queue depth
- Dead letter queue growth is monitored

**Metrics Collected:**
- `sdtw_queue_enqueued_total{platform}`
- `sdtw_queue_dequeued_total{platform}`
- `sdtw_queue_depth{platform}`
- `sdtw_queue_dlq_total{platform}`
- `sdtw_queue_retry_total{platform}`
- `sdtw_queue_latency_seconds{platform,operation}`

---

## Scope

### In Scope
1. **SQLite Backend with WAL Mode**
   - Single-file database with write-ahead logging
   - Concurrent reader/writer support
   - Automatic crash recovery

2. **Queue Operations**
   - Enqueue: Add message with priority
   - Dequeue: Get next message atomically
   - DequeueBatch: Get multiple messages
   - Ack: Mark as delivered
   - Nack: Schedule retry
   - Peek: Inspect without state change
   - Get: Retrieve specific message
   - Requeue: Return to pending

3. **Dead Letter Queue**
   - Automatic DLQ movement on max retries
   - DLQ listing and filtering
   - Manual retry from DLQ
   - Auto-cleanup of old reviewed items

4. **Resilience Features**
   - Exponential backoff with jitter
   - Priority ordering (high → normal → low)
   - Retry limits (configurable, default: 3)
   - Message TTL with auto-expiration

5. **Performance Optimizations**
   - Connection pooling (10 connections default)
   - Prepared statements for queries
   - Indexed lookups (status, platform, priority, retry time)
   - PRAGMA tuning (cache_size, mmap_size, synchronous)

### Out of Scope
- Platform-specific API integrations (handled by adapters)
- Policy evaluation logic (separate component)
- Message content transformation (PII scrubbing is separate)
- WebRTC voice integration
- Matrix E2EE operations

---

## Expected Deliverable

1. **Working queue package** at `bridge/internal/queue/`
   - `queue.go`: Main queue implementation
   - `backoff.go`: Retry strategy
   - `dlq.go`: Dead letter operations
   - `batch.go`: Batch processing
   - `metrics.go`: Prometheus metrics
   - `stats.go`: Statistics

2. **Comprehensive test suite**
   - `queue_test.go`: 20+ unit tests
   - `integration_test.go`: Persistence and concurrency tests
   - `benchmark_test.go`: Performance benchmarks

3. **Documentation**
   - Queue configuration guide
   - RPC API updates (if queue methods exposed)
   - Metrics reference

4. **Performance benchmarks**
   - Enqueue: <50ms p95
   - Dequeue: <50ms p95
   - Throughput: 1000 messages/minute
   - Memory: <100MB per instance

---

## Technical Constraints

### Database Requirements
- SQLite 3.38+ required for WAL mode optimization
- Database file must be on local filesystem (no network storage)
- Backup strategy required for production deployment

### Performance Requirements
- Support 1000 messages/minute sustained throughput
- Handle 100+ concurrent operations without blocking
- Memory footprint under 100MB per queue instance
- Database size under 1GB for 10K queued messages

### Platform Considerations
- Queue works across all SDTW platforms
- Per-platform queue isolation via platform field
- Platform-specific rate limits respected via retry timing

---

## Success Metrics

- [ ] 85%+ test coverage
- [ ] All unit tests passing
- [ ] Concurrency tests handle 1000+ concurrent operations
- [ ] Benchmarks meet performance targets
- [ ] Zero message loss in failure scenarios
- [ ] Clean WAL recovery after unclean shutdown
- [ ] Prometheus metrics correctly exported

---

## Dependencies

**Required:**
- ArmorClaw Phase 1 (Production-Ready)
- Go 1.24+ with modernc.org/sqlite driver
- Prometheus client library

**Blocked By:**
- None (foundational component)

**Blocks:**
- SDTW Adapter Implementation (Phase 1)
- Policy Engine Implementation (Phase 2)
- Circuit Breaker Implementation

---

**Implementation Ready:** Tasks defined in `tasks.md`
