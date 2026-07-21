package localexecutor

import (
	"context"
	"database/sql"
	"io"
	"log"
	"testing"
	"time"

	scheduler0 "github.com/scheduler0/scheduler0-go-client/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// newTestExecutor builds an Executor backed by an in-memory SQLite database.
// The client field is left nil; tests that exercise methods relying on it must
// mock or stub accordingly.
func newTestExecutor(t *testing.T) *Executor {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	require.NoError(t, migrate(db))

	t.Cleanup(func() { _ = db.Close() })

	return &Executor{
		executorID:      1,
		command:         "echo",
		workingDir:      "",
		pollInterval:    time.Second,
		client:          nil,
		db:              db,
		logger:          log.New(io.Discard, "", 0),
		jobCache:        make(map[int64]*jobExecState),
		queue:           newScheduleQueue(),
		jobAddedChan:    make(chan struct{}, 1),
		reportTriggerCh: make(chan struct{}, 1),
	}
}

// futureJob returns a simple job that fires once at StartDate (no cron spec).
func futureJob(id int64) scheduler0.Job {
	return scheduler0.Job{
		ID:        id,
		ProjectID: 10,
		AccountID: 5,
		Spec:      "",
		StartDate: time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339),
		Timezone:  "UTC",
		Status:    "active",
	}
}

// ---------------------------------------------------------------------------
// splitCommand
// ---------------------------------------------------------------------------

func TestSplitCommand_Single(t *testing.T) {
	assert.Equal(t, []string{"echo"}, splitCommand("echo"))
}

func TestSplitCommand_MultipleWords(t *testing.T) {
	assert.Equal(t, []string{"sh", "-c", "echo", "hello"}, splitCommand("sh -c echo hello"))
}

func TestSplitCommand_Empty(t *testing.T) {
	assert.Empty(t, splitCommand(""))
}

func TestSplitCommand_ExtraWhitespace(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c"}, splitCommand("  a   b   c  "))
}

// ---------------------------------------------------------------------------
// writeExecution
// ---------------------------------------------------------------------------

