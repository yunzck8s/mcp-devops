package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metrics "k8s.io/metrics-server/pkg/client/clientset/versioned"
)

// ClusterHealthTool 获取集群健康状态的工具函数
func ClusterHealthTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	fmt.Println("ai 正在调用mcp server的tool: cluster_health")

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取节点列表
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取节点列表失败: %v", err)), err
	}

	// 获取命名空间列表
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取命名空间列表失败: %v", err)), err
	}

	// 获取所有Pod
	allPods := make([]corev1.Pod, 0)
	for _, ns := range namespaces.Items {
		pods, err := clientset.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}
		allPods = append(allPods, pods.Items...)
	}

	// 统计节点状态
	var readyNodes, notReadyNodes int
	for _, node := range nodes.Items {
		isReady := false
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				isReady = true
				break
			}
		}
		if isReady {
			readyNodes++
		} else {
			notReadyNodes++
		}
	}

	// 统计Pod状态
	var runningPods, pendingPods, failedPods, unknownPods, otherPods int
	for _, pod := range allPods {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			runningPods++
		case corev1.PodPending:
			pendingPods++
		case corev1.PodFailed:
			failedPods++
		case corev1.PodUnknown:
			unknownPods++
		default:
			otherPods++
		}
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString("集群健康状态概览:\n\n")
	
	// 节点状态
	result.WriteString(fmt.Sprintf("节点状态 (%d 总计):\n", len(nodes.Items)))
	result.WriteString(fmt.Sprintf("  就绪: %d\n", readyNodes))
	result.WriteString(fmt.Sprintf("  未就绪: %d\n", notReadyNodes))
	
	// Pod状态
	result.WriteString(fmt.Sprintf("\nPod状态 (%d 总计):\n", len(allPods)))
	result.WriteString(fmt.Sprintf("  运行中: %d\n", runningPods))
	result.WriteString(fmt.Sprintf("  等待中: %d\n", pendingPods))
	result.WriteString(fmt.Sprintf("  失败: %d\n", failedPods))
	result.WriteString(fmt.Sprintf("  未知: %d\n", unknownPods))
	result.WriteString(fmt.Sprintf("  其他: %d\n", otherPods))
	
	// 命名空间状态
	result.WriteString(fmt.Sprintf("\n命名空间 (%d 总计):\n", len(namespaces.Items)))
	for _, ns := range namespaces.Items {
		result.WriteString(fmt.Sprintf("  %s\n", ns.Name))
	}

	return mcp.NewToolResultText(result.String()), nil
}

