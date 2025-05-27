package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// AddClickhouseTools 注册 ClickHouse 链路分析工具
func AddClickhouseTools(svr *server.MCPServer) {
	// 查找 span 数量超过阈值的 trace_id 列表
	svr.AddTool(mcp.NewTool("find_heavy_traces",
		mcp.WithDescription("查找 span 数量超过阈值的 trace_id 列表"),
		mcp.WithNumber("threshold",
			mcp.Description("span 数量告警阈值"),
			mcp.DefaultNumber(1000),
		),
		mcp.WithNumber("limit",
			mcp.Description("结果限制条数"),
			mcp.DefaultNumber(100),
		),
	), FindHeavyTracesTool)

	// 查找 span 最大耗时排序的 trace_id 列表
	svr.AddTool(mcp.NewTool("find_top_slowest_traces",
		mcp.WithDescription("按最大 span 耗时排序，获取 trace_id 列表"),
		mcp.WithNumber("limit",
			mcp.Description("结果限制条数"),
			mcp.DefaultNumber(10),
		),
	), FindTopSlowTracesTool)

	// 查找包含错误的 trace_id 列表
	svr.AddTool(mcp.NewTool("find_error_traces",
		mcp.WithDescription("查找包含错误 spans 的 trace_id 列表"),
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

	// 查找 span 数量大于1000 的 trace_id 列表
	svr.AddTool(mcp.NewTool("find_spans_gt_1000",
		mcp.WithDescription("查找 span 数量大于1000 的 trace_id 列表"),
	), FindSpansGT1000Tool)
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
}

// FindHeavyTracesTool 列出 span 数量超过阈值的 trace_id
func FindHeavyTracesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	th := int64(request.Params.Arguments["threshold"].(float64))
	lim := int(request.Params.Arguments["limit"].(float64))
	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接 ClickHouse 失败: %v", err)), err
	}
	defer conn.Close()

	sql := fmt.Sprintf(
		`SELECT trace_id, count() AS cnt FROM signoz_traces.signoz_index_v3
         GROUP BY trace_id HAVING cnt > %d
         ORDER BY cnt DESC LIMIT %d`, th, lim)
	rows, err := conn.Query(ctx, sql)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询 heavy traces 失败: %v", err)), err
	}
	defer rows.Close()

	type Item struct {
		TraceID string
		Count   uint64
	}
	var list []Item
	for rows.Next() {
		var id string
		var cnt uint64
		if rows.Scan(&id, &cnt) == nil {
			list = append(list, Item{id, cnt})
		}
	}
	out, _ := json.MarshalIndent(list, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// FindTopSlowTracesTool 按最大 span 耗时排序的 trace_id
func FindTopSlowTracesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lim := int(request.Params.Arguments["limit"].(float64))
	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接失败: %v", err)), err
	}
	defer conn.Close()
	sql := fmt.Sprintf(
		`SELECT trace_id, max(duration_nano) AS max_dur FROM signoz_traces.signoz_index_v3
         GROUP BY trace_id ORDER BY max_dur DESC LIMIT %d`, lim)
	rows, err := conn.Query(ctx, sql)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询 slow traces 失败: %v", err)), err
	}
	defer rows.Close()

	type Item struct {
		TraceID string
		MaxDur  uint64
	}
	var list []Item
	for rows.Next() {
		var id string
		var dur uint64
		if rows.Scan(&id, &dur) == nil {
			list = append(list, Item{id, dur})
		}
	}
	out, _ := json.MarshalIndent(list, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// FindErrorTracesTool 列出包含错误 spans 的 trace_id
func FindErrorTracesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lim := int(request.Params.Arguments["limit"].(float64))
	conn, err := NewClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("连接失败: %v", err)), err
	}
	defer conn.Close()
	sql := fmt.Sprintf(
		`SELECT trace_id, countIf(has_error) AS err_count FROM signoz_traces.signoz_index_v3
         GROUP BY trace_id HAVING err_count > 0 ORDER BY err_count DESC LIMIT %d`, lim)
	rows, err := conn.Query(ctx, sql)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("查询 error traces 失败: %v", err)), err
	}
	defer rows.Close()

	type Item struct {
		TraceID  string
		ErrCount uint64
	}
	var list []Item
	for rows.Next() {
		var id string
		var cnt uint64
		if rows.Scan(&id, &cnt) == nil {
			list = append(list, Item{id, cnt})
		}
	}
	out, _ := json.MarshalIndent(list, "", "  ")
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
		"span_patterns":     patterns,
		"redis_analysis":    redisAnalysis,
		"anomaly_analysis":  anomalyAnalysis,
		"service_behaviors": serviceBehaviors,
		"summary": map[string]interface{}{
			"total_pattern_types": len(patterns),
			"potential_issues":    len(anomalyAnalysis.Anomalies),
			"services_involved":   len(serviceBehaviors),
		},
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

	// 计算时间范围（纳秒时间戳）
	nowNano := uint64(time.Now().UnixNano())
	timeRangeNano := uint64(timeRangeHours * 3600 * 1000000000) // 转换为纳秒
	startTimeNano := nowNano - timeRangeNano

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
		WHERE serviceName = ? AND timestamp >= ?
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
	metricsRow := conn.QueryRow(ctx, metricsSQL, timeRangeHours, serviceName, startTimeNano)
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
			END as performance_pattern
		FROM signoz_traces.signoz_index_v3 
		WHERE serviceName = ? AND timestamp >= ?
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

	operationRows, err := conn.Query(ctx, operationSQL, serviceName, startTimeNano, limit)
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
			toHour(toDateTime(timestamp/1000000000)) as hour,
			count() as span_count,
			countIf(has_error = 1) as error_count,
			round(avg(duration_nano)/1000000, 2) as avg_duration_ms
		FROM signoz_traces.signoz_index_v3 
		WHERE serviceName = ? AND timestamp >= ?
		GROUP BY hour
		ORDER BY hour
	`

	type HourlyPattern struct {
		Hour          uint8   `json:"hour"`
		SpanCount     uint64  `json:"span_count"`
		ErrorCount    uint64  `json:"error_count"`
		AvgDurationMs float64 `json:"avg_duration_ms"`
	}

	timeRows, err := conn.Query(ctx, timeDistributionSQL, serviceName, startTimeNano)
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
	}

	// 5. 依赖服务分析
	dependencySQL := `
		SELECT 
			t2.serviceName as target_service,
			count() as call_count,
			round(avg(t2.duration_nano)/1000000, 2) as avg_duration_ms,
			countIf(t2.has_error = 1) as error_count
		FROM signoz_traces.signoz_index_v3 t1
		JOIN signoz_traces.signoz_index_v3 t2 ON t1.trace_id = t2.trace_id
		WHERE t1.serviceName = ? 
		  AND t2.serviceName != ? 
		  AND t1.timestamp >= ?
		  AND t2.timestamp >= ?
		GROUP BY t2.serviceName
		ORDER BY call_count DESC
		LIMIT 20
	`

	type DependencyInfo struct {
		TargetService string  `json:"target_service"`
		CallCount     uint64  `json:"call_count"`
		AvgDurationMs float64 `json:"avg_duration_ms"`
		ErrorCount    uint64  `json:"error_count"`
	}

	depRows, err := conn.Query(ctx, dependencySQL, serviceName, serviceName, startTimeNano, startTimeNano)
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
		WHERE serviceName = ? AND timestamp >= ?
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

	slowRows, err := conn.Query(ctx, slowTracesSQL, serviceName, startTimeNano)
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
