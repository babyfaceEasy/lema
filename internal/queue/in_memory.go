package queue

import (
	"context"
	"errors"
	"sync"

	"github.com/babyfaceeasy/lema/internal/domain"
	"go.uber.org/zap"
)

type InMemoryQueue struct {
	logger        *zap.Logger
	tasks         chan Task
	workers       int
	wg            sync.WaitGroup
	commitService domain.CommitService
	repoService   domain.RepositoryService
}

func NewInMemoryQueue(workerCount int, queueSize int, logger *zap.Logger, commitSvc domain.CommitService, repoSvc domain.RepositoryService) TaskQueue {
	logger = logger.With(zap.String("package", "queue"))
	q := &InMemoryQueue{
		logger:        logger,
		tasks:         make(chan Task, queueSize),
		workers:       workerCount,
		commitService: commitSvc,
		repoService:   repoSvc,
	}
	q.startWorkers()
	return q
}

func (q *InMemoryQueue) startWorkers() {
	for i := 0; i < q.workers; i++ {
		go q.worker()
	}
}

/*
func (q *InMemoryQueue) worker() {
	for task := range q.tasks {
		data := payload.(map[string]string)
		ctx := context.Background()
		q.logger.Info("processing task", zap.String("task_type", task.Type))
		if task.Type == "ops:reset_commits" {
			err := q.commitService.ResetCommits(ctx, data["RepositoryOwner"], data["RepositoryName"])
			if err != nil {

			}
		}
		q.wg.Done()
	}
}
*/

func (q *InMemoryQueue) worker() {
	for task := range q.tasks {
		q.logger.Info("processing task", zap.String("task_type", task.Type))
		// implement the code / call the function to run here
		q.wg.Done()
	}
}

func (q *InMemoryQueue) Enqueue(ctx context.Context, task Task) error {
	select {
	case q.tasks <- task:
		q.wg.Add(1)
		return nil
	case <-ctx.Done():
		return errors.New("failed to enqueue task: context cancelled")
	}
}

func (q *InMemoryQueue) Wait() {
	q.wg.Wait()
}
