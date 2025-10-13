package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"fleet-tracking/internal/model"

	"github.com/redis/go-redis/v9"
)

type CacheService interface {
	GetVehicleStatus(ctx context.Context, vehicleID string) (*model.VehicleStatus, error)
	SetVehicleStatus(ctx context.Context, vehicleID string, status *model.VehicleStatus, ttl time.Duration) error
	InvalidateVehicleStatus(ctx context.Context, vehicleID string) error
}

type cacheService struct {
	client *redis.Client
}

func NewCacheService(client *redis.Client) CacheService {
	return &cacheService{client: client}
}

func (c *cacheService) GetVehicleStatus(ctx context.Context, vehicleID string) (*model.VehicleStatus, error) {
	key := fmt.Sprintf("vehicle:status:%s", vehicleID)

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var status model.VehicleStatus
	if err := json.Unmarshal([]byte(val), &status); err != nil {
		return nil, err
	}

	return &status, nil
}

func (c *cacheService) SetVehicleStatus(ctx context.Context, vehicleID string, status *model.VehicleStatus, ttl time.Duration) error {
	key := fmt.Sprintf("vehicle:status:%s", vehicleID)

	data, err := json.Marshal(status)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *cacheService) InvalidateVehicleStatus(ctx context.Context, vehicleID string) error {
	key := fmt.Sprintf("vehicle:status:%s", vehicleID)
	return c.client.Del(ctx, key).Err()
}