// PodDiagnosticTool 诊断Pod问题的工具函数
func PodDiagnosticTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	podName := request.Params.Arguments["pod_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: pod_diagnostic, pod_name=", podName, ", namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取Pod详情
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Pod详情失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Pod %s 诊断报告:\n\n", podName))
	
	// 基本信息
	result.WriteString("基本信息:\n")
	result.WriteString(fmt.Sprintf("  名称: %s\n", pod.Name))
	result.WriteString(fmt.Sprintf("  命名空间: %s\n", pod.Namespace))
	result.WriteString(fmt.Sprintf("  状态: %s\n", string(pod.Status.Phase)))
	result.WriteString(fmt.Sprintf("  节点: %s\n", pod.Spec.NodeName))
	result.WriteString(fmt.Sprintf("  IP: %s\n", pod.Status.PodIP))
	
	// 容器状态
	result.WriteString("\n容器状态:\n")
	for _, containerStatus := range pod.Status.ContainerStatuses {
		result.WriteString(fmt.Sprintf("  容器: %s\n", containerStatus.Name))
		result.WriteString(fmt.Sprintf("    就绪: %t\n", containerStatus.Ready))
		result.WriteString(fmt.Sprintf("    重启次数: %d\n", containerStatus.RestartCount))
		
		// 容器当前状态
		if containerStatus.State.Running != nil {
			result.WriteString(fmt.Sprintf("    状态: Running (开始于 %s)\n", containerStatus.State.Running.StartedAt.Format(time.RFC3339)))
		} else if containerStatus.State.Waiting != nil {
			result.WriteString(fmt.Sprintf("    状态: Waiting (原因: %s, 消息: %s)\n", 
				containerStatus.State.Waiting.Reason, 
				containerStatus.State.Waiting.Message))
		} else if containerStatus.State.Terminated != nil {
			result.WriteString(fmt.Sprintf("    状态: Terminated (原因: %s, 退出码: %d, 消息: %s)\n", 
				containerStatus.State.Terminated.Reason, 
				containerStatus.State.Terminated.ExitCode,
				containerStatus.State.Terminated.Message))
		}
		
		// 上次终止状态
		if containerStatus.LastTerminationState.Terminated != nil {
			result.WriteString(fmt.Sprintf("    上次终止: (原因: %s, 退出码: %d, 消息: %s)\n", 
				containerStatus.LastTerminationState.Terminated.Reason, 
				containerStatus.LastTerminationState.Terminated.ExitCode,
				containerStatus.LastTerminationState.Terminated.Message))
		}
	}
	
	// 初始化容器状态
	if len(pod.Status.InitContainerStatuses) > 0 {
		result.WriteString("\n初始化容器状态:\n")
		for _, initStatus := range pod.Status.InitContainerStatuses {
			result.WriteString(fmt.Sprintf("  容器: %s\n", initStatus.Name))
			result.WriteString(fmt.Sprintf("    就绪: %t\n", initStatus.Ready))
			result.WriteString(fmt.Sprintf("    重启次数: %d\n", initStatus.RestartCount))
			
			// 容器当前状态
			if initStatus.State.Running != nil {
				result.WriteString(fmt.Sprintf("    状态: Running (开始于 %s)\n", initStatus.State.Running.StartedAt.Format(time.RFC3339)))
			} else if initStatus.State.Waiting != nil {
				result.WriteString(fmt.Sprintf("    状态: Waiting (原因: %s, 消息: %s)\n", 
					initStatus.State.Waiting.Reason, 
					initStatus.State.Waiting.Message))
			} else if initStatus.State.Terminated != nil {
				result.WriteString(fmt.Sprintf("    状态: Terminated (原因: %s, 退出码: %d, 消息: %s)\n", 
					initStatus.State.Terminated.Reason, 
					initStatus.State.Terminated.ExitCode,
					initStatus.State.Terminated.Message))
			}
		}
	}
	
	// 条件
	if len(pod.Status.Conditions) > 0 {
		result.WriteString("\n条件:\n")
		for _, condition := range pod.Status.Conditions {
			result.WriteString(fmt.Sprintf("  %s: %s (原因: %s, 消息: %s)\n", 
				condition.Type, 
				condition.Status, 
				condition.Reason, 
				condition.Message))
		}
	}
	
	// 获取相关事件
	events, err := getEventsForPod(ctx, clientset, pod)
	if err == nil && len(events.Items) > 0 {
		result.WriteString("\n最近事件:\n")
		for _, event := range events.Items {
			result.WriteString(fmt.Sprintf("  %s [%s] %s: %s\n", 
				formatAge(event.LastTimestamp.Time),
				event.Type,
				event.Reason,
				event.Message))
		}
	}
	
	// 诊断建议
	result.WriteString("\n诊断建议:\n")
	
	// 检查Pod状态
	if pod.Status.Phase != corev1.PodRunning {
		result.WriteString(fmt.Sprintf("  • Pod状态为 %s, 不是 Running\n", pod.Status.Phase))
		
		if pod.Status.Phase == corev1.PodPending {
			// 检查是否有调度问题
			for _, condition := range pod.Status.Conditions {
				if condition.Type == corev1.PodScheduled && condition.Status != corev1.ConditionTrue {
					result.WriteString(fmt.Sprintf("    - Pod调度问题: %s\n", condition.Message))
				}
			}
		}
	}
	
	// 检查容器状态
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if !containerStatus.Ready {
			result.WriteString(fmt.Sprintf("  • 容器 %s 未就绪\n", containerStatus.Name))
		}
		
		if containerStatus.RestartCount > 0 {
			result.WriteString(fmt.Sprintf("  • 容器 %s 已重启 %d 次\n", containerStatus.Name, containerStatus.RestartCount))
		}
		
		if containerStatus.State.Waiting != nil {
			result.WriteString(fmt.Sprintf("  • 容器 %s 处于等待状态: %s\n", containerStatus.Name, containerStatus.State.Waiting.Reason))
			
			// 特定原因的建议
			switch containerStatus.State.Waiting.Reason {
			case "CrashLoopBackOff":
				result.WriteString("    - 容器反复崩溃，请检查应用日志和配置\n")
				result.WriteString("    - 建议使用 pod_logs 工具查看容器日志\n")
			case "ImagePullBackOff", "ErrImagePull":
				result.WriteString("    - 无法拉取容器镜像，请检查镜像名称和仓库访问权限\n")
			case "CreateContainerConfigError":
				result.WriteString("    - 创建容器配置错误，请检查Pod配置\n")
			}
		}
		
		if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode != 0 {
			result.WriteString(fmt.Sprintf("  • 容器 %s 异常终止，退出码: %d\n", 
				containerStatus.Name, 
				containerStatus.State.Terminated.ExitCode))
		}
	}
	
	// 检查初始化容器
	for _, initStatus := range pod.Status.InitContainerStatuses {
		if !initStatus.Ready && pod.Status.Phase == corev1.PodPending {
			result.WriteString(fmt.Sprintf("  • 初始化容器 %s 未完成，阻止了Pod启动\n", initStatus.Name))
		}
	}
	
	// 如果没有发现明显问题
	if pod.Status.Phase == corev1.PodRunning {
		allContainersReady := true
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if !containerStatus.Ready {
				allContainersReady = false
				break
			}
		}
		
		if allContainersReady {
			result.WriteString("  • Pod看起来运行正常，所有容器都已就绪\n")
		}
	}

	return mcp.NewToolResultText(result.String()), nil
}

