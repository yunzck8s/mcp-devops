package k8s

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"strings"
	"time"
)

// 列出Pod的工具函数
func ListPodsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: list_pods, namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取Pod列表
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Pod列表失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("命名空间: %s\n\n", namespace))
	result.WriteString("NAME\tREADY\tSTATUS\tRESTARTS\tAGE\tIP\tNODE\n")

	for _, pod := range pods.Items {
		// 计算容器就绪数
		var readyContainers int
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Ready {
				readyContainers++
			}
		}

		// 计算重启次数
		var restarts int32
		for _, containerStatus := range pod.Status.ContainerStatuses {
			restarts += containerStatus.RestartCount
		}

		// 计算运行时间
		age := formatAge(pod.CreationTimestamp.Time)

		result.WriteString(fmt.Sprintf("%s\t%d/%d\t%s\t%d\t%s\t%s\t%s\n",
			pod.Name,
			readyContainers,
			len(pod.Spec.Containers),
			string(pod.Status.Phase),
			restarts,
			age,
			pod.Status.PodIP,
			pod.Spec.NodeName))
	}

	return mcp.NewToolResultText(result.String()), nil
}

func DsscribePodTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	podName := request.Params.Arguments["pod_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: describe_pod, pod_name=", podName, ", namespace=", namespace)

	// 创建kubernetes客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获得pod的详情
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Pod详情失败: %v", err)), err
	}

	//格式化输出
	var result *strings.Builder
	result.WriteString(fmt.Sprintf("Name:				%s\n", pod.Name))
	result.WriteString(fmt.Sprintf("Namespace:			%s\n", pod.Namespace))
	result.WriteString(fmt.Sprintf("Priority:			%s\n", getPodPriority(pod)))
	result.WriteString(fmt.Sprintf("Node:         		%s\n", pod.Spec.NodeName))
	result.WriteString(fmt.Sprintf("Start Time:   		%s\n", pod.CreationTimestamp.Format(time.RFC3339)))
	result.WriteString(fmt.Sprintf("Labels:       		%s\n", formatLabels(pod.Labels)))
	result.WriteString(fmt.Sprintf("Annotations:  		%s\n", formatLabels(pod.Annotations)))
	result.WriteString(fmt.Sprintf("Status:       		%s\n", string(pod.Status.Phase)))
	result.WriteString(fmt.Sprintf("IP:           		%s\n", pod.Status.PodIP))
	result.WriteString(fmt.Sprintf("IPs:          		%s\n", formatPodIPs(pod.Status.PodIPs)))

	//容器详情
	result.WriteString("\nContainers:\n")
	for _, container := range pod.Spec.Containers {
		result.WriteString(fmt.Sprintf("  %s:\n", container.Name))
		result.WriteString(fmt.Sprintf("    Image: %s\n", container.Image))
		result.WriteString(fmt.Sprintf("    Ports:       %s\n", formatContainerPorts(container.Ports)))
		result.WriteString(fmt.Sprintf("    Host Ports:  %s\n", formatContainerHostPorts(container.Ports)))
		result.WriteString(fmt.Sprintf("    Command:     %s\n", strings.Join(container.Command, " ")))
		result.WriteString(fmt.Sprintf("    Args:        %s\n", strings.Join(container.Args, " ")))
		result.WriteString(fmt.Sprintf("    Environment: %s\n", formatEnvVars(container.Env)))
		result.WriteString(fmt.Sprintf("    Mounts:      %s\n", formatVolumeMounts(container.VolumeMounts)))

		// 容器状态
		var containerStatus *corev1.ContainerStatus
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == container.Name {
				containerStatus = &status
				break
			}
		}

		if containerStatus != nil {
			result.WriteString(fmt.Sprintf("    State:       %s\n", formatContainerState(containerStatus.State)))
			result.WriteString(fmt.Sprintf("    Ready:       %t\n", containerStatus.Ready))
			result.WriteString(fmt.Sprintf("    Restarts:    %d\n", containerStatus.RestartCount))
			result.WriteString(fmt.Sprintf("    Image ID:    %s\n", containerStatus.ImageID))
		}

		// 卷详情
		if len(pod.Spec.Volumes) > 0 {
			result.WriteString("\nVolumes:\n")
			for _, volume := range pod.Spec.Volumes {
				result.WriteString(fmt.Sprintf("  %s:\n", volume.Name))
				if volume.PersistentVolumeClaim != nil {
					result.WriteString(fmt.Sprintf("    Type:        PersistentVolumeClaim\n"))
					result.WriteString(fmt.Sprintf("    ClaimName:   %s\n", volume.PersistentVolumeClaim.ClaimName))
				} else if volume.ConfigMap != nil {
					result.WriteString(fmt.Sprintf("    Type:        ConfigMap\n"))
					result.WriteString(fmt.Sprintf("    Name:        %s\n", volume.ConfigMap.Name))
				} else if volume.Secret != nil {
					result.WriteString(fmt.Sprintf("    Type:        Secret\n"))
					result.WriteString(fmt.Sprintf("    SecretName:  %s\n", volume.Secret.SecretName))
				} else if volume.EmptyDir != nil {
					result.WriteString(fmt.Sprintf("    Type:        EmptyDir\n"))
				} else if volume.HostPath != nil {
					result.WriteString(fmt.Sprintf("    Type:        HostPath\n"))
					result.WriteString(fmt.Sprintf("    Path:        %s\n", volume.HostPath.Path))
				} else {
					result.WriteString(fmt.Sprintf("    Type:        Other\n"))
				}
			}
		}

	}
	// 事件
	events, err := getEventsForPod(ctx, clientset, pod)
	if err == nil && len(events.Items) > 0 {
		result.WriteString("\nEvents:\n")
		result.WriteString("LAST SEEN\tTYPE\tREASON\tOBJECT\tMESSAGE\n")
		for _, event := range events.Items {
			age := formatAge(event.LastTimestamp.Time)
			result.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n",
				age,
				event.Type,
				event.Reason,
				event.InvolvedObject.Kind+"/"+event.InvolvedObject.Name,
				event.Message))
		}
	}

	return mcp.NewToolResultText(result.String()), nil
}

