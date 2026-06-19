// Package scheduler runs in-process periodic background work for the API.
//
// Today it powers the daily pre-match debate generator (see prewarm.go). Kept
// dependency-light on purpose: a plain time.Timer for "next 04:00 UTC", and
// Redis SetNX for cross-machine deduplication. If the worker surface grows
// beyond one or two jobs we should graduate to robfig/cron in a dedicated
// services/workers binary (see the team note in services/workers/main.go).
package scheduler

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

// Job is a single piece of background work. Implementations should be
// idempotent — the scheduler may invoke Run for the same logical day both at
// boot (catch-up) and at the regular tick if a prior run failed before
// acquiring its Redis lock.
type Job interface {
	// Name is used purely for logs.
	Name() string
	// Run does the work. The context is cancelled on graceful shutdown; jobs
	// should respect it and exit promptly.
	Run(ctx context.Context) error
}

// Options configures the Scheduler.
type Options struct {
	// DailyAtUTC is the wall-clock UTC time of day to run the job (only the
	// Hour/Minute/Second are read). Zero value (00:00:00) runs at UTC midnight.
	DailyAtUTC time.Time
	// RunOnStart, when true, runs the job once shortly after Start() returns
	// (after BootDelay) so a fresh deploy doesn't leave today's work undone.
	RunOnStart bool
	// BootDelay defers the catch-up run so the HTTP server has time to start
	// serving (the pre-warm uses loopback HTTP to itself). Default 15s.
	BootDelay time.Duration
	// JobTimeout caps how long a single Run is allowed to take. Default 30 min.
	JobTimeout time.Duration
}

// Scheduler invokes a Job on a daily schedule and (optionally) once on start.
type Scheduler struct {
	job  Job
	opts Options

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	doneCh    chan struct{}
}

// New constructs a Scheduler. Call Start to begin ticking, Stop for graceful
// shutdown.
func New(job Job, opts Options) *Scheduler {
	if opts.BootDelay <= 0 {
		opts.BootDelay = 15 * time.Second
	}
	if opts.JobTimeout <= 0 {
		opts.JobTimeout = 30 * time.Minute
	}
	return &Scheduler{
		job:    job,
		opts:   opts,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.startOnce.Do(func() {
		go s.run(ctx)
	})
}

// Stop signals the loop to exit and waits for the current iteration (if any)
// to settle, capped by the scheduler's job timeout.
func (s *Scheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	select {
	case <-s.doneCh:
	case <-time.After(s.opts.JobTimeout + 5*time.Second):
		log.Printf("[scheduler:%s] Stop: timed out waiting for run to settle", s.job.Name())
	}
}

func (s *Scheduler) run(ctx context.Context) {
	defer close(s.doneCh)
	name := s.job.Name()
	log.Printf("[scheduler:%s] starting (daily at %02d:%02d:%02d UTC, runOnStart=%v)", name,
		s.opts.DailyAtUTC.Hour(), s.opts.DailyAtUTC.Minute(), s.opts.DailyAtUTC.Second(), s.opts.RunOnStart)

	if s.opts.RunOnStart {
		select {
		case <-time.After(s.opts.BootDelay):
			s.runOnce(ctx, "boot-catchup")
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		}
	}

	for {
		next := nextOccurrenceUTC(time.Now().UTC(), s.opts.DailyAtUTC)
		delay := time.Until(next)
		log.Printf("[scheduler:%s] next run at %s UTC (in %s)", name, next.Format(time.RFC3339), delay.Truncate(time.Second))

		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
			s.runOnce(ctx, "daily-tick")
		case <-ctx.Done():
			timer.Stop()
			return
		case <-s.stopCh:
			timer.Stop()
			return
		}
	}
}

func (s *Scheduler) runOnce(parent context.Context, reason string) {
	name := s.job.Name()
	runCtx, cancel := context.WithTimeout(parent, s.opts.JobTimeout)
	defer cancel()
	start := time.Now()
	log.Printf("[scheduler:%s] run begin (reason=%s)", name, reason)
	if err := s.job.Run(runCtx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			log.Printf("[scheduler:%s] run aborted after %s (reason=%s): %v", name, time.Since(start).Truncate(time.Millisecond), reason, err)
			return
		}
		log.Printf("[scheduler:%s] run error after %s (reason=%s): %v", name, time.Since(start).Truncate(time.Millisecond), reason, err)
		return
	}
	log.Printf("[scheduler:%s] run ok in %s (reason=%s)", name, time.Since(start).Truncate(time.Millisecond), reason)
}

// nextOccurrenceUTC returns the next UTC instant whose H:M:S equals timeOfDay.
// If today's slot is already in the past, returns tomorrow's slot.
func nextOccurrenceUTC(now time.Time, timeOfDay time.Time) time.Time {
	now = now.UTC()
	candidate := time.Date(now.Year(), now.Month(), now.Day(),
		timeOfDay.Hour(), timeOfDay.Minute(), timeOfDay.Second(), 0, time.UTC)
	if !candidate.After(now) {
		candidate = candidate.Add(24 * time.Hour)
	}
	return candidate
}