// NodeDiagnosticTool 诊断节点问题的工具函数
func NodeDiagnosticTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	nodeName := request.Params.Arguments["node_name"].(string)

	fmt.Println("ai 正在调用mcp server的tool: node_diagnostic, node_name=", nodeName)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取节点详情
	node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取节点详情失败: %v", err)), err
	}

	// 获取节点上的Pod
	fieldSelector := fmt.Sprintf("spec.nodeName=%s", nodeName)
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取节点上的Pod失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("节点 %s 诊断报告:\n\n", nodeName))
	
	// 基本信息
	result.WriteString("基本信息:\n")
	result.WriteString(fmt.Sprintf("  名称: %s\n", node.Name))
	result.WriteString(fmt.Sprintf("  创建时间: %s\n", node.CreationTimestamp.Format(time.RFC3339)))
	
	// 标签
	result.WriteString("  标签:\n")
	for key, value := range node.Labels {
		result.WriteString(fmt.Sprintf("    %s: %s\n", key, value))
	}
	
	// 角色
	var roles []string
	for key, value := range node.Labels {
		if strings.HasPrefix(key, "node-role.kubernetes.io/") && value == "true" {
			role := strings.TrimPrefix(key, "node-role.kubernetes.io/")
			roles = append(roles, role)
		}
	}
	if len(roles) > 0 {
		result.WriteString(fmt.Sprintf("  角色: %s\n", strings.Join(roles, ", ")))
	} else {
		result.WriteString("  角色: <none>\n")
	}
	
	// 地址
	result.WriteString("  地址:\n")
	for _, address := range node.Status.Addresses {
		result.WriteString(fmt.Sprintf("    %s: %s\n", address.Type, address.Address))
	}
	
	// 容量和可分配资源
	result.WriteString("\n资源:\n")
	cpuCapacity := node.Status.Capacity.Cpu().String()
	memoryCapacity := node.Status.Capacity.Memory().String()
	podsCapacity := node.Status.Capacity.Pods().String()
	
	cpuAllocatable := node.Status.Allocatable.Cpu().String()
	memoryAllocatable := node.Status.Allocatable.Memory().String()
	podsAllocatable := node.Status.Allocatable.Pods().String()
	
	result.WriteString(fmt.Sprintf("  容量: cpu: %s, 内存: %s, pods: %s\n", cpuCapacity, memoryCapacity, podsCapacity))
	result.WriteString(fmt.Sprintf("  可分配: cpu: %s, 内存: %s, pods: %s\n", cpuAllocatable, memoryAllocatable, podsAllocatable))
	
	// 节点状态条件
	result.WriteString("\n状态条件:\n")
	for _, condition := range node.Status.Conditions {
		result.WriteString(fmt.Sprintf("  %s: %s (上次转换: %s)\n", 
			condition.Type, 
			condition.Status, 
			formatAge(condition.LastTransitionTime.Time)))
		if condition.Message != "" {
			result.WriteString(fmt.Sprintf("    原因: %s, 消息: %s\n", condition.Reason, condition.Message))
		}
	}
	
	// 节点上的Pod
	result.WriteString(fmt.Sprintf("\n节点上的Pod (%d 总计):\n", len(pods.Items)))
	
	// 按命名空间分组
	podsByNamespace := make(map[string][]corev1.Pod)
	for _, pod := range pods.Items {
		podsByNamespace[pod.Namespace] = append(podsByNamespace[pod.Namespace], pod)
	}
	
	for ns, nsPods := range podsByNamespace {
		result.WriteString(fmt.Sprintf("  命名空间: %s (%d pods)\n", ns, len(nsPods)))
		for _, pod := range nsPods {
			result.WriteString(fmt.Sprintf("    %s (%s)\n", pod.Name, pod.Status.Phase))
		}
	}
	
	// 获取相关事件
	events, err := getEventsForNode(ctx, clientset, node)
	if err == nil && len(events.Items) > 0 {
		result.WriteString("\n最近事件:\n")
		for _, event := range events.Items {
			result.WriteString(fmt.Sprintf("  %s [%s] %s: %s\n", 
				formatAge(event.LastTimestamp.Time),
				event.Type,
				event.Reason,
				event.Message))
		}
	}
	
	// 诊断建议
	result.WriteString("\n诊断建议:\n")
	
	// 检查节点状态
	nodeReady := false
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				nodeReady = true
			} else {
				result.WriteString(fmt.Sprintf("  • 节点未就绪: %s (%s)\n", condition.Reason, condition.Message))
			}
		} else if condition.Status != corev1.ConditionFalse && condition.Type != corev1.NodeNetworkUnavailable {
			// 对于大多数条件，ConditionFalse是正常的，除了NodeReady和NodeNetworkUnavailable
			result.WriteString(fmt.Sprintf("  • 节点条件 %s 为 %s: %s\n", condition.Type, condition.Status, condition.Message))
		}
	}
	
	if nodeReady {
		result.WriteString("  • 节点状态正常，已就绪\n")
	}
	
	// 检查Pod状态
	var runningPods, pendingPods, failedPods, otherPods int
	for _, pod := range pods.Items {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			runningPods++
		case corev1.PodPending:
			pendingPods++
		case corev1.PodFailed:
			failedPods++
		default:
			otherPods++
		}
	}
	
	if pendingPods > 0 {
		result.WriteString(fmt.Sprintf("  • 节点上有 %d 个处于Pending状态的Pod\n", pendingPods))
	}
	
	if failedPods > 0 {
		result.WriteString(fmt.Sprintf("  • 节点上有 %d 个处于Failed状态的Pod\n", failedPods))
	}
	
	// 如果没有发现明显问题
	if nodeReady && pendingPods == 0 && failedPods == 0 {
		result.WriteString(fmt.Sprintf("  • 节点看起来运行正常，有 %d 个正常运行的Pod\n", runningPods))
	}

	return mcp.NewToolResultText(result.String()), nil
}

