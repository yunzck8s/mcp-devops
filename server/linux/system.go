package linux

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"golang.org/x/crypto/ssh"
)

// sshClient is a reusable SSH client for command execution
var sshClient *ssh.Client
var sshConfig *ssh.ClientConfig

// initSSHConfig initializes SSH configuration for passwordless authentication
func initSSHConfig() error {
	// Use environment variable for SSH key path or default to ~/.ssh/id_rsa
	sshKeyPath := os.Getenv("SSH_KEY_PATH")
	if sshKeyPath == "" {
		sshKeyPath = os.Getenv("HOME") + "/.ssh/id_rsa"
	}

	// Read the private key
	key, err := os.ReadFile(sshKeyPath)
	if err != nil {
		return fmt.Errorf("无法读取SSH私钥文件 %s: %v", sshKeyPath, err)
	}

	// Create the signer for this private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return fmt.Errorf("解析SSH私钥失败: %v", err)
	}

	// Configure SSH client with passwordless auth
	sshConfig = &ssh.ClientConfig{
		User: os.Getenv("SSH_USER"),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, use a proper host key verification
		Timeout:         5 * time.Second,
	}

	if sshConfig.User == "" {
		sshConfig.User = os.Getenv("USER") // Fallback to current user
	}

	return nil
}

// testSSHConnection tests if SSH passwordless connection is working for a given host
func testSSHConnection(hostname string) error {
	if sshConfig == nil {
		if err := initSSHConfig(); err != nil {
			return err
		}
	}

	// Format host with port (default to 22 if not specified)
	hostWithPort := hostname
	if !strings.Contains(hostname, ":") {
		hostWithPort = hostname + ":22"
	}

	// Attempt to establish SSH connection
	client, err := ssh.Dial("tcp", hostWithPort, sshConfig)
	if err != nil {
		return fmt.Errorf("SSH连接测试失败，请确保已配置免密认证: %v", err)
	}
	defer client.Close()

	// Connection successful
	return nil
}

