package tasks

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

func (tsk *Task) LoggingMiddleware(h asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
		start := time.Now()

		tsk.logger.Info("start processing", zap.Any("task-type", t.Type()))

		err := h.ProcessTask(ctx, t)
		if err != nil {
			return err
		}
		tsk.logger.Info("finished processing", zap.Any("task-type", t.Type()), zap.Any("elapsed-time", time.Since(start)))
		return nil
	})
}
