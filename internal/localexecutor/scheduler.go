// Package localexecutor — scheduling primitives.
//
// This file is intentionally a self-contained port of the scheduling logic
// from scheduler0-private so that both projects stay algorithm-compatible.
// Any fix or feature (e.g. quiet-hours, DST handling) should be applied here
// first and then back-ported to the private server's equivalent files:
//   - pkg/models/schedule_queue.go
//   - pkg/models/job.go  (GetNextExecutionTime / GetNextExecutionId / HasJobEnded)
package localexecutor

import (
	"container/heap"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	scheduler0 "github.com/scheduler0/scheduler0-go-client/v2"

	"github.com/robfig/cron"
)

// ---------------------------------------------------------------------------
// Min-heap schedule queue (mirrors pkg/models/schedule_queue.go)
// ---------------------------------------------------------------------------

// scheduleQueueEntry mirrors models.JobScheduleKey.
type scheduleQueueEntry struct {
	JobId         int64
	ExecutionTime time.Time
}

type entryHeap []scheduleQueueEntry

func (h entryHeap) Len() int { return len(h) }

// Less sorts in ascending order of ExecutionTime (UTC) — earliest due first.
func (h entryHeap) Less(i, j int) bool {
	return h[i].ExecutionTime.UTC().Before(h[j].ExecutionTime.UTC())
}

func (h entryHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *entryHeap) Push(x interface{}) { *h = append(*h, x.(scheduleQueueEntry)) }

func (h *entryHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

func (h entryHeap) peek() scheduleQueueEntry {
	if len(h) == 0 {
		return scheduleQueueEntry{}
	}
	return h[0]
}

// scheduleQueue is a thread-safe priority queue of pending job executions.
type scheduleQueue struct {
	h  *entryHeap
	mu sync.Mutex
}

func newScheduleQueue() *scheduleQueue {
	h := &entryHeap{}
	heap.Init(h)
	return &scheduleQueue{h: h}
}

// Push adds an entry to the queue.
func (q *scheduleQueue) Push(e scheduleQueueEntry) {
	q.mu.Lock()
	heap.Push(q.h, e)
	q.mu.Unlock()
}

// Pop removes and returns the entry with the earliest execution time.
// Returns a zero-value entry when the queue is empty.
func (q *scheduleQueue) Pop() scheduleQueueEntry {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.h.Len() == 0 {
		return scheduleQueueEntry{}
	}
	return heap.Pop(q.h).(scheduleQueueEntry)
}

// Peek returns the entry with the earliest execution time without removing it.
// Returns a zero-value entry (JobId == 0) when the queue is empty.
func (q *scheduleQueue) Peek() scheduleQueueEntry {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.h.peek()
}

// Len returns the number of entries in the queue.
func (q *scheduleQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.h.Len()
}

// RemoveByJobId removes all entries for the given job ID.
func (q *scheduleQueue) RemoveByJobId(jobId int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	rebuilt := &entryHeap{}
	heap.Init(rebuilt)
	for q.h.Len() > 0 {
		e := heap.Pop(q.h).(scheduleQueueEntry)
		if e.JobId != jobId {
			heap.Push(rebuilt, e)
		}
	}
	q.h = rebuilt
}

// Clear empties the queue.
func (q *scheduleQueue) Clear() {
	q.mu.Lock()
	h := &entryHeap{}
	heap.Init(h)
	q.h = h
	q.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Scheduling helpers (mirrors pkg/models/job.go)
// ---------------------------------------------------------------------------

// getNextExecutionTime computes the next time a job should fire, respecting
// its cron spec, timezone, start date and the time of the last execution.
//
// Mirrors scheduler0-private's Job.GetNextExecutionTime().
func getNextExecutionTime(job scheduler0.Job, lastExecTime time.Time) (time.Time, error) {
	loc, err := loadLocation(job.Timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("getNextExecutionTime: invalid timezone %q: %w", job.Timezone, err)
	}

	if job.Spec == "" {
		// No cron spec — fire once at StartDate.
		startDate, err := parseJobTime(job.StartDate)
		if err != nil {
			return time.Time{}, fmt.Errorf("getNextExecutionTime: invalid start date %q: %w", job.StartDate, err)
		}
		return startDate.In(loc), nil
	}

	schedule, err := cron.Parse(job.Spec)
	if err != nil {
		return time.Time{}, fmt.Errorf("getNextExecutionTime: invalid cron spec %q: %w", job.Spec, err)
	}

	if lastExecTime.IsZero() && job.StartDate != "" {
		startDate, err := parseJobTime(job.StartDate)
		if err != nil {
			return time.Time{}, fmt.Errorf("getNextExecutionTime: invalid start date %q: %w", job.StartDate, err)
		}
		// Treat a zero-time string (e.g. "0001-01-01T00:00:00Z") the same as an
		// absent StartDate — fall through to the cron-advance logic below.
		if !startDate.IsZero() {
			return startDate.In(loc), nil
		}
	}

	// Advance through the cron schedule (in the job's timezone) until we find a
	// future time — mirrors the loop in Job.GetNextExecutionTime().
	nowInJobTZ := time.Now().UTC().In(loc)
	currentTime := lastExecTime.In(loc)
	for !currentTime.After(nowInJobTZ) {
		currentTime = schedule.Next(currentTime)
	}
	return currentTime, nil
}

// getNextExecutionID produces a stable, content-addressed ID for an execution.
// Mirrors scheduler0-private's Job.GetNextExecutionId().
func getNextExecutionID(job scheduler0.Job, lastExecDate, nextExecTime time.Time) string {
	raw := fmt.Sprintf("%d-%d-%s-%s",
		job.ProjectID,
		job.ID,
		lastExecDate.String(),
		nextExecTime.String(),
	)
	sha := sha256.New()
	return fmt.Sprintf("%x", sha.Sum([]byte(raw)))
}

// hasJobEnded returns true if the job's EndDate (in its timezone) is in the past.
// Mirrors scheduler0-private's Job.HasJobEnded().
func hasJobEnded(job scheduler0.Job) (bool, error) {
	if job.EndDate == "" {
		return false, nil
	}
	loc, err := loadLocation(job.Timezone)
	if err != nil {
		return false, fmt.Errorf("hasJobEnded: invalid timezone %q: %w", job.Timezone, err)
	}
	endDate, err := parseJobTime(job.EndDate)
	if err != nil {
		return false, fmt.Errorf("hasJobEnded: invalid end date %q: %w", job.EndDate, err)
	}
	// Treat a zero-time string (e.g. "0001-01-01T00:00:00Z") the same as an
	// absent EndDate so jobs without an end date are never falsely ended.
	if endDate.IsZero() {
		return false, nil
	}
	nowInJobTZ := time.Now().UTC().In(loc)
	return endDate.In(loc).Before(nowInJobTZ), nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// loadLocation wraps time.LoadLocation, treating an empty string as UTC.
func loadLocation(tz string) (*time.Location, error) {
	if tz == "" {
		return time.UTC, nil
	}
	return time.LoadLocation(tz)
}

// parseJobTime parses the date/time strings that the go-client returns for
// job StartDate / EndDate / LastExecutionDate. Multiple formats are tried in
// order to be resilient to server-side formatting differences.
func parseJobTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time %q", s)
}
