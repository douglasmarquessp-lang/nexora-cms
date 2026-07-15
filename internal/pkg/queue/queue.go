package queue

import (
	"context"
)

type Job struct {
	ID      string
	Type    string
	Payload interface{}
	SiteID  string
}

type Queue interface {
	Enqueue(ctx context.Context, job Job) error
	Dequeue(ctx context.Context, jobType string) (*Job, error)
	Ack(ctx context.Context, jobID string) error
	Fail(ctx context.Context, jobID string, err error) error
	Len(ctx context.Context, jobType string) (int, error)
}
