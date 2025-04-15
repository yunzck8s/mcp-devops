package mcp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// 定义错误常量
var (
	ErrClientClosed   = errors.New("MCP客户端已关闭")
	ErrConnectionFail = errors.New("无法连接到MCP服务器")
	ErrInitFail       = errors.New("初始化MCP客户端失败")
)

// 连接状态
type connectionState int

const (
	stateDisconnected connectionState = iota
	stateConnecting
	stateConnected
	stateFailed
)

// ClientManager 管理MCP客户端连接的结构体
type ClientManager struct {
	client               *client.SSEMCPClient
	serverURL            string
	mutex                sync.RWMutex
	state                connectionState
	lastError            error
	reconnect            chan struct{} // 用于触发重连的通道
	stateChange          chan struct{} // 状态变更通知通道
	maxRetries           int           // 最大重试次数
	retryInterval        time.Duration // 重试间隔
	connectTimeout       time.Duration // 连接超时
	closed               bool          // 客户端是否已关闭
	apiToken             string
	sessionID            string
	lastConnectTime      time.Time
	connectLock          sync.Mutex
	connectionFailed     bool
	lastConnectionError  error
	connectionFailedLock sync.RWMutex
	sessionExpiryTimer   *time.Timer
}

// ClientOption 是客户端配置选项函数
type ClientOption func(*ClientManager)

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(retries int) ClientOption {
	return func(cm *ClientManager) {
		if retries > 0 {
			cm.maxRetries = retries
		}
	}
}

// WithRetryInterval 设置重试间隔
func WithRetryInterval(interval time.Duration) ClientOption {
	return func(cm *ClientManager) {
		if interval > 0 {
			cm.retryInterval = interval
		}
	}
}

// WithConnectTimeout 设置连接超时
func WithConnectTimeout(timeout time.Duration) ClientOption {
	return func(cm *ClientManager) {
		if timeout > 0 {
			cm.connectTimeout = timeout
		}
	}
}

// NewClientManager 创建新的MCP客户端管理器
func NewClientManager(serverURL, apiToken string, options ...ClientOption) *ClientManager {
	cm := &ClientManager{
		serverURL:      serverURL,
		apiToken:       apiToken,
		reconnect:      make(chan struct{}, 1),
		stateChange:    make(chan struct{}, 5),
		state:          stateDisconnected,
		maxRetries:     5,
		retryInterval:  2 * time.Second,
		connectTimeout: 5 * time.Second,
	}

	// 应用选项
	for _, option := range options {
		option(cm)
	}

	return cm
}

// Start 启动客户端并开始监听重连信号
func (m *ClientManager) Start(ctx context.Context) error {
	// 首次连接
	if err := m.ensureConnected(ctx); err != nil {
		return err
	}

	// 启动重连监控协程
	go m.reconnectLoop(ctx)

	return nil
}

// reconnectLoop 持续监控重连信号和状态变更
func (m *ClientManager) reconnectLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case <-m.reconnect:
			m.mutex.RLock()
			if m.closed {
				m.mutex.RUnlock()
				return
			}
			m.mutex.RUnlock()

			// 等待一段时间再重连
			time.Sleep(m.retryInterval)

			// 尝试重新连接
			if err := m.ensureConnected(ctx); err != nil {
				// 记录错误但不返回，继续监听重连信号
				fmt.Printf("重连MCP服务器失败: %v\n", err)
			}

		case <-m.stateChange:
			// 处理状态变更，可以在这里添加状态通知逻辑
			continue
		}
	}
}

// GetClient 获取客户端，如果连接异常则尝试重新连接
func (m *ClientManager) GetClient(ctx context.Context) (*client.SSEMCPClient, error) {
	m.connectLock.Lock()
	defer m.connectLock.Unlock()

	// 检查是否需要重新建立连接
	if m.needsReconnect() {
		if err := m.connect(ctx); err != nil {
			return nil, err
		}
	}

	return m.client, nil
}

// ensureConnected 确保客户端已连接
func (m *ClientManager) ensureConnected(ctx context.Context) error {
	m.mutex.Lock()

	// 如果客户端已关闭，直接返回错误
	if m.closed {
		m.mutex.Unlock()
		return ErrClientClosed
	}

	// 如果状态是连接中或已连接，不需要重新连接
	if m.state == stateConnecting || m.state == stateConnected {
		m.mutex.Unlock()
		return nil
	}

	// 设置状态为连接中
	m.setState(stateConnecting)
	m.mutex.Unlock()

	// 使用带取消的上下文进行连接操作
	connectCtx, cancel := context.WithTimeout(ctx, m.connectTimeout)
	defer cancel()

	// 尝试连接
	err := m.connect(connectCtx)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if err != nil {
		m.setState(stateFailed)
		m.lastError = err
		return err
	}

	m.setState(stateConnected)
	return nil
}