// DeploymentDiagnosticTool 诊断Deployment问题的工具函数
func DeploymentDiagnosticTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deploymentName := request.Params.Arguments["deployment_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: deployment_diagnostic, deployment_name=", deploymentName, ", namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取Deployment详情
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Deployment详情失败: %v", err)), err
	}

	// 获取相关的ReplicaSet
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("解析标签选择器失败: %v", err)), err
	}
	
	replicaSets, err := clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取ReplicaSet列表失败: %v", err)), err
	}

	// 获取相关的Pod
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Pod列表失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Deployment %s 诊断报告:\n\n", deploymentName))
	
	// 基本信息
	result.WriteString("基本信息:\n")
	result.WriteString(fmt.Sprintf("  名称: %s\n", deployment.Name))
	result.WriteString(fmt.Sprintf("  命名空间: %s\n", deployment.Namespace))
	result.WriteString(fmt.Sprintf("  创建时间: %s\n", deployment.CreationTimestamp.Format(time.RFC3339)))
	result.WriteString(fmt.Sprintf("  标签选择器: %s\n", selector.String()))
	
	// 副本状态
	result.WriteString("\n副本状态:\n")
	result.WriteString(fmt.Sprintf("  期望副本数: %d\n", *deployment.Spec.Replicas))
	result.WriteString(fmt.Sprintf("  当前副本数: %d\n", deployment.Status.Replicas))
	result.WriteString(fmt.Sprintf("  就绪副本数: %d\n", deployment.Status.ReadyReplicas))
	result.WriteString(fmt.Sprintf("  可用副本数: %d\n", deployment.Status.AvailableReplicas))
	result.WriteString(fmt.Sprintf("  更新副本数: %d\n", deployment.Status.UpdatedReplicas))
	
	// 部署策略
	result.WriteString("\n部署策略:\n")
	result.WriteString(fmt.Sprintf("  类型: %s\n", deployment.Spec.Strategy.Type))
	if deployment.Spec.Strategy.RollingUpdate != nil {
		result.WriteString("  滚动更新配置:\n")
		if deployment.Spec.Strategy.RollingUpdate.MaxUnavailable != nil {
			result.WriteString(fmt.Sprintf("    最大不可用: %s\n", deployment.Spec.Strategy.RollingUpdate.MaxUnavailable.String()))
		}
		if deployment.Spec.Strategy.RollingUpdate.MaxSurge != nil {
			result.WriteString(fmt.Sprintf("    最大超出: %s\n", deployment.Spec.Strategy.RollingUpdate.MaxSurge.String()))
		}
	}
	
	// 容器配置
	result.WriteString("\n容器配置:\n")
	for _, container := range deployment.Spec.Template.Spec.Containers {
		result.WriteString(fmt.Sprintf("  容器: %s\n", container.Name))
		result.WriteString(fmt.Sprintf("    镜像: %s\n", container.Image))
		
		if len(container.Ports) > 0 {
			result.WriteString("    端口:\n")
			for _, port := range container.Ports {
				result.WriteString(fmt.Sprintf("      %s: %d/%s\n", port.Name, port.ContainerPort, port.Protocol))
			}
		}
		
		if container.Resources.Limits != nil || container.Resources.Requests != nil {
			result.WriteString("    资源:\n")
			if container.Resources.Limits != nil {
				result.WriteString("      限制:\n")
				for res, qty := range container.Resources.Limits {
					result.WriteString(fmt.Sprintf("        %s: %s\n", res, qty.String()))
				}
			}
			if container.Resources.Requests != nil {
				result.WriteString("      请求:\n")
				for res, qty := range container.Resources.Requests {
					result.WriteString(fmt.Sprintf("        %s: %s\n", res, qty.String()))
				}
			}
		}
		
		// 存活和就绪探针
		if container.LivenessProbe != nil {
			result.WriteString("    存活探针: 已配置\n")
		}
		if container.ReadinessProbe != nil {
			result.WriteString("    就绪探针: 已配置\n")
		}
	}
	
	// ReplicaSet信息
	result.WriteString(fmt.Sprintf("\nReplicaSet (%d 总计):\n", len(replicaSets.Items)))
	for _, rs := range replicaSets.Items {
		result.WriteString(fmt.Sprintf("  %s:\n", rs.Name))
		result.WriteString(fmt.Sprintf("    期望副本数: %d\n", *rs.Spec.Replicas))
		result.WriteString(fmt.Sprintf("    当前副本数: %d\n", rs.Status.Replicas))
		result.WriteString(fmt.Sprintf("    就绪副本数: %d\n", rs.Status.ReadyReplicas))
		result.WriteString(fmt.Sprintf("    可用副本数: %d\n", rs.Status.AvailableReplicas))
		
		// 检查是否是当前活动的ReplicaSet
		isCurrentRS := true
		for _, ownerRef := range rs.OwnerReferences {
			if ownerRef.Kind == "Deployment" && ownerRef.Name == deployment.Name {
				isCurrentRS = (rs.Spec.Replicas != nil && *rs.Spec.Replicas > 0)
				break
			}
		}
		
		if isCurrentRS {
			result.WriteString("    状态: 活动\n")
		} else {
			result.WriteString("    状态: 非活动\n")
		}
	}
	
	// Pod信息
	result.WriteString(fmt.Sprintf("\nPod (%d 总计):\n", len(pods.Items)))
	
	// 按状态分组
	podsByStatus := make(map[corev1.PodPhase][]corev1.Pod)
	for _, pod := range pods.Items {
		podsByStatus[pod.Status.Phase] = append(podsByStatus[pod.Status.Phase], pod)
	}
	
	for status, statusPods := range podsByStatus {
		result.WriteString(fmt.Sprintf("  %s (%d pods):\n", status, len(statusPods)))
		for _, pod := range statusPods {
			result.WriteString(fmt.Sprintf("    %s (创建于 %s)\n", pod.Name, formatAge(pod.CreationTimestamp.Time)))
			
			// 如果Pod不是Running状态，显示更多信息
			if status != corev1.PodRunning {
				for _, containerStatus := range pod.Status.ContainerStatuses {
					if !containerStatus.Ready {
						if containerStatus.State.Waiting != nil {
							result.WriteString(fmt.Sprintf("      容器 %s: Waiting (%s)\n", 
								containerStatus.Name, 
								containerStatus.State.Waiting.Reason))
						} else if containerStatus.State.Terminated != nil {
							result.WriteString(fmt.Sprintf("      容器 %s: Terminated (退出码: %d, 原因: %s)\n", 
								containerStatus.Name, 
								containerStatus.State.Terminated.ExitCode,
								containerStatus.State.Terminated.Reason))
						}
					}
				}
			}
		}
	}
	
	// 获取相关事件
	events, err := getEventsForDeployment(ctx, clientset, deployment)
	if err == nil && len(events.Items) > 0 {
		result.WriteString("\n最近事件:\n")
		for _, event := range events.Items {
			result.WriteString(fmt.Sprintf("  %s [%s] %s: %s\n", 
				formatAge(event.LastTimestamp.Time),
				event.Type,
				event.Reason,
				event.Message))
		}
	}
	
	// 诊断建议
	result.WriteString("\n诊断建议:\n")
	
	// 检查副本状态
	if deployment.Status.ReadyReplicas < *deployment.Spec.Replicas {
		result.WriteString(fmt.Sprintf("  • Deployment副本不足: 期望 %d, 就绪 %d\n", 
			*deployment.Spec.Replicas, 
			deployment.Status.ReadyReplicas))
	}
	
	// 检查更新状态
	if deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
		result.WriteString(fmt.Sprintf("  • Deployment更新未完成: 已更新 %d/%d\n", 
			deployment.Status.UpdatedReplicas, 
			*deployment.Spec.Replicas))
	}
	
	// 检查可用性
	if deployment.Status.AvailableReplicas < *deployment.Spec.Replicas {
		result.WriteString(fmt.Sprintf("  • Deployment可用副本不足: 可用 %d/%d\n", 
			deployment.Status.AvailableReplicas, 
			*deployment.Spec.Replicas))
	}
	
	// 检查Pod状态
	var pendingPods, failedPods int
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodPending {
			pendingPods++
		} else if pod.Status.Phase == corev1.PodFailed {
			failedPods++
		}
	}
	
	if pendingPods > 0 {
		result.WriteString(fmt.Sprintf("  • 有 %d 个Pod处于Pending状态\n", pendingPods))
		result.WriteString("    - 检查资源限制、节点调度和镜像拉取问题\n")
	}
	
	if failedPods > 0 {
		result.WriteString(fmt.Sprintf("  • 有 %d 个Pod处于Failed状态\n", failedPods))
		result.WriteString("    - 检查容器日志和事件以确定失败原因\n")
	}
	
	// 检查ReplicaSet
	if len(replicaSets.Items) > 1 {
		result.WriteString(fmt.Sprintf("  • 存在多个ReplicaSet (%d): 可能正在进行滚动更新或更新被暂停\n", len(replicaSets.Items)))
	}
	
	// 如果没有发现明显问题
	if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas && 
	   deployment.Status.UpdatedReplicas == *deployment.Spec.Replicas && 
	   deployment.Status.AvailableReplicas == *deployment.Spec.Replicas && 
	   pendingPods == 0 && failedPods == 0 {
		result.WriteString("  • Deployment看起来运行正常，所有副本都已就绪\n")
	}

	return mcp.NewToolResultText(result.String()), nil
}

