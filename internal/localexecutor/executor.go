// Package localexecutor implements the long-running local-executor service that runs
// on the user's machine via `scheduler0 local-executor start`. It polls the scheduler0-private
// server for jobs assigned to this executor, runs them locally as shell scripts, and
// batches execution reports back to the server even when temporarily offline.
package localexecutor

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	scheduler0 "github.com/scheduler0/scheduler0-go-client"
)

const (
	// stateScheduled / stateSuccess / stateFailedmirror ExecutionLogScheduleState etc.
	// in scheduler0-private/pkg/models/job_execution_log.go.
	stateScheduled = 0
	stateSuccess   = 1
	stateFailed    = 2

	// reportThreshold triggers an early report when this many unreported
	// execution rows have accumulated, subject to the minimum interval.
	reportThreshold = 50

	// reportMinInterval is the minimum time between two successive reports.
	reportMinInterval = 5 * time.Minute
)

// jobExecState holds the in-memory execution state for one job.
// Mirrors the combination of models.JobSchedule + models.MemJobExecution
// from the private server.
type jobExecState struct {
	job          scheduler0.Job
	execVersion  uint64
	lastExecTime time.Time // time of last completed execution (success or fail)
	lastState    int       // stateScheduled / stateSuccess / stateFailed
	failCount    uint64
}

// Executor is the main runtime for the local executor service.
type Executor struct {
	executorID   int64
	pollInterval time.Duration
	client       *scheduler0.Client
	db           *sql.DB
	logger       *log.Logger

	// executor config refreshed from the server (protected by cfgMu).
	cfgMu      sync.RWMutex
	command    string
	workingDir string

	// job execution state (protected by mu)
	mu       sync.Mutex
	jobCache map[int64]*jobExecState

	// schedule queue (has its own internal mutex)
	queue        *scheduleQueue
	jobAddedChan chan struct{}

	// reporter state
	reportTriggerCh chan struct{}
	reportMu        sync.Mutex
	lastReportedAt  time.Time
}

// New creates and initialises a new Executor.
// executorID must be > 0.
func New(
	client *scheduler0.Client,
	executorID int64,
	command string,
	workingDir string,
	pollInterval time.Duration,
	logger *log.Logger,
) (*Executor, error) {
	if logger == nil {
		logger = log.New(os.Stderr, "[local-executor] ", log.LstdFlags)
	}

	db, err := openDB()
	if err != nil {
		return nil, fmt.Errorf("local executor db: %w", err)
	}

	return &Executor{
		executorID:      executorID,
		command:         command,
		workingDir:      workingDir,
		pollInterval:    pollInterval,
		client:          client,
		db:              db,
		logger:          logger,
		jobCache:        make(map[int64]*jobExecState),
		queue:           newScheduleQueue(),
		jobAddedChan:    make(chan struct{}, 1),
		reportTriggerCh: make(chan struct{}, 1),
	}, nil
}

// Run starts the executor service.  It blocks until ctx is cancelled.
func (e *Executor) Run(ctx context.Context) {
	e.logger.Printf("starting local executor id=%d poll_interval=%s", e.executorID, e.pollInterval)

	// Pull the latest executor definition (command/workingDir) before we begin
	// so we always run with the server's current configuration.  On failure we
	// log a warning and keep using the locally configured values.
	if err := e.refreshExecutorConfig(); err != nil {
		e.logger.Printf("warning: failed to pull remote executor config at startup (using local config): %v", err)
	}

	// Pull the latest jobs from the server and refresh the cache before we begin
	// scheduling.  If the server is unreachable we log a warning and fall back to
	// whatever is already in the cache.
	if jobs, err := e.fetchRemoteJobs(); err != nil {
		e.logger.Printf("warning: failed to pull remote jobs at startup (using cache): %v", err)
	} else if cacheErr := e.cacheJobs(ctx, jobs); cacheErr != nil {
		e.logger.Printf("warning: failed to update job cache at startup: %v", cacheErr)
	}

	e.logger.Printf("successful updated cache")
	// Restore cached jobs from SQLite so we can continue scheduling even if the
	// server is unreachable at startup.
	e.scheduleFromCache(ctx)

	// listen() processes jobs off the priority queue.
	go e.listen(ctx)

	pollTicker := time.NewTicker(e.pollInterval)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(reportMinInterval)
	defer reportTicker.Stop()

	// Kick off an initial poll immediately.
	e.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			e.logger.Println("local executor shutting down — flushing execution reports")
			e.report(ctx)
			return

		case <-pollTicker.C:
			e.poll(ctx)

		case <-reportTicker.C:
			e.report(ctx)
			e.setLastReportedAt(time.Now())

		case <-e.reportTriggerCh:
			// Threshold-triggered early report; respect the minimum interval.
			if time.Since(e.getLastReportedAt()) >= reportMinInterval {
				e.report(ctx)
				e.setLastReportedAt(time.Now())
			}
		}
	}
}

