package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	jobQueueKey      = "lora-trainer:jobs:pending"
	processingSetKey = "lora-trainer:jobs:processing"
)

type RedisQueue struct {
	client *redis.Client
}

func NewRedisQueue(client *redis.Client) *RedisQueue {
	return &RedisQueue{client: client}
}

func (q *RedisQueue) Enqueue(ctx context.Context, jobID string) error {
	if err := q.client.LPush(ctx, jobQueueKey, jobID).Err(); err != nil {
		return fmt.Errorf("enqueueing job %s: %w", jobID, err)
	}
	return nil
}

func (q *RedisQueue) Dequeue(ctx context.Context, timeout time.Duration) (string, error) {
	result, err := q.client.BRPopLPush(ctx, jobQueueKey, processingSetKey, timeout).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", fmt.Errorf("dequeuing job: %w", err)
	}
	return result, nil
}

func (q *RedisQueue) Ack(ctx context.Context, jobID string) error {
	if err := q.client.LRem(ctx, processingSetKey, 1, jobID).Err(); err != nil {
		return fmt.Errorf("acking job %s: %w", jobID, err)
	}
	return nil
}

func (q *RedisQueue) Nack(ctx context.Context, jobID string) error {
	pipe := q.client.Pipeline()
	pipe.LRem(ctx, processingSetKey, 1, jobID)
	pipe.RPush(ctx, jobQueueKey, jobID)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("nacking job %s: %w", jobID, err)
	}
	return nil
}

func (q *RedisQueue) Depth(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, jobQueueKey).Result()
}

func (q *RedisQueue) ProcessingCount(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, processingSetKey).Result()
}
