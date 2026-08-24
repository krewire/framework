// Package worker provides the Krewire job-queue contract and its default
// in-memory backend (KWF-L5H2F §5.7): a Job interface, queue operations
// (Enqueue/Dequeue/Ack/Nack), priority and delay scheduling, cron recurrence,
// configurable retry with exponential backoff, a dead-letter queue, and a
// Runner that drains any Queue with bounded concurrency.
//
// Applier contract — opt-in batteries:
//
//   - Importing this package drags nothing else: it is stdlib-only
//     (log/slog included). Applications that never import worker keep their
//     binary size and startup time unchanged (KWF-L5H2F NFR1).
//   - Queue is the single extension point. The in-memory backend is the
//     default for local dev; NATS, Redis, and PostgreSQL backends are future
//     conformers of the same interface (FRK-SVC-063) and plug in without
//     changes to Runner or call sites.
//   - Retries and dead-lettering are part of the queue contract (FRK-SVC-062):
//     Nack consults the task's RetryPolicy, requeues with backoff while
//     attempts remain, and files exhausted tasks into DLQ() for programmatic
//     inspection (a future kiw worker dlq list wraps it).
//
// Scheduling semantics:
//
//   - Higher Options.Priority dequeues first; ties dequeue FIFO.
//   - Options.Delay gates delivery until enqueue-time + delay.
//   - Options.Cron accepts a 5-field spec (minute hour day-of-month month
//     day-of-week). Empty means one-shot. The first occurrence is deliverable
//     immediately; Ack of a cron occurrence re-arms the next one at
//     Schedule.NextFire computed from the previous scheduled time, so
//     occurrences neither drift nor overlap. A cron occurrence that exhausts
//     its retries dead-letters like any other poison job.
package worker
