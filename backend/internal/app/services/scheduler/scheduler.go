package scheduler

import (
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/infra/observability"

	"github.com/robfig/cron/v3"
)

type Job struct {
	Name    string
	Cron    string
	Handler func()
	Enabled bool
}

type Scheduler struct {
	cron     *cron.Cron
	logger   *slog.Logger
	timezone string
	enabled  bool
	jobs     []Job
	entryMap map[string]cron.EntryID
}

func NewScheduler(
	logger *slog.Logger,
	timezone string,
	enabled bool,
) *Scheduler {
	if logger == nil {
		logger = observability.GetLogger()
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		logger.Warn("Invalid timezone, using UTC", "timezone", timezone, "error", err)
		loc = time.UTC
	}

	return &Scheduler{
		cron:     cron.New(cron.WithLocation(loc)),
		logger:   logger,
		timezone: timezone,
		enabled:  enabled,
		jobs:     make([]Job, 0),
		entryMap: make(map[string]cron.EntryID),
	}
}

func (s *Scheduler) AddJob(job Job) {
	s.jobs = append(s.jobs, job)
}

func (s *Scheduler) Start() error {
	if !s.enabled {
		s.logger.Info("Scheduler is disabled")
		return nil
	}

	for _, job := range s.jobs {
		if !job.Enabled {
			s.logger.Info("Job disabled, skipping", "name", job.Name)
			continue
		}

		handler := s.wrapHandler(job.Name, job.Handler)
		entryID, err := s.cron.AddFunc(job.Cron, handler)
		if err != nil {
			return fmt.Errorf("failed to schedule %s: %w", job.Name, err)
		}

		s.entryMap[job.Name] = entryID
		s.logger.Info("Job registered", "name", job.Name, "cron", job.Cron)
	}

	s.cron.Start()
	s.logger.Info("Scheduler started")
	return nil
}

func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	s.logger.Info("Scheduler stopping, waiting for running jobs...")

	select {
	case <-ctx.Done():
		s.logger.Info("Scheduler stopped, all jobs completed")
	case <-time.After(5 * time.Second):
		s.logger.Warn("Scheduler stop timed out, some jobs may still be running")
	}
}

func (s *Scheduler) wrapHandler(jobName string, handler func()) func() {
	return func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("Job panicked", "name", jobName, "error", fmt.Errorf("panic: %v", r))
				panic(r)
			}
		}()

		handler()
	}
}