// connect 连接到MCP服务器
func (m *ClientManager) connect(ctx context.Context) error {
	// 清除旧客户端
	if m.client != nil {
		m.client.Close()
		m.client = nil
	}

	// 重置会话ID
	m.sessionID = ""

	// 创建新客户端
	var err error
	m.client, err = client.NewSSEMCPClient(m.serverURL)
	if err != nil {
		if Debug {
			fmt.Printf("[连接] 创建MCP客户端失败: %v\n", err)
		}
		return fmt.Errorf("创建MCP客户端失败: %w", err)
	}

	// 配置客户端
	// 注意: 由于SSEMCPClient没有SetToken方法，将apiToken通过其他方式处理
	// 如果需要设置token，可能需要在NewSSEMCPClient前后处理
	if m.apiToken != "" && Debug {
		fmt.Println("[连接] API令牌已配置")
	}

	// 使用完全独立的上下文进行连接，避免外部上下文取消导致SSE流关闭
	connectCtx := context.Background()

	// 建立连接
	fmt.Println("[连接] 正在连接到MCP服务器...")

	// 使用Start方法代替Connect方法
	err = m.client.Start(connectCtx)
	if err != nil {
		m.client.Close()
		m.client = nil
		if Debug {
			fmt.Printf("[连接] 连接失败: %v\n", err)
		}
		return fmt.Errorf("MCP客户端连接失败: %w", err)
	}

	// 等待连接稳定，增加稳定等待时间
	stabilizationWait := 5 * time.Second
	if Debug {
		fmt.Printf("[连接] 连接成功，等待 %v 稳定连接...\n", stabilizationWait)
	} else {
		fmt.Println("[连接] 连接成功，等待连接稳定...")
	}
	time.Sleep(stabilizationWait)

	// 发送初始化请求获取会话ID - 使用独立上下文
	initCtx := context.Background()
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "docker-cli",
		Version: "1.0.0",
	}

	// 使用独立上下文发送初始化请求
	res, err := m.client.Initialize(initCtx, initRequest)
	if err != nil {
		m.client.Close()
		m.client = nil
		if Debug {
			fmt.Printf("[连接] 初始化失败: %v\n", err)
		}
		return fmt.Errorf("MCP客户端初始化失败: %w", err)
	}

	if Debug {
		fmt.Printf("[连接] 初始化成功，响应: %+v\n", res)
	}

	// 尝试从响应中获取会话信息
	var sessionID string
	if res != nil && res.Result.Meta != nil {
		// 尝试从Meta中获取会话ID
		if sid, ok := res.Result.Meta["session_id"].(string); ok && sid != "" {
			sessionID = sid
			if Debug {
				fmt.Printf("[连接] 从响应Meta中获取到会话ID: %s\n", sessionID)
			}
		}
	}

	// 如果无法从响应中获取会话ID，生成一个本地唯一ID
	if sessionID == "" {
		sessionID = fmt.Sprintf("session-%d", time.Now().UnixNano())
		if Debug {
			fmt.Printf("[连接] 使用本地生成的会话ID: %s\n", sessionID)
		}
	}

	// 更新状态
	// 注意: 由于SSEMCPClient没有ID字段，只在本地保存会话ID
	m.sessionID = sessionID
	m.lastConnectTime = time.Now()

	// 清除连接失败标志
	m.connectionFailedLock.Lock()
	m.connectionFailed = false
	m.lastConnectionError = nil
	m.connectionFailedLock.Unlock()

	// 设置会话过期计时器，30分钟后自动标记为过期
	if m.sessionExpiryTimer != nil {
		m.sessionExpiryTimer.Stop()
	}
	m.sessionExpiryTimer = time.AfterFunc(30*time.Minute, func() {
		if Debug {
			fmt.Println("[连接] 会话过期计时器触发，标记连接需要刷新")
		}
		m.MarkConnectionFailed(fmt.Errorf("会话超时自动标记为过期"))
	})

	fmt.Printf("[连接] 建立新会话成功，会话ID: %s\n", sessionID)
	return nil
}

// initializeClient 初始化MCP客户端
func (m *ClientManager) initializeClient(ctx context.Context) error {
	// 初始化客户端
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "docker-cli",
		Version: "1.0.0",
	}

	// 使用单独的上下文，避免共享原始上下文
	// 这样即使原始上下文取消，初始化请求也能完成
	initCtx := context.Background()

	// 发送初始化请求
	_, err := m.client.Initialize(initCtx, initRequest)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInitFail, err)
	}

	return nil
}

// setState 设置连接状态并通知状态变更
func (m *ClientManager) setState(state connectionState) {
	if m.state != state {
		m.state = state
		// 通知状态变更
		select {
		case m.stateChange <- struct{}{}:
		default:
			// 通道已满，忽略
		}
	}
}

