package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"

	// 本地包导入
	"mcp-devops/client/pkg/mcp"
	"mcp-devops/client/pkg/model"
)

const (
	maxRetries      = 5  // 最大重试次数
	retryInterval   = 5  // 重试间隔(秒)
	agentTimeout    = 90 // 代理执行超时时间(秒)
	toolTimeout     = 30 // 工具执行超时时间(秒)
	toolUpdateTime  = 30 // 工具更新间隔(分钟)
	maxHistoryItems = 10 // 最大历史记录条数
	reconnectBuffer = 5  // 重连通道缓冲区大小
)

// Debug 是否开启调试模式
var Debug bool

// Application 应用程序结构体
type Application struct {
	ctx            context.Context
	cancel         context.CancelFunc
	clientManager  *mcp.ClientManager
	tools          []tool.BaseTool
	runner         *react.Agent
	dialog         []*schema.Message
	lastCommand    string
	pendingRetry   bool
	lastUpdateTime time.Time
}

// NewApplication 创建新的应用程序实例
func NewApplication() *Application {
	// 创建根上下文
	ctx, cancel := context.WithCancel(context.Background())

	return &Application{
		ctx:            ctx,
		cancel:         cancel,
		dialog:         make([]*schema.Message, 0),
		lastUpdateTime: time.Now().Add(-toolUpdateTime * time.Minute), // 强制首次更新
	}
}

// Initialize 初始化应用
func (app *Application) Initialize() error {
	fmt.Println("==== 云原生容器管理客户端启动 ====")
	fmt.Println("支持 Docker 和 Kubernetes 资源管理")

	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		return fmt.Errorf("加载环境变量失败: %w", err)
	}

	// 获取服务器URL
	serverURL := os.Getenv("MCP_SERVER_URL")
	if serverURL == "" {
		return fmt.Errorf("MCP_SERVER_URL 环境变量未设置")
	}

	fmt.Printf("使用服务器URL: %s\n", serverURL)

	// 清除之前可能存在的工具缓存
	mcp.ResetToolsCache()

	// 初始化客户端管理器，使用更长的超时时间
	app.clientManager = mcp.NewClientManager(
		serverURL,
		os.Getenv("MCP_API_TOKEN"),
		mcp.WithMaxRetries(maxRetries),
		mcp.WithRetryInterval(time.Duration(3)*time.Second),  // 增加重试间隔
		mcp.WithConnectTimeout(time.Duration(8)*time.Second), // 增加连接超时
	)

	fmt.Println("正在连接MCP服务器...")

	// 创建一个独立的上下文用于初始化，避免共享app.ctx
	initCtx := context.Background()

	// 启动客户端并等待连接稳定
	if err := app.clientManager.Start(initCtx); err != nil {
		return fmt.Errorf("启动MCP客户端失败: %w", err)
	}

	// 等待更长时间确保连接稳定
	fmt.Println("MCP连接已建立，等待连接稳定...")
	time.Sleep(5 * time.Second)

	// 初始化系统提示
	app.dialog = append(app.dialog, &schema.Message{
		Role: schema.System,
		Content: `
	作为云原生容器管理助手，你必须始终回复中文并且严格遵守以下规则：

	# 系统能力
	你可以管理Kubernetes资源，包括：
	- Kubernetes: Pod、Deployment、Service、命名空间等资源管理

	# 命令规则
	0. 记住永远不能一次执行多个命令，你应该执行一个命令等待结果后才执行下一个命令
	  不要使用会持续产生输出的命令

	1. 生成命令前必须：
	  - 检查危险操作（删除、清理等）
	  - 确认命令格式符合Windows要求，不要使用|或者grep等unix才有的命令
	  - 危险命令必须按此格式确认："【安全提示】即将执行：xxx，是否继续？(Y/N)"

	2. Kubernetes命令规则：
	  - 不要手动构造kubectl命令，使用提供的MCP工具，只有在真正无法实现的时候可以执行
	  - 操作前先检查相关资源是否存在
	  - 命名空间敏感操作需要先确认命名空间
	  - 删除资源操作需要二次确认

	3. 错误处理原则：
	  - 当命令执行失败时，用普通用户能理解的方式解释错误
	  - 不要尝试自动修复需要权限的操作
	  - 提供可能的解决方案
	  - **如果收到系统发送的错误信息，你应该分析错误原因，并尝试提出修复建议。**

	4. 关于操作超时：
	  - 执行stop、restart、remove等操作时可能需要较长时间
	  - 如果执行命令后长时间没有响应，可能是服务器处理超时
	  - 建议用户再次查看资源状态确认操作是否成功
	5. pod名称：
	  - 有时候用户只会输入pod名称中去除随机生成的内容，你要能识别出来
	  - 如果要查看pod或者容器日志，你应该知道pod中有哪些容器，哪个容器是用户想要的
	  - 有时候用户给的pod名称不全，你需要自己判断是否存在名称相近的pod

	示例对话：
	用户：删除所有停止的容器
	你：【安全提示】即将执行：docker system prune -a，这将删除所有未使用的容器、镜像和网络，是否继续？(Y/N)

	用户：查看所有的Kubernetes命名空间
	你：我将获取所有Kubernetes命名空间列表。
	`,
	})

	// 使用独立上下文尝试更新工具，避免上下文取消问题
	var err error
	fmt.Println("正在获取MCP工具...")

	for i := 0; i < 3; i++ {
		fmt.Printf("[系统] 第 %d 次尝试获取工具...\n", i+1)
		// 创建独立上下文进行工具获取
		toolCtx := context.Background()
		err = app.updateToolsWithContext(toolCtx, i == 0) // 第一次尝试时显示详细信息
		if err == nil {
			break
		}

		fmt.Printf("[系统] 获取工具失败: %v\n", err)

		// 如果是最后一次尝试，返回错误
		if i == 2 {
			return fmt.Errorf("获取MCP工具失败: %w", err)
		}

		// 重置连接并等待后重试
		fmt.Println("[系统] 重置连接并等待5秒后重试...")
		app.clientManager.MarkConnectionFailed(fmt.Errorf("工具获取失败，重置连接"))
		time.Sleep(5 * time.Second)
	}

	fmt.Printf("[系统] 成功获取 %d 个工具，初始化完成\n", len(app.tools))

	return nil
}

