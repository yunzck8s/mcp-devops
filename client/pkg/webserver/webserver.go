package webserver

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket" // 推荐使用 gorilla/websocket 处理 WebSocket
)

// 定义 WebSocket 的升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许所有来源的连接，生产环境中应配置更严格的检查
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// client 代表一个连接的 Web UI 客户端
type client struct {
	conn *websocket.Conn
	send chan []byte // 用于向该客户端发送消息的缓冲通道
}

// Server 结构体实现了 main.WebServer 接口
type Server struct {
	addr    string
	server  *http.Server
	clients map[*client]bool // 存储所有连接的客户端
	mu      sync.Mutex       // 用于保护 clients map 的互斥锁
	inputCh chan<- string    // 从 main 包传入的输入通道 (只写)
	broadcast chan []byte    // 用于广播消息给所有客户端的通道
	register  chan *client   // 用于注册新客户端的通道
	unregister chan *client   // 用于注销客户端的通道
}

// NewWebServer 创建一个新的 WebServer 实例
func NewWebServer(addr string) *Server {
	return &Server{
		addr:       addr,
		clients:    make(map[*client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *client),
		unregister: make(chan *client),
	}
}

// run 管理客户端连接和消息广播
func (s *Server) run() {
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()
			log.Println("[WebUI] 新客户端连接")

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
				log.Println("[WebUI] 客户端断开连接")
			}
			s.mu.Unlock()

		case message := <-s.broadcast:
			s.mu.Lock()
			for client := range s.clients {
				select {
				case client.send <- message:
				default: // 如果发送缓冲区满，则关闭连接
					close(client.send)
					delete(s.clients, client)
					log.Println("[WebUI] 客户端发送缓冲区满，断开连接")
				}
			}
			s.mu.Unlock()
		}
	}
}

// handleWebSocket 处理 WebSocket 连接
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebUI] WebSocket 升级失败: %v", err)
		return
	}

	// 创建新客户端
	c := &client{conn: conn, send: make(chan []byte, 256)}
	s.register <- c

	// 启动写协程和读协程
	go s.writePump(c)
	go s.readPump(c)
}

// readPump 从 WebSocket 读取消息并发送到 inputCh
func (s *Server) readPump(c *client) {
	defer func() {
		s.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512) // 设置最大消息大小
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // 设置读超时
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebUI] WebSocket 读取错误: %v", err)
			}
			break // 退出循环，触发 defer 中的注销和关闭
		}

		// 将收到的消息（用户输入）发送到主应用的 inputCh
		inputStr := string(message)
		log.Printf("[WebUI] 收到消息: %s", inputStr)
		select {
		case s.inputCh <- inputStr:
			// 发送成功
		default:
			log.Println("[警告] 主应用输入通道已满，Web UI 输入可能丢失。")
			// 可以考虑给客户端发送一个错误提示
			// c.send <- []byte("错误：服务器繁忙，请稍后再试。")
		}
		// 重置读超时
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	}
}

// writePump 将消息从 send 通道写入 WebSocket
func (s *Server) writePump(c *client) {
	ticker := time.NewTicker(50 * time.Second) // 定时发送 ping 消息保持连接
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)) // 设置写超时
			if !ok {
				// send 通道已关闭
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 如果通道中还有更多消息，一次性写入以提高效率
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'}) // 添加换行符分隔消息（可选）
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return // 发送 ping 失败，连接可能已断开
			}
		}
	}
}

// Start 启动 Web 服务器
func (s *Server) Start(inputCh chan<- string) error {
	s.inputCh = inputCh // 保存输入通道

	// 启动后台管理协程
	go s.run()

	// 设置 HTTP 路由
	mux := http.NewServeMux()
	// WebSocket 端点
	mux.HandleFunc("/ws", s.handleWebSocket)
	// 可以添加一个简单的 HTTP 端点用于健康检查或提供静态文件
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 简单的响应，表明服务器正在运行
		// 实际应用中，这里可以服务前端静态文件 (HTML, CSS, JS)
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintln(w, "MCP DevOps Web Server is running. Connect via WebSocket at /ws")
	})

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	log.Printf("[WebUI] 服务器正在监听 %s", s.addr)
	// 启动 HTTP 服务器
	// ListenAndServe 会阻塞，直到服务器关闭或出错
	err := s.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Printf("[WebUI] 服务器启动失败: %v", err)
		return err // 返回错误
	}
	log.Println("[WebUI] 服务器已停止。")
	return nil // 正常关闭时返回 nil
}

// Broadcast 将消息广播给所有连接的 Web UI 客户端
func (s *Server) Broadcast(message string) {
	log.Printf("[WebUI] 广播消息: %s", message)
	s.broadcast <- []byte(message)
}

// Shutdown 平滑关闭 Web 服务器
func (s *Server) Shutdown() {
	log.Println("[WebUI] 正在关闭服务器...")
	if s.server != nil {
		// 创建一个带有超时的上下文
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 尝试平滑关闭服务器
		if err := s.server.Shutdown(ctx); err != nil {
			log.Printf("[WebUI] 服务器关闭失败: %v", err)
		} else {
			log.Println("[WebUI] 服务器已成功关闭。")
		}
	}
	// 关闭管理通道（可选，取决于 run 协程的退出逻辑）
	// close(s.broadcast)
	// close(s.register)
	// close(s.unregister)
}