// Close releases resources held by the executor.
func (e *Executor) Close() {
	if e.db != nil {
		_ = e.db.Close()
	}
}

// ---------------------------------------------------------------------------
// listen — mirrors ListenForJobsToInvokeV1 in the private server
// ---------------------------------------------------------------------------

// listen is the single goroutine that drains the schedule queue.  It sleeps
// until the head of the min-heap is due, then fires the job in a new goroutine.
func (e *Executor) listen(ctx context.Context) {
	for {
		next := e.queue.Peek()

		var sleep time.Duration
		if next.JobId == 0 {
			// Queue is empty — poll every second.
			sleep = time.Second
		} else {
			sleep = time.Until(next.ExecutionTime)
			if sleep < 0 {
				sleep = 0
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-e.jobAddedChan:
			// A new job was added — re-evaluate the queue head immediately.
			continue
		case <-time.After(sleep):
		}

		// Re-peek after waking up (a new job may have a sooner execution time).
		next = e.queue.Peek()
		if next.JobId == 0 || time.Now().Before(next.ExecutionTime) {
			continue
		}

		e.queue.Pop()

		e.mu.Lock()
		state, ok := e.jobCache[next.JobId]
		e.mu.Unlock()
		if !ok {
			e.logger.Printf("listen: job %d no longer in cache, skipping", next.JobId)
			continue
		}

		// Run in a separate goroutine so the listener is never blocked.
		go e.runJob(ctx, state)
	}
}

// ---------------------------------------------------------------------------
// poll — fetch jobs from server and refresh the schedule queue
// ---------------------------------------------------------------------------

// poll fetches the latest job list from the server and updates the local
// schedule.  On network failure it logs a warning and continues using the
// cached jobs (offline execution).
func (e *Executor) poll(ctx context.Context) {
	e.logger.Printf("polling for jobs executor_id=%d", e.executorID)

	// Refresh the executor's own config (command/workingDir) alongside its jobs.
	if err := e.refreshExecutorConfig(); err != nil {
		e.logger.Printf("poll: failed to refresh executor config (using current config): %v", err)
	}

	jobs, err := e.fetchRemoteJobs()
	if err != nil {
		e.logger.Printf("poll failed (will use cache): %v", err)
		return
	}

	if cacheErr := e.cacheJobs(ctx, jobs); cacheErr != nil {
		e.logger.Printf("poll: %v", cacheErr)
		return
	}

	freshIDs := make(map[int64]bool, len(jobs))
	for _, job := range jobs {
		freshIDs[job.ID] = true
	}

	// Remove queue/cache entries for jobs that are no longer assigned.
	e.mu.Lock()
	for jobID := range e.jobCache {
		if !freshIDs[jobID] {
			delete(e.jobCache, jobID)
			e.queue.RemoveByJobId(jobID)
			e.logger.Printf("poll: removed stale job jobId=%d", jobID)
		}
	}
	e.mu.Unlock()

	// Add or refresh entries for fresh jobs.
	for _, job := range jobs {
		if job.Status != "" && job.Status != "active" {
			continue
		}

		ended, endErr := hasJobEnded(job)
		if endErr != nil {
			e.logger.Printf("poll: hasJobEnded error jobId=%d: %v", job.ID, endErr)
		}
		if ended {
			e.logger.Printf("poll: skipping ended job jobId=%d", job.ID)
			continue
		}

		e.mu.Lock()
		existing, exists := e.jobCache[job.ID]
		e.mu.Unlock()

		if !exists {
			// Brand new job — schedule it from scratch.
			e.scheduleJob(ctx, job, time.Time{}, 1)
		} else if existing.job.Spec != job.Spec {
			// Spec changed — remove the old entry and re-schedule.
			e.queue.RemoveByJobId(job.ID)
			e.scheduleJob(ctx, job, existing.lastExecTime, existing.execVersion)
		} else {
			// Keep the existing schedule; just update the job definition in cache
			// in case non-scheduling fields changed.
			e.mu.Lock()
			existing.job = job
			e.mu.Unlock()
		}
	}
}

// refreshExecutorConfig pulls the remote copy of this executor and updates the
// local configuration (command and working directory) so the CLI always runs
// with the latest definition.  It returns an error on transport failure or an
// unsuccessful response.
func (e *Executor) refreshExecutorConfig() error {
	resp, err := e.client.GetExecutor(strconv.FormatInt(e.executorID, 10))
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("server returned success=false")
	}

	e.cfgMu.Lock()
	defer e.cfgMu.Unlock()
	if resp.Data.Command != "" && resp.Data.Command != e.command {
		e.logger.Printf("updating command from remote: %q -> %q", e.command, resp.Data.Command)
		e.command = resp.Data.Command
	}
	if resp.Data.WorkingDir != e.workingDir {
		e.logger.Printf("updating workingDir from remote: %q -> %q", e.workingDir, resp.Data.WorkingDir)
		e.workingDir = resp.Data.WorkingDir
	}
	return nil
}

