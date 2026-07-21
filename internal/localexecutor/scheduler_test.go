package localexecutor

import (
	"testing"
	"time"

	scheduler0 "github.com/scheduler0/scheduler0-go-client/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// scheduleQueue
// ---------------------------------------------------------------------------

func TestScheduleQueue_PushPopOrder(t *testing.T) {
	q := newScheduleQueue()

	now := time.Now()
	q.Push(scheduleQueueEntry{JobId: 3, ExecutionTime: now.Add(3 * time.Hour)})
	q.Push(scheduleQueueEntry{JobId: 1, ExecutionTime: now.Add(1 * time.Hour)})
	q.Push(scheduleQueueEntry{JobId: 2, ExecutionTime: now.Add(2 * time.Hour)})

	first := q.Pop()
	second := q.Pop()
	third := q.Pop()

	assert.Equal(t, int64(1), first.JobId, "earliest job should come out first")
	assert.Equal(t, int64(2), second.JobId)
	assert.Equal(t, int64(3), third.JobId)
}

func TestScheduleQueue_PopEmpty(t *testing.T) {
	q := newScheduleQueue()
	entry := q.Pop()
	assert.Equal(t, int64(0), entry.JobId, "pop on empty queue should return zero entry")
}

func TestScheduleQueue_Peek(t *testing.T) {
	q := newScheduleQueue()
	now := time.Now()
	q.Push(scheduleQueueEntry{JobId: 10, ExecutionTime: now.Add(5 * time.Minute)})
	q.Push(scheduleQueueEntry{JobId: 5, ExecutionTime: now.Add(1 * time.Minute)})

	peeked := q.Peek()
	assert.Equal(t, int64(5), peeked.JobId, "peek should return the earliest entry")
	assert.Equal(t, 2, q.Len(), "peek should not remove the entry")
}

func TestScheduleQueue_PeekEmpty(t *testing.T) {
	q := newScheduleQueue()
	entry := q.Peek()
	assert.Equal(t, int64(0), entry.JobId)
}

func TestScheduleQueue_Len(t *testing.T) {
	q := newScheduleQueue()
	assert.Equal(t, 0, q.Len())

	q.Push(scheduleQueueEntry{JobId: 1, ExecutionTime: time.Now()})
	q.Push(scheduleQueueEntry{JobId: 2, ExecutionTime: time.Now()})
	assert.Equal(t, 2, q.Len())
}

func TestScheduleQueue_RemoveByJobId(t *testing.T) {
	q := newScheduleQueue()
	now := time.Now()
	q.Push(scheduleQueueEntry{JobId: 1, ExecutionTime: now.Add(1 * time.Hour)})
	q.Push(scheduleQueueEntry{JobId: 2, ExecutionTime: now.Add(2 * time.Hour)})
	q.Push(scheduleQueueEntry{JobId: 1, ExecutionTime: now.Add(3 * time.Hour)}) // duplicate job 1

	q.RemoveByJobId(1)

	assert.Equal(t, 1, q.Len(), "both entries for job 1 should be removed")
	remaining := q.Pop()
	assert.Equal(t, int64(2), remaining.JobId)
}

func TestScheduleQueue_Clear(t *testing.T) {
	q := newScheduleQueue()
	q.Push(scheduleQueueEntry{JobId: 1, ExecutionTime: time.Now()})
	q.Push(scheduleQueueEntry{JobId: 2, ExecutionTime: time.Now()})

	q.Clear()

	assert.Equal(t, 0, q.Len())
}

// ---------------------------------------------------------------------------
// getNextExecutionTime
// ---------------------------------------------------------------------------

func TestGetNextExecutionTime_NoSpecUsesStartDate(t *testing.T) {
	future := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	job := scheduler0.Job{
		Spec:      "",
		StartDate: future.Format(time.RFC3339),
		Timezone:  "UTC",
	}

	got, err := getNextExecutionTime(job, time.Time{})
	require.NoError(t, err)
	assert.Equal(t, future.UTC(), got.UTC())
}

func TestGetNextExecutionTime_ZeroLastExecWithStartDate(t *testing.T) {
	// When the job has a cron spec but lastExecTime is zero and StartDate is
	// set, the function should return StartDate (first-fire bootstrap).
	future := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	job := scheduler0.Job{
		Spec:      "*/5 * * * *",
		StartDate: future.Format(time.RFC3339),
		Timezone:  "UTC",
	}

	got, err := getNextExecutionTime(job, time.Time{})
	require.NoError(t, err)
	assert.Equal(t, future.UTC(), got.UTC())
}

func TestGetNextExecutionTime_ZeroStartDateStringFallsThrough(t *testing.T) {
	// When the server serializes a zero time.Time it produces "0001-01-01T00:00:00Z".
	// A cron job with that StartDate and no prior execution should still advance
	// to a real future time via the cron schedule, not schedule in year 1.
	job := scheduler0.Job{
		Spec:      "@every 1h",
		StartDate: "0001-01-01T00:00:00Z",
		Timezone:  "UTC",
	}

	got, err := getNextExecutionTime(job, time.Time{})
	require.NoError(t, err)
	assert.True(t, got.After(time.Now().UTC()), "next execution time should be in the future, not year 1")
	assert.Greater(t, got.Year(), 2000, "year 1 execution time indicates zero-date bug")
}

func TestGetNextExecutionTime_CronAdvancesPastNow(t *testing.T) {
	// lastExecTime in the past → result must be in the future.
	lastExec := time.Now().UTC().Add(-2 * time.Hour)
	job := scheduler0.Job{
		Spec:     "@every 1h",
		Timezone: "UTC",
	}

	got, err := getNextExecutionTime(job, lastExec)
	require.NoError(t, err)
	assert.True(t, got.After(time.Now().UTC()), "next execution time should be in the future")
}

func TestGetNextExecutionTime_TimezoneApplied(t *testing.T) {
	future := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	job := scheduler0.Job{
		Spec:      "",
		StartDate: future.UTC().Format(time.RFC3339),
		Timezone:  "America/New_York",
	}

	got, err := getNextExecutionTime(job, time.Time{})
	require.NoError(t, err)

	loc, _ := time.LoadLocation("America/New_York")
	assert.Equal(t, loc, got.Location())
}

func TestGetNextExecutionTime_EmptyTimezoneDefaultsToUTC(t *testing.T) {
	future := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	job := scheduler0.Job{
		Spec:      "",
		StartDate: future.Format(time.RFC3339),
		Timezone:  "",
	}

	got, err := getNextExecutionTime(job, time.Time{})
	require.NoError(t, err)
	assert.Equal(t, time.UTC, got.Location())
}

// ---------------------------------------------------------------------------
// getNextExecutionID
// ---------------------------------------------------------------------------

func TestGetNextExecutionID_Deterministic(t *testing.T) {
	job := scheduler0.Job{ID: 42, ProjectID: 7}
	last := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	next := time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)

	id1 := getNextExecutionID(job, last, next)
	id2 := getNextExecutionID(job, last, next)
	assert.Equal(t, id1, id2, "same inputs should produce the same ID")
	assert.NotEmpty(t, id1)
}

