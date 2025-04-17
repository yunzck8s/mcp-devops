package main

import (
	"fmt"
	"log"
	"mcp-devops/server/sse"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	var err error
	// 加载环境变量
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	fmt.Println("======================================")
	fmt.Println("Kubernetes MCP 服务器启动中...")
	fmt.Println("版本: 1.0.0")
	fmt.Println("======================================")

	// 获取配置参数
	address := os.Getenv("MCP_SERVER_ADDRESS")

	// 创建并配置 MCP 服务器
	svr, _ := sse.K8sServer()

	fmt.Println("======================================")
	fmt.Println("MCP服务器配置：")
	fmt.Println("无需鉴权，所有客户端都可以直接访问")
	fmt.Println("======================================")

	// 添加HTTP服务器
	sseServer := server.NewSSEServer(svr)

	// 启动服务器
	fmt.Printf("正在启动MCP服务器，监听地址: %s\n", address)
	err = http.ListenAndServe(address, sseServer)
	if err != nil {
		log.Fatal(err)
	}
}
