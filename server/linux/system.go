package linux

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// SystemInfoTool 获取系统信息的工具函数
func SystemInfoTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hostname, _ := request.Params.Arguments["hostname"].(string)
	
	fmt.Println("ai 正在调用mcp server的tool: system_info, hostname=", hostname)

	// 构建命令
	var command string
	if hostname != "" {
		command = fmt.Sprintf("ssh %s 'uname -a && cat /etc/os-release && uptime && free -h && df -h'", hostname)
	} else {
		command = "uname -a && cat /etc/os-release && uptime && free -h && df -h"
	}

	// 执行命令
	output, err := executeCommand(command)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取系统信息失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString("系统信息:\n\n")
	result.WriteString(output)

	return mcp.NewToolResultText(result.String()), nil
}

// ProcessInfoTool 获取进程信息的工具函数
func ProcessInfoTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hostname, _ := request.Params.Arguments["hostname"].(string)
	processName, _ := request.Params.Arguments["process_name"].(string)
	topCount, _ := request.Params.Arguments["top_count"].(float64)
	if topCount == 0 {
		topCount = 10 // 默认显示前10个进程
	}

	fmt.Println("ai 正在调用mcp server的tool: process_info, hostname=", hostname, ", process_name=", processName)

	// 构建命令
	var command string
	if hostname != "" {
		if processName != "" {
			command = fmt.Sprintf("ssh %s 'ps aux | grep %s | grep -v grep'", hostname, processName)
		} else {
			command = fmt.Sprintf("ssh %s 'ps aux --sort=-%cpu | head -%d'", hostname, int(topCount))
		}
	} else {
		if processName != "" {
			command = fmt.Sprintf("ps aux | grep %s | grep -v grep", processName)
		} else {
			command = fmt.Sprintf("ps aux --sort=-%cpu | head -%d", int(topCount))
		}
	}

	// 执行命令
	output, err := executeCommand(command)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取进程信息失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	if processName != "" {
		result.WriteString(fmt.Sprintf("进程 '%s' 信息:\n\n", processName))
	} else {
		result.WriteString(fmt.Sprintf("CPU使用率前 %d 的进程:\n\n", int(topCount)))
	}
	result.WriteString(output)

	return mcp.NewToolResultText(result.String()), nil
}

// ResourceUsageTool 获取资源使用情况的工具函数
func ResourceUsageTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hostname, _ := request.Params.Arguments["hostname"].(string)
	duration, _ := request.Params.Arguments["duration"].(float64)
	if duration == 0 {
		duration = 5 // 默认监控5秒
	}

	fmt.Println("ai 正在调用mcp server的tool: resource_usage, hostname=", hostname)

	// 构建命令
	var command string
	if hostname != "" {
		command = fmt.Sprintf("ssh %s 'top -b -n 2 -d %.1f | grep -A 15 \"%%Cpu\" | tail -n 15'", hostname, duration)
	} else {
		command = fmt.Sprintf("top -b -n 2 -d %.1f | grep -A 15 \"%%Cpu\" | tail -n 15", duration)
	}

	// 执行命令
	output, err := executeCommand(command)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取资源使用情况失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString("资源使用情况:\n\n")
	result.WriteString(output)

	// 添加内存使用情况
	var memCommand string
	if hostname != "" {
		memCommand = fmt.Sprintf("ssh %s 'free -h'", hostname)
	} else {
		memCommand = "free -h"
	}
	
	memOutput, err := executeCommand(memCommand)
	if err == nil {
		result.WriteString("\n内存使用情况:\n\n")
		result.WriteString(memOutput)
	}

	// 添加磁盘使用情况
	var diskCommand string
	if hostname != "" {
		diskCommand = fmt.Sprintf("ssh %s 'df -h'", hostname)
	} else {
		diskCommand = "df -h"
	}
	
	diskOutput, err := executeCommand(diskCommand)
	if err == nil {
		result.WriteString("\n磁盘使用情况:\n\n")
		result.WriteString(diskOutput)
	}

	return mcp.NewToolResultText(result.String()), nil
}

