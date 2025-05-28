package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// AddClickhouseTools 注册 ClickHouse 链路分析工具
func AddClickhouseTools(svr *server.MCPServer) {
	// 查找 span 数量超过阈值的 trace_id 列表（支持时间范围）
	svr.AddTool(mcp.NewTool("find_heavy_traces",
		mcp.WithDescription("查找 span 数量超过阈值的 trace_id 列表，支持指定时间范围"),
		mcp.WithNumber("threshold",
			mcp.Description("span 数量告警阈值"),
			mcp.DefaultNumber(1000),
		),
		mcp.WithNumber("time_range_hours",
			mcp.Description("查询时间范围（小时），默认为1小时"),
			mcp.DefaultNumber(1),
		),
		mcp.WithNumber("limit",
			mcp.Description("结果限制条数"),
			mcp.DefaultNumber(100),
		),
	), FindHeavyTracesTool)

	// 查找 span 最大耗时排序的 trace_id 列表（支持时间范围）
	svr.AddTool(mcp.NewTool("find_top_slowest_traces",
		mcp.WithDescription("按最大 span 耗时排序，获取 trace_id 列表，支持指定时间范围"),
		mcp.WithNumber("time_range_hours",
			mcp.Description("查询时间范围（小时），默认为1小时"),
			mcp.DefaultNumber(1),
		),
		mcp.WithNumber("limit",
			mcp.Description("结果限制条数"),
			mcp.DefaultNumber(10),
		),
	), FindTopSlowTracesTool)

	// 查找包含错误的 trace_id 列表（支持时间范围）
	svr.AddTool(mcp.NewTool("find_error_traces",
		mcp.WithDescription("查找包含错误 spans 的 trace_id 列表，支持指定时间范围"),
		mcp.WithNumber("time_range_hours",
			mcp.Description("查询时间范围（小时），默认为1小时"),
			mcp.DefaultNumber(1),
		),
		mcp.WithNumber("limit",
			mcp.Description("结果限制条数"),
			mcp.DefaultNumber(100),
		),
	), FindErrorTracesTool)

	// 综合报告单个 trace 链路详情，包括统计、分布和时延分位
	svr.AddTool(mcp.NewTool("report_trace",
		mcp.WithDescription("综合报告指定 trace_id 的链路详情"),
		mcp.WithString("trace_id",
			mcp.Description("要分析的 trace_id"),
			mcp.Required(),
		),
	), ReportTraceTool)

	// 查找特定服务的异常 trace
	svr.AddTool(mcp.NewTool("find_service_issues",
		mcp.WithDescription("查找特定服务的异常 trace，包括错误、慢查询等"),
		mcp.WithString("service_name",
			mcp.Description("要查询的服务名称"),
			mcp.Required(),
		),
		mcp.WithNumber("time_range_hours",
			mcp.Description("查询时间范围（小时），默认为1小时"),
			mcp.DefaultNumber(1),
		),
		mcp.WithNumber("min_duration_ms",
			mcp.Description("最小耗时阈值（毫秒），默认为1000ms"),
			mcp.DefaultNumber(1000),
		),
		mcp.WithNumber("limit",
			mcp.Description("结果限制条数"),
			mcp.DefaultNumber(50),
		),
	), FindServiceIssuesTool)

	// 查找频繁操作的 trace
	svr.AddTool(mcp.NewTool("find_frequent_operations",
		mcp.WithDescription("查找指定时间范围内频繁执行的操作和对应的 trace"),
		mcp.WithNumber("time_range_hours",
			mcp.Description("查询时间范围（小时），默认为1小时"),
			mcp.DefaultNumber(1),
		),
		mcp.WithNumber("min_frequency",
			mcp.Description("最小频率阈值，默认为100次"),
			mcp.DefaultNumber(100),
		),
		mcp.WithNumber("limit",
			mcp.Description("结果限制条数"),
			mcp.DefaultNumber(20),
		),
	), FindFrequentOperationsTool)

	// 查找异常模式的 trace
	svr.AddTool(mcp.NewTool("find_anomaly_traces",
		mcp.WithDescription("查找异常模式的 trace，如超长链路、错误集中等"),
		mcp.WithNumber("time_range_hours",
			mcp.Description("查询时间范围（小时），默认为1小时"),
			mcp.DefaultNumber(1),
		),
		mcp.WithNumber("limit",
			mcp.Description("结果限制条数"),
			mcp.DefaultNumber(50),
		),
	), FindAnomalyTracesTool)

	// 详细分析单个 trace_id
	svr.AddTool(mcp.NewTool("analyze_trace",
		mcp.WithDescription("详细分析单个 trace_id，包括 span 列表、时间线、错误信息等"),
		mcp.WithString("trace_id",
			mcp.Description("要分析的 trace_id"),
			mcp.Required(),
		),
		mcp.WithNumber("limit",
			mcp.Description("返回的 span 数量限制，默认为50"),
			mcp.DefaultNumber(50),
		),
	), AnalyzeTraceTool)

	// 深度分析 trace 中每个 span 节点的行为模式
	svr.AddTool(mcp.NewTool("deep_analyze_trace",
		mcp.WithDescription("深度分析trace中每个span节点的行为，识别不合理的模式和性能问题"),
		mcp.WithString("trace_id",
			mcp.Description("要深度分析的 trace_id"),
			mcp.Required(),
		),
	), DeepAnalyzeTraceTool)

	// 单个服务深度分析工具
	svr.AddTool(mcp.NewTool("analyze_service",
		mcp.WithDescription("深度分析单个服务的性能指标、调用模式和异常行为"),
		mcp.WithString("service_name",
			mcp.Description("要分析的服务名称"),
			mcp.Required(),
		),
		mcp.WithNumber("time_range_hours",
			mcp.Description("分析时间范围（小时），默认为24小时"),
			mcp.DefaultNumber(24),
		),
		mcp.WithNumber("limit",
			mcp.Description("返回的示例数量限制，默认为100"),
			mcp.DefaultNumber(100),
		),
	), AnalyzeServiceTool)
	// 服务间调用分析
	svr.AddTool(mcp.NewTool("analyze_service_dependencies",
		mcp.WithDescription("分析服务间的调用关系和依赖模式"),
		mcp.WithString("service_name",
			mcp.Description("要分析的服务名称"),
			mcp.Required(),
		),
		mcp.WithNumber("time_range_hours",
			mcp.Description("分析时间范围（小时），默认为1小时"),
			mcp.DefaultNumber(1),
		),
		mcp.WithNumber("limit",
			mcp.Description("返回的依赖服务数量限制，默认为20"),
			mcp.DefaultNumber(20),
		),
	), AnalyzeServiceDependenciesTool)
	// 分析服务接口负载，找出负载最重的trace
	svr.AddTool(mcp.NewTool("analyze_service_interface_load",
		mcp.WithDescription("分析指定服务的接口负载，找到指定时间内负载最重的top 10 trace。支持分析全部接口或特定接口"),
		mcp.WithString("service_name",
			mcp.Description("要分析的服务名称"),
			mcp.Required(),
		),
		mcp.WithString("interface_name",
			mcp.Description("要分析的特定接口名称（可选），如果不提供则分析全部接口"),
		),
		mcp.WithNumber("time_range_hours",
			mcp.Description("分析时间范围（小时），默认为2小时"),
			mcp.DefaultNumber(2),
		),
		mcp.WithNumber("top_count",
			mcp.Description("返回负载最重的trace数量，默认为10"),
			mcp.DefaultNumber(10),
		),
	), AnalyzeServiceInterfaceLoadTool)

	// 获取Signoz中的服务列表
	svr.AddTool(mcp.NewTool("list_services",
		mcp.WithDescription("获取Signoz中所有可用的服务列表，包括基本统计信息"),
		mcp.WithNumber("time_range_hours",
			mcp.Description("统计时间范围（小时），默认为24小时"),
			mcp.DefaultNumber(24),
		),
		mcp.WithNumber("limit",
			mcp.Description("返回服务数量限制，默认为100"),
			mcp.DefaultNumber(100),
		),
	), ListServicesTool)

	// 获取指定服务的接口列表
	svr.AddTool(mcp.NewTool("list_service_interfaces",
		mcp.WithDescription("获取指定服务的所有接口列表及其基本统计信息"),
		mcp.WithString("service_name",
			mcp.Description("要查询的服务名称"),
			mcp.Required(),
		),
		mcp.WithNumber("time_range_hours",
			mcp.Description("统计时间范围（小时），默认为24小时"),
			mcp.DefaultNumber(24),
		),
		mcp.WithNumber("limit",
			mcp.Description("返回接口数量限制，默认为100"),
			mcp.DefaultNumber(100),
		),
	), ListServiceInterfacesTool)
}