// getCommand returns the current command under cfgMu.
func (e *Executor) getCommand() string {
	e.cfgMu.RLock()
	defer e.cfgMu.RUnlock()
	return e.command
}

// getWorkingDir returns the current working directory under cfgMu.
func (e *Executor) getWorkingDir() string {
	e.cfgMu.RLock()
	defer e.cfgMu.RUnlock()
	return e.workingDir
}

// fetchRemoteJobs pulls the latest job list assigned to this executor from the
// server.  It returns an error on transport failure or an unsuccessful response.
func (e *Executor) fetchRemoteJobs() ([]scheduler0.Job, error) {
	resp, err := e.client.PullLocalExecutorJobs(e.executorID)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("server returned success=false")
	}
	return resp.Data, nil
}

// cacheJobs upserts the given jobs into the SQLite jobs_cache so they survive
// restarts and remain available while the server is unreachable.
func (e *Executor) cacheJobs(ctx context.Context, jobs []scheduler0.Job) error {
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin cache tx: %w", err)
	}

	for _, job := range jobs {
		_, upsertErr := tx.ExecContext(ctx, `
			INSERT INTO jobs_cache
				(id, executor_id, project_id, account_id, spec, data, timezone, timezone_offset,
				 start_date, end_date, retry_max, status, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(id) DO UPDATE SET
				spec=excluded.spec,
				data=excluded.data,
				status=excluded.status,
				updated_at=excluded.updated_at
		`,
			job.ID, e.executorID, job.ProjectID, job.AccountID,
			job.Spec, job.Data, job.Timezone, job.TimezoneOffset,
			job.StartDate, job.EndDate, job.RetryMax, job.Status,
		)
		if upsertErr != nil {
			e.logger.Printf("cacheJobs: cache upsert failed jobId=%d: %v", job.ID, upsertErr)
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("failed to commit cache tx: %w", commitErr)
	}
	return nil
}