// executeSSHCommand executes a command via SSH on a remote host
func executeSSHCommand(hostname, command string) (string, error) {
	if sshConfig == nil {
		if err := initSSHConfig(); err != nil {
			return "", err
		}
	}

	// Format host with port (default to 22 if not specified)
	hostWithPort := hostname
	if !strings.Contains(hostname, ":") {
		hostWithPort = hostname + ":22"
	}

	// Establish SSH connection
	client, err := ssh.Dial("tcp", hostWithPort, sshConfig)
	if err != nil {
		return "", fmt.Errorf("无法连接到 %s: %v", hostname, err)
	}
	defer client.Close()

	// Create a session
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建SSH会话失败: %v", err)
	}
	defer session.Close()

	// Execute the command
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(command)
	if err != nil {
		return stderr.String(), fmt.Errorf("执行命令失败: %v\n错误输出: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// executeLocalCommand executes a command locally on the server
func executeLocalCommand(command string) (string, error) {
	// Split the command into parts for exec.Command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("命令为空")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return stderr.String(), fmt.Errorf("执行本地命令失败: %v\n错误输出: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// executeCommand 辅助函数：执行命令
func executeCommand(command string) (string, error) {
	// Check if the command involves SSH to a remote host
	if strings.HasPrefix(command, "ssh ") {
		// Extract hostname from the command
		parts := strings.Fields(command)
		if len(parts) < 2 {
			return "", fmt.Errorf("SSH命令格式错误，无法解析主机名")
		}
		hostname := parts[1]

		// Test SSH connection first to ensure passwordless auth is set up
		err := testSSHConnection(hostname)
		if err != nil {
			return "", fmt.Errorf("SSH免密认证未配置或不可用: %v\n请确保已配置SSH免密登录到 %s", err, hostname)
		}

		// Extract the actual command to run on the remote host
		sshCmdIndex := strings.Index(command, "'")
		if sshCmdIndex == -1 {
			return "", fmt.Errorf("无法解析SSH命令内容")
		}
		remoteCommand := command[sshCmdIndex+1 : len(command)-1]

		// Execute the command via SSH
		output, err := executeSSHCommand(hostname, remoteCommand)
		if err != nil {
			return "", fmt.Errorf("SSH命令执行失败: %v", err)
		}
		return output, nil
	}

	// Local command execution
	output, err := executeLocalCommand(command)
	if err != nil {
		return "", fmt.Errorf("本地命令执行失败: %v", err)
	}
	return output, nil
}

// SystemInfoTool 获取系统信息的工具函数
func SystemInfoTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hostname, _ := request.Params.Arguments["hostname"].(string)

	fmt.Println("ai 正在调用mcp server的tool: system_info, hostname=", hostname)

	// 构建命令以获取系统信息
	var unameCmd, osReleaseCmd, uptimeCmd, memCmd, diskCmd string
	if hostname != "" {
		unameCmd = fmt.Sprintf("ssh %s 'uname -a'", hostname)
		osReleaseCmd = fmt.Sprintf("ssh %s 'cat /etc/os-release'", hostname)
		uptimeCmd = fmt.Sprintf("ssh %s 'uptime'", hostname)
		memCmd = fmt.Sprintf("ssh %s 'free -h'", hostname)
		diskCmd = fmt.Sprintf("ssh %s 'df -h'", hostname)
	} else {
		unameCmd = "uname -a"
		osReleaseCmd = "cat /etc/os-release"
		uptimeCmd = "uptime"
		memCmd = "free -h"
		diskCmd = "df -h"
	}

	// 执行命令并收集输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("<b>%s节点系统信息：</b><br><br>", hostname))

	// 获取内核信息
	unameOutput, err := executeCommand(unameCmd)
	if err != nil {
		result.WriteString(fmt.Sprintf("获取内核信息失败: %v<br>", err))
	} else {
		result.WriteString("<b>内核信息：</b><br>")
		result.WriteString(unameOutput)
		result.WriteString("<br><br>")
	}

	// 获取操作系统信息
	osReleaseOutput, err := executeCommand(osReleaseCmd)
	if err != nil {
		result.WriteString(fmt.Sprintf("获取操作系统信息失败: %v<br>", err))
	} else {
		result.WriteString("<b>操作系统信息：</b><br>")
		lines := strings.Split(osReleaseOutput, "\n")
		for _, line := range lines {
			if strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.Trim(parts[0], "\"")
					value := strings.Trim(parts[1], "\"")
					if key == "NAME" || key == "VERSION" || key == "PRETTY_NAME" {
						result.WriteString(fmt.Sprintf("%s: %s<br>", key, value))
					}
				}
			}
		}
		result.WriteString("<br>")
	}

	// 获取运行时间
	uptimeOutput, err := executeCommand(uptimeCmd)
	if err != nil {
		result.WriteString(fmt.Sprintf("获取运行时间失败: %v<br>", err))
	} else {
		result.WriteString("<b>运行时间：</b><br>")
		result.WriteString(uptimeOutput)
		result.WriteString("<br><br>")
	}

	// 获取内存使用情况
	memOutput, err := executeCommand(memCmd)
	if err != nil {
		result.WriteString(fmt.Sprintf("获取内存信息失败: %v<br>", err))
	} else {
		result.WriteString("<b>内存使用情况：</b><br>")
		lines := strings.Split(memOutput, "\n")
		for _, line := range lines {
			result.WriteString(line)
			result.WriteString("<br>")
		}
		result.WriteString("<br>")
	}

	// 获取磁盘使用情况
	diskOutput, err := executeCommand(diskCmd)
	if err != nil {
		result.WriteString(fmt.Sprintf("获取磁盘信息失败: %v<br>", err))
	} else {
		result.WriteString("<b>磁盘使用情况：</b><br>")
		lines := strings.Split(diskOutput, "\n")
		for _, line := range lines {
			result.WriteString(line)
			result.WriteString("<br>")
		}
		result.WriteString("<br>")
	}

	// 添加备注以确保内容不被截断
	result.WriteString("<i>注：以上为完整系统信息，未经截断。</i>")

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
