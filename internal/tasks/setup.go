package tasks

import (
	"os"
	"time"

	"github.com/babyfaceeasy/lema/config"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var client *asynq.Client

const (
	TypeQueueCritical = "critical"
	TypeQueueDefault  = "default"
)

// config for periodic task
type FileBasedConfigProvider struct {
	filename string
}

type PeriodicTaskConfigContainer struct {
	Configs []*Config `yaml:"configs"`
}

type Config struct {
	Cronspec string `yaml:"cronspec"`
	TaskType string `yaml:"task_type"`
}

// start worker
func StartWorker(t Task, config *config.Config) error {
	client = asynq.NewClient(asynq.RedisClientOpt{Addr: config.RedisAddress()})

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: config.RedisAddress()},
		asynq.Config{Concurrency: 10, Queues: map[string]int{
			TypeQueueCritical: 3,
			TypeQueueDefault:  1,
		}},
	)

	mux := asynq.NewServeMux()
	mux.Use(t.LoggingMiddleware)

	// tasks
	mux.HandleFunc("ops:load_commits", t.HandleLoadCommitsTask)
	mux.HandleFunc("ops:latest_commits", t.HandleLatestCommitsTask)
	mux.HandleFunc("ops:reset_commits", t.HandleResetCommitsTask)

	// cron
	mux.HandleFunc("cron:commits_update", t.HandleCommitsUpdateTask)

	go func() {
		if err := srv.Run(mux); err != nil {
			t.logger.Fatal("failed to start task server", zap.Error(err))
		}
		t.logger.Info("tasks server started successfully")
	}()

	// for the crons (dynamic periodic task)
	provider := &FileBasedConfigProvider{filename: "./cron.yaml"}

	mgr, err := asynq.NewPeriodicTaskManager(
		asynq.PeriodicTaskManagerOpts{
			RedisConnOpt:               asynq.RedisClientOpt{Addr: config.RedisAddress()},
			PeriodicTaskConfigProvider: provider,
			SyncInterval:               10 * time.Second,
		})
	if err != nil {
		return err
	}

	if err := mgr.Run(); err != nil {
		return err
	}
	t.logger.Info("dynamic periodic stack server started successfully")

	return nil
}

func (p *FileBasedConfigProvider) GetConfigs() ([]*asynq.PeriodicTaskConfig, error) {
	data, err := os.ReadFile(p.filename)
	if err != nil {
		return nil, err
	}

	var c PeriodicTaskConfigContainer
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	var configs []*asynq.PeriodicTaskConfig
	for _, cfg := range c.Configs {
		configs = append(configs, &asynq.PeriodicTaskConfig{Cronspec: cfg.Cronspec, Task: asynq.NewTask(cfg.TaskType, nil, asynq.Retention(24*time.Hour))})
	}
	return configs, nil
}
