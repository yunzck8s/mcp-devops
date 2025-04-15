package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// 列出Deployment的工具函数
func ListDeploymentsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: list_deployments, namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取Deployment列表
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Deployment列表失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("命名空间: %s\n\n", namespace))
	result.WriteString("NAME\tREADY\tUP-TO-DATE\tAVAILABLE\tAGE\tCONTAINERS\tIMAGES\n")

	for _, deployment := range deployments.Items {
		// 计算运行时间
		age := formatAge(deployment.CreationTimestamp.Time)

		// 获取容器和镜像信息
		var containers []string
		var images []string
		for _, container := range deployment.Spec.Template.Spec.Containers {
			containers = append(containers, container.Name)
			images = append(images, container.Image)
		}

		result.WriteString(fmt.Sprintf("%s\t%d/%d\t%d\t%d\t%s\t%s\t%s\n",
			deployment.Name,
			deployment.Status.ReadyReplicas,
			deployment.Status.Replicas,
			deployment.Status.UpdatedReplicas,
			deployment.Status.AvailableReplicas,
			age,
			strings.Join(containers, ","),
			strings.Join(images, ",")))
	}

	return mcp.NewToolResultText(result.String()), nil
}

// 获取Deployment详情的工具函数
func DescribeDeploymentTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deploymentName := request.Params.Arguments["deployment_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: describe_deployment, deployment_name=", deploymentName, ", namespace=", namespace)

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

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Name:               %s\n", deployment.Name))
	result.WriteString(fmt.Sprintf("Namespace:          %s\n", deployment.Namespace))
	result.WriteString(fmt.Sprintf("CreationTimestamp:  %s\n", deployment.CreationTimestamp.Format(time.RFC3339)))
	result.WriteString(fmt.Sprintf("Labels:             %s\n", formatLabels(deployment.Labels)))
	result.WriteString(fmt.Sprintf("Annotations:        %s\n", formatLabels(deployment.Annotations)))
	result.WriteString(fmt.Sprintf("Selector:           %s\n", formatSelector(deployment.Spec.Selector)))
	result.WriteString(fmt.Sprintf("Replicas:           %d desired | %d updated | %d total | %d available | %d unavailable\n",
		*deployment.Spec.Replicas,
		deployment.Status.UpdatedReplicas,
		deployment.Status.Replicas,
		deployment.Status.AvailableReplicas,
		deployment.Status.UnavailableReplicas))
	result.WriteString(fmt.Sprintf("StrategyType:       %s\n", deployment.Spec.Strategy.Type))

	if deployment.Spec.Strategy.RollingUpdate != nil {
		result.WriteString(fmt.Sprintf("RollingUpdateStrategy:  MaxUnavailable: %s, MaxSurge: %s\n",
			deployment.Spec.Strategy.RollingUpdate.MaxUnavailable.String(),
			deployment.Spec.Strategy.RollingUpdate.MaxSurge.String()))
	}

	result.WriteString(fmt.Sprintf("MinReadySeconds:    %d\n", deployment.Spec.MinReadySeconds))
	result.WriteString(fmt.Sprintf("RevisionHistoryLimit: %d\n", *deployment.Spec.RevisionHistoryLimit))

	// Pod模板
	result.WriteString("\nPod Template:\n")
	result.WriteString(fmt.Sprintf("  Labels:       %s\n", formatLabels(deployment.Spec.Template.Labels)))

	// 容器
	result.WriteString("  Containers:\n")
	for _, container := range deployment.Spec.Template.Spec.Containers {
		result.WriteString(fmt.Sprintf("   %s:\n", container.Name))
		result.WriteString(fmt.Sprintf("    Image:      %s\n", container.Image))
		result.WriteString(fmt.Sprintf("    Ports:      %s\n", formatContainerPorts(container.Ports)))
		result.WriteString(fmt.Sprintf("    Host Ports: %s\n", formatContainerHostPorts(container.Ports)))

		if len(container.Command) > 0 {
			result.WriteString(fmt.Sprintf("    Command:    %s\n", strings.Join(container.Command, " ")))
		}

		if len(container.Args) > 0 {
			result.WriteString(fmt.Sprintf("    Args:       %s\n", strings.Join(container.Args, " ")))
		}

		result.WriteString(fmt.Sprintf("    Environment: %s\n", formatEnvVars(container.Env)))
		result.WriteString(fmt.Sprintf("    Mounts:      %s\n", formatVolumeMounts(container.VolumeMounts)))

		if container.Resources.Limits != nil || container.Resources.Requests != nil {
			result.WriteString("    Resources:\n")
			if container.Resources.Limits != nil {
				result.WriteString(fmt.Sprintf("      Limits:\n"))
				for res, qty := range container.Resources.Limits {
					result.WriteString(fmt.Sprintf("        %s: %s\n", res, qty.String()))
				}
			}
			if container.Resources.Requests != nil {
				result.WriteString(fmt.Sprintf("      Requests:\n"))
				for res, qty := range container.Resources.Requests {
					result.WriteString(fmt.Sprintf("        %s: %s\n", res, qty.String()))
				}
			}
		}

		if container.LivenessProbe != nil {
			result.WriteString("    Liveness Probe:  Defined\n")
		}

		if container.ReadinessProbe != nil {
			result.WriteString("    Readiness Probe: Defined\n")
		}
	}

	// 卷
	if len(deployment.Spec.Template.Spec.Volumes) > 0 {
		result.WriteString("  Volumes:\n")
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			result.WriteString(fmt.Sprintf("   %s:\n", volume.Name))
			if volume.PersistentVolumeClaim != nil {
				result.WriteString(fmt.Sprintf("    Type:       PersistentVolumeClaim\n"))
				result.WriteString(fmt.Sprintf("    ClaimName:  %s\n", volume.PersistentVolumeClaim.ClaimName))
			} else if volume.ConfigMap != nil {
				result.WriteString(fmt.Sprintf("    Type:       ConfigMap\n"))
				result.WriteString(fmt.Sprintf("    Name:       %s\n", volume.ConfigMap.Name))
			} else if volume.Secret != nil {
				result.WriteString(fmt.Sprintf("    Type:       Secret\n"))
				result.WriteString(fmt.Sprintf("    SecretName: %s\n", volume.Secret.SecretName))
			} else if volume.EmptyDir != nil {
				result.WriteString(fmt.Sprintf("    Type:       EmptyDir\n"))
			} else if volume.HostPath != nil {
				result.WriteString(fmt.Sprintf("    Type:       HostPath\n"))
				result.WriteString(fmt.Sprintf("    Path:       %s\n", volume.HostPath.Path))
			} else {
				result.WriteString(fmt.Sprintf("    Type:       Other\n"))
			}
		}
	}

	// 获取相关事件
	events, err := getEventsForDeployment(ctx, clientset, deployment)
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

