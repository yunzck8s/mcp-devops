package linux

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// KubeletStatusTool 获取kubelet状态的工具函数
func KubeletStatusTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hostname, _ := request.Params.Arguments["hostname"].(string)

	fmt.Println("ai 正在调用mcp server的tool: kubelet_status, hostname=", hostname)

	// 构建命令
	var commands []string
	if hostname != "" {
		commands = []string{
			fmt.Sprintf("ssh %s 'systemctl status kubelet'", hostname),
			fmt.Sprintf("ssh %s 'journalctl -u kubelet --no-pager -n 50'", hostname),
		}
	} else {
		commands = []string{
			"systemctl status kubelet",
			"journalctl -u kubelet --no-pager -n 50",
		}
	}

	// 执行命令并收集输出
	var result strings.Builder
	result.WriteString("Kubelet状态:\n\n")

	for _, cmd := range commands {
		output, err := executeCommand(cmd)
		if err != nil {
			result.WriteString(fmt.Sprintf("执行命令 '%s' 失败: %v\n", cmd, err))
			continue
		}
		result.WriteString(output)
		result.WriteString("\n\n")
	}

	return mcp.NewToolResultText(result.String()), nil
}

// ContainerRuntimeStatusTool 获取容器运行时状态的工具函数
func ContainerRuntimeStatusTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hostname, _ := request.Params.Arguments["hostname"].(string)
	runtime, _ := request.Params.Arguments["runtime"].(string)
	if runtime == "" {
		runtime = "docker" // 默认为docker
	}

	fmt.Println("ai 正在调用mcp server的tool: container_runtime_status, hostname=", hostname, ", runtime=", runtime)

	// 构建命令
	var commands []string
	var serviceName string

	switch strings.ToLower(runtime) {
	case "docker":
		serviceName = "docker"
	case "containerd":
		serviceName = "containerd"
	case "crio", "cri-o":
		serviceName = "crio"
	default:
		serviceName = runtime
	}

	if hostname != "" {
		commands = []string{
			fmt.Sprintf("ssh %s 'systemctl status %s'", hostname, serviceName),
			fmt.Sprintf("ssh %s 'journalctl -u %s --no-pager -n 30'", hostname, serviceName),
		}

		// 添加特定于运行时的命令
		switch strings.ToLower(runtime) {
		case "docker":
			commands = append(commands, fmt.Sprintf("ssh %s 'docker info'", hostname))
			commands = append(commands, fmt.Sprintf("ssh %s 'docker ps'", hostname))
		case "containerd":
			commands = append(commands, fmt.Sprintf("ssh %s 'ctr version'", hostname))
			commands = append(commands, fmt.Sprintf("ssh %s 'ctr containers list'", hostname))
		case "crio", "cri-o":
			commands = append(commands, fmt.Sprintf("ssh %s 'crictl info'", hostname))
			commands = append(commands, fmt.Sprintf("ssh %s 'crictl ps'", hostname))
		}
	} else {
		commands = []string{
			fmt.Sprintf("systemctl status %s", serviceName),
			fmt.Sprintf("journalctl -u %s --no-pager -n 30", serviceName),
		}

		// 添加特定于运行时的命令
		switch strings.ToLower(runtime) {
		case "docker":
			commands = append(commands, "docker info")
			commands = append(commands, "docker ps")
		case "containerd":
			commands = append(commands, "ctr version")
			commands = append(commands, "ctr containers list")
		case "crio", "cri-o":
			commands = append(commands, "crictl info")
			commands = append(commands, "crictl ps")
		}
	}

	// 执行命令并收集输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("容器运行时 '%s' 状态:\n\n", runtime))

	for _, cmd := range commands {
		output, err := executeCommand(cmd)
		if err != nil {
			result.WriteString(fmt.Sprintf("执行命令 '%s' 失败: %v\n", cmd, err))
			continue
		}
		result.WriteString(output)
		result.WriteString("\n\n")
	}

	return mcp.NewToolResultText(result.String()), nil
}

