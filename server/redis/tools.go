package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Redis tool implementations

// InfoTool handles the redis-cli INFO command.
func InfoTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Failed to connect to Redis: %v", err)), err
	}
	defer client.Close()

	handler := NewHandler(client)
	section, _ := request.Params.Arguments["section"].(string)
	result, err := handler.GetInfo(section)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Failed to get Redis info: %v", err)), err
	}
	return mcp.NewToolResultText(result), nil
}

// SlowLogTool handles the redis-cli SLOWLOG command.
func SlowLogTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Failed to connect to Redis: %v", err)), err
	}
	defer client.Close()

	handler := NewHandler(client)
	count, _ := request.Params.Arguments["count"].(float64)
	result, err := handler.GetSlowLog(int64(count))
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Failed to get Redis slow log: %v", err)), err
	}
	return mcp.NewToolResultText(fmt.Sprintf("%v", result)), nil
}

// BigKeysTool handles the redis-cli --bigkeys command.
func BigKeysTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Failed to connect to Redis: %v", err)), err
	}
	defer client.Close()

	handler := NewHandler(client)
	result, err := handler.GetBigKeys()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Failed to get Redis big keys: %v", err)), err
	}
	return mcp.NewToolResultText(result), nil
}

// HotKeysTool handles the redis-cli --hotkeys command.
func HotKeysTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Failed to connect to Redis: %v", err)), err
	}
	defer client.Close()

	handler := NewHandler(client)
	result, err := handler.GetHotKeys()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Failed to get Redis hot keys: %v", err)), err
	}
	return mcp.NewToolResultText(result), nil
}

// MonitorTool handles the redis-cli MONITOR command.
func MonitorTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Failed to connect to Redis: %v", err)), err
	}
	defer client.Close()

	handler := NewHandler(client)
	// Note: MONITOR requires a streaming approach which might not fit well with MCP tool output
	err = handler.MonitorCommands(func(cmd string) {
		fmt.Println(cmd) // Placeholder for streaming output
	})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Failed to start Redis monitor: %v", err)), err
	}
	return mcp.NewToolResultText("Monitor started (output streaming not fully supported in this context)"), nil
}

// LatencyTool handles the redis-cli --latency command.
func LatencyTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Failed to connect to Redis: %v", err)), err
	}
	defer client.Close()

	handler := NewHandler(client)
	result, err := handler.GetLatency()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Failed to measure Redis latency: %v", err)), err
	}
	return mcp.NewToolResultText(result.String()), nil
}

// LatencyHistoryTool handles the redis-cli --latency-history command.
func LatencyHistoryTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Failed to connect to Redis: %v", err)), err
	}
	defer client.Close()

	handler := NewHandler(client)
	intervalSeconds, _ := request.Params.Arguments["interval"].(float64)
	count, _ := request.Params.Arguments["count"].(float64)
	result, err := handler.GetLatencyHistory(time.Duration(intervalSeconds)*time.Second, int(count))
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Failed to measure Redis latency history: %v", err)), err
	}
	latencies := make([]string, len(result))
	for i, lat := range result {
		latencies[i] = lat.String()
	}
	return mcp.NewToolResultText(fmt.Sprintf("%v", latencies)), nil
}

// StatTool handles the redis-cli --stat command.
func StatTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Failed to connect to Redis: %v", err)), err
	}
	defer client.Close()

	handler := NewHandler(client)
	result, err := handler.GetStats()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Failed to get Redis stats: %v", err)), err
	}
	return mcp.NewToolResultText(result), nil
}

// AddRedisTools adds all Redis tools to the MCP server.
func AddRedisTools(svr *server.MCPServer) {
	svr.AddTool(mcp.NewTool("redis_info",
		mcp.WithDescription("获取Redis服务器运行时信息"),
		mcp.WithString("section",
			mcp.Description("要查询的信息部分，例如SERVER, CLIENTS, MEMORY等，默认为ALL"),
			mcp.DefaultString("ALL"),
		),
	), InfoTool)

	svr.AddTool(mcp.NewTool("redis_slowlog",
		mcp.WithDescription("分析Redis慢查询日志"),
		mcp.WithNumber("count",
			mcp.Description("要获取的慢查询日志条目数，默认为10"),
			mcp.DefaultNumber(10),
		),
	), SlowLogTool)

	svr.AddTool(mcp.NewTool("redis_bigkeys",
		mcp.WithDescription("查找Redis中的大键"),
	), BigKeysTool)

	svr.AddTool(mcp.NewTool("redis_hotkeys",
		mcp.WithDescription("查找Redis中的热键"),
	), HotKeysTool)

	svr.AddTool(mcp.NewTool("redis_monitor",
		mcp.WithDescription("实时查看Redis执行的命令流（高负载，慎用）"),
	), MonitorTool)

	svr.AddTool(mcp.NewTool("redis_latency",
		mcp.WithDescription("测量Redis的延迟"),
	), LatencyTool)

	svr.AddTool(mcp.NewTool("redis_latency_history",
		mcp.WithDescription("测量Redis延迟并获取历史数据"),
		mcp.WithNumber("interval",
			mcp.Description("测量间隔时间（秒），默认为1秒"),
			mcp.DefaultNumber(1),
		),
		mcp.WithNumber("count",
			mcp.Description("测量次数，默认为10次"),
			mcp.DefaultNumber(10),
		),
	), LatencyHistoryTool)

	svr.AddTool(mcp.NewTool("redis_stat",
		mcp.WithDescription("实时查看Redis简洁统计信息"),
	), StatTool)
}

// JSON marshalling for SlowLog to ensure proper output format
func (s SlowLog) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID        int64    `json:"id"`
		Timestamp string   `json:"timestamp"`
		Duration  string   `json:"duration"`
		Args      []string `json:"args"`
	}{
		ID:        s.ID,
		Timestamp: s.Timestamp.Format(time.RFC3339),
		Duration:  s.Duration.String(),
		Args:      s.Args,
	})
}
