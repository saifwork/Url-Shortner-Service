package redisstore

import (
	"context"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/saifwork/url-shortner-service/app/configs"
)

type RedisService struct {
	Config *configs.Config
	Client *redis.Client
	Ctx    context.Context
}

func NewRedisService(c *configs.Config) *RedisService {
	rs := RedisService{
		Config: c,
		Ctx:    context.Background(),
	}
	rs.initialise()
	return &rs
}

func (r *RedisService) initialise() {
	// Connecting to Redis
	r.Client = redis.NewClient(&redis.Options{
		Addr:     r.Config.RedisHost,
		Username: "default",
		Password: r.Config.RedisPwd, // Redis password from config
		DB:       0,                 // Use default DB
	})

	_, err := r.Client.Ping(r.Ctx).Result()
	if err != nil {
		log.Fatalf("Redis server not reachable. Err: %s\n", err)
	}
}

// **Set a key-value pair with an optional expiration time**
func (r *RedisService) Set(key string, value string, expiration time.Duration) error {
	err := r.Client.Set(r.Ctx, key, value, expiration).Err()
	if err != nil {
		log.Printf("Error setting key %s in Redis: %s\n", key, err)
	}
	return err
}

// **Get a value by key**
func (r *RedisService) Get(key string) (string, error) {
	value, err := r.Client.Get(r.Ctx, key).Result()
	if err == redis.Nil {
		log.Printf("Key %s does not exist in Redis\n", key)
		return "", nil // Return empty string if key does not exist
	} else if err != nil {
		log.Printf("Error getting key %s from Redis: %s\n", key, err)
		return "", err
	}
	return value, nil
}

// **Delete a key**
func (r *RedisService) Delete(key string) error {
	_, err := r.Client.Del(r.Ctx, key).Result()
	if err != nil {
		log.Printf("Error deleting key %s from Redis: %s\n", key, err)
	}
	return err
}

// **Check if a key exists**
func (r *RedisService) Exists(key string) (bool, error) {
	count, err := r.Client.Exists(r.Ctx, key).Result()
	if err != nil {
		log.Printf("Error checking existence of key %s in Redis: %s\n", key, err)
		return false, err
	}
	return count > 0, nil
}