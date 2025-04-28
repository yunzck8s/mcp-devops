package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// FormattedLogEntry represents a formatted log entry for AI understanding
type FormattedLogEntry struct {
	Timestamp string            `json:"timestamp"`
	LogLine   string            `json:"log_line"`
	Labels    map[string]string `json:"labels"`
}

// ServiceLogsTool handles querying logs for a specific service for the last 30 minutes
func ServiceLogsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lokiAddress, ok := request.Params.Arguments["loki_address"].(string)
	if !ok {
		lokiAddress = "http://localhost:3100" // Default address
	}

	serviceName, ok := request.Params.Arguments["service_name"].(string)
	if !ok {
		return mcp.NewToolResultText("服务名未提供"), fmt.Errorf("服务名未提供")
	}

	endTime := time.Now()
	startTime := endTime.Add(-30 * time.Minute)
	logQL := fmt.Sprintf(`{app="%s"}`, serviceName)
	limit := 300
	direction := "backward"

	resp, err := QueryLoki(lokiAddress, logQL, startTime, endTime, limit, direction)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询服务日志失败: %v", err)), err
	}

	formattedEntries := formatLogEntries(resp)
	result := map[string]interface{}{
		"status":        resp.Status,
		"result_type":   resp.Data.ResultType,
		"log_entries":   formattedEntries,
		"total_entries": len(formattedEntries),
		"stats":         resp.Data.Stats,
	}
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("格式化日志数据失败: %v", err)), err
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// TimeRangeLogsTool handles querying logs for a specific service within a time range
func TimeRangeLogsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lokiAddress, ok := request.Params.Arguments["loki_address"].(string)
	if !ok {
		lokiAddress = "http://localhost:3100" // Default address
	}

	serviceName, ok := request.Params.Arguments["service_name"].(string)
	if !ok {
		return mcp.NewToolResultText("服务名未提供"), fmt.Errorf("服务名未提供")
	}

	startTimeStr, ok := request.Params.Arguments["start_time"].(string)
	if !ok {
		return mcp.NewToolResultText("起始时间未提供"), fmt.Errorf("起始时间未提供")
	}

	endTimeStr, ok := request.Params.Arguments["end_time"].(string)
	if !ok {
		return mcp.NewToolResultText("结束时间未提供"), fmt.Errorf("结束时间未提供")
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("解析起始时间失败: %v", err)), err
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("解析结束时间失败: %v", err)), err
	}

	logQL := fmt.Sprintf(`{app="%s"}`, serviceName)
	limit := 300
	direction := "backward"

	resp, err := QueryLoki(lokiAddress, logQL, startTime, endTime, limit, direction)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询服务日志失败: %v", err)), err
	}

	formattedEntries := formatLogEntries(resp)
	result := map[string]interface{}{
		"status":        resp.Status,
		"result_type":   resp.Data.ResultType,
		"log_entries":   formattedEntries,
		"total_entries": len(formattedEntries),
		"stats":         resp.Data.Stats,
	}
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("格式化日志数据失败: %v", err)), err
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// AddLokiTools adds all Loki tools to the MCP server
func AddLokiTools(svr *server.MCPServer) {
	svr.AddTool(mcp.NewTool("loki_service_logs",
		mcp.WithDescription("查询指定服务的日志（最近30分钟）"),
		mcp.WithString("loki_address",
			mcp.Description("Loki服务器地址，默认为http://localhost:3100"),
			mcp.DefaultString("http://localhost:3100"),
		),
		mcp.WithString("service_name",
			mcp.Description("要查询日志的服务名（app标签）"),
		),
	), ServiceLogsTool)

	svr.AddTool(mcp.NewTool("loki_time_range_logs",
		mcp.WithDescription("查询指定服务在指定时间范围内的日志"),
		mcp.WithString("loki_address",
			mcp.Description("Loki服务器地址，默认为http://localhost:3100"),
			mcp.DefaultString("http://localhost:3100"),
		),
		mcp.WithString("service_name",
			mcp.Description("要查询日志的服务名（app标签）"),
		),
		mcp.WithString("start_time",
			mcp.Description("查询起始时间（RFC3339格式，例如2023-10-01T00:00:00Z）"),
		),
		mcp.WithString("end_time",
			mcp.Description("查询结束时间（RFC3339格式，例如2023-10-01T01:00:00Z）"),
		),
	), TimeRangeLogsTool)
}

// formatLogEntries formats the log entries for AI understanding
func formatLogEntries(resp *LokiResponse) []FormattedLogEntry {
	var formattedEntries []FormattedLogEntry

	for _, stream := range resp.Data.Result {
		for _, valuePair := range stream.Values {
			if len(valuePair) == 2 {
				timestampNsStr := valuePair[0]
				logLine := valuePair[1]

				tsNanoInt, err := strconv.ParseInt(timestampNsStr, 10, 64)
				var displayTime string
				if err == nil {
					displayTime = time.Unix(0, tsNanoInt).UTC().Format(time.RFC3339)
				} else {
					displayTime = timestampNsStr
				}

				formattedEntries = append(formattedEntries, FormattedLogEntry{
					Timestamp: displayTime,
					LogLine:   logLine,
					Labels:    stream.Stream,
				})
			}
		}
	}

	return formattedEntries
}