// updateToolsWithContext 使用指定上下文更新工具列表并重新初始化代理
func (app *Application) updateToolsWithContext(ctx context.Context, verbose bool) error {
	var err error

	// 获取工具列表
	app.tools, err = mcp.GetMCPTools(ctx, app.clientManager, verbose, true)
	if err != nil {
		return fmt.Errorf("获取MCP工具失败: %w", err)
	}

	app.lastUpdateTime = time.Now()

	// 初始化聊天模型
	cm := model.NewChatModel(
		ctx, // 使用传入的上下文
		os.Getenv("OPENAI_API_KEY"),
		os.Getenv("OPENAI_BASE_URL"),
		os.Getenv("OPENAI_MODEL"),
	)

	// 初始化代理
	app.runner, err = react.NewAgent(ctx, &react.AgentConfig{
		Model: cm,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: app.tools,
		},
		MaxStep: 40,
	})

	if err != nil {
		return fmt.Errorf("初始化Agent失败: %w", err)
	}

	return nil
}

// updateTools 更新工具列表并重新初始化代理
func (app *Application) updateTools(verbose bool) error {
	// 创建独立上下文进行工具更新
	toolCtx := context.Background()
	return app.updateToolsWithContext(toolCtx, verbose)
}

// startReconnectMonitor 启动重连监控
func (app *Application) startReconnectMonitor() {
	go func() {
		for {
			select {
			case <-app.ctx.Done():
				return

			case <-app.clientManager.GetReconnectChannel():
				fmt.Println("\n[系统] 检测到连接重置信号，尝试重新初始化MCP工具...")

				// 等待一段时间再重连
				time.Sleep(time.Duration(retryInterval) * time.Second)

				if err := app.updateTools(false); err != nil {
					fmt.Printf("[系统] 重新连接MCP服务器失败: %v\n", err)
					continue
				}

				fmt.Println("[系统] MCP工具重新连接成功")

				// 设置重试标志
				if app.lastCommand != "" {
					app.pendingRetry = true
					fmt.Println("[系统] 检测到连接问题已解决，将自动重试上一次的命令...")
				}
			}
		}
	}()
}

// handleUserInput 处理用户输入
func (app *Application) handleUserInput() (string, bool) {
	fmt.Println("\nYou: ")

	// 如果需要自动重试上一条命令
	if app.pendingRetry && app.lastCommand != "" {
		message := app.lastCommand
		fmt.Println(message + " (自动重试)")
		app.pendingRetry = false
		return message, false
	}

	// 读取用户输入
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			fmt.Printf("[系统] 读取输入失败: %v\n", err)
		}
		return "", true
	}

	message := scanner.Text()

	// 检查是否要退出
	if message == "exit" || message == "quit" || message == "退出" {
		return "", true
	}

	// 保存最后一条命令
	if message != "" {
		app.lastCommand = message
	}

	return message, false
}

