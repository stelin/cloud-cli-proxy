package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Job struct {
	Name     string
	Interval time.Duration
	Fn       func(ctx context.Context) error
}

type Scheduler struct {
	logger *slog.Logger
	jobs   []Job
}

func New(logger *slog.Logger, jobs []Job) *Scheduler {
	return &Scheduler{logger: logger, jobs: jobs}
}

func (s *Scheduler) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, job := range s.jobs {
		wg.Add(1)
		go func(j Job) {
			defer wg.Done()
			ticker := time.NewTicker(j.Interval)
			defer ticker.Stop()
			s.logger.Info("scheduler job started", "job", j.Name, "interval", j.Interval)
			for {
				select {
				case <-ctx.Done():
					s.logger.Info("scheduler job stopping", "job", j.Name)
					return
				case <-ticker.C:
					if err := j.Fn(ctx); err != nil {
						s.logger.Error("scheduler job failed", "job", j.Name, "error", err)
					}
				}
			}
		}(job)
	}
	wg.Wait()
}
