package queue

import "context"

type Task struct {
	Type    string
	Payload interface{}
}

type TaskQueue interface {
	Enqueue(ctx context.Context, task Task) error
}