// scheduleFromCache initialises the schedule from the SQLite cache.  Called at
// startup so the executor can run jobs even if the server is unreachable.
func (e *Executor) scheduleFromCache(ctx context.Context) {
	// Seed each job's last execution time from its most recent uncommitted
	// execution in a single query. The LEFT JOIN against a correlated subquery
	// keeps the per-job lookup inside the same result set, so we never issue a
	// second query while iterating over these rows — which would otherwise
	// deadlock against the single-connection pool (db.SetMaxOpenConns(1)).
	rows, err := e.db.QueryContext(ctx, `
		SELECT j.id, j.spec, j.data, j.timezone, j.timezone_offset, j.start_date,
		       j.end_date, j.retry_max, j.status, j.account_id, j.project_id,
		       e.last_execution_time, e.execution_version
		FROM jobs_cache j
		LEFT JOIN executions_uncommitted e
		  ON e.id = (
		      SELECT eu.id
		      FROM executions_uncommitted eu
		      WHERE eu.job_id = j.id
		      ORDER BY eu.created_at DESC, eu.id DESC
		      LIMIT 1
		  )
		WHERE j.executor_id = ?
	`, e.executorID)
	if err != nil {
		e.logger.Printf("scheduleFromCache: failed to query cache: %v", err)
		return
	}
	defer rows.Close()

	// Buffer everything we need before scheduling. scheduleJob writes to the DB
	// (writeExecution), and we must not run a write while these rows are still
	// open — with a single-connection pool that would deadlock against the open
	// cursor. So we drain the result set, close it, then schedule.
	type pending struct {
		job          scheduler0.Job
		lastExecTime time.Time
		version      uint64
	}
	var toSchedule []pending

	for rows.Next() {
		var job scheduler0.Job
		var timezoneOffset int64
		var lastExecTimeStr sql.NullString
		var lastVersion sql.NullInt64
		if scanErr := rows.Scan(
			&job.ID, &job.Spec, &job.Data, &job.Timezone, &timezoneOffset,
			&job.StartDate, &job.EndDate, &job.RetryMax, &job.Status,
			&job.AccountID, &job.ProjectID,
			&lastExecTimeStr, &lastVersion,
		); scanErr != nil {
			e.logger.Printf("scheduleFromCache: scan error: %v", scanErr)
			continue
		}

		job.TimezoneOffset = timezoneOffset

		if job.Status != "" && job.Status != "active" {
			continue
		}
		if ended, _ := hasJobEnded(job); ended {
			continue
		}

		e.mu.Lock()
		_, alreadyScheduled := e.jobCache[job.ID]
		e.mu.Unlock()
		if alreadyScheduled {
			continue
		}

		var lastExecTime time.Time
		if lastExecTimeStr.Valid && lastExecTimeStr.String != "" {
			if t, parseErr := parseJobTime(lastExecTimeStr.String); parseErr == nil {
				lastExecTime = t
			}
		}

		toSchedule = append(toSchedule, pending{
			job:          job,
			lastExecTime: lastExecTime,
			version:      uint64(lastVersion.Int64) + 1,
		})
	}
	if err := rows.Err(); err != nil {
		e.logger.Printf("scheduleFromCache: row iteration error: %v", err)
	}
	rows.Close()

	for _, p := range toSchedule {
		e.scheduleJob(ctx, p.job, p.lastExecTime, p.version)
	}
}

// ---------------------------------------------------------------------------
// scheduleJob — compute next execution time and push to queue
// ---------------------------------------------------------------------------

// scheduleJob computes the next execution time for a job, writes a
// "scheduled" execution log, updates jobCache, and pushes the entry onto the
// priority queue.  It mirrors AddJobSchedule / ScheduleJobs in the private
// server.
func (e *Executor) scheduleJob(ctx context.Context, job scheduler0.Job, lastExecTime time.Time, execVersion uint64) {
	nextTime, err := getNextExecutionTime(job, lastExecTime)
	if err != nil {
		e.logger.Printf("scheduleJob: failed to compute next execution time jobId=%d: %v", job.ID, err)
		return
	}

	uniqueID := getNextExecutionID(job, lastExecTime, nextTime)

	// Write the "scheduled" execution log — this is the authoritative record
	// that a job has been queued for execution, matching the private server's
	// behaviour.
	e.writeExecution(ctx, job.ID, uniqueID, stateScheduled, lastExecTime, nextTime, execVersion)

	e.logger.Printf("scheduled job jobId=%d nextExecTime=%s", job.ID, nextTime.Format(time.RFC3339))

	// Update in-memory cache.
	e.mu.Lock()
	e.jobCache[job.ID] = &jobExecState{
		job:          job,
		execVersion:  execVersion,
		lastExecTime: lastExecTime,
		lastState:    stateScheduled,
	}
	e.mu.Unlock()

	// Push onto the priority queue and wake the listener.
	e.queue.Push(scheduleQueueEntry{JobId: job.ID, ExecutionTime: nextTime})
	select {
	case e.jobAddedChan <- struct{}{}:
	default:
	}
}

// ---------------------------------------------------------------------------
// runJob — execute the shell command
// ---------------------------------------------------------------------------

