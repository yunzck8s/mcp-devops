package k8s

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

// ListDaemonSetsTool 列出DaemonSet的工具函数
func ListDaemonSetsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: list_daemonsets, namespace=", namespace)

	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	daemonSets, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取DaemonSet列表失败: %v", err)), err
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("命名空间: %s\n\n", namespace))
	result.WriteString("NAME\tDESIRED\tCURRENT\tREADY\tAGE\tCONTAINERS\tIMAGES\n")

	for _, ds := range daemonSets.Items {
		age := formatAge(ds.CreationTimestamp.Time)
		var containers, images []string
		for _, c := range ds.Spec.Template.Spec.Containers {
			containers = append(containers, c.Name)
			images = append(images, c.Image)
		}

		result.WriteString(fmt.Sprintf("%s\t%d\t%d\t%d\t%s\t%s\t%s\n",
			ds.Name,
			ds.Status.DesiredNumberScheduled,
			ds.Status.CurrentNumberScheduled,
			ds.Status.NumberReady,
			age,
			strings.Join(containers, ","),
			strings.Join(images, ",")))
	}

	return mcp.NewToolResultText(result.String()), nil
}

// DescribeDaemonSetTool 获取DaemonSet详情
func DescribeDaemonSetTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dsName := request.Params.Arguments["daemonset_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: describe_daemonset, daemonset_name=", dsName, ", namespace=", namespace)

	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	ds, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, dsName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取DaemonSet失败: %v", err)), err
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Name:               %s\n", ds.Name))
	result.WriteString(fmt.Sprintf("Namespace:          %s\n", ds.Namespace))
	result.WriteString(fmt.Sprintf("CreationTimestamp:  %s\n", ds.CreationTimestamp.Format(time.RFC3339)))
	result.WriteString(fmt.Sprintf("Labels:             %s\n", formatLabels(ds.Labels)))
	result.WriteString(fmt.Sprintf("Annotations:        %s\n", formatLabels(ds.Annotations)))
	result.WriteString(fmt.Sprintf("Selector:           %s\n", formatSelector(ds.Spec.Selector)))
	result.WriteString(fmt.Sprintf("UpdateStrategy:     %s\n", ds.Spec.UpdateStrategy.Type))
	result.WriteString(fmt.Sprintf("MinReadySeconds:    %d\n", ds.Spec.MinReadySeconds))

	// 容器信息
	result.WriteString("Containers:\n")
	for _, c := range ds.Spec.Template.Spec.Containers {
		result.WriteString(fmt.Sprintf("  Name:   %s\n", c.Name))
		result.WriteString(fmt.Sprintf("  Image:  %s\n", c.Image))
		result.WriteString(fmt.Sprintf("  Ports:  %s\n", formatContainerPorts(c.Ports)))
	}

	return mcp.NewToolResultText(result.String()), nil
}

// RestartDaemonSetTool 优雅重启DaemonSet
func RestartDaemonSetTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dsName := request.Params.Arguments["daemonset_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: restart_daemonset, daemonset_name=", dsName, ", namespace=", namespace)

	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	ds, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, dsName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取DaemonSet失败: %v", err)), err
	}

	if ds.Spec.Template.Annotations == nil {
		ds.Spec.Template.Annotations = make(map[string]string)
	}
	ds.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = clientset.AppsV1().DaemonSets(namespace).Update(ctx, ds, metav1.UpdateOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("重启DaemonSet失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("DaemonSet %s 已成功触发重启", dsName)), nil
}
