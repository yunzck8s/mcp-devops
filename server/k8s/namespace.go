package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// 列出Namespace的工具函数
func ListNamespacesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	fmt.Println("ai 正在调用mcp server的tool: list_namespaces")

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取Namespace列表
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Namespace列表失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString("NAME\tSTATUS\tAGE\n")

	for _, ns := range namespaces.Items {
		// 计算运行时间
		age := formatAge(ns.CreationTimestamp.Time)

		result.WriteString(fmt.Sprintf("%s\t%s\t%s\n",
			ns.Name,
			string(ns.Status.Phase),
			age))
	}

	return mcp.NewToolResultText(result.String()), nil
}

// 获取Namespace详情的工具函数
func DescribeNamespaceTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceName := request.Params.Arguments["namespace_name"].(string)

	fmt.Println("ai 正在调用mcp server的tool: describe_namespace, namespace_name=", namespaceName)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取Namespace详情
	namespace, err := clientset.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Namespace详情失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Name:              %s\n", namespace.Name))
	result.WriteString(fmt.Sprintf("Labels:            %s\n", formatLabels(namespace.Labels)))
	result.WriteString(fmt.Sprintf("Annotations:       %s\n", formatLabels(namespace.Annotations)))
	result.WriteString(fmt.Sprintf("Status:            %s\n", string(namespace.Status.Phase)))
	result.WriteString(fmt.Sprintf("CreationTimestamp: %s\n", namespace.CreationTimestamp.Format(time.RFC3339)))

	// 获取资源配额
	quotas, err := clientset.CoreV1().ResourceQuotas(namespace.Name).List(ctx, metav1.ListOptions{})
	if err == nil && len(quotas.Items) > 0 {
		result.WriteString("\nResource Quotas:\n")
		for _, quota := range quotas.Items {
			result.WriteString(fmt.Sprintf("  %s:\n", quota.Name))
			if len(quota.Spec.Hard) > 0 {
				result.WriteString("    Hard:\n")
				for res, qty := range quota.Spec.Hard {
					result.WriteString(fmt.Sprintf("      %s: %s\n", res, qty.String()))
				}
			}
			if len(quota.Status.Used) > 0 {
				result.WriteString("    Used:\n")
				for res, qty := range quota.Status.Used {
					result.WriteString(fmt.Sprintf("      %s: %s\n", res, qty.String()))
				}
			}
		}
	}

	// 获取资源限制范围
	limits, err := clientset.CoreV1().LimitRanges(namespace.Name).List(ctx, metav1.ListOptions{})
	if err == nil && len(limits.Items) > 0 {
		result.WriteString("\nLimit Ranges:\n")
		for _, limit := range limits.Items {
			result.WriteString(fmt.Sprintf("  %s:\n", limit.Name))
			for _, item := range limit.Spec.Limits {
				result.WriteString(fmt.Sprintf("    Type: %s\n", item.Type))

				if len(item.Max) > 0 {
					result.WriteString("    Max:\n")
					for res, qty := range item.Max {
						result.WriteString(fmt.Sprintf("      %s: %s\n", res, qty.String()))
					}
				}

				if len(item.Min) > 0 {
					result.WriteString("    Min:\n")
					for res, qty := range item.Min {
						result.WriteString(fmt.Sprintf("      %s: %s\n", res, qty.String()))
					}
				}

				if len(item.Default) > 0 {
					result.WriteString("    Default:\n")
					for res, qty := range item.Default {
						result.WriteString(fmt.Sprintf("      %s: %s\n", res, qty.String()))
					}
				}

				if len(item.DefaultRequest) > 0 {
					result.WriteString("    DefaultRequest:\n")
					for res, qty := range item.DefaultRequest {
						result.WriteString(fmt.Sprintf("      %s: %s\n", res, qty.String()))
					}
				}
			}
		}
	}

	// 获取已知的工作负载数量
	deployCount, _ := getDeploymentCount(ctx, clientset, namespace.Name)
	svcCount, _ := getServiceCount(ctx, clientset, namespace.Name)
	podCount, _ := getPodCount(ctx, clientset, namespace.Name)
	cmCount, _ := getConfigMapCount(ctx, clientset, namespace.Name)
	secretCount, _ := getSecretCount(ctx, clientset, namespace.Name)

	result.WriteString("\nWorkload Summary:\n")
	result.WriteString(fmt.Sprintf("  Deployments: %d\n", deployCount))
	result.WriteString(fmt.Sprintf("  Services: %d\n", svcCount))
	result.WriteString(fmt.Sprintf("  Pods: %d\n", podCount))
	result.WriteString(fmt.Sprintf("  ConfigMaps: %d\n", cmCount))
	result.WriteString(fmt.Sprintf("  Secrets: %d\n", secretCount))

	// 获取相关事件
	events, err := getEventsForNamespace(ctx, clientset, namespace)
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

// 创建Namespace的工具函数
func CreateNamespaceTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceName := request.Params.Arguments["namespace_name"].(string)

	fmt.Println("ai 正在调用mcp server的tool: create_namespace, namespace_name=", namespaceName)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 创建Namespace对象
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}

	// 创建Namespace
	_, err = clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Namespace失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("Namespace %s 创建成功", namespaceName)), nil
}

// 删除Namespace的工具函数
func DeleteNamespaceTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceName := request.Params.Arguments["namespace_name"].(string)

	fmt.Println("ai 正在调用mcp server的tool: delete_namespace, namespace_name=", namespaceName)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 删除Namespace
	err = clientset.CoreV1().Namespaces().Delete(ctx, namespaceName, metav1.DeleteOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("删除Namespace失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("Namespace %s 删除成功（删除过程可能需要一些时间才能完成）", namespaceName)), nil
}

// 辅助函数：获取Namespace相关事件
func getEventsForNamespace(ctx context.Context, clientset *kubernetes.Clientset, namespace *corev1.Namespace) (*corev1.EventList, error) {
	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Namespace", namespace.Name)

	return clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
}

// 辅助函数：获取命名空间中的Deployment数量
func getDeploymentCount(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (int, error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	return len(deployments.Items), nil
}

// 辅助函数：获取命名空间中的Service数量
func getServiceCount(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (int, error) {
	services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	return len(services.Items), nil
}

// 辅助函数：获取命名空间中的Pod数量
func getPodCount(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (int, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	return len(pods.Items), nil
}

// 辅助函数：获取命名空间中的ConfigMap数量
func getConfigMapCount(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (int, error) {
	configmaps, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	return len(configmaps.Items), nil
}

// 辅助函数：获取命名空间中的Secret数量
func getSecretCount(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (int, error) {
	secrets, err := clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	return len(secrets.Items), nil
}