// 删除pod的工具函数
func DeletePodTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	podName := request.Params.Arguments["pod_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}
	force, _ := request.Params.Arguments["force"].(bool)

	fmt.Println("ai 正在调用mcp server的tool: delete_pod, pod_name=", podName, ", namespace=", namespace, ", force=", force)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}
	// 设置删除选项
	deleteOptions := metav1.DeleteOptions{}
	if force {
		gracePeriod := int64(0)
		deleteOptions.GracePeriodSeconds = &gracePeriod
	}
	err = clientset.CoreV1().Pods(namespace).Delete(ctx, podName, deleteOptions)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("删除Pod失败: %v", err)), err
	}
	return mcp.NewToolResultText(fmt.Sprintf("Pod %s 在命名空间 %s 中已成功删除", podName, namespace)), nil
}

// 获取Pod日志的工具函数
func PodLogsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	podName := request.Params.Arguments["pod_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}
	container, _ := request.Params.Arguments["container"].(string)
	tail, _ := request.Params.Arguments["tail"].(float64)
	if tail == 0 {
		tail = 100 // 默认值
	}

	fmt.Println("ai 正在调用mcp server的tool: pod_logs, pod_name=", podName, ", namespace=", namespace, ", container=", container)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 设置日志选项
	tailLines := int64(tail)
	podLogOptions := corev1.PodLogOptions{
		TailLines: &tailLines,
	}
	if container != "" {
		podLogOptions.Container = container
	}

	// 获取Pod日志
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOptions)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Pod日志失败: %v", err)), err
	}
	defer podLogs.Close()

	// 读取日志
	buf := new(strings.Builder)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("读取Pod日志失败: %v", err)), err
	}

	return mcp.NewToolResultText(buf.String()), nil
}

