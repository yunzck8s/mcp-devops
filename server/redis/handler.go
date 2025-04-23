package redis

import (
	"time"

	"github.com/go-redis/redis/v8"
)

// Handler provides methods to handle Redis operations, typically for API or CLI integration.
type Handler struct {
	client *Client
}

// NewHandler creates a new Handler with a Redis client.
func NewHandler(client *Client) *Handler {
	return &Handler{client: client}
}

// GetInfo handles the redis-cli INFO command.
func (h *Handler) GetInfo(section string) (string, error) {
	return h.client.Info(section)
}

// GetSlowLog handles the redis-cli SLOWLOG command.
func (h *Handler) GetSlowLog(count int64) ([]SlowLog, error) {
	logs, err := h.client.SlowLog(count)
	if err != nil {
		return nil, err
	}
	return FormatSlowLog(logs), nil
}

// GetBigKeys handles the redis-cli --bigkeys command.
func (h *Handler) GetBigKeys() (string, error) {
	return h.client.BigKeys()
}

// GetHotKeys handles the redis-cli --hotkeys command.
func (h *Handler) GetHotKeys() (string, error) {
	return h.client.HotKeys()
}

// MonitorCommands handles the redis-cli MONITOR command.
func (h *Handler) MonitorCommands(callback func(string)) error {
	return h.client.Monitor(callback)
}

// GetLatency handles the redis-cli --latency command.
func (h *Handler) GetLatency() (time.Duration, error) {
	return h.client.Latency()
}

// GetLatencyHistory handles the redis-cli --latency-history command.
func (h *Handler) GetLatencyHistory(interval time.Duration, count int) ([]time.Duration, error) {
	return h.client.LatencyHistory(interval, count)
}

// GetStats handles the redis-cli --stat command.
func (h *Handler) GetStats() (string, error) {
	return h.client.Stat()
}

// SlowLog represents a slow log entry (simplified for handler output).
type SlowLog struct {
	ID        int64
	Timestamp time.Time
	Duration  time.Duration
	Args      []string
}

// FormatSlowLog converts redis.SlowLog to handler's SlowLog format.
func FormatSlowLog(logs []redis.SlowLog) []SlowLog {
	result := make([]SlowLog, len(logs))
	for i, log := range logs {
		result[i] = SlowLog{
			ID:        log.ID,
			Timestamp: log.Time,
			Duration:  time.Duration(log.Duration),
			Args:      log.Args,
		}
	}
	return result
}
