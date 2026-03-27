package repository

import (
	"database/sql"

	"github.com/bobacgo/cron-job/internal/config"
	"github.com/bobacgo/cron-job/internal/repository/dependency"
	"github.com/bobacgo/cron-job/internal/repository/job"
	"github.com/bobacgo/cron-job/internal/repository/jobrun"
	"github.com/bobacgo/cron-job/internal/repository/log"
)

type Repo struct {
	Dependencies dependency.Repository
	Job          job.Repository
	JobRun       jobrun.Repository
	Log          log.Repository
}

func NewRepo(cfg *config.Config, db *sql.DB) *Repo {
	logRepo, err := log.NewFileRepository(cfg.LogDir)
	if err != nil {
		panic(err)
	}

	return &Repo{
		Dependencies: dependency.NewMySQLRepository(db),
		Job:          job.NewMySQLRepository(db),
		JobRun:       jobrun.NewMySQLRepository(db),
		Log:          logRepo,
	}
}
