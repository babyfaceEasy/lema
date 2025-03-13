package tasks

import (
	"context"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

func (t *Task) HandleCommitsUpdateTask(ctx context.Context, a *asynq.Task) error {
	t.logger.Info("inside HandleCommitsUpdateTask ", zap.String("val", "testing"))
	return nil
}