// runJob executes the shell command for a job.  It must be called in its own
// goroutine.  After completion it re-schedules the job for its next execution.
func (e *Executor) runJob(ctx context.Context, state *jobExecState) {
	jobID := state.job.ID

	// Snapshot the executor config so a concurrent refresh can't change it mid-run.
	command := e.getCommand()
	workingDir := e.getWorkingDir()
	e.logger.Printf("running job jobId=%d command=%s", jobID, command)

	parts := splitCommand(command)
	if len(parts) == 0 {
		e.logger.Printf("runJob: empty command for executor, skipping jobId=%d", jobID)
		return
	}

	// Snapshot the state we need before running — the cache may be mutated
	// concurrently by poll().
	e.mu.Lock()
	current, ok := e.jobCache[jobID]
	e.mu.Unlock()
	if !ok {
		e.logger.Printf("runJob: job %d no longer in cache, skipping", jobID)
		return
	}

	jobData := current.job.Data
	jobSpec := current.job.Spec
	execVersion := current.execVersion
	lastExecTime := current.lastExecTime

	// Derive the uniqueID we wrote when the job was scheduled so success/failed
	// logs share the same uniqueID as their corresponding scheduled log.
	nextTime, nextErr := getNextExecutionTime(current.job, lastExecTime)
	if nextErr != nil {
		e.logger.Printf("runJob: cannot derive uniqueID for jobId=%d: %v", jobID, nextErr)
		return
	}
	uniqueID := getNextExecutionID(current.job, lastExecTime, nextTime)

	executionStart := time.Now().UTC()

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Stdin = strings.NewReader(jobData)
	cmd.Env = append(os.Environ(),
		"SCHEDULER0_JOB_ID="+strconv.FormatInt(jobID, 10),
		"SCHEDULER0_JOB_DATA="+jobData,
		"SCHEDULER0_JOB_SPEC="+jobSpec,
		"SCHEDULER0_EXEC_UNIQUE_ID="+uniqueID,
	)
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	e.logger.Printf("job START jobId=%d exec_version=%d uniqueId=%s command=%q stdin=%q",
		jobID, execVersion, uniqueID, e.command, jobData)

	runErr := cmd.Run()
	completedAt := time.Now().UTC()
	duration := completedAt.Sub(executionStart)

	stdoutStr := stdoutBuf.String()
	stderrStr := stderrBuf.String()

	e.logger.Printf("job IO jobId=%d exec_version=%d duration=%s\nstdout: %s\nstderr: %s",
		jobID, execVersion, duration, stdoutStr, stderrStr)

	var resultState int
	if runErr != nil {
		resultState = stateFailed
		e.logger.Printf("job FAILED jobId=%d exec_version=%d duration=%s: %v",
			jobID, execVersion, duration, runErr)
	} else {
		resultState = stateSuccess
		e.logger.Printf("job SUCCEEDED jobId=%d exec_version=%d duration=%s", jobID, execVersion, duration)
	}

	e.writeExecution(ctx, jobID, uniqueID, resultState, executionStart, completedAt, execVersion)

	// Update state and re-schedule the job for its next execution.
	newVersion := execVersion + 1
	newLastExecTime := executionStart

	e.mu.Lock()
	if s, exists := e.jobCache[jobID]; exists {
		s.lastExecTime = newLastExecTime
		s.execVersion = newVersion
		s.lastState = resultState
		if resultState == stateFailed {
			s.failCount++
		} else {
			s.failCount = 0
		}
	}
	e.mu.Unlock()

	// Re-read the job from cache (it may have been updated by a concurrent poll).
	e.mu.Lock()
	updatedJob := e.jobCache[jobID]
	e.mu.Unlock()

	if updatedJob != nil {
		e.scheduleJob(ctx, updatedJob.job, newLastExecTime, newVersion)
	}
}

// ---------------------------------------------------------------------------
// writeExecution — persist to SQLite and maybe trigger the reporter
// ---------------------------------------------------------------------------

