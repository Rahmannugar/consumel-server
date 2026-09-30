package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	consumptionservices "github.com/Rahmannugar/consumel-server/internal/core/consumption/services"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const operationStreamBatchSize = 100

type OperationSource struct {
	redis *redis.Client
}

func NewOperationSource(redisClient *redis.Client) *OperationSource {
	return &OperationSource{redis: redisClient}
}

func (source *OperationSource) Read(
	ctx context.Context,
	after string,
	block time.Duration,
) ([]consumptionservices.OperationStreamNotice, string, error) {
	if after == "" {
		after = "$"
	}
	streams, err := source.redis.XRead(ctx, &redis.XReadArgs{
		Streams: []string{StreamName, after}, Count: operationStreamBatchSize, Block: block,
	}).Result()
	if errors.Is(err, redis.Nil) {
		return nil, after, nil
	}
	if err != nil {
		return nil, after, fmt.Errorf("read usage operation stream: %w", err)
	}
	notices := make([]consumptionservices.OperationStreamNotice, 0, operationStreamBatchSize)
	latest := after
	for _, stream := range streams {
		for _, message := range stream.Messages {
			latest = message.ID
			event, err := ParseStreamEvent(message)
			if err != nil {
				return nil, latest, err
			}
			if event.Type != "usage.consumed.v1" && event.Type != "usage.denied.v1" {
				continue
			}
			var payload struct {
				ProjectEnvironmentID string `json:"projectEnvironmentId"`
				UsageEventID         string `json:"usageEventId"`
			}
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return nil, latest, fmt.Errorf("decode usage operation stream payload: %w", err)
			}
			environmentID, err := uuid.Parse(payload.ProjectEnvironmentID)
			if err != nil {
				return nil, latest, fmt.Errorf("parse usage operation environment ID: %w", err)
			}
			operationID, err := uuid.Parse(payload.UsageEventID)
			if err != nil || operationID != event.AggregateID {
				return nil, latest, fmt.Errorf("validate usage operation stream ID")
			}
			notices = append(notices, consumptionservices.OperationStreamNotice{
				Cursor: message.ID, ProjectEnvironmentID: environmentID, OperationID: operationID,
			})
		}
	}
	return notices, latest, nil
}