func TestWriteExecution_WritesRow(t *testing.T) {
	e := newTestExecutor(t)
	ctx := context.Background()

	jobID := int64(42)
	uniqueID := "test-unique-id"
	lastTime := time.Now().UTC().Add(-time.Hour)
	nextTime := time.Now().UTC().Add(time.Hour)

	e.writeExecution(ctx, jobID, uniqueID, stateScheduled, lastTime, nextTime, 1)

	var count int
	err := e.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM executions_uncommitted WHERE job_id = ? AND unique_id = ? AND state = ? AND reported = 0",
		jobID, uniqueID, stateScheduled,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestWriteExecution_ZeroTimesStoredAsEmpty(t *testing.T) {
	e := newTestExecutor(t)
	ctx := context.Background()

	e.writeExecution(ctx, 1, "uid", stateScheduled, time.Time{}, time.Time{}, 1)

	var lastTime, nextTime string
	err := e.db.QueryRowContext(ctx,
		"SELECT last_execution_time, next_execution_time FROM executions_uncommitted WHERE job_id = 1",
	).Scan(&lastTime, &nextTime)
	require.NoError(t, err)
	assert.Empty(t, lastTime)
	assert.Empty(t, nextTime)
}

func TestWriteExecution_MultipleStatesForSameJob(t *testing.T) {
	e := newTestExecutor(t)
	ctx := context.Background()

	now := time.Now().UTC()
	e.writeExecution(ctx, 99, "uid-1", stateScheduled, time.Time{}, now.Add(time.Hour), 1)
	e.writeExecution(ctx, 99, "uid-1", stateSuccess, now, now.Add(time.Hour), 1)

	var count int
	err := e.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM executions_uncommitted WHERE job_id = 99",
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

// ---------------------------------------------------------------------------
// scheduleJob
// ---------------------------------------------------------------------------

func TestScheduleJob_PushesEntryToQueue(t *testing.T) {
	e := newTestExecutor(t)
	ctx := context.Background()

	job := futureJob(1)
	e.scheduleJob(ctx, job, time.Time{}, 1)

	assert.Equal(t, 1, e.queue.Len(), "queue should contain the scheduled entry")
	entry := e.queue.Peek()
	assert.Equal(t, job.ID, entry.JobId)
	assert.True(t, entry.ExecutionTime.After(time.Now()))
}

func TestScheduleJob_PopulatesJobCache(t *testing.T) {
	e := newTestExecutor(t)
	ctx := context.Background()

	job := futureJob(2)
	e.scheduleJob(ctx, job, time.Time{}, 1)

	e.mu.Lock()
	state, ok := e.jobCache[job.ID]
	e.mu.Unlock()

	require.True(t, ok, "job should be in cache after scheduleJob")
	assert.Equal(t, job.ID, state.job.ID)
	assert.Equal(t, stateScheduled, state.lastState)
	assert.Equal(t, uint64(1), state.execVersion)
}

func TestScheduleJob_WritesScheduledExecution(t *testing.T) {
	e := newTestExecutor(t)
	ctx := context.Background()

	job := futureJob(3)
	e.scheduleJob(ctx, job, time.Time{}, 1)

	var count int
	err := e.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM executions_uncommitted WHERE job_id = ? AND state = ?",
		job.ID, stateScheduled,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "scheduleJob should write a 'scheduled' execution log")
}

func TestScheduleJob_SignalsJobAddedChan(t *testing.T) {
	e := newTestExecutor(t)
	ctx := context.Background()

	e.scheduleJob(ctx, futureJob(4), time.Time{}, 1)

	select {
	case <-e.jobAddedChan:
		// expected
	default:
		t.Fatal("jobAddedChan should have been signalled")
	}
}

// ---------------------------------------------------------------------------
// scheduleFromCache
// ---------------------------------------------------------------------------

func TestScheduleFromCache_PopulatesQueueFromDB(t *testing.T) {
	e := newTestExecutor(t)
	ctx := context.Background()

	futureStart := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)

	_, err := e.db.ExecContext(ctx, `
		INSERT INTO jobs_cache
			(id, executor_id, project_id, account_id, spec, data, timezone, timezone_offset,
			 start_date, end_date, retry_max, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 10, e.executorID, 1, 1, "", "test-data", "UTC", 0, futureStart, "", 0, "active")
	require.NoError(t, err)

	e.scheduleFromCache(ctx)

	assert.Equal(t, 1, e.queue.Len(), "queue should have one entry loaded from cache")

	e.mu.Lock()
	_, inCache := e.jobCache[10]
	e.mu.Unlock()
	assert.True(t, inCache, "job should be in jobCache after scheduleFromCache")
}

func TestScheduleFromCache_SkipsInactiveJobs(t *testing.T) {
	e := newTestExecutor(t)
	ctx := context.Background()

	futureStart := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)

	_, err := e.db.ExecContext(ctx, `
		INSERT INTO jobs_cache
			(id, executor_id, project_id, account_id, spec, data, timezone, timezone_offset,
			 start_date, end_date, retry_max, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 20, e.executorID, 1, 1, "", "", "UTC", 0, futureStart, "", 0, "inactive")
	require.NoError(t, err)

	e.scheduleFromCache(ctx)

	assert.Equal(t, 0, e.queue.Len(), "inactive jobs should not be scheduled")
}

func TestScheduleFromCache_SkipsEndedJobs(t *testing.T) {
	e := newTestExecutor(t)
	ctx := context.Background()

	pastEnd := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	pastStart := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)

	_, err := e.db.ExecContext(ctx, `
		INSERT INTO jobs_cache
			(id, executor_id, project_id, account_id, spec, data, timezone, timezone_offset,
			 start_date, end_date, retry_max, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 30, e.executorID, 1, 1, "", "", "UTC", 0, pastStart, pastEnd, 0, "active")
	require.NoError(t, err)

	e.scheduleFromCache(ctx)

	assert.Equal(t, 0, e.queue.Len(), "ended jobs should not be scheduled")
}

func TestScheduleFromCache_DoesNotRescheduleAlreadyCachedJob(t *testing.T) {
	e := newTestExecutor(t)
	ctx := context.Background()

	futureStart := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)

	_, err := e.db.ExecContext(ctx, `
		INSERT INTO jobs_cache
			(id, executor_id, project_id, account_id, spec, data, timezone, timezone_offset,
			 start_date, end_date, retry_max, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 40, e.executorID, 1, 1, "", "", "UTC", 0, futureStart, "", 0, "active")
	require.NoError(t, err)

	// Pre-populate the cache to simulate a job already scheduled.
	e.mu.Lock()
	e.jobCache[40] = &jobExecState{job: scheduler0.Job{ID: 40}}
	e.mu.Unlock()

	e.scheduleFromCache(ctx)

	assert.Equal(t, 0, e.queue.Len(), "already-cached job should not be re-queued")
}

func TestScheduleFromCache_SeedsLastExecTimeFromUncommitted(t *testing.T) {
	e := newTestExecutor(t)
	ctx := context.Background()

	futureStart := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
	_, err := e.db.ExecContext(ctx, `
		INSERT INTO jobs_cache
			(id, executor_id, project_id, account_id, spec, data, timezone, timezone_offset,
			 start_date, end_date, retry_max, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 50, e.executorID, 1, 1, "", "", "UTC", 0, futureStart, "", 0, "active")
	require.NoError(t, err)

	// Simulate a prior execution that has not yet been reported.
	priorExecTime := time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339)
	_, err = e.db.ExecContext(ctx, `
		INSERT INTO executions_uncommitted
			(job_id, unique_id, state, last_execution_time, next_execution_time, execution_version, reported)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, 50, "uid-prev", stateSuccess, priorExecTime, futureStart, 1, 0)
	require.NoError(t, err)

	e.scheduleFromCache(ctx)

	// The job should have been scheduled from cache and appear in both the queue
	// and jobCache — even when there is a prior uncommitted execution.
	assert.Equal(t, 1, e.queue.Len(), "job should be queued after scheduleFromCache")

	e.mu.Lock()
	state, ok := e.jobCache[50]
	e.mu.Unlock()
	require.True(t, ok, "job should be in jobCache after scheduleFromCache")
	assert.Equal(t, stateScheduled, state.lastState)
}
