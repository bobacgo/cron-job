package repository

import (
	"database/sql"

	"github.com/bobacgo/cron-job/internal/config"
)

type Repo struct {
	Job          JobRepository
	JobRun       JobRunRepository
	Dependencies DependencyRepository
	Log          LogRepository
}

func NewRepo(cfg *config.Config, db *sql.DB) *Repo {
	logRepo, err := NewFileLogRepository(cfg.LogDir)
	if err != nil {
		panic(err)
	}
	return &Repo{
		Job:          &jobRepo{db: db},
		JobRun:       &jobRunRepo{db: db},
		Dependencies: &dependencyRepo{db: db},
		Log:          logRepo,
	}
}
