package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
)

var (
	// toolsCache 缓存工具实例，提高性能
	toolsCache     []tool.BaseTool
	toolsCacheLock sync.RWMutex
	lastFetchTime  time.Time
	// toolCacheTTL 工具缓存有效期，超过这个时间需要重新获取
	toolCacheTTL = 10 * time.Minute
)

// 调试模式变量
var Debug bool

// 初始化函数
func init() {
	// 检查是否开启调试模式
	debugEnv := os.Getenv("DEBUG")
	Debug = debugEnv == "1" || debugEnv == "true" || debugEnv == "yes"
}

// GetMCPTools 获取MCP工具列表
// verbose参数控制是否打印详细工具列表
// forceRefresh参数控制是否强制刷新工具缓存
func GetMCPTools(ctx context.Context, clientManager *ClientManager, options ...bool) ([]tool.BaseTool, error) {
	// 解析选项参数
	verbose, forceRefresh := false, false
	if len(options) > 0 {
		verbose = options[0]
	}
	if len(options) > 1 {
		forceRefresh = options[1]
	}

	// 检查缓存是否有效且未被强制刷新
	if !forceRefresh {
		toolsCacheLock.RLock()
		if toolsCache != nil && len(toolsCache) > 0 && time.Since(lastFetchTime) < toolCacheTTL {
			tools := toolsCache
			toolsCacheLock.RUnlock()
			if verbose {
				fmt.Printf("使用缓存的 %d 个 MCP 工具\n", len(tools))
			}
			return tools, nil
		}
		toolsCacheLock.RUnlock()
	}

	// 使用完全独立的上下文进行工具获取，使用超长超时时间
	toolCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 获取MCP客户端
	cli, err := clientManager.GetClient(ctx)
	if err != nil {
		if Debug || verbose {
			fmt.Printf("[调试] 获取MCP客户端失败: %v\n", err)
		}
		return nil, fmt.Errorf("获取MCP客户端失败: %w", err)
	}

	if verbose || Debug {
		fmt.Println("[工具] 获取Docker和Kubernetes工具列表...")
	}

	// 尝试获取工具
	var tools []tool.BaseTool
	var getErr error

	// 添加重试逻辑，最多尝试4次，增加重试次数
	for attempt := 0; attempt < 4; attempt++ {
		// 每次尝试时显示进度
		if verbose || Debug {
			fmt.Printf("[工具] 尝试获取工具 (%d/4)...\n", attempt+1)
		}

		// 使用超长超时时间的mcpp配置
		startTime := time.Now()
		tools, getErr = mcpp.GetTools(toolCtx, &mcpp.Config{
			Cli: cli,
		})
		duration := time.Since(startTime)

		// 工具获取后刷新会话，防止会话失效
		clientManager.RefreshSession()

		if Debug {
			fmt.Printf("[调试] 工具获取耗时: %v\n", duration)
		}

		if getErr == nil {
			if Debug {
				fmt.Printf("[调试] 成功获取工具，数量: %d\n", len(tools))
			}
			break // 成功获取工具
		}

		// 检查错误类型
		errMsg := getErr.Error()
		isConnectionErr := strings.Contains(errMsg, "Invalid session ID") ||
			strings.Contains(errMsg, "connection") ||
			strings.Contains(errMsg, "stream closed") ||
			strings.Contains(errMsg, "deadline exceeded")

		if isConnectionErr {
			// 记录错误并重新建立连接
			fmt.Printf("[工具] 工具获取失败 (尝试 %d/4): %v\n", attempt+1, getErr)

			if Debug {
				fmt.Printf("[调试] 错误详情: %s\n", errMsg)
			}

			// 如果是最后一次尝试，标记连接失败并返回错误
			if attempt == 3 {
				clientManager.MarkConnectionFailed(getErr)
				return nil, fmt.Errorf("获取MCP工具失败，可能是超时或会话ID无效: %w", getErr)
			}

			// 标记连接失败
			clientManager.MarkConnectionFailed(getErr)

			// 获取新的客户端
			cli, err = clientManager.GetClient(ctx)
			if err != nil {
				if Debug {
					fmt.Printf("[调试] 重新获取客户端失败: %v\n", err)
				}
				return nil, fmt.Errorf("重新获取MCP客户端失败: %w", err)
			}

			// 等待更长时间后重试
			retryWait := 5 * time.Second
			if Debug {
				fmt.Printf("[调试] 等待 %v 后重试...\n", retryWait)
			}
			time.Sleep(retryWait)
			continue
		}

		// 其他类型错误直接返回
		if Debug {
			fmt.Printf("[调试] 非连接错误: %v\n", getErr)
		}
		return nil, fmt.Errorf("获取MCP工具失败: %w", getErr)
	}

	// 检查是否成功获取了工具
	if len(tools) == 0 {
		return nil, fmt.Errorf("获取到的工具列表为空")
	}

	// 更新缓存
	toolsCacheLock.Lock()
	toolsCache = tools
	lastFetchTime = time.Now()
	toolsCacheLock.Unlock()

	if verbose {
		fmt.Printf("成功获取 %d 个 MCP 工具\n", len(tools))
		printToolsList(toolCtx, tools)
	}

	return tools, nil
}

// ResetToolsCache 重置工具缓存
func ResetToolsCache() {
	toolsCacheLock.Lock()
	defer toolsCacheLock.Unlock()

	toolsCache = nil
	lastFetchTime = time.Time{}
}

// SetToolCacheTTL 设置工具缓存有效期
func SetToolCacheTTL(ttl time.Duration) {
	if ttl > 0 {
		toolCacheTTL = ttl
	}
}

// printToolsList 打印工具列表
func printToolsList(ctx context.Context, tools []tool.BaseTool) {
	if len(tools) == 0 {
		fmt.Println("  没有可用的工具")
		return
	}

	// 使用超时上下文，避免获取工具信息时阻塞
	infoCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for i, t := range tools {
		// 使用独立的上下文获取每个工具的信息，避免一个错误影响全部
		info, err := t.Info(infoCtx)
		if err != nil {
			fmt.Printf("  %d. [获取工具信息失败: %v]\n", i+1, err)
			continue
		}

		// 只显示工具名称，避免结构体字段访问错误
		fmt.Printf("  %d. %s\n", i+1, info.Name)
	}
}