// 辅助函数：格式化Pod优先级
func getPodPriority(pod *corev1.Pod) int32 {
	if pod.Spec.Priority != nil {
		return *pod.Spec.Priority
	}
	return 0
}

// 辅助函数：格式化标签
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return "<none>"
	}

	var parts []string
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ", ")
}

// 辅助函数：格式化PodIP列表
func formatPodIPs(podIPs []corev1.PodIP) string {
	if len(podIPs) == 0 {
		return "<none>"
	}

	var ips []string
	for _, ip := range podIPs {
		ips = append(ips, ip.IP)
	}
	return strings.Join(ips, ", ")
}

// 辅助函数：格式化容器端口
func formatContainerPorts(ports []corev1.ContainerPort) string {
	if len(ports) == 0 {
		return "<none>"
	}

	var result []string
	for _, port := range ports {
		result = append(result, fmt.Sprintf("%d/%s", port.ContainerPort, port.Protocol))
	}
	return strings.Join(result, ", ")
}

// 辅助函数：格式化容器主机端口
func formatContainerHostPorts(ports []corev1.ContainerPort) string {
	if len(ports) == 0 {
		return "<none>"
	}

	var result []string
	for _, port := range ports {
		if port.HostPort > 0 {
			result = append(result, fmt.Sprintf("%d/%s", port.HostPort, port.Protocol))
		}
	}

	if len(result) == 0 {
		return "<none>"
	}
	return strings.Join(result, ", ")
}

// 辅助函数：格式化环境变量
func formatEnvVars(envVars []corev1.EnvVar) string {
	if len(envVars) == 0 {
		return "<none>"
	}

	var result []string
	for _, env := range envVars {
		if env.ValueFrom != nil {
			if env.ValueFrom.ConfigMapKeyRef != nil {
				result = append(result, fmt.Sprintf("%s=<from ConfigMap %s>", env.Name, env.ValueFrom.ConfigMapKeyRef.Name))
			} else if env.ValueFrom.SecretKeyRef != nil {
				result = append(result, fmt.Sprintf("%s=<from Secret %s>", env.Name, env.ValueFrom.SecretKeyRef.Name))
			} else {
				result = append(result, fmt.Sprintf("%s=<from other source>", env.Name))
			}
		} else {
			result = append(result, fmt.Sprintf("%s=%s", env.Name, env.Value))
		}
	}
	return strings.Join(result, ", ")
}

// 辅助函数：格式化卷挂载
func formatVolumeMounts(mounts []corev1.VolumeMount) string {
	if len(mounts) == 0 {
		return "<none>"
	}

	var result []string
	for _, mount := range mounts {
		readOnly := ""
		if mount.ReadOnly {
			readOnly = " (ro)"
		}
		result = append(result, fmt.Sprintf("%s:%s%s", mount.Name, mount.MountPath, readOnly))
	}
	return strings.Join(result, ", ")
}

// 辅助函数：格式化容器状态
func formatContainerState(state corev1.ContainerState) string {
	if state.Running != nil {
		return fmt.Sprintf("Running since %s", state.Running.StartedAt.Format(time.RFC3339))
	}
	if state.Waiting != nil {
		return fmt.Sprintf("Waiting: %s", state.Waiting.Reason)
	}
	if state.Terminated != nil {
		return fmt.Sprintf("Terminated: %s (exit code: %d)", state.Terminated.Reason, state.Terminated.ExitCode)
	}
	return "Unknown"
}

// 辅助函数：获取Pod相关事件
func getEventsForPod(ctx context.Context, clientset *kubernetes.Clientset, pod *corev1.Pod) (*corev1.EventList, error) {
	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.namespace=%s,involvedObject.kind=Pod",
		pod.Name, pod.Namespace)

	return clientset.CoreV1().Events(pod.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
}

// 辅助函数：格式化年龄
func formatAge(t time.Time) string {
	duration := time.Since(t)

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