// processCommand 处理命令
func (app *Application) processCommand(message string) bool {
	// 检查是否是特殊命令
	if message == "更新工具" || message == "刷新工具" || message == "重新连接" {
		fmt.Println("[系统] 正在更新容器管理工具...")

		// 使用独立上下文进行工具更新
		toolCtx := context.Background()
		if err := app.updateToolsWithContext(toolCtx, true); err != nil {
			fmt.Printf("[系统] 获取MCP工具失败: %v\n", err)
			fmt.Println("AI: 很抱歉，我无法连接到容器管理服务器，请检查服务器是否正常运行。")
			return false
		}

		fmt.Printf("[系统] 成功获取 %d 个容器管理工具，已完成更新\n", len(app.tools))
		return true
	}

	// 检查是否需要更新工具（超出更新间隔或客户端需要重连）
	needUpdateTools := time.Since(app.lastUpdateTime) > time.Duration(toolUpdateTime)*time.Minute ||
		app.clientManager.NeedsReconnect()

	if needUpdateTools {
		fmt.Println("[系统] 正在更新容器管理工具...")

		// 使用独立上下文进行工具更新
		toolCtx := context.Background()
		if err := app.updateToolsWithContext(toolCtx, false); err != nil {
			fmt.Printf("[系统] 获取MCP工具失败: %v\n", err)
			fmt.Println("AI: 很抱歉，我无法连接到容器管理服务器，请检查服务器是否正常运行。")
			return false
		}
	}

	// 添加用户消息到对话历史
	app.dialog = append(app.dialog, &schema.Message{
		Role:    schema.User,
		Content: message,
	})

	// 创建独立上下文进行命令执行，避免共享app.ctx导致的上下文取消问题
	cmdCtx := context.Background()

	// 设置超时上下文，增加时间为90秒，给复杂命令更多时间
	generateCtx, generateCancel := context.WithTimeout(cmdCtx, time.Duration(90)*time.Second)
	defer generateCancel()

	// 在执行命令前先刷新会话
	if app.clientManager != nil {
		app.clientManager.RefreshSession()
	}

	// 执行生成
	done := make(chan struct{})
	var out *schema.Message
	var generateErr error

	// 显示等待提示
	fmt.Print("AI: 正在处理您的请求")
	waitIndicator := time.NewTicker(1 * time.Second)
	defer waitIndicator.Stop()

	// 创建工作协程
	go func() {
		// 使用更长的超时时间进行生成
		out, generateErr = app.runner.Generate(generateCtx, app.dialog, agent.WithComposeOptions())
		close(done)
	}()

	// 等待生成完成或超时
	waiting := true
	for waiting {
		select {
		case <-done:
			waiting = false
			fmt.Println() // 换行，结束进度指示

			// 命令执行完毕后刷新会话
			if app.clientManager != nil {
				app.clientManager.RefreshSession()
			}

			if generateErr != nil {
				// 检查是否是连接或超时问题
				if isConnectionError(generateErr) {
					fmt.Printf("\n[系统] 检测到连接问题: %v\n尝试重新连接MCP服务器...\n", generateErr)
					app.clientManager.MarkConnectionFailed(generateErr)
					fmt.Println("AI: 很抱歉，连接服务器时出现问题，正在尝试重新连接，请稍后再试。")
					return false
				}
				//// 输出AI回应
				//output := out.Content
				//fmt.Println("AI: " + output)
				//
				//// 添加AI回复到对话历史
				//app.dialog = append(app.dialog, &schema.Message{
				//	Role:    schema.Assistant,
				//	Content: output,
				//})
				// 检查是否是工具执行超时问题
				if isToolTimeoutError(generateErr) {
					fmt.Printf("\n[系统] 工具执行超时: %v\n", generateErr)
					fmt.Println("AI: 很抱歉，执行命令时超时。这通常发生在处理大量数据或系统负载高时。请尝试简化命令或稍后再试。")
					return false
				}

				fmt.Printf("\n[系统] 运行Agent失败: %v\n", generateErr)
				fmt.Println("AI: 我在处理您的请求时遇到了问题，请稍后再试或尝试不同的命令。")
				return false
			}

			// 输出AI回应
			output := out.Content
			fmt.Println("AI: " + output)

			// 添加AI回复到对话历史
			app.dialog = append(app.dialog, &schema.Message{
				Role:    schema.Assistant,
				Content: output,
			})

		case <-waitIndicator.C:
			// 显示等待动画
			fmt.Print(".")

		case <-time.After(time.Duration(100) * time.Second):
			// 超时
			waiting = false
			fmt.Println() // 换行，结束进度指示
			generateCancel()
			fmt.Println("\n[系统] 命令执行超时")
			fmt.Println("AI: 处理您的请求时间过长，可能是服务器响应缓慢或命令过于复杂。请尝试更简单的命令或稍后再试。")

			// 标记连接可能有问题
			app.clientManager.MarkConnectionFailed(fmt.Errorf("命令执行超时"))
			return false
		}
	}

	return false
}