// writeExecution inserts one execution row into executions_uncommitted.
// If the number of unreported rows reaches reportThreshold it signals the
// reporter channel.
func (e *Executor) writeExecution(ctx context.Context, jobID int64, uniqueID string, state int,
	lastTime, nextTime time.Time, version uint64) {

	lastTimeStr := ""
	nextTimeStr := ""
	if !lastTime.IsZero() {
		lastTimeStr = lastTime.Format(time.RFC3339)
	}
	if !nextTime.IsZero() {
		nextTimeStr = nextTime.Format(time.RFC3339)
	}

	_, err := e.db.ExecContext(ctx, `
		INSERT INTO executions_uncommitted
			(job_id, unique_id, state, last_execution_time, next_execution_time,
			 execution_version, reported)
		VALUES (?, ?, ?, ?, ?, ?, 0)
	`, jobID, uniqueID, state, lastTimeStr, nextTimeStr, version)
	if err != nil {
		e.logger.Printf("writeExecution: failed to write execution jobId=%d state=%d: %v",
			jobID, state, err)
		return
	}

	// Count unreported rows and trigger early reporting when the threshold is reached.
	var count int
	if scanErr := e.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM executions_uncommitted WHERE reported = 0`,
	).Scan(&count); scanErr == nil && count >= reportThreshold {
		select {
		case e.reportTriggerCh <- struct{}{}:
		default:
		}
	}
}

// ---------------------------------------------------------------------------
// report — batch-send unreported executions to the server
// ---------------------------------------------------------------------------

// report reads unreported executions from SQLite, sends them to the server,
// and marks them as reported on success.  On failure it retains them for
// the next attempt (offline-tolerant catch-up).
func (e *Executor) report(ctx context.Context) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT id, job_id, unique_id, state,
		       last_execution_time, next_execution_time,
		       execution_version, job_queue_version
		FROM executions_uncommitted
		WHERE reported = 0
		ORDER BY created_at
		LIMIT 200
	`)
	if err != nil {
		e.logger.Printf("report: failed to query unreported executions: %v", err)
		return
	}

	type uncommitted struct {
		rowID  int64
		report scheduler0.LocalExecutionReport
	}

	var batch []uncommitted
	for rows.Next() {
		var u uncommitted
		if scanErr := rows.Scan(
			&u.rowID,
			&u.report.JobID,
			&u.report.UniqueID,
			&u.report.State,
			&u.report.LastExecutionTime,
			&u.report.NextExecutionTime,
			&u.report.ExecutionVersion,
			&u.report.JobQueueVersion,
		); scanErr != nil {
			e.logger.Printf("report: scan error: %v", scanErr)
			continue
		}
		batch = append(batch, u)
	}
	rows.Close()

	if len(batch) == 0 {
		return
	}

	reports := make([]scheduler0.LocalExecutionReport, 0, len(batch))
	for _, u := range batch {
		reports = append(reports, u.report)
	}

	_, reportErr := e.client.ReportLocalExecutions(e.executorID, reports)
	if reportErr != nil {
		e.logger.Printf("report: failed to report %d executions (will retry): %v",
			len(reports), reportErr)
		return
	}

	// Mark rows as reported.
	ids := make([]interface{}, 0, len(batch))
	placeholders := make([]string, 0, len(batch))
	for _, u := range batch {
		ids = append(ids, u.rowID)
		placeholders = append(placeholders, "?")
	}

	query := fmt.Sprintf(
		"UPDATE executions_uncommitted SET reported = 1 WHERE id IN (%s)",
		strings.Join(placeholders, ","),
	)
	if _, updateErr := e.db.ExecContext(ctx, query, ids...); updateErr != nil {
		e.logger.Printf("report: failed to mark executions as reported: %v", updateErr)
		return
	}

	e.logger.Printf("report: reported %d executions to server", len(reports))
}

// ---------------------------------------------------------------------------
// lastReportedAt helpers (mutex-protected)
// ---------------------------------------------------------------------------

func (e *Executor) getLastReportedAt() time.Time {
	e.reportMu.Lock()
	defer e.reportMu.Unlock()
	return e.lastReportedAt
}

func (e *Executor) setLastReportedAt(t time.Time) {
	e.reportMu.Lock()
	defer e.reportMu.Unlock()
	e.lastReportedAt = t
}

// ---------------------------------------------------------------------------
// splitCommand
// ---------------------------------------------------------------------------

// splitCommand tokenises a simple shell command string by whitespace.
func splitCommand(command string) []string {
	return strings.Fields(command)
}
