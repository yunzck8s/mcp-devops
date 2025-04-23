package redis

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// Client holds the Redis connection and provides methods for various Redis operations.
type Client struct {
	rdb *redis.Client
	ctx context.Context
}

// NewClient creates a new Redis client by reading connection details from environment variables.
func NewClient() (*Client, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379" // Default fallback
	}

	password := os.Getenv("REDIS_PASSWORD")
	dbStr := os.Getenv("REDIS_DB")
	db := 0 // Default DB
	if dbStr != "" {
		var dbNum int
		_, err := fmt.Sscanf(dbStr, "%d", &dbNum)
		if err == nil {
			db = dbNum
		}
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return &Client{rdb: rdb, ctx: ctx}, nil
}

// Close closes the Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}

// Info retrieves server runtime information.
func (c *Client) Info(section string) (string, error) {
	if section == "" {
		section = "ALL"
	}
	return c.rdb.Info(c.ctx, section).Result()
}

// SlowLog retrieves slow query log entries.
func (c *Client) SlowLog(count int64) ([]redis.SlowLog, error) {
	return c.rdb.SlowLogGet(c.ctx, count).Result()
}

// BigKeys scans for big keys in the database.
func (c *Client) BigKeys() (string, error) {
	var cursor uint64
	var bigKeys []string
	threshold := int64(1024 * 1024) // 1MB threshold for considering a key as "big"

	for {
		keys, nextCursor, err := c.rdb.Scan(c.ctx, cursor, "*", 100).Result()
		if err != nil {
			return fmt.Sprintf("Error scanning keys: %v", err), err
		}

		for _, key := range keys {
			// Get the memory usage of the key
			memUsage, err := c.rdb.MemoryUsage(c.ctx, key).Result()
			if err != nil {
				continue // Skip keys with errors
			}

			if memUsage > threshold {
				bigKeys = append(bigKeys, fmt.Sprintf("Key: %s, Size: %d bytes", key, memUsage))
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if len(bigKeys) == 0 {
		return "No big keys found (threshold: 1MB)", nil
	}

	return fmt.Sprintf("Found %d big keys (threshold: 1MB):\n%s", len(bigKeys), strings.Join(bigKeys, "\n")), nil
}

// HotKeys scans for hot keys in the database.
func (c *Client) HotKeys() (string, error) {
	// Placeholder for hot keys scanning logic
	// This may require custom logic or external tools
	return "Hot keys scanning not fully implemented yet", nil
}

// Monitor starts monitoring Redis commands in real-time.
func (c *Client) Monitor(callback func(string)) error {
	// Placeholder for MONITOR command
	// This requires a streaming approach
	return fmt.Errorf("MONITOR command not implemented yet")
}

// Latency measures Redis latency.
func (c *Client) Latency() (time.Duration, error) {
	start := time.Now()
	_, err := c.rdb.Ping(c.ctx).Result()
	if err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

// LatencyHistory measures latency over time and returns historical data.
func (c *Client) LatencyHistory(interval time.Duration, count int) ([]time.Duration, error) {
	var latencies []time.Duration
	for i := 0; i < count; i++ {
		lat, err := c.Latency()
		if err != nil {
			return nil, err
		}
		latencies = append(latencies, lat)
		time.Sleep(interval)
	}
	return latencies, nil
}

// Stat retrieves real-time statistics.
func (c *Client) Stat() (string, error) {
	return c.rdb.Info(c.ctx, "STATS").Result()
}
