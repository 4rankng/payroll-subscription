package scheduler_test

import (
	"testing"

	"api-server/internal/app/services/scheduler"
	"api-server/internal/infra/observability"

	"github.com/stretchr/testify/assert"
)

func TestNewScheduler(t *testing.T) {
	logger := observability.GetLogger()
	s := scheduler.NewScheduler(logger, "UTC", true)
	assert.NotNil(t, s)
}

func TestScheduler_AddJob(t *testing.T) {
	logger := observability.GetLogger()
	s := scheduler.NewScheduler(logger, "UTC", true)

	job := scheduler.Job{
		Name:    "test_job",
		Cron:    "* * * * *",
		Enabled: true,
		Handler: func() {
			// Handler function
		},
	}

	s.AddJob(job)
	// We can't easily inspect private fields, but we can verify no panic
	assert.NotNil(t, s)
}