// 扩缩Deployment的工具函数
func ScaleDeploymentTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deploymentName := request.Params.Arguments["deployment_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	replicas, ok := request.Params.Arguments["replicas"].(float64)
	if !ok {
		return mcp.NewToolResultText("缺少必要的参数: replicas"), fmt.Errorf("缺少replicas参数")
	}

	replicasInt := int32(replicas)

	fmt.Println("ai 正在调用mcp server的tool: scale_deployment, deployment_name=", deploymentName, ", namespace=", namespace, ", replicas=", replicasInt)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取当前Deployment
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Deployment失败: %v", err)), err
	}

	// 记录原副本数
	oldReplicas := *deployment.Spec.Replicas

	// 更新副本数
	deployment.Spec.Replicas = &replicasInt

	// 应用更新
	_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("扩缩Deployment失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("已将Deployment %s 在命名空间 %s 中的副本数从 %d 扩缩到 %d",
		deploymentName, namespace, oldReplicas, replicasInt)), nil
}

// 重启Deployment的工具函数
func RestartDeploymentTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deploymentName := request.Params.Arguments["deployment_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: restart_deployment, deployment_name=", deploymentName, ", namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取当前Deployment
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Deployment失败: %v", err)), err
	}

	// 添加或更新重启注解
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	// 应用更新
	_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("重启Deployment失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("Deployment %s 在命名空间 %s 中已开始重启", deploymentName, namespace)), nil
}

// 辅助函数：获取Deployment相关事件
func getEventsForDeployment(ctx context.Context, clientset *kubernetes.Clientset, deployment *appsv1.Deployment) (*corev1.EventList, error) {
	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.namespace=%s,involvedObject.kind=Deployment",
		deployment.Name, deployment.Namespace)

	return clientset.CoreV1().Events(deployment.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
}

// 辅助函数：格式化选择器
func formatSelector(selector *metav1.LabelSelector) string {
	if selector == nil || (len(selector.MatchLabels) == 0 && len(selector.MatchExpressions) == 0) {
		return "<none>"
	}

	var parts []string

	// 处理MatchLabels
	for k, v := range selector.MatchLabels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}

	// 处理MatchExpressions
	for _, expr := range selector.MatchExpressions {
		switch expr.Operator {
		case metav1.LabelSelectorOpIn:
			parts = append(parts, fmt.Sprintf("%s in (%s)", expr.Key, strings.Join(expr.Values, ",")))
		case metav1.LabelSelectorOpNotIn:
			parts = append(parts, fmt.Sprintf("%s notin (%s)", expr.Key, strings.Join(expr.Values, ",")))
		case metav1.LabelSelectorOpExists:
			parts = append(parts, fmt.Sprintf("%s", expr.Key))
		case metav1.LabelSelectorOpDoesNotExist:
			parts = append(parts, fmt.Sprintf("!%s", expr.Key))
		}
	}

	return strings.Join(parts, ", ")
}