// isConnectionError 检查是否是连接错误
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	return strings.Contains(errMsg, "connection") ||
		strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "EOF") ||
		strings.Contains(errMsg, "Invalid session ID")
}

// isToolTimeoutError 检查是否是工具执行超时错误
func isToolTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	return (strings.Contains(errMsg, "deadline exceeded") ||
		strings.Contains(errMsg, "context deadline exceeded")) &&
		(strings.Contains(errMsg, "execute node[tools]") ||
			strings.Contains(errMsg, "tool call") ||
			strings.Contains(errMsg, "invoke tool"))
}

// trimDialogHistory 裁剪对话历史
func (app *Application) trimDialogHistory() {
	// 保留系统消息和最近的对话记录
	if len(app.dialog) > maxHistoryItems+1 {
		// 保留系统消息和最近的对话记录
		app.dialog = append(app.dialog[:1], app.dialog[len(app.dialog)-(maxHistoryItems):]...)
	}
}

// Start 启动应用
func (app *Application) Start() {
	// 启动重连监控
	app.startReconnectMonitor()

	fmt.Println("客户端准备就绪，请输入您的命令 (输入'exit'退出):")

	for {
		// 处理用户输入
		message, shouldExit := app.handleUserInput()
		if shouldExit {
			break
		}

		// 处理命令
		skipCurrentCommand := app.processCommand(message)
		if skipCurrentCommand {
			continue
		}

		// 裁剪对话历史
		app.trimDialogHistory()
	}

	fmt.Println("客户端已退出")
}

// Shutdown 优雅关闭应用
func (app *Application) Shutdown() {
	if app.clientManager != nil {
		app.clientManager.Close()
	}
	app.cancel()
}

// runHealthCheck 启动会话健康检查
func (app *Application) runHealthCheck(ctx context.Context) {
	// 创建一个每5分钟触发一次的定时器
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 上下文被取消，退出
			if Debug {
				fmt.Println("[健康] 健康检查协程退出")
			}
			return
		case <-ticker.C:
			// 执行健康检查
			if app.clientManager != nil {
				if !app.clientManager.HealthCheck() {
					if Debug {
						fmt.Println("[健康] 会话健康检查失败，尝试更新工具...")
					}
					// 健康检查失败，尝试更新工具列表 - 使用独立上下文
					healthCtx := context.Background()
					_ = app.updateToolsWithContext(healthCtx, false)
				} else {
					// 健康检查通过，主动刷新会话
					if Debug {
						fmt.Println("[健康] 会话健康检查通过，刷新会话...")
					}
					app.clientManager.RefreshSession()
				}
			}
		}
	}
}

// 主函数
func main() {
	// 创建带取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 初始化应用
	app := &Application{
		ctx:    ctx,
		cancel: cancel,
		dialog: make([]*schema.Message, 0, maxHistoryItems),
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n接收到终止信号，正在关闭客户端...")
		app.Shutdown()
		os.Exit(0)
	}()

	// 尝试初始化应用，失败重试最多3次
	var initErr error
	for i := 0; i < 3; i++ {
		fmt.Printf("=== 第 %d 次尝试初始化应用 ===\n", i+1)

		initErr = app.Initialize()
		if initErr == nil {
			break
		}

		fmt.Printf("初始化失败: %v\n", initErr)

		// 在重试前清理旧的资源
		if app.clientManager != nil {
			app.clientManager.Close()
			app.clientManager = nil
		}

		if i < 2 {
			fmt.Println("等待10秒后重试...")
			time.Sleep(10 * time.Second)
		}
	}

	// 如果初始化失败，退出应用
	if initErr != nil {
		log.Fatalf("初始化应用失败: %v\n", initErr)
	}

	// 启动会话健康检查
	go app.runHealthCheck(ctx)

	// 启动应用
	app.Start()
}