// NetworkInfoTool 获取网络信息的工具函数
func NetworkInfoTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hostname, _ := request.Params.Arguments["hostname"].(string)
	interface_, _ := request.Params.Arguments["interface"].(string)

	fmt.Println("ai 正在调用mcp server的tool: network_info, hostname=", hostname, ", interface=", interface_)

	// 构建命令
	var commands []string
	if hostname != "" {
		if interface_ != "" {
			commands = []string{
				fmt.Sprintf("ssh %s 'ip addr show %s'", hostname, interface_),
				fmt.Sprintf("ssh %s 'ip -s link show %s'", hostname, interface_),
			}
		} else {
			commands = []string{
				fmt.Sprintf("ssh %s 'ip addr'", hostname),
				fmt.Sprintf("ssh %s 'netstat -tuln'", hostname),
				fmt.Sprintf("ssh %s 'ip route'", hostname),
			}
		}
	} else {
		if interface_ != "" {
			commands = []string{
				fmt.Sprintf("ip addr show %s", interface_),
				fmt.Sprintf("ip -s link show %s", interface_),
			}
		} else {
			commands = []string{
				"ip addr",
				"netstat -tuln",
				"ip route",
			}
		}
	}

	// 执行命令并收集输出
	var result strings.Builder
	if interface_ != "" {
		result.WriteString(fmt.Sprintf("网络接口 '%s' 信息:\n\n", interface_))
	} else {
		result.WriteString("网络信息:\n\n")
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

// LogAnalysisTool 分析日志文件的工具函数
func LogAnalysisTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hostname, _ := request.Params.Arguments["hostname"].(string)
	logPath, _ := request.Params.Arguments["log_path"].(string)
	if logPath == "" {
		logPath = "/var/log/syslog" // 默认日志文件
	}
	
	pattern, _ := request.Params.Arguments["pattern"].(string)
	lines, _ := request.Params.Arguments["lines"].(float64)
	if lines == 0 {
		lines = 50 // 默认显示50行
	}

	fmt.Println("ai 正在调用mcp server的tool: log_analysis, hostname=", hostname, ", log_path=", logPath)

	// 构建命令
	var command string
	if hostname != "" {
		if pattern != "" {
			command = fmt.Sprintf("ssh %s 'grep \"%s\" %s | tail -%d'", hostname, pattern, logPath, int(lines))
		} else {
			command = fmt.Sprintf("ssh %s 'tail -%d %s'", hostname, int(lines), logPath)
		}
	} else {
		if pattern != "" {
			command = fmt.Sprintf("grep \"%s\" %s | tail -%d", pattern, logPath, int(lines))
		} else {
			command = fmt.Sprintf("tail -%d %s", int(lines), logPath)
		}
	}

	// 执行命令
	output, err := executeCommand(command)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("分析日志失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	if pattern != "" {
		result.WriteString(fmt.Sprintf("日志文件 '%s' 中包含 '%s' 的最后 %d 行:\n\n", logPath, pattern, int(lines)))
	} else {
		result.WriteString(fmt.Sprintf("日志文件 '%s' 的最后 %d 行:\n\n", logPath, int(lines)))
	}
	result.WriteString(output)

	// 添加错误和警告统计
	if pattern == "" {
		var errorCommand, warnCommand string
		if hostname != "" {
			errorCommand = fmt.Sprintf("ssh %s 'grep -i \"error\" %s | wc -l'", hostname, logPath)
			warnCommand = fmt.Sprintf("ssh %s 'grep -i \"warn\" %s | wc -l'", hostname, logPath)
		} else {
			errorCommand = fmt.Sprintf("grep -i \"error\" %s | wc -l", logPath)
			warnCommand = fmt.Sprintf("grep -i \"warn\" %s | wc -l", logPath)
		}
		
		errorCount, err1 := executeCommand(errorCommand)
		warnCount, err2 := executeCommand(warnCommand)
		
		if err1 == nil && err2 == nil {
			result.WriteString("\n日志统计:\n")
			result.WriteString(fmt.Sprintf("  错误 (Error) 数量: %s", errorCount))
			result.WriteString(fmt.Sprintf("  警告 (Warning) 数量: %s", warnCount))
		}
	}

	return mcp.NewToolResultText(result.String()), nil
}

// ServiceStatusTool 获取服务状态的工具函数
func ServiceStatusTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hostname, _ := request.Params.Arguments["hostname"].(string)
	serviceName, _ := request.Params.Arguments["service_name"].(string)

	fmt.Println("ai 正在调用mcp server的tool: service_status, hostname=", hostname, ", service_name=", serviceName)

	// 构建命令
	var command string
	if hostname != "" {
		if serviceName != "" {
			command = fmt.Sprintf("ssh %s 'systemctl status %s'", hostname, serviceName)
		} else {
			command = fmt.Sprintf("ssh %s 'systemctl list-units --type=service --state=running'", hostname)
		}
	} else {
		if serviceName != "" {
			command = fmt.Sprintf("systemctl status %s", serviceName)
		} else {
			command = "systemctl list-units --type=service --state=running"
		}
	}

	// 执行命令
	output, err := executeCommand(command)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取服务状态失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	if serviceName != "" {
		result.WriteString(fmt.Sprintf("服务 '%s' 状态:\n\n", serviceName))
	} else {
		result.WriteString("运行中的服务列表:\n\n")
	}
	result.WriteString(output)

	return mcp.NewToolResultText(result.String()), nil
}

// 辅助函数：执行命令
func executeCommand(command string) (string, error) {
	// 这里应该实现实际的命令执行逻辑
	// 在实际实现中，可能需要使用os/exec包来执行命令
	// 但由于这是一个示例，我们只返回一个模拟的输出
	
	// 模拟命令执行延迟
	time.Sleep(500 * time.Millisecond)
	
	// 返回模拟输出
	return fmt.Sprintf("执行命令: %s\n\n[命令输出将在实际环境中显示]", command), nil
}
