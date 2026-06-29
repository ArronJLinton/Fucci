package scheduler

import (
	"context"
	"log"
	"sync"
	"time"
)

// IntervalJob runs on a fixed interval (e.g. every 15 minutes).
type IntervalJob interface {
	Job
}

// IntervalOptions configures IntervalScheduler.
type IntervalOptions struct {
	Every    time.Duration
	JobTimeout time.Duration
}

// IntervalScheduler invokes a Job on a fixed interval.
type IntervalScheduler struct {
	job  IntervalJob
	opts IntervalOptions

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	doneCh    chan struct{}
}

func NewInterval(job IntervalJob, opts IntervalOptions) *IntervalScheduler {
	if opts.Every <= 0 {
		opts.Every = 15 * time.Minute
	}
	if opts.JobTimeout <= 0 {
		opts.JobTimeout = 5 * time.Minute
	}
	return &IntervalScheduler{
		job:    job,
		opts:   opts,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

func (s *IntervalScheduler) Start(ctx context.Context) {
	s.startOnce.Do(func() {
		go s.run(ctx)
	})
}

func (s *IntervalScheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	select {
	case <-s.doneCh:
	case <-time.After(s.opts.JobTimeout + 5*time.Second):
		log.Printf("[interval:%s] Stop: timed out", s.job.Name())
	}
}

func (s *IntervalScheduler) run(ctx context.Context) {
	defer close(s.doneCh)
	name := s.job.Name()
	log.Printf("[interval:%s] starting (every %s)", name, s.opts.Every.Truncate(time.Second))
	ticker := time.NewTicker(s.opts.Every)
	defer ticker.Stop()

	s.runOnce(ctx, "start")
	for {
		select {
		case <-ticker.C:
			s.runOnce(ctx, "tick")
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		}
	}
}

func (s *IntervalScheduler) runOnce(parent context.Context, reason string) {
	runCtx, cancel := context.WithTimeout(parent, s.opts.JobTimeout)
	defer cancel()
	if err := s.job.Run(runCtx); err != nil {
		log.Printf("[interval:%s] run error (reason=%s): %v", s.job.Name(), reason, err)
	}
}
