package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"log"
	"mcp-devops/server/k8s"
	"net/http"
	"os"
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
	svr := server.NewMCPServer("Kubernetes MCP Server", mcp.LATEST_PROTOCOL_VERSION)

	fmt.Println("======================================")
	fmt.Println("MCP服务器配置：")
	fmt.Println("无需鉴权，所有客户端都可以直接访问")
	fmt.Println("======================================")

	// 添加HTTP服务器
	sseServer := server.NewSSEServer(svr)

	// 添加tool， tool 三大要素，名称，描述，参数，参数中也要有描述
	// 添加kubernetes pod相关工具
	svr.AddTool(mcp.NewTool("list_pods",
		mcp.WithDescription("列出指定命名空间中的所有Pod"),
		mcp.WithString("namespace",
			mcp.Description("要查询的命名空间, 默认为default"),
			mcp.DefaultString("default"),
		),
	), k8s.ListPodsTool)
	svr.AddTool(mcp.NewTool("describe_pod",
		mcp.WithDescription("查看pod的详细信息"),
		mcp.WithString("pod_name",
			mcp.Required(),
			mcp.Description("要查看的pod名称"),
		),
		mcp.WithString("namespace",
			mcp.Description("Pod所在的名称空间，默认为default"),
			mcp.Required(),
			mcp.DefaultString("default"),
		),
	), k8s.DsscribePodTool)

	svr.AddTool(mcp.NewTool("delete_pod",
		mcp.WithDescription("删除指定的Pod"),
		mcp.WithString("pod_name",
			mcp.Required(),
			mcp.Description("要删除的Pod名称"),
		),
		mcp.WithString("namespace",
			mcp.Description("Pod所在的命名空间, 默认为default"),
			mcp.DefaultString("default"),
		),
		mcp.WithBoolean("force",
			mcp.Description("是否强制删除"),
			mcp.DefaultBool(false),
		),
	), k8s.DeletePodTool)

	svr.AddTool(mcp.NewTool("pod_logs",
		mcp.WithDescription("获取Pod的日志"),
		mcp.WithString("pod_name",
			mcp.Required(),
			mcp.Description("要查看日志的Pod名称"),
		),
		mcp.WithString("namespace",
			mcp.Description("Pod所在的命名空间, 默认为default"),
			mcp.DefaultString("default"),
		),
		mcp.WithString("container",
			mcp.Description("要查看日志的容器名称, 如果Pod中只有一个容器则可以省略"),
		),
		mcp.WithNumber("tail",
			mcp.Description("要查看的日志行数"),
			mcp.DefaultNumber(100.0),
		),
	), k8s.PodLogsTool)

	// 添加Kubernetes Deployment相关工具
	svr.AddTool(mcp.NewTool("list_deployments",
		mcp.WithDescription("列出指定命名空间中的所有Deployment"),
		mcp.WithString("namespace",
			mcp.Description("要查询的命名空间, 默认为default"),
			mcp.DefaultString("default"),
		),
	), k8s.ListDeploymentsTool)

	svr.AddTool(mcp.NewTool("describe_deployment",
		mcp.WithDescription("查看Deployment的详细信息"),
		mcp.WithString("deployment_name",
			mcp.Required(),
			mcp.Description("要查看的Deployment名称"),
		),
		mcp.WithString("namespace",
			mcp.Description("Deployment所在的命名空间, 默认为default"),
			mcp.DefaultString("default"),
		),
	), k8s.DescribeDeploymentTool)

	svr.AddTool(mcp.NewTool("scale_deployment",
		mcp.WithDescription("调整Deployment的副本数"),
		mcp.WithString("deployment_name",
			mcp.Required(),
			mcp.Description("要调整的Deployment名称"),
		),
		mcp.WithString("namespace",
			mcp.Description("Deployment所在的命名空间, 默认为default"),
			mcp.DefaultString("default"),
		),
		mcp.WithNumber("replicas",
			mcp.Required(),
			mcp.Description("要设置的副本数"),
		),
	), k8s.ScaleDeploymentTool)

	svr.AddTool(mcp.NewTool("restart_deployment",
		mcp.WithDescription("重启Deployment的所有Pod"),
		mcp.WithString("deployment_name",
			mcp.Required(),
			mcp.Description("要重启的Deployment名称"),
		),
		mcp.WithString("namespace",
			mcp.Description("Deployment所在的命名空间, 默认为default"),
			mcp.DefaultString("default"),
		),
	), k8s.RestartDeploymentTool)

	// 添加Kubernetes Service相关工具
	svr.AddTool(mcp.NewTool("list_services",
		mcp.WithDescription("列出指定命名空间中的所有Service"),
		mcp.WithString("namespace",
			mcp.Description("要查询的命名空间, 默认为default"),
			mcp.DefaultString("default"),
		),
	), k8s.ListServicesTool)

	svr.AddTool(mcp.NewTool("describe_service",
		mcp.WithDescription("查看Service的详细信息"),
		mcp.WithString("service_name",
			mcp.Required(),
			mcp.Description("要查看的Service名称"),
		),
		mcp.WithString("namespace",
			mcp.Description("Service所在的命名空间, 默认为default"),
			mcp.DefaultString("default"),
		),
	), k8s.DescribeServiceTool)

	// 添加Kubernetes Namespace相关工具
	svr.AddTool(mcp.NewTool("list_namespaces",
		mcp.WithDescription("列出所有命名空间"),
	), k8s.ListNamespacesTool)

	svr.AddTool(mcp.NewTool("describe_namespace",
		mcp.WithDescription("查看命名空间的详细信息"),
		mcp.WithString("namespace_name",
			mcp.Required(),
			mcp.Description("要查看的命名空间名称"),
		),
	), k8s.DescribeNamespaceTool)

	svr.AddTool(mcp.NewTool("create_namespace",
		mcp.WithDescription("创建新的命名空间"),
		mcp.WithString("namespace_name",
			mcp.Required(),
			mcp.Description("要创建的命名空间名称"),
		),
	), k8s.CreateNamespaceTool)

	svr.AddTool(mcp.NewTool("delete_namespace",
		mcp.WithDescription("删除指定的命名空间"),
		mcp.WithString("namespace_name",
			mcp.Required(),
			mcp.Description("要删除的命名空间名称"),
		),
	), k8s.DeleteNamespaceTool)

	// 启动服务器
	fmt.Printf("正在启动MCP服务器，监听地址: %s\n", address)
	err = http.ListenAndServe(address, sseServer)
	if err != nil {
		log.Fatal(err)
	}
}