// FindHeavyTracesTool 列出 span 数量超过阈值的 trace_id（支持时间范围）
func FindHeavyTracesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	th := int64(request.Params.Arguments["threshold"].(float64))
	lim := int(request.Params.Arguments["limit"].(float64))

	// 获取时间范围，默认1小时
	timeRangeHours := 1.0
	if timeVal, ok := request.Params.Arguments["time_range_hours"]; ok && timeVal != nil {
		if timeFloat, ok := timeVal.(float64); ok {
			timeRangeHours = timeFloat
		}
	}

	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接 ClickHouse 失败: %v", err)), err
	}
	defer conn.Close()
	// 使用ClickHouse的时间函数避免溢出问题
	sql := `
		SELECT 
			trace_id, 
			count() AS cnt,
			round(max(duration_nano)/1000000, 2) as max_duration_ms,
			countIf(has_error = 1) as error_count,
			uniq(serviceName) as service_count
		FROM signoz_traces.signoz_index_v3
		WHERE timestamp >= now() - INTERVAL ? HOUR
		GROUP BY trace_id 
		HAVING cnt > ?
		ORDER BY cnt DESC 
		LIMIT ?`

	rows, err := conn.Query(ctx, sql, timeRangeHours, th, lim)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询 heavy traces 失败: %v", err)), err
	}
	defer rows.Close()

	type HeavyTrace struct {
		TraceID       string  `json:"trace_id"`
		SpanCount     uint64  `json:"span_count"`
		MaxDurationMs float64 `json:"max_duration_ms"`
		ErrorCount    uint64  `json:"error_count"`
		ServiceCount  uint64  `json:"service_count"`
		Severity      string  `json:"severity"`
	}

	var list []HeavyTrace
	for rows.Next() {
		var item HeavyTrace
		if rows.Scan(&item.TraceID, &item.SpanCount, &item.MaxDurationMs, &item.ErrorCount, &item.ServiceCount) == nil {
			// 评估严重程度
			if item.SpanCount > 5000 {
				item.Severity = "CRITICAL"
			} else if item.SpanCount > 2000 {
				item.Severity = "HIGH"
			} else {
				item.Severity = "MEDIUM"
			}
			list = append(list, item)
		}
	}

	result := map[string]interface{}{
		"query_time_range_hours": timeRangeHours,
		"threshold":              th,
		"total_found":            len(list),
		"heavy_traces":           list, "summary": map[string]interface{}{
			"critical_traces": len(list), // 可以进一步统计各严重程度的数量
		},
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// FindTopSlowTracesTool 按最大 span 耗时排序的 trace_id（支持时间范围）
func FindTopSlowTracesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lim := int(request.Params.Arguments["limit"].(float64))

	// 获取时间范围，默认1小时
	timeRangeHours := 1.0
	if timeVal, ok := request.Params.Arguments["time_range_hours"]; ok && timeVal != nil {
		if timeFloat, ok := timeVal.(float64); ok {
			timeRangeHours = timeFloat
		}
	}

	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接失败: %v", err)), err
	}
	defer conn.Close()
	// 使用ClickHouse的时间函数避免溢出问题
	sql := `
		SELECT 
			trace_id, 
			round(max(duration_nano)/1000000, 2) as max_duration_ms,
			count() as span_count,
			countIf(has_error = 1) as error_count,
			uniq(serviceName) as service_count
		FROM signoz_traces.signoz_index_v3
		WHERE timestamp >= now() - INTERVAL ? HOUR
		GROUP BY trace_id 
		ORDER BY max(duration_nano) DESC 
		LIMIT ?`

	rows, err := conn.Query(ctx, sql, timeRangeHours, lim)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询 slow traces 失败: %v", err)), err
	}
	defer rows.Close()

	type SlowTrace struct {
		TraceID       string  `json:"trace_id"`
		MaxDurationMs float64 `json:"max_duration_ms"`
		SpanCount     uint64  `json:"span_count"`
		ErrorCount    uint64  `json:"error_count"`
		ServiceCount  uint64  `json:"service_count"`
	}

	var list []SlowTrace
	for rows.Next() {
		var item SlowTrace
		if rows.Scan(&item.TraceID, &item.MaxDurationMs, &item.SpanCount, &item.ErrorCount, &item.ServiceCount) == nil {
			list = append(list, item)
		}
	}

	result := map[string]interface{}{
		"query_time_range_hours": timeRangeHours,
		"total_found":            len(list),
		"slowest_traces":         list,
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// FindErrorTracesTool 列出包含错误 spans 的 trace_id（支持时间范围）
func FindErrorTracesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lim := int(request.Params.Arguments["limit"].(float64))

	// 获取时间范围，默认1小时
	timeRangeHours := 1.0
	if timeVal, ok := request.Params.Arguments["time_range_hours"]; ok && timeVal != nil {
		if timeFloat, ok := timeVal.(float64); ok {
			timeRangeHours = timeFloat
		}
	}

	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接失败: %v", err)), err
	}
	defer conn.Close()
	// 使用ClickHouse的时间函数避免溢出问题
	sql := `
		SELECT 
			trace_id, 
			countIf(has_error = 1) as error_count,
			count() as total_spans,
			round(max(duration_nano)/1000000, 2) as max_duration_ms,
			uniq(serviceName) as service_count
		FROM signoz_traces.signoz_index_v3
		WHERE timestamp >= now() - INTERVAL ? HOUR AND has_error = 1
		GROUP BY trace_id 
		ORDER BY error_count DESC 
		LIMIT ?`

	rows, err := conn.Query(ctx, sql, timeRangeHours, lim)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询 error traces 失败: %v", err)), err
	}
	defer rows.Close()

	type ErrorTrace struct {
		TraceID       string  `json:"trace_id"`
		ErrorCount    uint64  `json:"error_count"`
		TotalSpans    uint64  `json:"total_spans"`
		MaxDurationMs float64 `json:"max_duration_ms"`
		ServiceCount  uint64  `json:"service_count"`
	}

	var list []ErrorTrace
	for rows.Next() {
		var item ErrorTrace
		if rows.Scan(&item.TraceID, &item.ErrorCount, &item.TotalSpans, &item.MaxDurationMs, &item.ServiceCount) == nil {
			list = append(list, item)
		}
	}

	result := map[string]interface{}{
		"query_time_range_hours": timeRangeHours,
		"total_found":            len(list),
		"error_traces":           list,
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// ReportTraceTool 综合报告单个 trace 链路详情
func ReportTraceTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tid := request.Params.Arguments["trace_id"].(string)
	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接失败: %v", err)), err
	}
	defer conn.Close()

	// 统计和分位耗时
	var total uint64
	var avg, max uint64
	var errs uint64
	var p50, p95, p99 float64
	row := conn.QueryRow(ctx,
		`SELECT count(), avg(duration_nano), max(duration_nano), countIf(has_error),
			 quantile(0.5)(duration_nano), quantile(0.95)(duration_nano), quantile(0.99)(duration_nano)
		   FROM signoz_traces.signoz_index_v3 WHERE trace_id = ?`, tid)
	_ = row.Scan(&total, &avg, &max, &errs, &p50, &p95, &p99)

	// 操作分布
	opRows, _ := conn.Query(ctx,
		`SELECT name, count() AS cnt FROM signoz_traces.signoz_index_v3
		   WHERE trace_id = ? GROUP BY name ORDER BY cnt DESC LIMIT 10`, tid)
	defer opRows.Close()
	ops := make([]map[string]interface{}, 0)
	for opRows.Next() {
		var name string
		var cnt uint64
		if opRows.Scan(&name, &cnt) == nil {
			ops = append(ops, map[string]interface{}{"name": name, "count": cnt})
		}
	}
	result := map[string]interface{}{
		"trace_id":   tid,
		"span_count": total,
		"avg_dur":    avg,
		"max_dur":    max,
		"errors":     errs,
		"p50":        p50, "p95": p95, "p99": p99,
		"top_ops": ops,
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// FindSpansGT1000Tool 查找 span 数量大于1000 的 trace_id
func FindSpansGT1000Tool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	const threshold = 1000
	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接失败: %v", err)), err
	}
	defer conn.Close()

	sql := fmt.Sprintf(
		`SELECT trace_id, count() AS cnt FROM signoz_traces.signoz_index_v3
         GROUP BY trace_id HAVING cnt > %d SETTINGS max_bytes_before_external_group_by = 8000000000`, threshold)
	rows, err := conn.Query(ctx, sql)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询失败: %v", err)), err
	}
	defer rows.Close()

	type Item struct {
		TraceID string `json:"trace_id"`
		Count   uint64 `json:"count"`
	}
	var list []Item
	for rows.Next() {
		var id string
		var cnt uint64
		if rows.Scan(&id, &cnt) == nil {
			list = append(list, Item{TraceID: id, Count: cnt})
		}
	}
	out, _ := json.MarshalIndent(list, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// AnalyzeTraceTool 详细分析单个 trace_id
func AnalyzeTraceTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tid := request.Params.Arguments["trace_id"].(string)

	// 安全地获取 limit 参数，如果没有提供则使用默认值
	limit := 50 // 默认值
	if limitVal, ok := request.Params.Arguments["limit"]; ok && limitVal != nil {
		if limitFloat, ok := limitVal.(float64); ok {
			limit = int(limitFloat)
		}
	}

	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接失败: %v", err)), err
	}
	defer conn.Close()

	// 1. 基本统计信息
	var spanCount uint64
	var minTime, maxTime uint64
	var avgDuration, maxDuration uint64
	var errorCount uint64
	var rootSpanName string

	statsRow := conn.QueryRow(ctx,
		`SELECT 
			count() as span_count,
			min(timestamp) as min_time,
			max(timestamp) as max_time,
			avg(duration_nano) as avg_duration,
			max(duration_nano) as max_duration,
			countIf(has_error = true) as error_count,
			argMaxIf(name, parent_span_id = '') as root_span_name
		FROM signoz_traces.signoz_index_v3 
		WHERE trace_id = ?`, tid)
	_ = statsRow.Scan(&spanCount, &minTime, &maxTime, &avgDuration, &maxDuration, &errorCount, &rootSpanName)

	if spanCount == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("未找到 trace_id: %s", tid)), nil
	}

	// 2. 获取详细的 span 信息
	spansSQL := fmt.Sprintf(`
		SELECT 
			span_id,
			parent_span_id,
			name,
			kind,
			duration_nano,
			timestamp,
			has_error,
			service_name
		FROM signoz_traces.signoz_index_v3 
		WHERE trace_id = ? 
		ORDER BY timestamp ASC 
		LIMIT %d`, limit)
	spansRows, err := conn.Query(ctx, spansSQL, tid)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询 spans 失败: %v", err)), err
	}
	defer spansRows.Close()

	type SpanInfo struct {
		SpanID       string `json:"span_id"`
		ParentSpanID string `json:"parent_span_id"`
		Name         string `json:"name"`
		Kind         string `json:"kind"`
		Duration     uint64 `json:"duration_nano"`
		Timestamp    uint64 `json:"timestamp"`
		HasError     bool   `json:"has_error"`
		ServiceName  string `json:"service_name"`
	}

	var spans []SpanInfo
	for spansRows.Next() {
		var span SpanInfo
		if spansRows.Scan(&span.SpanID, &span.ParentSpanID, &span.Name, &span.Kind,
			&span.Duration, &span.Timestamp, &span.HasError, &span.ServiceName) == nil {
			spans = append(spans, span)
		}
	}

	// 3. 服务分布统计
	serviceSQL := `
		SELECT 
			service_name,
			count() as span_count,
			avg(duration_nano) as avg_duration,
			countIf(has_error = true) as error_count
		FROM signoz_traces.signoz_index_v3 
		WHERE trace_id = ? 
		GROUP BY service_name 
		ORDER BY span_count DESC`

	serviceRows, err := conn.Query(ctx, serviceSQL, tid)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询服务统计失败: %v", err)), err
	}
	defer serviceRows.Close()

	type ServiceStat struct {
		ServiceName string `json:"service_name"`
		SpanCount   uint64 `json:"span_count"`
		AvgDuration uint64 `json:"avg_duration"`
		ErrorCount  uint64 `json:"error_count"`
	}

	var services []ServiceStat
	for serviceRows.Next() {
		var service ServiceStat
		if serviceRows.Scan(&service.ServiceName, &service.SpanCount,
			&service.AvgDuration, &service.ErrorCount) == nil {
			services = append(services, service)
		}
	}

	// 4. 构建分析结果
	totalDuration := maxTime - minTime
	result := map[string]interface{}{
		"trace_id": tid,
		"summary": map[string]interface{}{
			"span_count":     spanCount,
			"total_duration": totalDuration,
			"avg_duration":   avgDuration,
			"max_duration":   maxDuration,
			"error_count":    errorCount,
			"root_span_name": rootSpanName,
			"start_time":     minTime,
			"end_time":       maxTime,
		},
		"services": services,
		"spans":    spans,
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// DeepAnalyzeTraceTool 深度分析trace中每个span节点的行为模式
func DeepAnalyzeTraceTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	traceID := request.Params.Arguments["trace_id"].(string)

	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接 ClickHouse 失败: %v", err)), err
	}
	defer conn.Close()
	// 1. 获取基础span模式分析
	spanPatternSQL := `		SELECT 
			name,
			serviceName,
			count() as occurrence_count,
			round(avg(duration_nano)/1000000, 2) as avg_duration_ms,
			round(max(duration_nano)/1000000, 2) as max_duration_ms,
			round(min(duration_nano)/1000000, 2) as min_duration_ms,
			quantile(0.95)(duration_nano/1000000) as p95_duration_ms,
			countIf(has_error = 1) as error_count,
			CASE 
				WHEN count() > 100 THEN 'EXCESSIVE_FREQUENCY'
				WHEN count() > 50 THEN 'HIGH_FREQUENCY' 
				WHEN count() > 10 THEN 'MEDIUM_FREQUENCY'
				ELSE 'LOW_FREQUENCY' 
			END as frequency_pattern,
			CASE 
				WHEN avg(duration_nano) > 1000000000 THEN 'SLOW_OPERATION'
				WHEN avg(duration_nano) > 500000000 THEN 'MODERATE_OPERATION'
				ELSE 'FAST_OPERATION'
			END as performance_pattern,
			CASE 
				WHEN count() > 100 AND avg(duration_nano) > 100000000 THEN 'POTENTIAL_ISSUE_HIGH_FREQ_SLOW'
				WHEN countIf(has_error = 1) > count() * 0.1 THEN 'POTENTIAL_ISSUE_HIGH_ERROR_RATE'
				WHEN max(duration_nano) > avg(duration_nano) * 10 THEN 'POTENTIAL_ISSUE_INCONSISTENT_PERFORMANCE'
				ELSE 'NORMAL'
			END as anomaly_flag
		FROM signoz_traces.signoz_index_v3 
		WHERE trace_id = ?
		GROUP BY name, serviceName
		ORDER BY occurrence_count DESC
	`

	rows, err := conn.Query(ctx, spanPatternSQL, traceID)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询span模式失败: %v", err)), err
	}

	type SpanPattern struct {
		Operation          string  `json:"operation"`
		Service            string  `json:"service"`
		OccurrenceCount    uint64  `json:"occurrence_count"`
		AvgDurationMs      float64 `json:"avg_duration_ms"`
		MaxDurationMs      float64 `json:"max_duration_ms"`
		MinDurationMs      float64 `json:"min_duration_ms"`
		P95DurationMs      float64 `json:"p95_duration_ms"`
		ErrorCount         uint64  `json:"error_count"`
		FrequencyPattern   string  `json:"frequency_pattern"`
		PerformancePattern string  `json:"performance_pattern"`
		AnomalyFlag        string  `json:"anomaly_flag"`
	}

	var patterns []SpanPattern
	for rows.Next() {
		var p SpanPattern
		if err := rows.Scan(&p.Operation, &p.Service, &p.OccurrenceCount,
			&p.AvgDurationMs, &p.MaxDurationMs, &p.MinDurationMs, &p.P95DurationMs,
			&p.ErrorCount, &p.FrequencyPattern, &p.PerformancePattern, &p.AnomalyFlag); err == nil {
			patterns = append(patterns, p)
		}
	}
	rows.Close()
	// 2. Redis操作分析
	redisSQL := `
		SELECT 
			count() as redis_ops_count,
			round(avg(duration_nano)/1000000, 2) as avg_duration_ms,
			round(max(duration_nano)/1000000, 2) as max_duration_ms,
			countIf(has_error = 1) as error_count,
			groupUniqArray(name) as operations
		FROM signoz_traces.signoz_index_v3 
		WHERE trace_id = ? AND (
			name LIKE '%redis%' OR 
			name LIKE '%PING%' OR 
			name LIKE '%GET%' OR 
			name LIKE '%SET%' OR
			name LIKE '%PUBLISH%' OR
			name LIKE '%PTTL%'
		)
	`

	redisRows, err := conn.Query(ctx, redisSQL, traceID)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Redis分析查询失败: %v", err)), err
	}

	type RedisAnalysis struct {
		OpsCount      uint64   `json:"ops_count"`
		AvgDurationMs float64  `json:"avg_duration_ms"`
		MaxDurationMs float64  `json:"max_duration_ms"`
		ErrorCount    uint64   `json:"error_count"`
		Operations    []string `json:"operations"`
	}

	var redisAnalysis RedisAnalysis
	if redisRows.Next() {
		redisRows.Scan(&redisAnalysis.OpsCount, &redisAnalysis.AvgDurationMs,
			&redisAnalysis.MaxDurationMs, &redisAnalysis.ErrorCount, &redisAnalysis.Operations)
	}
	redisRows.Close()

	// 3. 异常检测分析
	anomalySQL := `
		SELECT 
			count() as total_spans,
			countIf(has_error = 1) as error_spans,
			round(avg(duration_nano)/1000000, 2) as avg_duration_ms,
			round(max(duration_nano)/1000000, 2) as max_duration_ms,
			uniq(serviceName) as service_count,
			countIf(duration_nano > 1000000000) as slow_spans_1s,
			countIf(duration_nano > 5000000000) as slow_spans_5s
		FROM signoz_traces.signoz_index_v3 
		WHERE trace_id = ?
	`

	anomalyRows, err := conn.Query(ctx, anomalySQL, traceID)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("异常检测查询失败: %v", err)), err
	}

	type AnomalyAnalysis struct {
		TotalSpans    uint64   `json:"total_spans"`
		ErrorSpans    uint64   `json:"error_spans"`
		AvgDurationMs float64  `json:"avg_duration_ms"`
		MaxDurationMs float64  `json:"max_duration_ms"`
		ServiceCount  uint64   `json:"service_count"`
		SlowSpans1s   uint64   `json:"slow_spans_1s"`
		SlowSpans5s   uint64   `json:"slow_spans_5s"`
		Anomalies     []string `json:"anomalies"`
	}

	var anomalyAnalysis AnomalyAnalysis
	if anomalyRows.Next() {
		anomalyRows.Scan(&anomalyAnalysis.TotalSpans, &anomalyAnalysis.ErrorSpans,
			&anomalyAnalysis.AvgDurationMs, &anomalyAnalysis.MaxDurationMs,
			&anomalyAnalysis.ServiceCount, &anomalyAnalysis.SlowSpans1s, &anomalyAnalysis.SlowSpans5s)
	}
	anomalyRows.Close()

	// 检测异常模式
	if anomalyAnalysis.TotalSpans > 1000 {
		anomalyAnalysis.Anomalies = append(anomalyAnalysis.Anomalies, fmt.Sprintf("异常：Span数量过多 (%d)", anomalyAnalysis.TotalSpans))
	}
	if anomalyAnalysis.ErrorSpans > 10 {
		anomalyAnalysis.Anomalies = append(anomalyAnalysis.Anomalies, fmt.Sprintf("异常：错误数量过多 (%d)", anomalyAnalysis.ErrorSpans))
	}
	if anomalyAnalysis.AvgDurationMs > 2000 {
		anomalyAnalysis.Anomalies = append(anomalyAnalysis.Anomalies, fmt.Sprintf("异常：平均耗时过长 (%.2fms)", anomalyAnalysis.AvgDurationMs))
	}
	if anomalyAnalysis.SlowSpans5s > 0 {
		anomalyAnalysis.Anomalies = append(anomalyAnalysis.Anomalies, fmt.Sprintf("异常：存在超长耗时操作 (%d个>5s)", anomalyAnalysis.SlowSpans5s))
	}
	if redisAnalysis.OpsCount > 100 {
		anomalyAnalysis.Anomalies = append(anomalyAnalysis.Anomalies, fmt.Sprintf("异常：Redis操作频率过高 (%d次)", redisAnalysis.OpsCount))
	}

	// 4. 服务行为分析
	serviceSQL := `		SELECT 
			serviceName,
			count() as span_count,
			round(avg(duration_nano)/1000000, 2) as avg_duration_ms,
			countIf(has_error = 1) as error_count
		FROM signoz_traces.signoz_index_v3 
		WHERE trace_id = ?
		GROUP BY serviceName
		ORDER BY span_count DESC
	`

	serviceRows, err := conn.Query(ctx, serviceSQL, traceID)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("服务分析查询失败: %v", err)), err
	}
	defer serviceRows.Close()

	type ServiceBehavior struct {
		ServiceName   string  `json:"service_name"`
		SpanCount     uint64  `json:"span_count"`
		AvgDurationMs float64 `json:"avg_duration_ms"`
		ErrorCount    uint64  `json:"error_count"`
	}

	var serviceBehaviors []ServiceBehavior
	for serviceRows.Next() {
		var sb ServiceBehavior
		if err := serviceRows.Scan(&sb.ServiceName, &sb.SpanCount, &sb.AvgDurationMs, &sb.ErrorCount); err == nil {
			serviceBehaviors = append(serviceBehaviors, sb)
		}
	}
	serviceRows.Close()
	// 构建最终结果
	result := map[string]interface{}{
		"trace_id":          traceID,
		"pattern_analysis":  patterns,
		"redis_analysis":    redisAnalysis,
		"anomaly_analysis":  anomalyAnalysis,
		"service_behaviors": serviceBehaviors,
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// AnalyzeServiceTool 深度分析单个服务的性能指标、调用模式和异常行为
func AnalyzeServiceTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceName := request.Params.Arguments["service_name"].(string)

	// 获取时间范围，默认24小时
	timeRangeHours := 24.0
	if timeVal, ok := request.Params.Arguments["time_range_hours"]; ok && timeVal != nil {
		if timeFloat, ok := timeVal.(float64); ok {
			timeRangeHours = timeFloat
		}
	}

	// 获取限制数量，默认100
	limit := 100
	if limitVal, ok := request.Params.Arguments["limit"]; ok && limitVal != nil {
		if limitFloat, ok := limitVal.(float64); ok {
			limit = int(limitFloat)
		}
	}

	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接 ClickHouse 失败: %v", err)), err
	}
	defer conn.Close()
	// 使用ClickHouse的时间函数避免溢出问题
	// 1. 服务基本性能指标
	metricsSQL := `
		SELECT 
			count() as total_spans,
			countIf(has_error = 1) as error_spans,
			round(count() / ?, 2) as spans_per_hour,
			round(avg(duration_nano)/1000000, 2) as avg_duration_ms,
			round(max(duration_nano)/1000000, 2) as max_duration_ms,
			round(min(duration_nano)/1000000, 2) as min_duration_ms,
			quantile(0.5)(duration_nano/1000000) as p50_duration_ms,
			quantile(0.95)(duration_nano/1000000) as p95_duration_ms,
			quantile(0.99)(duration_nano/1000000) as p99_duration_ms,
			uniq(trace_id) as unique_traces,
			countIf(duration_nano > 1000000000) as slow_spans_1s,
			countIf(duration_nano > 5000000000) as slow_spans_5s
		FROM signoz_traces.signoz_index_v3 
		WHERE serviceName = ? AND timestamp >= now() - INTERVAL ? HOUR
	`

	type ServiceMetrics struct {
		TotalSpans    uint64  `json:"total_spans"`
		ErrorSpans    uint64  `json:"error_spans"`
		SpansPerHour  float64 `json:"spans_per_hour"`
		AvgDurationMs float64 `json:"avg_duration_ms"`
		MaxDurationMs float64 `json:"max_duration_ms"`
		MinDurationMs float64 `json:"min_duration_ms"`
		P50DurationMs float64 `json:"p50_duration_ms"`
		P95DurationMs float64 `json:"p95_duration_ms"`
		P99DurationMs float64 `json:"p99_duration_ms"`
		UniqueTraces  uint64  `json:"unique_traces"`
		SlowSpans1s   uint64  `json:"slow_spans_1s"`
		SlowSpans5s   uint64  `json:"slow_spans_5s"`
		ErrorRate     float64 `json:"error_rate"`
	}
	var metrics ServiceMetrics
	metricsRow := conn.QueryRow(ctx, metricsSQL, timeRangeHours, serviceName, timeRangeHours)
	err = metricsRow.Scan(&metrics.TotalSpans, &metrics.ErrorSpans, &metrics.SpansPerHour,
		&metrics.AvgDurationMs, &metrics.MaxDurationMs, &metrics.MinDurationMs,
		&metrics.P50DurationMs, &metrics.P95DurationMs, &metrics.P99DurationMs,
		&metrics.UniqueTraces, &metrics.SlowSpans1s, &metrics.SlowSpans5s)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询服务指标失败: %v", err)), err
	}

	// 计算错误率
	if metrics.TotalSpans > 0 {
		metrics.ErrorRate = float64(metrics.ErrorSpans) / float64(metrics.TotalSpans) * 100
	}

	// 2. 操作模式分析
	operationSQL := `
		SELECT 
			name,
			count() as call_count,
			round(avg(duration_nano)/1000000, 2) as avg_duration_ms,
			round(max(duration_nano)/1000000, 2) as max_duration_ms,
			quantile(0.95)(duration_nano/1000000) as p95_duration_ms,
			countIf(has_error = 1) as error_count,
			round(countIf(has_error = 1) * 100.0 / count(), 2) as error_rate_pct,
			CASE 
				WHEN count() > 1000 THEN 'HIGH_FREQUENCY'
				WHEN count() > 100 THEN 'MEDIUM_FREQUENCY'
				ELSE 'LOW_FREQUENCY'
			END as frequency_pattern,
			CASE 
				WHEN avg(duration_nano) > 1000000000 THEN 'SLOW_OPERATION'
				WHEN avg(duration_nano) > 500000000 THEN 'MODERATE_OPERATION'
				ELSE 'FAST_OPERATION'
			END as performance_pattern		FROM signoz_traces.signoz_index_v3 
		WHERE serviceName = ? AND timestamp >= now() - INTERVAL ? HOUR
		GROUP BY name
		ORDER BY call_count DESC
		LIMIT ?
	`

	type OperationPattern struct {
		Name               string  `json:"name"`
		CallCount          uint64  `json:"call_count"`
		AvgDurationMs      float64 `json:"avg_duration_ms"`
		MaxDurationMs      float64 `json:"max_duration_ms"`
		P95DurationMs      float64 `json:"p95_duration_ms"`
		ErrorCount         uint64  `json:"error_count"`
		ErrorRatePct       float64 `json:"error_rate_pct"`
		FrequencyPattern   string  `json:"frequency_pattern"`
		PerformancePattern string  `json:"performance_pattern"`
	}

	operationRows, err := conn.Query(ctx, operationSQL, serviceName, timeRangeHours, limit)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询操作模式失败: %v", err)), err
	}
	defer operationRows.Close()

	var operations []OperationPattern
	for operationRows.Next() {
		var op OperationPattern
		if err := operationRows.Scan(&op.Name, &op.CallCount, &op.AvgDurationMs, &op.MaxDurationMs,
			&op.P95DurationMs, &op.ErrorCount, &op.ErrorRatePct, &op.FrequencyPattern, &op.PerformancePattern); err == nil {
			operations = append(operations, op)
		}
	}
	// 3. 时间分布分析
	timeDistributionSQL := `
		SELECT 
			toHour(timestamp) as hour,
			count() as span_count,
			countIf(has_error = 1) as error_count,
			round(avg(duration_nano)/1000000, 2) as avg_duration_ms
		FROM signoz_traces.signoz_index_v3 
		WHERE serviceName = ? AND timestamp >= now() - INTERVAL ? HOUR
		GROUP BY hour
		ORDER BY hour
	`

	type HourlyPattern struct {
		Hour          uint8   `json:"hour"`
		SpanCount     uint64  `json:"span_count"`
		ErrorCount    uint64  `json:"error_count"`
		AvgDurationMs float64 `json:"avg_duration_ms"`
	}

	timeRows, err := conn.Query(ctx, timeDistributionSQL, serviceName, timeRangeHours)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询时间分布失败: %v", err)), err
	}
	defer timeRows.Close()

	var timeDistribution []HourlyPattern
	for timeRows.Next() {
		var hp HourlyPattern
		if err := timeRows.Scan(&hp.Hour, &hp.SpanCount, &hp.ErrorCount, &hp.AvgDurationMs); err == nil {
			timeDistribution = append(timeDistribution, hp)
		}
	}

	// 4. 异常和性能问题检测
	var issues []string
	var recommendations []string

	// 错误率分析
	if metrics.ErrorRate > 5.0 {
		issues = append(issues, fmt.Sprintf("高错误率: %.2f%% (超过5%%阈值)", metrics.ErrorRate))
		recommendations = append(recommendations, "建议检查错误日志，分析主要错误原因")
	}

	// 性能分析
	if metrics.P95DurationMs > 5000 {
		issues = append(issues, fmt.Sprintf("P95延迟过高: %.2fms (超过5秒)", metrics.P95DurationMs))
		recommendations = append(recommendations, "建议优化慢操作，检查数据库查询和外部API调用")
	}

	// 频率分析
	if metrics.SpansPerHour > 10000 {
		issues = append(issues, fmt.Sprintf("调用频率过高: %.0f spans/hour", metrics.SpansPerHour))
		recommendations = append(recommendations, "考虑添加缓存或限流措施")
	}

	// 慢操作分析
	if metrics.SlowSpans5s > 0 {
		issues = append(issues, fmt.Sprintf("存在极慢操作: %d个span超过5秒", metrics.SlowSpans5s))
		recommendations = append(recommendations, "紧急排查超长耗时操作")
	} // 5. 依赖服务分析
	dependencySQL := `
		SELECT 
			t2.resources_string['service.name'] as target_service,
			count() as call_count,
			round(avg(t2.duration_nano)/1000000, 2) as avg_duration_ms,
			countIf(t2.has_error = 1) as error_count
		FROM signoz_traces.signoz_index_v3 t1
		JOIN signoz_traces.signoz_index_v3 t2 ON t1.trace_id = t2.trace_id
		WHERE t1.resources_string['service.name'] = ? 
		  AND t2.resources_string['service.name'] != ? 
		  AND t1.timestamp >= now() - INTERVAL ? HOUR
		  AND t2.timestamp >= now() - INTERVAL ? HOUR
		GROUP BY t2.resources_string['service.name']
		ORDER BY call_count DESC
		LIMIT 20
	`

	type DependencyInfo struct {
		TargetService string  `json:"target_service"`
		CallCount     uint64  `json:"call_count"`
		AvgDurationMs float64 `json:"avg_duration_ms"`
		ErrorCount    uint64  `json:"error_count"`
	}

	depRows, err := conn.Query(ctx, dependencySQL, serviceName, serviceName, timeRangeHours, timeRangeHours)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询依赖服务失败: %v", err)), err
	}
	defer depRows.Close()

	var dependencies []DependencyInfo
	for depRows.Next() {
		var dep DependencyInfo
		if err := depRows.Scan(&dep.TargetService, &dep.CallCount, &dep.AvgDurationMs, &dep.ErrorCount); err == nil {
			dependencies = append(dependencies, dep)
		}
	}
	// 6. 最慢的trace示例
	slowTracesSQL := `
		SELECT 
			trace_id,
			max(duration_nano) as max_duration_nano,
			count() as span_count,
			countIf(has_error = 1) as error_count
		FROM signoz_traces.signoz_index_v3 
		WHERE serviceName = ? AND timestamp >= now() - INTERVAL ? HOUR
		GROUP BY trace_id
		ORDER BY max_duration_nano DESC
		LIMIT 10
	`

	type SlowTrace struct {
		TraceID       string  `json:"trace_id"`
		MaxDurationMs float64 `json:"max_duration_ms"`
		SpanCount     uint64  `json:"span_count"`
		ErrorCount    uint64  `json:"error_count"`
	}

	slowRows, err := conn.Query(ctx, slowTracesSQL, serviceName, timeRangeHours)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询慢trace失败: %v", err)), err
	}
	defer slowRows.Close()

	var slowTraces []SlowTrace
	for slowRows.Next() {
		var st SlowTrace
		var maxDurationNano uint64
		if err := slowRows.Scan(&st.TraceID, &maxDurationNano, &st.SpanCount, &st.ErrorCount); err == nil {
			st.MaxDurationMs = float64(maxDurationNano) / 1000000.0
			slowTraces = append(slowTraces, st)
		}
	}

	// 构建分析结果
	result := map[string]interface{}{
		"service_name":      serviceName,
		"analysis_period":   fmt.Sprintf("%.1f hours", timeRangeHours),
		"metrics":           metrics,
		"operations":        operations,
		"time_distribution": timeDistribution,
		"dependencies":      dependencies,
		"slow_traces":       slowTraces,
		"health_assessment": map[string]interface{}{
			"overall_status":  getHealthStatus(metrics.ErrorRate, metrics.P95DurationMs),
			"issues":          issues,
			"recommendations": recommendations,
		},
		"summary": map[string]interface{}{
			"total_operations":   len(operations),
			"total_dependencies": len(dependencies),
			"analysis_timestamp": time.Now().Format(time.RFC3339),
		},
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// getHealthStatus 根据错误率和P95延迟评估服务健康状态
func getHealthStatus(errorRate, p95Duration float64) string {
	if errorRate > 10 || p95Duration > 10000 {
		return "CRITICAL"
	}
	if errorRate > 5 || p95Duration > 5000 {
		return "WARNING"
	}
	if errorRate > 1 || p95Duration > 2000 {
		return "ATTENTION"
	}
	return "HEALTHY"
}

// FindServiceIssuesTool 查找特定服务的异常 trace
func FindServiceIssuesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceName := request.Params.Arguments["service_name"].(string)

	// 获取时间范围，默认1小时
	timeRangeHours := 1.0
	if timeVal, ok := request.Params.Arguments["time_range_hours"]; ok && timeVal != nil {
		if timeFloat, ok := timeVal.(float64); ok {
			timeRangeHours = timeFloat
		}
	}

	// 获取最小耗时阈值，默认1000ms
	minDurationMs := 1000.0
	if durationVal, ok := request.Params.Arguments["min_duration_ms"]; ok && durationVal != nil {
		if durationFloat, ok := durationVal.(float64); ok {
			minDurationMs = durationFloat
		}
	}

	limit := int(request.Params.Arguments["limit"].(float64))
	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接 ClickHouse 失败: %v", err)), err
	}
	defer conn.Close()

	// 使用ClickHouse的时间函数避免溢出问题
	minDurationNano := uint64(minDurationMs * 1000000)

	sql := `
		SELECT 
			trace_id,
			count() as span_count,
			round(max(duration_nano)/1000000, 2) as max_duration_ms,
			round(avg(duration_nano)/1000000, 2) as avg_duration_ms,
			countIf(has_error = 1) as error_count,
			uniq(name) as operation_count,
			CASE 
				WHEN countIf(has_error = 1) > 0 AND max(duration_nano) > ? THEN 'ERROR_AND_SLOW'
				WHEN countIf(has_error = 1) > 0 THEN 'ERROR_ONLY'
				WHEN max(duration_nano) > ? THEN 'SLOW_ONLY'
				ELSE 'OTHER'
			END as issue_type
		FROM signoz_traces.signoz_index_v3
		WHERE serviceName = ? 
		  AND timestamp >= now() - INTERVAL ? HOUR
		  AND (has_error = 1 OR duration_nano > ?)
		GROUP BY trace_id
		ORDER BY 
			CASE issue_type 
				WHEN 'ERROR_AND_SLOW' THEN 1
				WHEN 'ERROR_ONLY' THEN 2
				WHEN 'SLOW_ONLY' THEN 3
				ELSE 4
			END,
			max_duration_ms DESC
		LIMIT ?`

	rows, err := conn.Query(ctx, sql, minDurationNano, minDurationNano, serviceName, timeRangeHours, minDurationNano, limit)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询服务异常失败: %v", err)), err
	}
	defer rows.Close()

	type ServiceIssue struct {
		TraceID        string  `json:"trace_id"`
		SpanCount      uint64  `json:"span_count"`
		MaxDurationMs  float64 `json:"max_duration_ms"`
		AvgDurationMs  float64 `json:"avg_duration_ms"`
		ErrorCount     uint64  `json:"error_count"`
		OperationCount uint64  `json:"operation_count"`
		IssueType      string  `json:"issue_type"`
	}

	var issues []ServiceIssue
	issueStats := map[string]int{
		"ERROR_AND_SLOW": 0,
		"ERROR_ONLY":     0,
		"SLOW_ONLY":      0,
		"OTHER":          0,
	}

	for rows.Next() {
		var issue ServiceIssue
		if rows.Scan(&issue.TraceID, &issue.SpanCount, &issue.MaxDurationMs, &issue.AvgDurationMs,
			&issue.ErrorCount, &issue.OperationCount, &issue.IssueType) == nil {
			issues = append(issues, issue)
			issueStats[issue.IssueType]++
		}
	}

	result := map[string]interface{}{
		"service_name":           serviceName,
		"query_time_range_hours": timeRangeHours,
		"min_duration_ms":        minDurationMs,
		"total_found":            len(issues),
		"issue_statistics":       issueStats,
		"service_issues":         issues,
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// FindFrequentOperationsTool 查找频繁执行的操作和对应的 trace
func FindFrequentOperationsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取时间范围，默认1小时
	timeRangeHours := 1.0
	if timeVal, ok := request.Params.Arguments["time_range_hours"]; ok && timeVal != nil {
		if timeFloat, ok := timeVal.(float64); ok {
			timeRangeHours = timeFloat
		}
	}

	// 获取最小频率阈值，默认100次
	minFrequency := 100
	if freqVal, ok := request.Params.Arguments["min_frequency"]; ok && freqVal != nil {
		if freqFloat, ok := freqVal.(float64); ok {
			minFrequency = int(freqFloat)
		}
	}

	limit := int(request.Params.Arguments["limit"].(float64))
	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接 ClickHouse 失败: %v", err)), err
	}
	defer conn.Close()

	// 使用ClickHouse的时间函数避免溢出问题
	sql := `
		SELECT 
			name,
			serviceName,
			count() as call_frequency,
			round(avg(duration_nano)/1000000, 2) as avg_duration_ms,
			round(max(duration_nano)/1000000, 2) as max_duration_ms,
			countIf(has_error = 1) as error_count,
			uniq(trace_id) as unique_traces,
			round(count() / ?, 2) as calls_per_hour,
			any(trace_id) as sample_trace_id
		FROM signoz_traces.signoz_index_v3
		WHERE timestamp >= now() - INTERVAL ? HOUR
		GROUP BY name, serviceName
		HAVING call_frequency >= ?
		ORDER BY call_frequency DESC
		LIMIT ?`

	rows, err := conn.Query(ctx, sql, timeRangeHours, timeRangeHours, minFrequency, limit)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询频繁操作失败: %v", err)), err
	}
	defer rows.Close()

	type FrequentOperation struct {
		Name          string  `json:"name"`
		ServiceName   string  `json:"service_name"`
		CallFrequency uint64  `json:"call_frequency"`
		AvgDurationMs float64 `json:"avg_duration_ms"`
		MaxDurationMs float64 `json:"max_duration_ms"`
		ErrorCount    uint64  `json:"error_count"`
		UniqueTraces  uint64  `json:"unique_traces"`
		CallsPerHour  float64 `json:"calls_per_hour"`
		SampleTraceID string  `json:"sample_trace_id"`
	}

	var operations []FrequentOperation
	for rows.Next() {
		var op FrequentOperation
		if rows.Scan(&op.Name, &op.ServiceName, &op.CallFrequency, &op.AvgDurationMs,
			&op.MaxDurationMs, &op.ErrorCount, &op.UniqueTraces, &op.CallsPerHour, &op.SampleTraceID) == nil {
			operations = append(operations, op)
		}
	}

	result := map[string]interface{}{
		"query_time_range_hours": timeRangeHours,
		"min_frequency":          minFrequency,
		"total_found":            len(operations),
		"frequent_operations":    operations,
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// FindAnomalyTracesTool 查找异常模式的 trace
func FindAnomalyTracesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取时间范围，默认1小时
	timeRangeHours := 1.0
	if timeVal, ok := request.Params.Arguments["time_range_hours"]; ok && timeVal != nil {
		if timeFloat, ok := timeVal.(float64); ok {
			timeRangeHours = timeFloat
		}
	}

	limit := int(request.Params.Arguments["limit"].(float64))

	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接 ClickHouse 失败: %v", err)), err
	}
	defer conn.Close()
	// 使用ClickHouse的时间函数避免溢出问题
	sql := `
		SELECT 
			trace_id,
			count() as span_count,
			round(max(duration_nano)/1000000, 2) as max_duration_ms,
			round(avg(duration_nano)/1000000, 2) as avg_duration_ms,
			countIf(has_error = 1) as error_count,
			uniq(serviceName) as service_count,
			uniq(name) as operation_count,
			max(duration_nano) - min(duration_nano) as trace_duration_nano,
			-- 计算负载分数：span数量权重40%，最大持续时间权重30%，错误数权重20%，操作数权重10%
			(count() * 0.4 + (max(duration_nano)/1000000) * 0.3 + countIf(has_error = 1) * 20 * 0.2 + uniq(name) * 0.1) as load_score,
			argMax(name, duration_nano) as slowest_operation,
			any(name) as sample_operation
		FROM signoz_traces.signoz_index_v3
		WHERE timestamp >= now() - INTERVAL ? HOUR
		GROUP BY trace_id		HAVING 
			count() > 1000 OR 
			countIf(has_error = 1) > 5 OR 
			max(duration_nano) > 10000000000 OR
			uniq(serviceName) > 15 OR
			countIf(duration_nano > 5000000000) > 5
		ORDER BY span_count DESC
		LIMIT ?`

	rows, err := conn.Query(ctx, sql, timeRangeHours, limit)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询异常trace失败: %v", err)), err
	}
	defer rows.Close()
	type AnomalyTrace struct {
		TraceID          string  `json:"trace_id"`
		SpanCount        uint64  `json:"span_count"`
		MaxDurationMs    float64 `json:"max_duration_ms"`
		AvgDurationMs    float64 `json:"avg_duration_ms"`
		ErrorCount       uint64  `json:"error_count"`
		ServiceCount     uint64  `json:"service_count"`
		OperationCount   uint64  `json:"operation_count"`
		TraceDurationMs  float64 `json:"trace_duration_ms"`
		LoadScore        float64 `json:"load_score"`
		SlowestOperation string  `json:"slowest_operation"`
		SampleOperation  string  `json:"sample_operation"`
		LoadLevel        string  `json:"load_level"`
		IssueDescription string  `json:"issue_description"`
	}
	var anomalies []AnomalyTrace
	anomalyStats := make(map[string]int)

	for rows.Next() {
		var anomaly AnomalyTrace
		var traceDurationNano uint64
		if rows.Scan(&anomaly.TraceID, &anomaly.SpanCount, &anomaly.MaxDurationMs, &anomaly.AvgDurationMs,
			&anomaly.ErrorCount, &anomaly.ServiceCount, &anomaly.OperationCount, &traceDurationNano,
			&anomaly.LoadScore, &anomaly.SlowestOperation, &anomaly.SampleOperation) == nil {

			anomaly.TraceDurationMs = float64(traceDurationNano) / 1000000.0

			// Determine anomaly type based on the values
			anomalyType := "NORMAL"
			if anomaly.SpanCount > 1000 {
				anomalyType = "EXCESSIVE_SPANS"
			} else if anomaly.ErrorCount > 5 {
				anomalyType = "HIGH_ERROR_RATE"
			} else if anomaly.MaxDurationMs > 10000 {
				anomalyType = "EXTREMELY_SLOW"
			} else if anomaly.ServiceCount > 15 {
				anomalyType = "TOO_MANY_SERVICES"
			}

			anomalies = append(anomalies, anomaly)
			anomalyStats[anomalyType]++
		}
	}

	result := map[string]interface{}{
		"query_time_range_hours": timeRangeHours,
		"total_found":            len(anomalies),
		"anomaly_statistics":     anomalyStats,
		"anomaly_traces":         anomalies,
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// AnalyzeServiceDependenciesTool 分析服务间的调用关系和依赖模式
func AnalyzeServiceDependenciesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceName := request.Params.Arguments["service_name"].(string)

	// 获取时间范围，默认1小时
	timeRangeHours := 1.0
	if timeVal, ok := request.Params.Arguments["time_range_hours"]; ok && timeVal != nil {
		if timeFloat, ok := timeVal.(float64); ok {
			timeRangeHours = timeFloat
		}
	}

	limit := int(request.Params.Arguments["limit"].(float64))

	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接 ClickHouse 失败: %v", err)), err
	}
	defer conn.Close() // 使用ClickHouse的时间函数避免溢出问题
	// 1. 查找该服务调用的下游服务
	downstreamSQL := `
		SELECT 
			downstream.resources_string['service.name'] as downstream_service,
			count() as call_count,
			round(avg(downstream.duration_nano)/1000000, 2) as avg_duration_ms,
			round(max(downstream.duration_nano)/1000000, 2) as max_duration_ms,
			countIf(downstream.has_error = 1) as error_count,
			uniq(downstream.trace_id) as unique_traces,
			round(countIf(downstream.has_error = 1) * 100.0 / count(), 2) as error_rate_pct
		FROM signoz_traces.signoz_index_v3 upstream
		JOIN signoz_traces.signoz_index_v3 downstream ON upstream.trace_id = downstream.trace_id
		WHERE upstream.resources_string['service.name'] = ?
		  AND downstream.resources_string['service.name'] != ?
		  AND upstream.timestamp >= now() - INTERVAL ? HOUR
		  AND downstream.timestamp >= now() - INTERVAL ? HOUR
		  AND downstream.timestamp > upstream.timestamp
		GROUP BY downstream.resources_string['service.name']
		ORDER BY call_count DESC
		LIMIT ?`

	downstreamRows, err := conn.Query(ctx, downstreamSQL, serviceName, serviceName, timeRangeHours, timeRangeHours, limit)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询下游服务失败: %v", err)), err
	}
	defer downstreamRows.Close()

	type ServiceDependency struct {
		ServiceName   string  `json:"service_name"`
		CallCount     uint64  `json:"call_count"`
		AvgDurationMs float64 `json:"avg_duration_ms"`
		MaxDurationMs float64 `json:"max_duration_ms"`
		ErrorCount    uint64  `json:"error_count"`
		UniqueTraces  uint64  `json:"unique_traces"`
		ErrorRatePct  float64 `json:"error_rate_pct"`
	}

	var downstreamServices []ServiceDependency
	for downstreamRows.Next() {
		var dep ServiceDependency
		if err := downstreamRows.Scan(&dep.ServiceName, &dep.CallCount, &dep.AvgDurationMs,
			&dep.MaxDurationMs, &dep.ErrorCount, &dep.UniqueTraces, &dep.ErrorRatePct); err == nil {
			downstreamServices = append(downstreamServices, dep)
		}
	} // 2. 查找调用该服务的上游服务
	upstreamSQL := `
		SELECT 
			upstream.resources_string['service.name'] as upstream_service,
			count() as call_count,
			round(avg(downstream.duration_nano)/1000000, 2) as avg_response_time_ms,
			countIf(downstream.has_error = 1) as error_count,
			uniq(upstream.trace_id) as unique_traces,
			round(countIf(downstream.has_error = 1) * 100.0 / count(), 2) as error_rate_pct
		FROM signoz_traces.signoz_index_v3 upstream
		JOIN signoz_traces.signoz_index_v3 downstream ON upstream.trace_id = downstream.trace_id
		WHERE downstream.resources_string['service.name'] = ?
		  AND upstream.resources_string['service.name'] != ?
		  AND upstream.timestamp >= now() - INTERVAL ? HOUR
		  AND downstream.timestamp >= now() - INTERVAL ? HOUR
		  AND downstream.timestamp > upstream.timestamp
		GROUP BY upstream.resources_string['service.name']
		ORDER BY call_count DESC
		LIMIT ?`

	upstreamRows, err := conn.Query(ctx, upstreamSQL, serviceName, serviceName, timeRangeHours, timeRangeHours, limit)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询上游服务失败: %v", err)), err
	}
	defer upstreamRows.Close()

	var upstreamServices []ServiceDependency
	for upstreamRows.Next() {
		var dep ServiceDependency
		if err := upstreamRows.Scan(&dep.ServiceName, &dep.CallCount, &dep.AvgDurationMs,
			&dep.MaxDurationMs, &dep.ErrorCount, &dep.UniqueTraces, &dep.ErrorRatePct); err == nil {
			upstreamServices = append(upstreamServices, dep)
		}
	}

	// 3. 分析调用模式和健康状况
	var healthIssues []string
	var recommendations []string

	for _, dep := range downstreamServices {
		if dep.ErrorRatePct > 10 {
			healthIssues = append(healthIssues, fmt.Sprintf("下游服务 %s 错误率过高: %.2f%%", dep.ServiceName, dep.ErrorRatePct))
		}
		if dep.AvgDurationMs > 5000 {
			healthIssues = append(healthIssues, fmt.Sprintf("下游服务 %s 响应时间过长: %.2fms", dep.ServiceName, dep.AvgDurationMs))
		}
	}

	for _, dep := range upstreamServices {
		if dep.ErrorRatePct > 10 {
			healthIssues = append(healthIssues, fmt.Sprintf("上游服务 %s 调用失败率过高: %.2f%%", dep.ServiceName, dep.ErrorRatePct))
		}
	}

	if len(downstreamServices) > 10 {
		recommendations = append(recommendations, "考虑减少下游依赖数量，简化服务架构")
	}

	if len(upstreamServices) > 15 {
		recommendations = append(recommendations, "该服务被过多上游服务依赖，考虑拆分功能")
	}

	result := map[string]interface{}{
		"service_name":           serviceName,
		"query_time_range_hours": timeRangeHours,
		"downstream_services":    downstreamServices,
		"upstream_services":      upstreamServices,
		"dependency_summary": map[string]interface{}{
			"downstream_count": len(downstreamServices),
			"upstream_count":   len(upstreamServices),
			"health_issues":    healthIssues,
			"recommendations":  recommendations,
		},
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// ListServicesTool 获取Signoz中所有可用的服务列表及其基本统计信息
func ListServicesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取时间范围，默认24小时
	timeRangeHours := 24.0
	if timeVal, ok := request.Params.Arguments["time_range_hours"]; ok && timeVal != nil {
		if timeFloat, ok := timeVal.(float64); ok {
			timeRangeHours = timeFloat
		}
	}

	// 获取限制数量，默认100
	limit := 100
	if limitVal, ok := request.Params.Arguments["limit"]; ok && limitVal != nil {
		if limitFloat, ok := limitVal.(float64); ok {
			limit = int(limitFloat)
		}
	}

	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接 ClickHouse 失败: %v", err)), err
	}
	defer conn.Close()

	// 查询所有服务及其基本统计信息
	servicesSQL := `
		SELECT 
			serviceName as service_name,
			count() as total_spans,
			uniq(trace_id) as unique_traces,
			uniq(name) as unique_operations,
			round(avg(duration_nano)/1000000, 2) as avg_duration_ms,
			round(max(duration_nano)/1000000, 2) as max_duration_ms,
			countIf(has_error = 1) as error_spans,
			round(countIf(has_error = 1) * 100.0 / count(), 2) as error_rate_pct,
			round(count() / ?, 2) as spans_per_hour,
			min(timestamp) as first_seen,
			max(timestamp) as last_seen
		FROM signoz_traces.signoz_index_v3
		WHERE timestamp >= now() - INTERVAL ? HOUR
		  AND serviceName != ''
		GROUP BY serviceName
		HAVING total_spans > 0
		ORDER BY total_spans DESC
		LIMIT ?
	`

	type ServiceInfo struct {
		ServiceName      string    `json:"service_name"`
		TotalSpans       uint64    `json:"total_spans"`
		UniqueTraces     uint64    `json:"unique_traces"`
		UniqueOperations uint64    `json:"unique_operations"`
		AvgDurationMs    float64   `json:"avg_duration_ms"`
		MaxDurationMs    float64   `json:"max_duration_ms"`
		ErrorSpans       uint64    `json:"error_spans"`
		ErrorRatePct     float64   `json:"error_rate_pct"`
		SpansPerHour     float64   `json:"spans_per_hour"`
		FirstSeen        time.Time `json:"first_seen"`
		LastSeen         time.Time `json:"last_seen"`
		HealthStatus     string    `json:"health_status"`
		ActivityLevel    string    `json:"activity_level"`
	}

	serviceRows, err := conn.Query(ctx, servicesSQL, timeRangeHours, timeRangeHours, limit)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询服务列表失败: %v", err)), err
	}
	defer serviceRows.Close()

	var services []ServiceInfo
	for serviceRows.Next() {
		var service ServiceInfo
		if err := serviceRows.Scan(&service.ServiceName, &service.TotalSpans, &service.UniqueTraces,
			&service.UniqueOperations, &service.AvgDurationMs, &service.MaxDurationMs,
			&service.ErrorSpans, &service.ErrorRatePct, &service.SpansPerHour,
			&service.FirstSeen, &service.LastSeen); err == nil {

			// 评估健康状态
			if service.ErrorRatePct > 10 || service.AvgDurationMs > 5000 {
				service.HealthStatus = "CRITICAL"
			} else if service.ErrorRatePct > 5 || service.AvgDurationMs > 2000 {
				service.HealthStatus = "WARNING"
			} else if service.ErrorRatePct > 1 || service.AvgDurationMs > 1000 {
				service.HealthStatus = "ATTENTION"
			} else {
				service.HealthStatus = "HEALTHY"
			}

			// 评估活跃程度
			if service.SpansPerHour > 1000 {
				service.ActivityLevel = "HIGH"
			} else if service.SpansPerHour > 100 {
				service.ActivityLevel = "MEDIUM"
			} else {
				service.ActivityLevel = "LOW"
			}

			services = append(services, service)
		}
	}

	// 生成统计摘要
	var totalSpans, totalTraces uint64
	healthCounts := map[string]int{"HEALTHY": 0, "ATTENTION": 0, "WARNING": 0, "CRITICAL": 0}
	activityCounts := map[string]int{"HIGH": 0, "MEDIUM": 0, "LOW": 0}

	for _, service := range services {
		totalSpans += service.TotalSpans
		totalTraces += service.UniqueTraces
		healthCounts[service.HealthStatus]++
		activityCounts[service.ActivityLevel]++
	}

	result := map[string]interface{}{
		"query_time_range_hours": timeRangeHours,
		"query_timestamp":        time.Now().Format(time.RFC3339),
		"services":               services,
		"summary": map[string]interface{}{
			"total_services":        len(services),
			"total_spans":           totalSpans,
			"total_traces":          totalTraces,
			"health_distribution":   healthCounts,
			"activity_distribution": activityCounts,
		},
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// ListServiceInterfacesTool 获取指定服务的所有接口列表及其基本统计信息
func ListServiceInterfacesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceName := request.Params.Arguments["service_name"].(string)

	// 获取时间范围，默认24小时
	timeRangeHours := 24.0
	if timeVal, ok := request.Params.Arguments["time_range_hours"]; ok && timeVal != nil {
		if timeFloat, ok := timeVal.(float64); ok {
			timeRangeHours = timeFloat
		}
	}

	// 获取限制数量，默认100
	limit := 100
	if limitVal, ok := request.Params.Arguments["limit"]; ok && limitVal != nil {
		if limitFloat, ok := limitVal.(float64); ok {
			limit = int(limitFloat)
		}
	}

	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接 ClickHouse 失败: %v", err)), err
	}
	defer conn.Close()

	// 查询指定服务的所有接口及其统计信息
	interfacesSQL := `
		SELECT 
			name as interface_name,
			count() as total_calls,
			uniq(trace_id) as unique_traces,
			round(avg(duration_nano)/1000000, 2) as avg_duration_ms,
			round(max(duration_nano)/1000000, 2) as max_duration_ms,
			round(min(duration_nano)/1000000, 2) as min_duration_ms,
			quantile(0.5)(duration_nano/1000000) as p50_duration_ms,
			quantile(0.95)(duration_nano/1000000) as p95_duration_ms,
			quantile(0.99)(duration_nano/1000000) as p99_duration_ms,
			countIf(has_error = 1) as error_calls,
			round(countIf(has_error = 1) * 100.0 / count(), 2) as error_rate_pct,
			round(count() / ?, 2) as calls_per_hour,
			min(timestamp) as first_seen,
			max(timestamp) as last_seen,
			CASE 
				WHEN count() > 1000 THEN 'HIGH_FREQUENCY'
				WHEN count() > 100 THEN 'MEDIUM_FREQUENCY'
				ELSE 'LOW_FREQUENCY'
			END as frequency_pattern,
			CASE 
				WHEN avg(duration_nano) > 1000000000 THEN 'SLOW'
				WHEN avg(duration_nano) > 500000000 THEN 'MODERATE'
				ELSE 'FAST'
			END as performance_pattern
		FROM signoz_traces.signoz_index_v3
		WHERE serviceName = ? 
		  AND timestamp >= now() - INTERVAL ? HOUR
		  AND name != ''
		GROUP BY name
		HAVING total_calls > 0
		ORDER BY total_calls DESC
		LIMIT ?
	`

	type InterfaceDetail struct {
		InterfaceName      string    `json:"interface_name"`
		TotalCalls         uint64    `json:"total_calls"`
		UniqueTraces       uint64    `json:"unique_traces"`
		AvgDurationMs      float64   `json:"avg_duration_ms"`
		MaxDurationMs      float64   `json:"max_duration_ms"`
		MinDurationMs      float64   `json:"min_duration_ms"`
		P50DurationMs      float64   `json:"p50_duration_ms"`
		P95DurationMs      float64   `json:"p95_duration_ms"`
		P99DurationMs      float64   `json:"p99_duration_ms"`
		ErrorCalls         uint64    `json:"error_calls"`
		ErrorRatePct       float64   `json:"error_rate_pct"`
		CallsPerHour       float64   `json:"calls_per_hour"`
		FirstSeen          time.Time `json:"first_seen"`
		LastSeen           time.Time `json:"last_seen"`
		FrequencyPattern   string    `json:"frequency_pattern"`
		PerformancePattern string    `json:"performance_pattern"`
		HealthStatus       string    `json:"health_status"`
		IssueDescription   string    `json:"issue_description"`
	}

	interfaceRows, err := conn.Query(ctx, interfacesSQL, timeRangeHours, serviceName, timeRangeHours, limit)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询服务接口失败: %v", err)), err
	}
	defer interfaceRows.Close()

	var interfaces []InterfaceDetail
	for interfaceRows.Next() {
		var iface InterfaceDetail
		if err := interfaceRows.Scan(&iface.InterfaceName, &iface.TotalCalls, &iface.UniqueTraces,
			&iface.AvgDurationMs, &iface.MaxDurationMs, &iface.MinDurationMs,
			&iface.P50DurationMs, &iface.P95DurationMs, &iface.P99DurationMs,
			&iface.ErrorCalls, &iface.ErrorRatePct, &iface.CallsPerHour,
			&iface.FirstSeen, &iface.LastSeen, &iface.FrequencyPattern, &iface.PerformancePattern); err == nil {

			// 评估健康状态和问题描述
			var issues []string
			if iface.ErrorRatePct > 10 {
				iface.HealthStatus = "CRITICAL"
				issues = append(issues, fmt.Sprintf("高错误率(%.2f%%)", iface.ErrorRatePct))
			} else if iface.ErrorRatePct > 5 {
				iface.HealthStatus = "WARNING"
				issues = append(issues, fmt.Sprintf("错误率偏高(%.2f%%)", iface.ErrorRatePct))
			} else if iface.P95DurationMs > 5000 {
				iface.HealthStatus = "WARNING"
				issues = append(issues, fmt.Sprintf("P95延迟过高(%.2fms)", iface.P95DurationMs))
			} else if iface.P95DurationMs > 2000 || iface.ErrorRatePct > 1 {
				iface.HealthStatus = "ATTENTION"
				if iface.P95DurationMs > 2000 {
					issues = append(issues, fmt.Sprintf("P95延迟较高(%.2fms)", iface.P95DurationMs))
				}
				if iface.ErrorRatePct > 1 {
					issues = append(issues, fmt.Sprintf("存在少量错误(%.2f%%)", iface.ErrorRatePct))
				}
			} else {
				iface.HealthStatus = "HEALTHY"
			}

			// 检查其他潜在问题
			if iface.CallsPerHour > 1000 {
				issues = append(issues, fmt.Sprintf("调用频率过高(%.0f/小时)", iface.CallsPerHour))
			}
			if iface.MaxDurationMs > 30000 {
				issues = append(issues, fmt.Sprintf("最大耗时极长(%.2fms)", iface.MaxDurationMs))
			}

			if len(issues) > 0 {
				iface.IssueDescription = fmt.Sprintf("%v", issues)
			} else {
				iface.IssueDescription = "无明显问题"
			}

			interfaces = append(interfaces, iface)
		}
	}

	// 检查服务是否存在
	if len(interfaces) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("未找到服务 '%s' 在过去 %.1f 小时内的接口数据。请检查服务名称是否正确。", serviceName, timeRangeHours)), nil
	}

	// 生成统计摘要
	var totalCalls, totalTraces uint64
	healthCounts := map[string]int{"HEALTHY": 0, "ATTENTION": 0, "WARNING": 0, "CRITICAL": 0}
	frequencyCounts := map[string]int{"HIGH_FREQUENCY": 0, "MEDIUM_FREQUENCY": 0, "LOW_FREQUENCY": 0}
	performanceCounts := map[string]int{"FAST": 0, "MODERATE": 0, "SLOW": 0}

	for _, iface := range interfaces {
		totalCalls += iface.TotalCalls
		totalTraces += iface.UniqueTraces
		healthCounts[iface.HealthStatus]++
		frequencyCounts[iface.FrequencyPattern]++
		performanceCounts[iface.PerformancePattern]++
	}

	// 生成建议
	var recommendations []string
	if healthCounts["CRITICAL"] > 0 {
		recommendations = append(recommendations, fmt.Sprintf("发现 %d 个关键问题接口，需要立即处理", healthCounts["CRITICAL"]))
	}
	if healthCounts["WARNING"] > 0 {
		recommendations = append(recommendations, fmt.Sprintf("发现 %d 个预警接口，建议及时优化", healthCounts["WARNING"]))
	}
	if performanceCounts["SLOW"] > 0 {
		recommendations = append(recommendations, fmt.Sprintf("发现 %d 个慢接口，建议性能优化", performanceCounts["SLOW"]))
	}
	if frequencyCounts["HIGH_FREQUENCY"] > 3 {
		recommendations = append(recommendations, "多个高频接口，考虑添加缓存或负载均衡")
	}

	result := map[string]interface{}{
		"service_name":           serviceName,
		"query_time_range_hours": timeRangeHours,
		"query_timestamp":        time.Now().Format(time.RFC3339),
		"interfaces":             interfaces,
		"summary": map[string]interface{}{
			"total_interfaces":         len(interfaces),
			"total_calls":              totalCalls,
			"total_traces":             totalTraces,
			"health_distribution":      healthCounts,
			"frequency_distribution":   frequencyCounts,
			"performance_distribution": performanceCounts,
		},
		"recommendations": recommendations,
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// AnalyzeServiceInterfaceLoadTool 分析指定服务的接口负载，找到指定时间内负载最重的top trace
func AnalyzeServiceInterfaceLoadTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceName := request.Params.Arguments["service_name"].(string)

	// 获取接口名称（可选）
	var interfaceName string
	if ifaceVal, ok := request.Params.Arguments["interface_name"]; ok && ifaceVal != nil {
		if ifaceStr, ok := ifaceVal.(string); ok {
			interfaceName = ifaceStr
		}
	}

	// 获取时间范围，默认2小时
	timeRangeHours := 2.0
	if timeVal, ok := request.Params.Arguments["time_range_hours"]; ok && timeVal != nil {
		if timeFloat, ok := timeVal.(float64); ok {
			timeRangeHours = timeFloat
		}
	}

	// 获取返回数量，默认10
	topCount := 10
	if countVal, ok := request.Params.Arguments["top_count"]; ok && countVal != nil {
		if countFloat, ok := countVal.(float64); ok {
			topCount = int(countFloat)
		}
	}

	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接 ClickHouse 失败: %v", err)), err
	}
	defer conn.Close()

	// 构建查询条件
	conditions := []string{"serviceName = ?"}
	queryArgs := []interface{}{serviceName}

	if interfaceName != "" {
		conditions = append(conditions, "name = ?")
		queryArgs = append(queryArgs, interfaceName)
	}

	whereClause := strings.Join(conditions, " AND ")

	// 查询负载最重的trace，考虑span数量、总耗时、错误数等因素
	loadSQL := fmt.Sprintf(`
		SELECT 
			trace_id,
			count() as span_count,
			round(sum(duration_nano)/1000000, 2) as total_duration_ms,
			round(max(duration_nano)/1000000, 2) as max_span_duration_ms,
			round(avg(duration_nano)/1000000, 2) as avg_span_duration_ms,
			countIf(has_error = 1) as error_count,
			uniq(name) as operation_count,
			-- 负载分数计算：span数量权重40%%，总耗时权重30%%，错误数权重20%%，操作复杂度权重10%%
			(count() * 0.4 + (sum(duration_nano)/1000000) * 0.0003 + countIf(has_error = 1) * 50 * 0.2 + uniq(name) * 2 * 0.1) as load_score,
			argMax(name, duration_nano) as slowest_operation,
			min(timestamp) as start_time,
			max(timestamp) as end_time
		FROM signoz_traces.signoz_index_v3
		WHERE %s AND timestamp >= now() - INTERVAL ? HOUR
		GROUP BY trace_id
		HAVING span_count > 1
		ORDER BY load_score DESC
		LIMIT ?
	`, whereClause)

	loadArgs := append(queryArgs, timeRangeHours, topCount)
	loadRows, err := conn.Query(ctx, loadSQL, loadArgs...)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询负载trace失败: %v", err)), err
	}
	defer loadRows.Close()

	type LoadTrace struct {
		TraceID           string    `json:"trace_id"`
		SpanCount         uint64    `json:"span_count"`
		TotalDurationMs   float64   `json:"total_duration_ms"`
		MaxSpanDurationMs float64   `json:"max_span_duration_ms"`
		AvgSpanDurationMs float64   `json:"avg_span_duration_ms"`
		ErrorCount        uint64    `json:"error_count"`
		OperationCount    uint64    `json:"operation_count"`
		LoadScore         float64   `json:"load_score"`
		SlowestOperation  string    `json:"slowest_operation"`
		StartTime         time.Time `json:"start_time"`
		EndTime           time.Time `json:"end_time"`
		LoadLevel         string    `json:"load_level"`
		IssueDescription  string    `json:"issue_description"`
	}

	var heavyTraces []LoadTrace
	for loadRows.Next() {
		var trace LoadTrace
		if err := loadRows.Scan(&trace.TraceID, &trace.SpanCount, &trace.TotalDurationMs,
			&trace.MaxSpanDurationMs, &trace.AvgSpanDurationMs, &trace.ErrorCount,
			&trace.OperationCount, &trace.LoadScore, &trace.SlowestOperation,
			&trace.StartTime, &trace.EndTime); err == nil {

			// 评估负载等级
			if trace.LoadScore > 500 {
				trace.LoadLevel = "EXTREME"
			} else if trace.LoadScore > 200 {
				trace.LoadLevel = "HIGH"
			} else if trace.LoadScore > 100 {
				trace.LoadLevel = "MEDIUM"
			} else {
				trace.LoadLevel = "LOW"
			}

			// 生成问题描述
			var issues []string
			if trace.SpanCount > 1000 {
				issues = append(issues, fmt.Sprintf("Span数量过多(%d)", trace.SpanCount))
			}
			if trace.TotalDurationMs > 30000 {
				issues = append(issues, fmt.Sprintf("总耗时过长(%.2fms)", trace.TotalDurationMs))
			}
			if trace.ErrorCount > 0 {
				issues = append(issues, fmt.Sprintf("包含错误(%d个)", trace.ErrorCount))
			}
			if trace.OperationCount > 50 {
				issues = append(issues, fmt.Sprintf("操作复杂度高(%d种操作)", trace.OperationCount))
			}

			if len(issues) > 0 {
				trace.IssueDescription = strings.Join(issues, ", ")
			} else {
				trace.IssueDescription = "轻度负载"
			}

			heavyTraces = append(heavyTraces, trace)
		}
	}

	// 获取服务和接口的整体统计
	statsSQL := fmt.Sprintf(`
		SELECT 
			count() as total_spans,
			uniq(trace_id) as total_traces,
			round(avg(duration_nano)/1000000, 2) as avg_duration_ms,
			countIf(has_error = 1) as error_spans,
			uniq(name) as unique_operations
		FROM signoz_traces.signoz_index_v3
		WHERE %s AND timestamp >= now() - INTERVAL ? HOUR
	`, whereClause)

	statsArgs := append(queryArgs, timeRangeHours)
	var totalSpans, totalTraces, errorSpans, uniqueOperations uint64
	var avgDurationMs float64

	statsRow := conn.QueryRow(ctx, statsSQL, statsArgs...)
	_ = statsRow.Scan(&totalSpans, &totalTraces, &avgDurationMs, &errorSpans, &uniqueOperations)

	// 生成分析建议
	var recommendations []string
	if len(heavyTraces) > 0 {
		extremeCount := 0
		highCount := 0
		for _, trace := range heavyTraces {
			if trace.LoadLevel == "EXTREME" {
				extremeCount++
			} else if trace.LoadLevel == "HIGH" {
				highCount++
			}
		}

		if extremeCount > 0 {
			recommendations = append(recommendations, fmt.Sprintf("发现%d个极高负载trace，需要立即优化", extremeCount))
		}
		if highCount > 0 {
			recommendations = append(recommendations, fmt.Sprintf("发现%d个高负载trace，建议优化", highCount))
		}

		// 检查是否有共同的慢操作
		slowOps := make(map[string]int)
		for _, trace := range heavyTraces {
			if trace.MaxSpanDurationMs > 5000 {
				slowOps[trace.SlowestOperation]++
			}
		}
		for op, count := range slowOps {
			if count > 1 {
				recommendations = append(recommendations, fmt.Sprintf("操作'%s'在多个trace中表现较慢，建议重点优化", op))
			}
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "未发现明显的负载问题")
	}

	analysisTarget := serviceName
	if interfaceName != "" {
		analysisTarget = fmt.Sprintf("%s::%s", serviceName, interfaceName)
	}

	result := map[string]interface{}{
		"analysis_target":  analysisTarget,
		"time_range_hours": timeRangeHours,
		"query_timestamp":  time.Now().Format(time.RFC3339),
		"heavy_traces":     heavyTraces,
		"overall_statistics": map[string]interface{}{
			"total_spans":       totalSpans,
			"total_traces":      totalTraces,
			"avg_duration_ms":   avgDurationMs,
			"error_spans":       errorSpans,
			"unique_operations": uniqueOperations,
			"analyzed_traces":   len(heavyTraces),
		},
		"load_analysis": map[string]interface{}{
			"extreme_load_traces": func() int {
				count := 0
				for _, t := range heavyTraces {
					if t.LoadLevel == "EXTREME" {
						count++
					}
				}
				return count
			}(),
			"high_load_traces": func() int {
				count := 0
				for _, t := range heavyTraces {
					if t.LoadLevel == "HIGH" {
						count++
					}
				}
				return count
			}(),
		},
		"recommendations": recommendations,
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}
