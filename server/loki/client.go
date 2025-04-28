package loki

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// LokiResponse represents the structure of Loki API response
type LokiResponse struct {
	Status string   `json:"status"` // "success" or "error"
	Data   LokiData `json:"data"`
}

// LokiData contains the result data from Loki query
type LokiData struct {
	ResultType string       `json:"resultType"` // for log queries, should be "streams"
	Result     []LokiStream `json:"result"`     // log stream results array
	Stats      LokiStats    `json:"stats"`      // query statistics
}

// LokiStream represents a single log stream with labels and log entries
type LokiStream struct {
	Stream map[string]string `json:"stream"` // log stream labels, e.g. {"app": "myapp", "level": "info"}
	Values [][]string        `json:"values"` // [ [<timestamp_ns_string>, <log_line_string>], ... ]
}

// LokiStats contains query statistics
type LokiStats struct {
	Summary struct {
		BytesProcessedPerSecond int64   `json:"bytesProcessedPerSecond"`
		LinesProcessedPerSecond int64   `json:"linesProcessedPerSecond"`
		TotalBytesProcessed     int64   `json:"totalBytesProcessed"`
		TotalLinesProcessed     int64   `json:"totalLinesProcessed"`
		ExecTimeSeconds         float64 `json:"execTime"`
		QueueTimeSeconds        float64 `json:"queueTime,omitempty"`            // may exist
		Subqueries              int     `json:"subqueries,omitempty"`           // may exist
		TotalEntriesReturned    int     `json:"totalEntriesReturned,omitempty"` // may exist
	} `json:"summary"`
	Querier struct {
		// ... Querier specific stats
	} `json:"querier,omitempty"`
	Ingester struct {
		// ... Ingester specific stats
	} `json:"ingester,omitempty"`
	// ... other possible stats sections
}

// QueryLoki sends a query to Loki and returns the response
func QueryLoki(lokiAddress, logQL string, startTime, endTime time.Time, limit int, direction string) (*LokiResponse, error) {
	// Build request URL
	apiEndpoint := fmt.Sprintf("%s/loki/api/v1/query_range", lokiAddress)
	queryParams := url.Values{}
	queryParams.Add("query", logQL)
	queryParams.Add("start", strconv.FormatInt(startTime.UnixNano(), 10))
	queryParams.Add("end", strconv.FormatInt(endTime.UnixNano(), 10))
	queryParams.Add("limit", strconv.Itoa(limit))
	queryParams.Add("direction", direction)

	fullURL := apiEndpoint + "?" + queryParams.Encode()

	// Create HTTP client and request
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP GET 请求失败: %v", err)
	}

	// Add necessary headers
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行 HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %v", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("请求未成功 (状态码: %d)，响应内容: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse JSON response
	var lokiResp LokiResponse
	err = json.Unmarshal(bodyBytes, &lokiResp)
	if err != nil {
		return nil, fmt.Errorf("解析 Loki JSON 响应失败: %v\n原始响应: %s", err, string(bodyBytes))
	}

	if lokiResp.Status != "success" {
		return nil, fmt.Errorf("Loki 查询 API 返回状态非 'success': %s", lokiResp.Status)
	}

	return &lokiResp, nil
}
