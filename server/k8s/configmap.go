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

// ListConfigMapsTool 列出ConfigMap的工具函数
func ListConfigMapsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: list_configmaps, namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取ConfigMap列表
	configmaps, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取ConfigMap列表失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("命名空间: %s\n\n", namespace))
	result.WriteString("NAME\tDATA\tAGE\n")

	for _, cm := range configmaps.Items {
		// 计算运行时间
		age := formatAge(cm.CreationTimestamp.Time)

		result.WriteString(fmt.Sprintf("%s\t%d\t%s\n",
			cm.Name,
			len(cm.Data),
			age))
	}

	return mcp.NewToolResultText(result.String()), nil
}

// DescribeConfigMapTool 查看ConfigMap详细信息的工具函数
func DescribeConfigMapTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	configMapName := request.Params.Arguments["configmap_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: describe_configmap, configmap_name=", configMapName, ", namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取ConfigMap详情
	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取ConfigMap详情失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Name:         %s\n", cm.Name))
	result.WriteString(fmt.Sprintf("Namespace:    %s\n", cm.Namespace))
	result.WriteString(fmt.Sprintf("Labels:       %s\n", formatLabels(cm.Labels)))
	result.WriteString(fmt.Sprintf("Annotations:  %s\n", formatLabels(cm.Annotations)))
	result.WriteString(fmt.Sprintf("创建时间:     %s\n\n", cm.CreationTimestamp.Format(time.RFC3339)))

	// 数据
	result.WriteString("数据:\n")
	if len(cm.Data) == 0 {
		result.WriteString("  <无数据>\n")
	} else {
		for key, value := range cm.Data {
			result.WriteString(fmt.Sprintf("  %s:\n", key))
			// 如果值很长，只显示前100个字符
			if len(value) > 100 {
				result.WriteString(fmt.Sprintf("    %s...\n", value[:100]))
			} else {
				result.WriteString(fmt.Sprintf("    %s\n", value))
			}
		}
	}

	// 二进制数据
	if len(cm.BinaryData) > 0 {
		result.WriteString("\n二进制数据:\n")
		for key, _ := range cm.BinaryData {
			result.WriteString(fmt.Sprintf("  %s: <二进制数据>\n", key))
		}
	}

	// 获取相关事件
	events, err := getEventsForConfigMap(ctx, clientset, cm)
	if err == nil && len(events.Items) > 0 {
		result.WriteString("\n事件:\n")
		result.WriteString("LAST SEEN\tTYPE\tREASON\tOBJECT\tMESSAGE\n")
		for _, event := range events.Items {
			age := formatAge(event.LastTimestamp.Time)
			result.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s/%s\t%s\n",
				age,
				event.Type,
				event.Reason,
				event.InvolvedObject.Kind,
				event.InvolvedObject.Name,
				event.Message))
		}
	}

	return mcp.NewToolResultText(result.String()), nil
}

// CreateConfigMapTool 创建ConfigMap的工具函数
func CreateConfigMapTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	configMapName := request.Params.Arguments["configmap_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	// 获取数据
	dataMap := make(map[string]string)
	dataJson, ok := request.Params.Arguments["data"].(map[string]interface{})
	if ok {
		for k, v := range dataJson {
			if strValue, ok := v.(string); ok {
				dataMap[k] = strValue
			}
		}
	}

	// 获取标签
	labels := make(map[string]string)
	labelsJson, ok := request.Params.Arguments["labels"].(map[string]interface{})
	if ok {
		for k, v := range labelsJson {
			if strValue, ok := v.(string); ok {
				labels[k] = strValue
			}
		}
	}

	// 获取注释
	annotations := make(map[string]string)
	annotationsJson, ok := request.Params.Arguments["annotations"].(map[string]interface{})
	if ok {
		for k, v := range annotationsJson {
			if strValue, ok := v.(string); ok {
				annotations[k] = strValue
			}
		}
	}

	// 从文件创建
	fromFile, _ := request.Params.Arguments["from_file"].(string)
	fromLiteral, _ := request.Params.Arguments["from_literal"].(string)

	fmt.Println("ai 正在调用mcp server的tool: create_configmap, configmap_name=", configMapName, ", namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 创建ConfigMap对象
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        configMapName,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Data: dataMap,
	}

	// 处理from_literal参数
	if fromLiteral != "" {
		// 解析key=value格式
		parts := strings.Split(fromLiteral, "=")
		if len(parts) >= 2 {
			key := parts[0]
			value := strings.Join(parts[1:], "=")
			if configMap.Data == nil {
				configMap.Data = make(map[string]string)
			}
			configMap.Data[key] = value
		}
	}

	// 处理from_file参数 (这里只是示例，实际上需要读取文件内容)
	if fromFile != "" {
		// 解析key=文件路径格式
		parts := strings.Split(fromFile, "=")
		if len(parts) == 2 {
			key := parts[0]
			// 在实际实现中，这里应该读取文件内容
			// 但由于我们无法直接读取文件，这里只是示例
			if configMap.Data == nil {
				configMap.Data = make(map[string]string)
			}
			configMap.Data[key] = fmt.Sprintf("<从文件 %s 读取的内容>", parts[1])
		}
	}

	// 创建ConfigMap
	createdConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建ConfigMap失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("ConfigMap %s 在命名空间 %s 中创建成功", createdConfigMap.Name, createdConfigMap.Namespace)), nil
}

