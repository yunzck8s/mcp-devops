package k8s

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// 列出Service的工具函数
func ListServicesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: list_services, namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取Service列表
	services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Service列表失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("命名空间: %s\n\n", namespace))
	result.WriteString("NAME\tTYPE\tCLUSTER-IP\tEXTERNAL-IP\tPORT(S)\tAGE\tSELECTOR\n")

	for _, service := range services.Items {
		// 计算运行时间
		age := formatAge(service.CreationTimestamp.Time)

		// 格式化端口
		var ports []string
		for _, port := range service.Spec.Ports {
			if port.NodePort > 0 {
				ports = append(ports, fmt.Sprintf("%d:%d/%s", port.Port, port.NodePort, port.Protocol))
			} else {
				ports = append(ports, fmt.Sprintf("%d/%s", port.Port, port.Protocol))
			}
		}
		portStr := strings.Join(ports, ", ")

		// 格式化外部IP
		externalIPs := "<none>"
		if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
			if len(service.Status.LoadBalancer.Ingress) > 0 {
				var ips []string
				for _, ingress := range service.Status.LoadBalancer.Ingress {
					if ingress.IP != "" {
						ips = append(ips, ingress.IP)
					} else if ingress.Hostname != "" {
						ips = append(ips, ingress.Hostname)
					}
				}
				if len(ips) > 0 {
					externalIPs = strings.Join(ips, ", ")
				}
			}
		} else if len(service.Spec.ExternalIPs) > 0 {
			externalIPs = strings.Join(service.Spec.ExternalIPs, ", ")
		}

		// 格式化选择器
		selectorStr := "<none>"
		if len(service.Spec.Selector) > 0 {
			var selectors []string
			for k, v := range service.Spec.Selector {
				selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
			}
			selectorStr = strings.Join(selectors, ", ")
		}

		result.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			service.Name,
			service.Spec.Type,
			service.Spec.ClusterIP,
			externalIPs,
			portStr,
			age,
			selectorStr))
	}

	return mcp.NewToolResultText(result.String()), nil
}

// 获取Service详情的工具函数
func DescribeServiceTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceName := request.Params.Arguments["service_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: describe_service, service_name=", serviceName, ", namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取Service详情
	service, err := clientset.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Service详情失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Name:              %s\n", service.Name))
	result.WriteString(fmt.Sprintf("Namespace:         %s\n", service.Namespace))
	result.WriteString(fmt.Sprintf("Labels:            %s\n", formatLabels(service.Labels)))
	result.WriteString(fmt.Sprintf("Annotations:       %s\n", formatLabels(service.Annotations)))
	result.WriteString(fmt.Sprintf("Selector:          %s\n", formatLabels(service.Spec.Selector)))
	result.WriteString(fmt.Sprintf("Type:              %s\n", service.Spec.Type))
	result.WriteString(fmt.Sprintf("IP:                %s\n", service.Spec.ClusterIP))

	if len(service.Spec.ExternalIPs) > 0 {
		result.WriteString(fmt.Sprintf("External IPs:      %s\n", strings.Join(service.Spec.ExternalIPs, ", ")))
	}

	if service.Spec.LoadBalancerIP != "" {
		result.WriteString(fmt.Sprintf("LoadBalancer IP:   %s\n", service.Spec.LoadBalancerIP))
	}

	if len(service.Status.LoadBalancer.Ingress) > 0 {
		var ips []string
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			if ingress.IP != "" {
				ips = append(ips, ingress.IP)
			} else if ingress.Hostname != "" {
				ips = append(ips, ingress.Hostname)
			}
		}
		if len(ips) > 0 {
			result.WriteString(fmt.Sprintf("LoadBalancer Ingress: %s\n", strings.Join(ips, ", ")))
		}
	}

	result.WriteString(fmt.Sprintf("Port:\n"))
	for _, port := range service.Spec.Ports {
		nodePort := ""
		if port.NodePort > 0 {
			nodePort = fmt.Sprintf("%d", port.NodePort)
		}
		targetPort := ""
		if port.TargetPort.IntVal > 0 {
			targetPort = fmt.Sprintf("%d", port.TargetPort.IntVal)
		} else if port.TargetPort.StrVal != "" {
			targetPort = port.TargetPort.StrVal
		}
		result.WriteString(fmt.Sprintf("  - Name:        %s\n", port.Name))
		result.WriteString(fmt.Sprintf("    Protocol:    %s\n", port.Protocol))
		result.WriteString(fmt.Sprintf("    Port:        %d\n", port.Port))
		result.WriteString(fmt.Sprintf("    TargetPort:  %s\n", targetPort))
		if nodePort != "" {
			result.WriteString(fmt.Sprintf("    NodePort:    %s\n", nodePort))
		}
	}

	result.WriteString(fmt.Sprintf("Session Affinity: %s\n", service.Spec.SessionAffinity))

	if service.Spec.ExternalTrafficPolicy != "" {
		result.WriteString(fmt.Sprintf("External Traffic Policy: %s\n", service.Spec.ExternalTrafficPolicy))
	}

	if service.Spec.Type == corev1.ServiceTypeExternalName {
		result.WriteString(fmt.Sprintf("External Name:    %s\n", service.Spec.ExternalName))
	}

	// 获取和此服务匹配的端点
	endpoints, err := clientset.CoreV1().Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err == nil {
		result.WriteString("\nEndpoints:\n")
		if len(endpoints.Subsets) == 0 {
			result.WriteString("  <none>\n")
		} else {
			for i, subset := range endpoints.Subsets {
				// 收集所有地址
				var addresses []string
				for _, addr := range subset.Addresses {
					addrStr := addr.IP
					if addr.TargetRef != nil && addr.TargetRef.Kind == "Pod" {
						addrStr = fmt.Sprintf("%s (%s)", addr.IP, addr.TargetRef.Name)
					}
					addresses = append(addresses, addrStr)
				}

				// 收集所有端口
				var ports []string
				for _, port := range subset.Ports {
					ports = append(ports, fmt.Sprintf("%s:%d", port.Name, port.Port))
				}

				result.WriteString(fmt.Sprintf("  Subset %d:\n", i+1))
				result.WriteString(fmt.Sprintf("    Addresses: %s\n", strings.Join(addresses, ", ")))
				if len(subset.NotReadyAddresses) > 0 {
					var notReadyAddrs []string
					for _, addr := range subset.NotReadyAddresses {
						addrStr := addr.IP
						if addr.TargetRef != nil && addr.TargetRef.Kind == "Pod" {
							addrStr = fmt.Sprintf("%s (%s)", addr.IP, addr.TargetRef.Name)
						}
						notReadyAddrs = append(notReadyAddrs, addrStr)
					}
					result.WriteString(fmt.Sprintf("    NotReadyAddresses: %s\n", strings.Join(notReadyAddrs, ", ")))
				}
				result.WriteString(fmt.Sprintf("    Ports: %s\n", strings.Join(ports, ", ")))
			}
		}
	}

	// 如果是LoadBalancer类型，尝试获取相关的事件
	if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
		events, err := getEventsForService(ctx, clientset, service)
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
	}

	return mcp.NewToolResultText(result.String()), nil
}

// 辅助函数：获取Service相关事件
func getEventsForService(ctx context.Context, clientset *kubernetes.Clientset, service *corev1.Service) (*corev1.EventList, error) {
	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.namespace=%s,involvedObject.kind=Service",
		service.Name, service.Namespace)

	return clientset.CoreV1().Events(service.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
}
