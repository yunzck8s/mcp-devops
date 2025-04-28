package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv" // 用于将时间戳转换为字符串
	"time"
)

// --- Loki API 响应结构体 ---
// 结构与 Loki /query_range API 返回的 JSON 格式匹配

type LokiResponse struct {
	Status string   `json:"status"` // "success" 或 "error"
	Data   LokiData `json:"data"`
}

type LokiData struct {
	ResultType string       `json:"resultType"` // 对于日志查询，应为 "streams"
	Result     []LokiStream `json:"result"`     // 日志流结果数组
	Stats      LokiStats    `json:"stats"`      // 查询统计信息
}

type LokiStream struct {
	Stream map[string]string `json:"stream"` // 日志流的标签集，如 {"app": "myapp", "level": "info"}
	Values [][]string        `json:"values"` // [ [<timestamp_ns_string>, <log_line_string>], ... ]
}

// 查询统计信息结构体 (根据文档和实际响应可能包含更多字段)
type LokiStats struct {
	Summary struct {
		BytesProcessedPerSecond int64   `json:"bytesProcessedPerSecond"`
		LinesProcessedPerSecond int64   `json:"linesProcessedPerSecond"`
		TotalBytesProcessed     int64   `json:"totalBytesProcessed"`
		TotalLinesProcessed     int64   `json:"totalLinesProcessed"`
		ExecTimeSeconds         float64 `json:"execTime"`
		QueueTimeSeconds        float64 `json:"queueTime,omitempty"`            // 可能存在
		Subqueries              int     `json:"subqueries,omitempty"`           // 可能存在
		TotalEntriesReturned    int     `json:"totalEntriesReturned,omitempty"` // 可能存在
	} `json:"summary"`
	Querier struct {
		// ... Querier specific stats
	} `json:"querier,omitempty"`
	Ingester struct {
		// ... Ingester specific stats
	} `json:"ingester,omitempty"`
	// ... 其他可能的统计部分
}