// MarkConnectionFailed 标记连接失败
func (m *ClientManager) MarkConnectionFailed(err error) {
	m.connectionFailedLock.Lock()
	defer m.connectionFailedLock.Unlock()

	m.connectionFailed = true
	m.lastConnectionError = err

	// 通知需要重连
	select {
	case m.reconnect <- struct{}{}:
	default:
		// 通道已满，不再发送
	}
}

// Close 关闭MCP客户端管理器
func (m *ClientManager) Close() error {
	m.connectLock.Lock()
	defer m.connectLock.Unlock()

	// 停止会话过期计时器
	if m.sessionExpiryTimer != nil {
		m.sessionExpiryTimer.Stop()
		m.sessionExpiryTimer = nil
	}

	// 如果客户端存在，关闭它
	if m.client != nil {
		if Debug {
			fmt.Println("[连接] 正在关闭MCP客户端...")
		}

		// 关闭客户端
		err := m.client.Close()
		m.client = nil
		m.sessionID = ""

		if Debug && err != nil {
			fmt.Printf("[连接] 关闭客户端时出错: %v\n", err)
		}
	}

	// 设置相关标志
	m.connectionFailedLock.Lock()
	m.connectionFailed = true
	m.lastConnectionError = nil
	m.connectionFailedLock.Unlock()

	if Debug {
		fmt.Println("[连接] MCP客户端管理器已关闭")
	}

	return nil
}

// IsConnected 检查客户端是否已连接
func (m *ClientManager) IsConnected() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.state == stateConnected && m.client != nil && !m.closed
}

// GetReconnectChannel 获取重连通道
func (m *ClientManager) GetReconnectChannel() <-chan struct{} {
	return m.reconnect
}

// NeedsReconnect 检查是否需要重新连接
func (m *ClientManager) NeedsReconnect() bool {
	return m.needsReconnect()
}

// needsReconnect 内部方法，检查是否需要重新连接
func (m *ClientManager) needsReconnect() bool {
	// 检查连接失败标志
	m.connectionFailedLock.RLock()
	connectionFailed := m.connectionFailed
	m.connectionFailedLock.RUnlock()

	if connectionFailed {
		return true
	}

	// 检查客户端是否为空
	if m.client == nil {
		return true
	}

	// 检查会话是否过期（超过25分钟）
	if time.Since(m.lastConnectTime) > 25*time.Minute {
		return true
	}

	return false
}

// GetStateChannel 获取状态变更通道
func (m *ClientManager) GetStateChannel() <-chan struct{} {
	return m.stateChange
}

// GetLastError 获取最后一次错误
func (m *ClientManager) GetLastError() error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.lastError
}

// RefreshSession 刷新当前的会话，重置会话过期计时器
func (m *ClientManager) RefreshSession() {
	m.connectLock.Lock()
	defer m.connectLock.Unlock()

	// 如果客户端不存在，无需刷新
	if m.client == nil {
		return
	}

	// 重置会话过期计时器，延长会话有效期
	if m.sessionExpiryTimer != nil {
		m.sessionExpiryTimer.Reset(30 * time.Minute)
	} else {
		// 如果定时器不存在，创建一个新的
		m.sessionExpiryTimer = time.AfterFunc(30*time.Minute, func() {
			m.MarkConnectionFailed(fmt.Errorf("会话超时自动标记为过期"))
		})
	}

	// 更新最后连接时间，减少不必要的重连
	m.lastConnectTime = time.Now()

	// 清除连接失败标志
	m.connectionFailedLock.Lock()
	m.connectionFailed = false
	m.lastConnectionError = nil
	m.connectionFailedLock.Unlock()
}

// HealthCheck 执行健康检查，确保会话正常
// 返回true表示健康，false表示不健康
func (m *ClientManager) HealthCheck() bool {
	// 首先获取读锁，以便快速检查
	m.connectionFailedLock.RLock()
	if m.connectionFailed {
		m.connectionFailedLock.RUnlock()
		if Debug {
			fmt.Println("[健康] 连接已标记为失败，需要重连")
		}
		return false
	}
	m.connectionFailedLock.RUnlock()

	// 获取连接锁
	m.connectLock.Lock()
	defer m.connectLock.Unlock()

	// 检查客户端是否存在
	if m.client == nil {
		if Debug {
			fmt.Println("[健康] 客户端不存在，需要重连")
		}
		return false
	}

	// 检查会话ID是否为空
	if m.sessionID == "" {
		if Debug {
			fmt.Println("[健康] 会话ID为空，需要重连")
		}
		return false
	}

	// 检查最后连接时间，如果超过20分钟，认为会话可能不健康
	if time.Since(m.lastConnectTime) > 20*time.Minute {
		if Debug {
			fmt.Printf("[健康] 会话时间过长 (%v)，需要刷新\n", time.Since(m.lastConnectTime))
		}
		// 尝试刷新会话而不是立即重连
		m.RefreshSession()
		return true
	}

	// 会话健康
	return true
}
