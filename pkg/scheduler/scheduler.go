package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CronJob represents a scheduled job.
type CronJob struct {
	ID       string
	Interval time.Duration
	Func     func()
	running  bool
	stopCh   chan struct{}
}

// CronScheduler manages periodic tasks.
type CronScheduler struct {
	mu    sync.RWMutex
	jobs  map[string]*CronJob
	ctx   context.Context
	cancel context.CancelFunc
}

// NewCronScheduler creates a new cron scheduler.
func NewCronScheduler() *CronScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &CronScheduler{
		jobs:   make(map[string]*CronJob),
		ctx:    ctx,
		cancel: cancel,
	}
}

// AddJob adds a periodic job.
func (cs *CronScheduler) AddJob(id string, interval time.Duration, fn func()) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if _, exists := cs.jobs[id]; exists {
		return fmt.Errorf("job %s already exists", id)
	}
	job := &CronJob{ID: id, Interval: interval, Func: fn, stopCh: make(chan struct{})}
	cs.jobs[id] = job
	go cs.runJob(job)
	return nil
}

func (cs *CronScheduler) runJob(job *CronJob) {
	cs.mu.Lock()
	job.running = true
	cs.mu.Unlock()

	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-cs.ctx.Done():
			return
		case <-job.stopCh:
			return
		case <-ticker.C:
			job.Func()
		}
	}
}

// RemoveJob removes a job.
func (cs *CronScheduler) RemoveJob(id string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if job, ok := cs.jobs[id]; ok {
		close(job.stopCh)
		delete(cs.jobs, id)
	}
}

// Stop stops all jobs.
func (cs *CronScheduler) Stop() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.cancel()
	for _, job := range cs.jobs {
		if job.running {
			close(job.stopCh)
		}
	}
	cs.jobs = make(map[string]*CronJob)
}

// JobCount returns the number of active jobs.
func (cs *CronScheduler) JobCount() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return len(cs.jobs)
}

// RetryConfig holds retry parameters.
type RetryConfig struct {
	MaxRetries int
	Delay      time.Duration
}

// RetryFunc executes a function with retry logic.
func RetryFunc(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error
	for i := 0; i <= config.MaxRetries; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := fn(); err != nil {
			lastErr = err
			if i < config.MaxRetries {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(config.Delay):
				}
			}
		} else {
			return nil
		}
	}
	return fmt.Errorf("max retries exceeded: %w", lastErr)
}