func main() {
	// --- 配置 ---
	// 替换为你的 Loki 查询器(Querier)或查询前端(Query Frontend)的地址
	lokiAddress := "http://localhost:3100"
	// LogQL 查询语句 (示例: 查询所有来自 'my-app' 且包含 'error' 或 'failed' 的日志)
	logQL := `{app="cilium-agent"}`
	// 查询时间范围
	endTime := time.Now()
	startTime := endTime.Add(-30 * time.Minute) // 查询过去 30 分钟
	// 其他查询参数
	limit := 1000           // 最多返回 1000 条
	direction := "backward" // "backward" (默认) 或 "forward"

	// --- 构建请求 URL ---
	// 使用文档中确认的 /loki/api/v1/query_range 端点
	apiEndpoint := fmt.Sprintf("%s/loki/api/v1/query_range", lokiAddress)

	// 准备 URL 查询参数
	queryParams := url.Values{}
	queryParams.Add("query", logQL)
	// 根据文档，时间需要是纳秒级 Unix 时间戳或 RFC3339Nano 字符串
	// 这里使用纳秒时间戳字符串
	queryParams.Add("start", strconv.FormatInt(startTime.UnixNano(), 10))
	queryParams.Add("end", strconv.FormatInt(endTime.UnixNano(), 10))
	queryParams.Add("limit", strconv.Itoa(limit))
	queryParams.Add("direction", direction)
	// 可选：如果想用 RFC3339Nano 格式时间
	// queryParams.Add("start", startTime.Format(time.RFC3339Nano))
	// queryParams.Add("end", endTime.Format(time.RFC3339Nano))

	// 组合基础 URL 和编码后的查询参数
	fullURL := apiEndpoint + "?" + queryParams.Encode()
	fmt.Println("请求 URL:", fullURL)

	// --- 创建 HTTP 客户端和请求 ---
	client := &http.Client{
		Timeout: 60 * time.Second, // 设置一个合理的超时时间
	}

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Fatalf("创建 HTTP GET 请求失败: %v", err)
	}

	// --- 添加必要的请求头 ---
	req.Header.Set("Accept", "application/json")
	// 如果 Loki 部署在多租户模式下，需要设置 X-Scope-OrgID
	// req.Header.Set("X-Scope-OrgID", "your_tenant_id")

	// 如果需要认证 (例如 Basic Auth 或 Bearer Token)，在这里添加
	// req.SetBasicAuth("username", "password")
	// req.Header.Set("Authorization", "Bearer your_api_token")

	// --- 发送请求 ---
	fmt.Println("正在发送请求...")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("执行 HTTP 请求失败: %v", err)
	}
	// 务必在函数结束前关闭响应体
	defer resp.Body.Close()

	// --- 处理响应 ---
	fmt.Printf("收到响应，状态码: %s\n", resp.Status)

	// 读取响应体内容
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("读取响应体失败: %v", err)
	}

	// 检查状态码是否为 2xx 成功
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Fatalf("请求未成功 (状态码: %d)，响应内容: %s", resp.StatusCode, string(bodyBytes))
	}

	// --- 解析 JSON 响应 ---
	var lokiResp LokiResponse
	err = json.Unmarshal(bodyBytes, &lokiResp)
	if err != nil {
		// 如果解析失败，打印原始响应体可能有助于调试
		log.Fatalf("解析 Loki JSON 响应失败: %v\n原始响应: %s", err, string(bodyBytes))
	}

	// --- 打印查询结果 ---
	if lokiResp.Status != "success" {
		log.Printf("Loki 查询 API 返回状态非 'success': %s\n", lokiResp.Status)
		// 可以在这里打印更多错误信息，如果 API 返回了的话
		return
	}

	fmt.Printf("\n查询状态: %s\n", lokiResp.Status)
	fmt.Printf("结果类型: %s\n", lokiResp.Data.ResultType) // 应为 "streams"
	fmt.Printf("找到 %d 个日志流\n", len(lokiResp.Data.Result))
	fmt.Println("========================================")

	totalLogsFound := 0
	for i, stream := range lokiResp.Data.Result {
		fmt.Printf("日志流 #%d\n", i+1)
		fmt.Printf("  标签 (Stream): %v\n", stream.Stream)
		fmt.Printf("  日志条数: %d\n", len(stream.Values))
		fmt.Printf("  日志内容 (Values):\n")
		for _, valuePair := range stream.Values {
			if len(valuePair) == 2 {
				// valuePair[0] 是纳秒时间戳字符串, valuePair[1] 是日志行内容
				timestampNsStr := valuePair[0]
				logLine := valuePair[1]

				// 将纳秒时间戳字符串转换为 time.Time 对象
				tsNanoInt, err := strconv.ParseInt(timestampNsStr, 10, 64)
				var displayTime string
				if err == nil {
					// 使用 UTC 时间或本地时间格式化
					displayTime = time.Unix(0, tsNanoInt).UTC().Format(time.RFC3339)
					// 或者本地时间：displayTime = time.Unix(0, tsNanoInt).Local().Format("2006-01-02 15:04:05.999")
				} else {
					displayTime = timestampNsStr // 转换失败则显示原始字符串
				}

				fmt.Printf("    [%s] %s\n", displayTime, logLine)
				totalLogsFound++
			} else {
				fmt.Printf("    [格式错误的数据对: %v]\n", valuePair)
			}
		}
		fmt.Println("----------------------------------------")
	}
	fmt.Printf("========================================\n")
	fmt.Printf("在所有流中总共找到 %d 条日志记录\n", totalLogsFound)

	// 打印查询统计信息
	fmt.Printf("\n查询统计 (摘要):\n")
	stats := lokiResp.Data.Stats.Summary
	fmt.Printf("  执行时间: %.4fs\n", stats.ExecTimeSeconds)
	fmt.Printf("  队列时间: %.4fs\n", stats.QueueTimeSeconds)
	fmt.Printf("  返回条数: %d\n", stats.TotalEntriesReturned)
	fmt.Printf("  总处理行数: %d\n", stats.TotalLinesProcessed)
	fmt.Printf("  总处理字节: %d\n", stats.TotalBytesProcessed)
	fmt.Printf("  处理速度 (行/秒): %d\n", stats.LinesProcessedPerSecond)
	fmt.Printf("  处理速度 (字节/秒): %d\n", stats.BytesProcessedPerSecond)
	fmt.Printf("  子查询数: %d\n", stats.Subqueries)
	fmt.Println("========================================")
}