// AlertAnalysisTool 分析告警信息的工具函数
func AlertAnalysisTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alertName, _ := request.Params.Arguments["alert_name"].(string)
	alertSeverity, _ := request.Params.Arguments["severity"].(string)
	alertStatus, _ := request.Params.Arguments["status"].(string)
	alertDescription, _ := request.Params.Arguments["description"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	podName, _ := request.Params.Arguments["pod_name"].(string)
	nodeName, _ := request.Params.Arguments["node_name"].(string)

	fmt.Println("ai 正在调用mcp server的tool: alert_analysis, alert_name=", alertName)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString("告警分析报告:\n\n")
	
	// 告警基本信息
	result.WriteString("告警信息:\n")
	result.WriteString(fmt.Sprintf("  名称: %s\n", alertName))
	if alertSeverity != "" {
		result.WriteString(fmt.Sprintf("  严重性: %s\n", alertSeverity))
	}
	if alertStatus != "" {
		result.WriteString(fmt.Sprintf("  状态: %s\n", alertStatus))
	}
	if alertDescription != "" {
		result.WriteString(fmt.Sprintf("  描述: %s\n", alertDescription))
	}
	
	// 相关资源信息
	result.WriteString("\n相关资源:\n")
	
	// 如果提供了Pod信息
	if podName != "" && namespace != "" {
		result.WriteString(fmt.Sprintf("  Pod: %s (命名空间: %s)\n", podName, namespace))
		
		// 尝试获取Pod信息
		pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err == nil {
			// Pod状态
			result.WriteString(fmt.Sprintf("    状态: %s\n", pod.Status.Phase))
			result.WriteString(fmt.Sprintf("    节点: %s\n", pod.Spec.NodeName))
			
			// 容器状态
			result.WriteString("    容器状态:\n")
			for _, containerStatus := range pod.Status.ContainerStatuses {
				result.WriteString(fmt.Sprintf("      %s: ", containerStatus.Name))
				if containerStatus.Ready {
					result.WriteString("就绪\n")
				} else {
					result.WriteString("未就绪\n")
					
					if containerStatus.State.Waiting != nil {
						result.WriteString(fmt.Sprintf("        等待原因: %s\n", containerStatus.State.Waiting.Reason))
					} else if containerStatus.State.Terminated != nil {
						result.WriteString(fmt.Sprintf("        终止原因: %s (退出码: %d)\n", 
							containerStatus.State.Terminated.Reason, 
							containerStatus.State.Terminated.ExitCode))
					}
				}
				
				if containerStatus.RestartCount > 0 {
					result.WriteString(fmt.Sprintf("        重启次数: %d\n", containerStatus.RestartCount))
				}
			}
			
			// 获取最近事件
			events, err := getEventsForPod(ctx, clientset, pod)
			if err == nil && len(events.Items) > 0 {
				result.WriteString("    最近事件:\n")
				for i, event := range events.Items {
					if i >= 5 {
						break // 只显示最近5个事件
					}
					result.WriteString(fmt.Sprintf("      %s [%s] %s: %s\n", 
						formatAge(event.LastTimestamp.Time),
						event.Type,
						event.Reason,
						event.Message))
				}
			}
		} else {
			result.WriteString(fmt.Sprintf("    无法获取Pod信息: %v\n", err))
		}
	}
	
	// 如果提供了节点信息
	if nodeName != "" {
		result.WriteString(fmt.Sprintf("  节点: %s\n", nodeName))
		
		// 尝试获取节点信息
		node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err == nil {
			// 节点状态
			nodeReady := "未就绪"
			for _, condition := range node.Status.Conditions {
				if condition.Type == corev1.NodeReady {
					if condition.Status == corev1.ConditionTrue {
						nodeReady = "就绪"
					}
					break
				}
			}
			result.WriteString(fmt.Sprintf("    状态: %s\n", nodeReady))
			
			// 资源信息
			cpuCapacity := node.Status.Capacity.Cpu().String()
			memoryCapacity := node.Status.Capacity.Memory().String()
			result.WriteString(fmt.Sprintf("    CPU容量: %s\n", cpuCapacity))
			result.WriteString(fmt.Sprintf("    内存容量: %s\n", memoryCapacity))
			
			// 获取最近事件
			events, err := getEventsForNode(ctx, clientset, node)
			if err == nil && len(events.Items) > 0 {
				result.WriteString("    最近事件:\n")
				for i, event := range events.Items {
					if i >= 5 {
						break // 只显示最近5个事件
					}
					result.WriteString(fmt.Sprintf("      %s [%s] %s: %s\n", 
						formatAge(event.LastTimestamp.Time),
						event.Type,
						event.Reason,
						event.Message))
				}
			}
		} else {
			result.WriteString(fmt.Sprintf("    无法获取节点信息: %v\n", err))
		}
	}
	
	// 告警分析和建议
	result.WriteString("\n分析和建议:\n")
	
	// 根据告警名称进行分析
	if alertName != "" {
		switch {
		case strings.Contains(strings.ToLower(alertName), "cpu") && strings.Contains(strings.ToLower(alertName), "high"):
			result.WriteString("  • CPU使用率高告警\n")
			result.WriteString("    - 检查是否有异常进程占用CPU\n")
			result.WriteString("    - 考虑增加资源限制或扩展副本数\n")
			result.WriteString("    - 使用 pod_diagnostic 工具查看Pod详情\n")
			
		case strings.Contains(strings.ToLower(alertName), "memory") && strings.Contains(strings.ToLower(alertName), "high"):
			result.WriteString("  • 内存使用率高告警\n")
			result.WriteString("    - 检查是否有内存泄漏\n")
			result.WriteString("    - 考虑增加内存限制或优化应用\n")
			result.WriteString("    - 使用 pod_diagnostic 工具查看Pod详情\n")
			
		case strings.Contains(strings.ToLower(alertName), "disk") && strings.Contains(strings.ToLower(alertName), "pressure"):
			result.WriteString("  • 磁盘空间压力告警\n")
			result.WriteString("    - 清理不必要的文件和日志\n")
			result.WriteString("    - 考虑扩展存储卷\n")
			
		case strings.Contains(strings.ToLower(alertName), "pod") && strings.Contains(strings.ToLower(alertName), "restart"):
			result.WriteString("  • Pod重启告警\n")
			result.WriteString("    - 检查容器日志查找崩溃原因\n")
			result.WriteString("    - 检查资源限制是否合理\n")
			result.WriteString("    - 使用 pod_logs 工具查看容器日志\n")
			
		case strings.Contains(strings.ToLower(alertName), "node") && strings.Contains(strings.ToLower(alertName), "notready"):
			result.WriteString("  • 节点未就绪告警\n")
			result.WriteString("    - 检查节点状态和事件\n")
			result.WriteString("    - 检查kubelet日志\n")
			result.WriteString("    - 使用 node_diagnostic 工具查看节点详情\n")
			
		default:
			result.WriteString("  • 通用告警处理建议\n")
			result.WriteString("    - 检查相关资源的状态和事件\n")
			result.WriteString("    - 查看应用日志寻找错误信息\n")
			result.WriteString("    - 使用相应的诊断工具获取更多信息\n")
		}
	}
	
	// 根据严重性提供建议
	if alertSeverity != "" {
		result.WriteString(fmt.Sprintf("\n  基于严重性(%s)的建议:\n", alertSeverity))
		switch strings.ToLower(alertSeverity) {
		case "critical":
			result.WriteString("    - 立即处理，可能需要人工干预\n")
			result.WriteString("    - 考虑通知相关团队\n")
		case "warning":
			result.WriteString("    - 尽快调查，但可能不需要立即干预\n")
			result.WriteString("    - 监控情况是否恶化\n")
		case "info":
			result.WriteString("    - 作为信息记录，通常不需要立即行动\n")
			result.WriteString("    - 定期检查是否有模式\n")
		}
	}

	return mcp.NewToolResultText(result.String()), nil
}

// 辅助函数：获取节点相关事件
func getEventsForNode(ctx context.Context, clientset *kubernetes.Clientset, node *corev1.Node) (*corev1.EventList, error) {
	fieldSelector := fmt.Sprintf("involvedObject.kind=Node,involvedObject.name=%s", node.Name)
	return clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
}

// 辅助函数：获取Deployment相关事件
func getEventsForDeployment(ctx context.Context, clientset *kubernetes.Clientset, deployment *appsv1.Deployment) (*corev1.EventList, error) {
	fieldSelector := fmt.Sprintf("involvedObject.kind=Deployment,involvedObject.name=%s,involvedObject.namespace=%s",
		deployment.Name, deployment.Namespace)
	return clientset.CoreV1().Events(deployment.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
}
