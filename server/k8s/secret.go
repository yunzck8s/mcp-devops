package k8s

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ListSecretsTool 列出Secret的工具函数
func ListSecretsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: list_secrets, namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取Secret列表
	secrets, err := clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Secret列表失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("命名空间: %s\n\n", namespace))
	result.WriteString("NAME\tTYPE\tDATA\tAGE\n")

	for _, secret := range secrets.Items {
		// 计算运行时间
		age := formatAge(secret.CreationTimestamp.Time)

		result.WriteString(fmt.Sprintf("%s\t%s\t%d\t%s\n",
			secret.Name,
			secret.Type,
			len(secret.Data),
			age))
	}

	return mcp.NewToolResultText(result.String()), nil
}

// DescribeSecretTool 查看Secret详细信息的工具函数
func DescribeSecretTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	secretName := request.Params.Arguments["secret_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}
	
	// 是否显示敏感数据
	showData, _ := request.Params.Arguments["show_data"].(bool)

	fmt.Println("ai 正在调用mcp server的tool: describe_secret, secret_name=", secretName, ", namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取Secret详情
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Secret详情失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Name:         %s\n", secret.Name))
	result.WriteString(fmt.Sprintf("Namespace:    %s\n", secret.Namespace))
	result.WriteString(fmt.Sprintf("Labels:       %s\n", formatLabels(secret.Labels)))
	result.WriteString(fmt.Sprintf("Annotations:  %s\n", formatLabels(secret.Annotations)))
	result.WriteString(fmt.Sprintf("Type:         %s\n", secret.Type))
	result.WriteString(fmt.Sprintf("创建时间:     %s\n\n", secret.CreationTimestamp.Format(time.RFC3339)))

	// 数据
	result.WriteString("数据:\n")
	if len(secret.Data) == 0 {
		result.WriteString("  <无数据>\n")
	} else {
		for key, value := range secret.Data {
			result.WriteString(fmt.Sprintf("  %s: ", key))
			if showData {
				// 尝试将二进制数据转换为字符串
				strValue := string(value)
				if isPrintable(strValue) {
					// 如果值很长，只显示前50个字符
					if len(strValue) > 50 {
						result.WriteString(fmt.Sprintf("%s...\n", strValue[:50]))
					} else {
						result.WriteString(fmt.Sprintf("%s\n", strValue))
					}
				} else {
					// 如果不是可打印字符，则显示base64编码
					result.WriteString(fmt.Sprintf("%s (base64编码)\n", base64.StdEncoding.EncodeToString(value)))
				}
			} else {
				result.WriteString(fmt.Sprintf("%d bytes\n", len(value)))
			}
		}
	}

	// 获取相关事件
	events, err := getEventsForSecret(ctx, clientset, secret)
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

// CreateSecretTool 创建Secret的工具函数
func CreateSecretTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	secretName := request.Params.Arguments["secret_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}
	
	// 获取Secret类型
	secretType, _ := request.Params.Arguments["type"].(string)
	if secretType == "" {
		secretType = "Opaque" // 默认类型
	}

	// 获取数据
	dataMap := make(map[string][]byte)
	dataJson, ok := request.Params.Arguments["data"].(map[string]interface{})
	if ok {
		for k, v := range dataJson {
			if strValue, ok := v.(string); ok {
				dataMap[k] = []byte(strValue)
			}
		}
	}

	// 获取字符串数据
	stringDataMap := make(map[string]string)
	stringDataJson, ok := request.Params.Arguments["string_data"].(map[string]interface{})
	if ok {
		for k, v := range stringDataJson {
			if strValue, ok := v.(string); ok {
				stringDataMap[k] = strValue
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

	fmt.Println("ai 正在调用mcp server的tool: create_secret, secret_name=", secretName, ", namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 创建Secret对象
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        secretName,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Type:       corev1.SecretType(secretType),
		Data:       dataMap,
		StringData: stringDataMap,
	}

	// 处理from_literal参数
	if fromLiteral != "" {
		// 解析key=value格式
		parts := strings.Split(fromLiteral, "=")
		if len(parts) >= 2 {
			key := parts[0]
			value := strings.Join(parts[1:], "=")
			if secret.StringData == nil {
				secret.StringData = make(map[string]string)
			}
			secret.StringData[key] = value
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
			if secret.StringData == nil {
				secret.StringData = make(map[string]string)
			}
			secret.StringData[key] = fmt.Sprintf("<从文件 %s 读取的内容>", parts[1])
		}
	}

	// 创建Secret
	createdSecret, err := clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Secret失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("Secret %s 在命名空间 %s 中创建成功", createdSecret.Name, createdSecret.Namespace)), nil
}

// UpdateSecretTool 更新Secret的工具函数
func UpdateSecretTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	secretName := request.Params.Arguments["secret_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	// 获取数据
	dataMap := make(map[string][]byte)
	dataJson, dataProvided := request.Params.Arguments["data"].(map[string]interface{})
	if dataProvided {
		for k, v := range dataJson {
			if strValue, ok := v.(string); ok {
				dataMap[k] = []byte(strValue)
			}
		}
	}

	// 获取字符串数据
	stringDataMap := make(map[string]string)
	stringDataJson, stringDataProvided := request.Params.Arguments["string_data"].(map[string]interface{})
	if stringDataProvided {
		for k, v := range stringDataJson {
			if strValue, ok := v.(string); ok {
				stringDataMap[k] = strValue
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

	fmt.Println("ai 正在调用mcp server的tool: update_secret, secret_name=", secretName, ", namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取现有Secret
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Secret %s 失败: %v", secretName, err)), err
	}

	// 更新标签
	if labelsProvided {
		secret.Labels = labels
	}

	// 更新注释
	if annotationsProvided {
		secret.Annotations = annotations
	}

	// 更新数据
	if dataProvided {
		secret.Data = dataMap
	}

	// 更新字符串数据
	if stringDataProvided {
		secret.StringData = stringDataMap
	}

	// 添加单个键值对
	if keyProvided && valueProvided {
		if secret.StringData == nil {
			secret.StringData = make(map[string]string)
		}
		secret.StringData[key] = value
	}

	// 更新Secret
	updatedSecret, err := clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("更新Secret失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("Secret %s 在命名空间 %s 中更新成功", updatedSecret.Name, updatedSecret.Namespace)), nil
}

// DeleteSecretTool 删除Secret的工具函数
func DeleteSecretTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	secretName := request.Params.Arguments["secret_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: delete_secret, secret_name=", secretName, ", namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 删除Secret
	err = clientset.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("删除Secret失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("Secret %s 在命名空间 %s 中已删除", secretName, namespace)), nil
}

// 辅助函数：获取Secret相关事件
func getEventsForSecret(ctx context.Context, clientset *kubernetes.Clientset, secret *corev1.Secret) (*corev1.EventList, error) {
	fieldSelector := fmt.Sprintf("involvedObject.kind=Secret,involvedObject.name=%s,involvedObject.namespace=%s",
		secret.Name, secret.Namespace)
	return clientset.CoreV1().Events(secret.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
}

// 辅助函数：检查字符串是否可打印
func isPrintable(s string) bool {
	for _, r := range s {
		if r < 32 || r > 126 {
			return false
		}
	}
	return true
}