// KubeProxyStatusTool 获取kube-proxy状态的工具函数
func KubeProxyStatusTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hostname, _ := request.Params.Arguments["hostname"].(string)

	fmt.Println("ai 正在调用mcp server的tool: kube_proxy_status, hostname=", hostname)

	// 构建命令
	var commands []string
	if hostname != "" {
		commands = []string{
			fmt.Sprintf("ssh %s 'ps aux | grep kube-proxy | grep -v grep'", hostname),
			fmt.Sprintf("ssh %s 'iptables-save | grep -i kube-proxy'", hostname),
			fmt.Sprintf("ssh %s 'ipvsadm -ln'", hostname),
		}
	} else {
		commands = []string{
			"ps aux | grep kube-proxy | grep -v grep",
			"iptables-save | grep -i kube-proxy",
			"ipvsadm -ln",
		}
	}

	// 执行命令并收集输出
	var result strings.Builder
	result.WriteString("Kube-Proxy状态:\n\n")

	for _, cmd := range commands {
		output, err := executeCommand(cmd)
		if err != nil {
			result.WriteString(fmt.Sprintf("执行命令 '%s' 失败: %v\n", cmd, err))
			continue
		}
		result.WriteString(output)
		result.WriteString("\n\n")
	}

	return mcp.NewToolResultText(result.String()), nil
}

// NodeNetworkDebugTool 节点网络调试工具函数
func NodeNetworkDebugTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hostname, _ := request.Params.Arguments["hostname"].(string)
	target, _ := request.Params.Arguments["target"].(string)
	port, _ := request.Params.Arguments["port"].(float64)
	if port == 0 {
		port = 80 // 默认端口
	}

	fmt.Println("ai 正在调用mcp server的tool: node_network_debug, hostname=", hostname, ", target=", target)

	// 构建命令
	var commands []string
	if hostname != "" {
		if target != "" {
			commands = []string{
				fmt.Sprintf("ssh %s 'ping -c 4 %s'", hostname, target),
				fmt.Sprintf("ssh %s 'traceroute %s'", hostname, target),
				fmt.Sprintf("ssh %s 'nc -zv %s %d'", hostname, target, int(port)),
				fmt.Sprintf("ssh %s 'curl -I -m 5 %s:%d'", hostname, target, int(port)),
			}
		} else {
			commands = []string{
				fmt.Sprintf("ssh %s 'ip route'", hostname),
				fmt.Sprintf("ssh %s 'cat /etc/resolv.conf'", hostname),
				fmt.Sprintf("ssh %s 'iptables -L -n'", hostname),
			}
		}
	} else {
		if target != "" {
			commands = []string{
				fmt.Sprintf("ping -c 4 %s", target),
				fmt.Sprintf("traceroute %s", target),
				fmt.Sprintf("nc -zv %s %d", target, int(port)),
				fmt.Sprintf("curl -I -m 5 %s:%d", target, int(port)),
			}
		} else {
			commands = []string{
				"ip route",
				"cat /etc/resolv.conf",
				"iptables -L -n",
			}
		}
	}

	// 执行命令并收集输出
	var result strings.Builder
	if target != "" {
		result.WriteString(fmt.Sprintf("网络连接测试 (目标: %s:%d):\n\n", target, int(port)))
	} else {
		result.WriteString("节点网络配置:\n\n")
	}

	for _, cmd := range commands {
		output, err := executeCommand(cmd)
		if err != nil {
			result.WriteString(fmt.Sprintf("执行命令 '%s' 失败: %v\n", cmd, err))
			continue
		}
		result.WriteString(output)
		result.WriteString("\n\n")
	}

	return mcp.NewToolResultText(result.String()), nil
}