func TestGetNextExecutionID_DifferentInputsProduceDifferentIDs(t *testing.T) {
	job := scheduler0.Job{ID: 42, ProjectID: 7}
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)
	t3 := time.Date(2024, 1, 1, 2, 0, 0, 0, time.UTC)

	idA := getNextExecutionID(job, t1, t2)
	idB := getNextExecutionID(job, t1, t3)
	assert.NotEqual(t, idA, idB)
}

// ---------------------------------------------------------------------------
// hasJobEnded
// ---------------------------------------------------------------------------

func TestHasJobEnded_NoEndDate(t *testing.T) {
	job := scheduler0.Job{Timezone: "UTC"}
	ended, err := hasJobEnded(job)
	require.NoError(t, err)
	assert.False(t, ended)
}

func TestHasJobEnded_FutureEndDate(t *testing.T) {
	future := time.Now().UTC().Add(48 * time.Hour)
	job := scheduler0.Job{
		EndDate:  future.Format(time.RFC3339),
		Timezone: "UTC",
	}
	ended, err := hasJobEnded(job)
	require.NoError(t, err)
	assert.False(t, ended)
}

func TestHasJobEnded_PastEndDate(t *testing.T) {
	past := time.Now().UTC().Add(-48 * time.Hour)
	job := scheduler0.Job{
		EndDate:  past.Format(time.RFC3339),
		Timezone: "UTC",
	}
	ended, err := hasJobEnded(job)
	require.NoError(t, err)
	assert.True(t, ended)
}

func TestHasJobEnded_ZeroTimeString(t *testing.T) {
	// "0001-01-01T00:00:00Z" is the zero time.Time marshaled to JSON when the
	// server omits EndDate. It must not be treated as a past end-date.
	job := scheduler0.Job{
		EndDate:  "0001-01-01T00:00:00Z",
		Timezone: "UTC",
	}
	ended, err := hasJobEnded(job)
	require.NoError(t, err)
	assert.False(t, ended, "zero-time EndDate must not mark the job as ended")
}

// ---------------------------------------------------------------------------
// parseJobTime
// ---------------------------------------------------------------------------

func TestParseJobTime_RFC3339(t *testing.T) {
	s := "2024-06-01T12:00:00Z"
	got, err := parseJobTime(s)
	require.NoError(t, err)
	assert.Equal(t, 2024, got.Year())
	assert.Equal(t, time.June, got.Month())
	assert.Equal(t, 12, got.Hour())
}

func TestParseJobTime_SpaceFormat(t *testing.T) {
	s := "2024-06-01 12:00:00"
	got, err := parseJobTime(s)
	require.NoError(t, err)
	assert.Equal(t, 2024, got.Year())
	assert.Equal(t, 12, got.Hour())
}

func TestParseJobTime_DateOnly(t *testing.T) {
	s := "2024-06-01"
	got, err := parseJobTime(s)
	require.NoError(t, err)
	assert.Equal(t, 2024, got.Year())
	assert.Equal(t, time.June, got.Month())
	assert.Equal(t, 1, got.Day())
}

func TestParseJobTime_EmptyString(t *testing.T) {
	got, err := parseJobTime("")
	require.NoError(t, err)
	assert.True(t, got.IsZero())
}

// ---------------------------------------------------------------------------
// loadLocation
// ---------------------------------------------------------------------------

func TestLoadLocation_Empty(t *testing.T) {
	loc, err := loadLocation("")
	require.NoError(t, err)
	assert.Equal(t, time.UTC, loc)
}

func TestLoadLocation_UTC(t *testing.T) {
	loc, err := loadLocation("UTC")
	require.NoError(t, err)
	assert.Equal(t, time.UTC, loc)
}

func TestLoadLocation_NamedZone(t *testing.T) {
	loc, err := loadLocation("America/New_York")
	require.NoError(t, err)
	assert.Equal(t, "America/New_York", loc.String())
}