// UpdateConfigMapTool 更新ConfigMap的工具函数
func UpdateConfigMapTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	configMapName := request.Params.Arguments["configmap_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	// 获取数据
	dataMap := make(map[string]string)
	dataJson, dataProvided := request.Params.Arguments["data"].(map[string]interface{})
	if dataProvided {
		for k, v := range dataJson {
			if strValue, ok := v.(string); ok {
				dataMap[k] = strValue
			}
		}
	}

	// 获取标签
	labels := make(map[string]string)
	labelsJson, labelsProvided := request.Params.Arguments["labels"].(map[string]interface{})
	if labelsProvided {
		for k, v := range labelsJson {
			if strValue, ok := v.(string); ok {
				labels[k] = strValue
			}
		}
	}

	// 获取注释
	annotations := make(map[string]string)
	annotationsJson, annotationsProvided := request.Params.Arguments["annotations"].(map[string]interface{})
	if annotationsProvided {
		for k, v := range annotationsJson {
			if strValue, ok := v.(string); ok {
				annotations[k] = strValue
			}
		}
	}

	// 添加单个键值对
	key, keyProvided := request.Params.Arguments["key"].(string)
	value, valueProvided := request.Params.Arguments["value"].(string)

	fmt.Println("ai 正在调用mcp server的tool: update_configmap, configmap_name=", configMapName, ", namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取现有ConfigMap
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取ConfigMap %s 失败: %v", configMapName, err)), err
	}

	// 更新标签
	if labelsProvided {
		configMap.Labels = labels
	}

	// 更新注释
	if annotationsProvided {
		configMap.Annotations = annotations
	}

	// 更新数据
	if dataProvided {
		configMap.Data = dataMap
	}

	// 添加单个键值对
	if keyProvided && valueProvided {
		if configMap.Data == nil {
			configMap.Data = make(map[string]string)
		}
		configMap.Data[key] = value
	}

	// 更新ConfigMap
	updatedConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("更新ConfigMap失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("ConfigMap %s 在命名空间 %s 中更新成功", updatedConfigMap.Name, updatedConfigMap.Namespace)), nil
}

// DeleteConfigMapTool 删除ConfigMap的工具函数
func DeleteConfigMapTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	configMapName := request.Params.Arguments["configmap_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: delete_configmap, configmap_name=", configMapName, ", namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 删除ConfigMap
	err = clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, configMapName, metav1.DeleteOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("删除ConfigMap失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("ConfigMap %s 在命名空间 %s 中已删除", configMapName, namespace)), nil
}

// 辅助函数：获取ConfigMap相关事件
func getEventsForConfigMap(ctx context.Context, clientset *kubernetes.Clientset, cm *corev1.ConfigMap) (*corev1.EventList, error) {
	fieldSelector := fmt.Sprintf("involvedObject.kind=ConfigMap,involvedObject.name=%s,involvedObject.namespace=%s",
		cm.Name, cm.Namespace)
	return clientset.CoreV1().Events(cm.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
}