// CNIStatusTool 获取CNI状态的工具函数
func CNIStatusTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hostname, _ := request.Params.Arguments["hostname"].(string)
	cniType, _ := request.Params.Arguments["cni_type"].(string)

	fmt.Println("ai 正在调用mcp server的tool: cni_status, hostname=", hostname, ", cni_type=", cniType)

	// 构建命令
	var commands []string
	if hostname != "" {
		// 基本CNI配置检查
		commands = []string{
			fmt.Sprintf("ssh %s 'ls -la /etc/cni/net.d/'", hostname),
			fmt.Sprintf("ssh %s 'cat /etc/cni/net.d/*.conf'", hostname),
		}

		// 根据CNI类型添加特定命令
		if cniType != "" {
			switch strings.ToLower(cniType) {
			case "calico":
				commands = append(commands, fmt.Sprintf("ssh %s 'calicoctl node status'", hostname))
				commands = append(commands, fmt.Sprintf("ssh %s 'ps aux | grep calico'", hostname))
			case "flannel":
				commands = append(commands, fmt.Sprintf("ssh %s 'ps aux | grep flannel'", hostname))
				commands = append(commands, fmt.Sprintf("ssh %s 'ip route | grep flannel'", hostname))
			case "weave":
				commands = append(commands, fmt.Sprintf("ssh %s 'weave status'", hostname))
				commands = append(commands, fmt.Sprintf("ssh %s 'ps aux | grep weave'", hostname))
			case "cilium":
				commands = append(commands, fmt.Sprintf("ssh %s 'cilium status'", hostname))
				commands = append(commands, fmt.Sprintf("ssh %s 'ps aux | grep cilium'", hostname))
			}
		}
	} else {
		// 基本CNI配置检查
		commands = []string{
			"ls -la /etc/cni/net.d/",
			"cat /etc/cni/net.d/*.conf",
		}

		// 根据CNI类型添加特定命令
		if cniType != "" {
			switch strings.ToLower(cniType) {
			case "calico":
				commands = append(commands, "calicoctl node status")
				commands = append(commands, "ps aux | grep calico")
			case "flannel":
				commands = append(commands, "ps aux | grep flannel")
				commands = append(commands, "ip route | grep flannel")
			case "weave":
				commands = append(commands, "weave status")
				commands = append(commands, "ps aux | grep weave")
			case "cilium":
				commands = append(commands, "cilium status")
				commands = append(commands, "ps aux | grep cilium")
			}
		}
	}

	// 执行命令并收集输出
	var result strings.Builder
	if cniType != "" {
		result.WriteString(fmt.Sprintf("CNI '%s' 状态:\n\n", cniType))
	} else {
		result.WriteString("CNI配置:\n\n")
	}

	for _, cmd := range commands {
		output, err := executeCommand(cmd)
		if err != nil {
			result.WriteString(fmt.Sprintf("执行命令 '%s' 失败: %v\n", cmd, err))
			continue
		}
		result.WriteString(output)
		result.WriteString("\n\n")
	}

	return mcp.NewToolResultText(result.String()), nil
}

// KubeComponentLogsTool 获取Kubernetes组件日志的工具函数
func KubeComponentLogsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hostname, _ := request.Params.Arguments["hostname"].(string)
	component, _ := request.Params.Arguments["component"].(string)
	if component == "" {
		component = "kubelet" // 默认组件
	}

	lines, _ := request.Params.Arguments["lines"].(float64)
	if lines == 0 {
		lines = 50 // 默认行数
	}

	fmt.Println("ai 正在调用mcp server的tool: kube_component_logs, hostname=", hostname, ", component=", component)

	// 构建命令
	var command string
	if hostname != "" {
		command = fmt.Sprintf("ssh %s 'journalctl -u %s --no-pager -n %d'", hostname, component, int(lines))
	} else {
		command = fmt.Sprintf("journalctl -u %s --no-pager -n %d", component, int(lines))
	}

	// 执行命令
	output, err := executeCommand(command)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取组件日志失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Kubernetes组件 '%s' 日志 (最后 %d 行):\n\n", component, int(lines)))
	result.WriteString(output)

	// 添加错误和警告统计
	var errorCommand, warnCommand string
	if hostname != "" {
		errorCommand = fmt.Sprintf("ssh %s 'journalctl -u %s --no-pager | grep -i \"error\" | wc -l'", hostname, component)
		warnCommand = fmt.Sprintf("ssh %s 'journalctl -u %s --no-pager | grep -i \"warn\" | wc -l'", hostname, component)
	} else {
		errorCommand = fmt.Sprintf("journalctl -u %s --no-pager | grep -i \"error\" | wc -l", component)
		warnCommand = fmt.Sprintf("journalctl -u %s --no-pager | grep -i \"warn\" | wc -l", component)
	}

	errorCount, err1 := executeCommand(errorCommand)
	warnCount, err2 := executeCommand(warnCommand)

	if err1 == nil && err2 == nil {
		result.WriteString("\n日志统计:\n")
		result.WriteString(fmt.Sprintf("  错误 (Error) 数量: %s", errorCount))
		result.WriteString(fmt.Sprintf("  警告 (Warning) 数量: %s", warnCount))
	}

	return mcp.NewToolResultText(result.String()), nil
}

// ContainerInspectTool 检查容器详情的工具函数
func ContainerInspectTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hostname, _ := request.Params.Arguments["hostname"].(string)
	containerID, _ := request.Params.Arguments["container_id"].(string)
	runtime, _ := request.Params.Arguments["runtime"].(string)
	if runtime == "" {
		runtime = "docker" // 默认为docker
	}

	fmt.Println("ai 正在调用mcp server的tool: container_inspect, hostname=", hostname, ", container_id=", containerID)

	// 构建命令
	var commands []string

	switch strings.ToLower(runtime) {
	case "docker":
		if hostname != "" {
			commands = []string{
				fmt.Sprintf("ssh %s 'docker inspect %s'", hostname, containerID),
				fmt.Sprintf("ssh %s 'docker logs --tail 50 %s'", hostname, containerID),
			}
		} else {
			commands = []string{
				fmt.Sprintf("docker inspect %s", containerID),
				fmt.Sprintf("docker logs --tail 50 %s", containerID),
			}
		}
	case "containerd":
		if hostname != "" {
			commands = []string{
				fmt.Sprintf("ssh %s 'crictl inspect %s'", hostname, containerID),
				fmt.Sprintf("ssh %s 'crictl logs %s --tail 50'", hostname, containerID),
			}
		} else {
			commands = []string{
				fmt.Sprintf("crictl inspect %s", containerID),
				fmt.Sprintf("crictl logs %s --tail 50", containerID),
			}
		}
	case "crio", "cri-o":
		if hostname != "" {
			commands = []string{
				fmt.Sprintf("ssh %s 'crictl inspect %s'", hostname, containerID),
				fmt.Sprintf("ssh %s 'crictl logs %s --tail 50'", hostname, containerID),
			}
		} else {
			commands = []string{
				fmt.Sprintf("crictl inspect %s", containerID),
				fmt.Sprintf("crictl logs %s --tail 50", containerID),
			}
		}
	default:
		if hostname != "" {
			commands = []string{
				fmt.Sprintf("ssh %s 'docker inspect %s'", hostname, containerID),
				fmt.Sprintf("ssh %s 'docker logs --tail 50 %s'", hostname, containerID),
			}
		} else {
			commands = []string{
				fmt.Sprintf("docker inspect %s", containerID),
				fmt.Sprintf("docker logs --tail 50 %s", containerID),
			}
		}
	}

	// 执行命令并收集输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("容器 '%s' 详情:\n\n", containerID))

	for _, cmd := range commands {
		output, err := executeCommand(cmd)
		if err != nil {
			result.WriteString(fmt.Sprintf("执行命令 '%s' 失败: %v\n", cmd, err))
			continue
		}
		result.WriteString(output)
		result.WriteString("\n\n")
	}

	return mcp.NewToolResultText(result.String()), nil
}
