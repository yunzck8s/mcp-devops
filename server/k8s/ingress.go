package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ListIngressesTool 列出Ingress的工具函数
func ListIngressesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: list_ingresses, namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取Ingress列表
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Ingress列表失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("命名空间: %s\n\n", namespace))
	result.WriteString("NAME\tHOSTS\tADDRESS\tPORTS\tAGE\tCLASS\n")

	for _, ing := range ingresses.Items {
		// 获取主机列表
		var hosts []string
		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
		}
		hostStr := "<none>"
		if len(hosts) > 0 {
			hostStr = strings.Join(hosts, ",")
		}

		// 获取地址列表
		var addresses []string
		for _, lbi := range ing.Status.LoadBalancer.Ingress {
			if lbi.IP != "" {
				addresses = append(addresses, lbi.IP)
			} else if lbi.Hostname != "" {
				addresses = append(addresses, lbi.Hostname)
			}
		}
		addressStr := "<none>"
		if len(addresses) > 0 {
			addressStr = strings.Join(addresses, ",")
		}

		// 获取端口列表
		var ports []string
		if len(ing.Spec.TLS) > 0 {
			ports = append(ports, "443")
		}
		// 检查是否有HTTP规则
		for _, rule := range ing.Spec.Rules {
			if rule.HTTP != nil && len(rule.HTTP.Paths) > 0 {
				ports = append(ports, "80")
				break
			}
		}
		portStr := "<none>"
		if len(ports) > 0 {
			portStr = strings.Join(ports, ",")
		}

		// 获取Ingress Class
		ingressClass := "<none>"
		if ing.Spec.IngressClassName != nil {
			ingressClass = *ing.Spec.IngressClassName
		}

		// 计算运行时间
		age := formatAge(ing.CreationTimestamp.Time)

		result.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\n",
			ing.Name,
			hostStr,
			addressStr,
			portStr,
			age,
			ingressClass))
	}

	return mcp.NewToolResultText(result.String()), nil
}

// DescribeIngressTool 查看Ingress详细信息的工具函数
func DescribeIngressTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ingressName := request.Params.Arguments["ingress_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: describe_ingress, ingress_name=", ingressName, ", namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取Ingress详情
	ing, err := clientset.NetworkingV1().Ingresses(namespace).Get(ctx, ingressName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Ingress详情失败: %v", err)), err
	}

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Name:               %s\n", ing.Name))
	result.WriteString(fmt.Sprintf("Namespace:          %s\n", ing.Namespace))
	result.WriteString(fmt.Sprintf("CreationTimestamp:  %s\n", ing.CreationTimestamp.Format(time.RFC3339)))
	result.WriteString(fmt.Sprintf("Labels:             %s\n", formatLabels(ing.Labels)))
	result.WriteString(fmt.Sprintf("Annotations:        %s\n", formatLabels(ing.Annotations)))

	// Ingress Class
	if ing.Spec.IngressClassName != nil {
		result.WriteString(fmt.Sprintf("IngressClass:       %s\n", *ing.Spec.IngressClassName))
	} else {
		result.WriteString("IngressClass:       <none>\n")
	}

	// TLS配置
	if len(ing.Spec.TLS) > 0 {
		result.WriteString("\nTLS:\n")
		for i, tls := range ing.Spec.TLS {
			result.WriteString(fmt.Sprintf("  TLS-%d:\n", i+1))
			if tls.SecretName != "" {
				result.WriteString(fmt.Sprintf("    SecretName: %s\n", tls.SecretName))
			}
			if len(tls.Hosts) > 0 {
				result.WriteString(fmt.Sprintf("    Hosts:      %s\n", strings.Join(tls.Hosts, ", ")))
			}
		}
	}

	// 规则配置
	if len(ing.Spec.Rules) > 0 {
		result.WriteString("\nRules:\n")
		for i, rule := range ing.Spec.Rules {
			result.WriteString(fmt.Sprintf("  Rule-%d:\n", i+1))
			if rule.Host != "" {
				result.WriteString(fmt.Sprintf("    Host: %s\n", rule.Host))
			}
			if rule.HTTP != nil && len(rule.HTTP.Paths) > 0 {
				result.WriteString("    HTTP Paths:\n")
				for j, path := range rule.HTTP.Paths {
					result.WriteString(fmt.Sprintf("      Path-%d:\n", j+1))
					result.WriteString(fmt.Sprintf("        Path:     %s\n", path.Path))
					result.WriteString(fmt.Sprintf("        PathType: %s\n", string(*path.PathType)))
					result.WriteString("        Backend:\n")
					if path.Backend.Service != nil {
						result.WriteString(fmt.Sprintf("          Service Name: %s\n", path.Backend.Service.Name))
						if path.Backend.Service.Port.Number > 0 {
							result.WriteString(fmt.Sprintf("          Service Port: %d\n", path.Backend.Service.Port.Number))
						} else if path.Backend.Service.Port.Name != "" {
							result.WriteString(fmt.Sprintf("          Service Port: %s\n", path.Backend.Service.Port.Name))
						}
					}
					if path.Backend.Resource != nil {
						result.WriteString(fmt.Sprintf("          Resource: %s/%s\n", 
							path.Backend.Resource.APIGroup, 
							path.Backend.Resource.Kind))
						result.WriteString(fmt.Sprintf("          Resource Name: %s\n", path.Backend.Resource.Name))
					}
				}
			}
		}
	}

	// 默认后端
	if ing.Spec.DefaultBackend != nil {
		result.WriteString("\nDefault Backend:\n")
		if ing.Spec.DefaultBackend.Service != nil {
			result.WriteString(fmt.Sprintf("  Service Name: %s\n", ing.Spec.DefaultBackend.Service.Name))
			if ing.Spec.DefaultBackend.Service.Port.Number > 0 {
				result.WriteString(fmt.Sprintf("  Service Port: %d\n", ing.Spec.DefaultBackend.Service.Port.Number))
			} else if ing.Spec.DefaultBackend.Service.Port.Name != "" {
				result.WriteString(fmt.Sprintf("  Service Port: %s\n", ing.Spec.DefaultBackend.Service.Port.Name))
			}
		}
		if ing.Spec.DefaultBackend.Resource != nil {
			result.WriteString(fmt.Sprintf("  Resource: %s/%s\n", 
				ing.Spec.DefaultBackend.Resource.APIGroup, 
				ing.Spec.DefaultBackend.Resource.Kind))
			result.WriteString(fmt.Sprintf("  Resource Name: %s\n", ing.Spec.DefaultBackend.Resource.Name))
		}
	}

	// 状态
	result.WriteString("\nStatus:\n")
	if len(ing.Status.LoadBalancer.Ingress) > 0 {
		result.WriteString("  LoadBalancer Ingress:\n")
		for _, lbi := range ing.Status.LoadBalancer.Ingress {
			if lbi.IP != "" {
				result.WriteString(fmt.Sprintf("    IP: %s\n", lbi.IP))
			}
			if lbi.Hostname != "" {
				result.WriteString(fmt.Sprintf("    Hostname: %s\n", lbi.Hostname))
			}
		}
	} else {
		result.WriteString("  LoadBalancer Ingress: <none>\n")
	}

	// 获取相关事件
	events, err := getEventsForIngress(ctx, clientset, ing)
	if err == nil && len(events.Items) > 0 {
		result.WriteString("\nEvents:\n")
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

// CreateIngressTool 创建Ingress的工具函数
func CreateIngressTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ingressName := request.Params.Arguments["ingress_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}
	
	host, _ := request.Params.Arguments["host"].(string)
	serviceName, _ := request.Params.Arguments["service_name"].(string)
	servicePort, _ := request.Params.Arguments["service_port"].(float64)
	if servicePort == 0 {
		servicePort = 80 // 默认端口
	}
	
	pathType, _ := request.Params.Arguments["path_type"].(string)
	if pathType == "" {
		pathType = "Prefix" // 默认路径类型
	}
	
	path, _ := request.Params.Arguments["path"].(string)
	if path == "" {
		path = "/" // 默认路径
	}
	
	ingressClassName, _ := request.Params.Arguments["ingress_class_name"].(string)
	
	tlsEnabled, _ := request.Params.Arguments["tls_enabled"].(bool)
	tlsSecretName, _ := request.Params.Arguments["tls_secret_name"].(string)

	fmt.Println("ai 正在调用mcp server的tool: create_ingress, ingress_name=", ingressName, 
		", namespace=", namespace, 
		", host=", host, 
		", service_name=", serviceName, 
		", service_port=", servicePort)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 检查服务是否存在
	_, err = clientset.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("服务 %s 在命名空间 %s 中不存在: %v", serviceName, namespace, err)), err
	}

	// 创建Ingress对象
	pathTypeValue := networkingv1.PathType(pathType)
	portNumber := int32(servicePort)
	
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressName,
			Namespace: namespace,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     path,
									PathType: &pathTypeValue,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: serviceName,
											Port: networkingv1.ServiceBackendPort{
												Number: portNumber,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	
	// 设置Ingress Class
	if ingressClassName != "" {
		ingress.Spec.IngressClassName = &ingressClassName
	}
	
	// 设置TLS
	if tlsEnabled && host != "" {
		if tlsSecretName == "" {
			tlsSecretName = ingressName + "-tls"
		}
		
		ingress.Spec.TLS = []networkingv1.IngressTLS{
			{
				Hosts:      []string{host},
				SecretName: tlsSecretName,
			},
		}
	}

	// 创建Ingress
	createdIngress, err := clientset.NetworkingV1().Ingresses(namespace).Create(ctx, ingress, metav1.CreateOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Ingress失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("Ingress %s 在命名空间 %s 中创建成功", createdIngress.Name, createdIngress.Namespace)), nil
}

// UpdateIngressTool 更新Ingress的工具函数
func UpdateIngressTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ingressName := request.Params.Arguments["ingress_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}
	
	host, hostProvided := request.Params.Arguments["host"].(string)
	serviceName, serviceNameProvided := request.Params.Arguments["service_name"].(string)
	servicePort, servicePortProvided := request.Params.Arguments["service_port"].(float64)
	
	pathType, pathTypeProvided := request.Params.Arguments["path_type"].(string)
	path, pathProvided := request.Params.Arguments["path"].(string)
	
	ingressClassName, ingressClassNameProvided := request.Params.Arguments["ingress_class_name"].(string)
	
	tlsEnabled, tlsEnabledProvided := request.Params.Arguments["tls_enabled"].(bool)
	tlsSecretName, tlsSecretNameProvided := request.Params.Arguments["tls_secret_name"].(string)

	fmt.Println("ai 正在调用mcp server的tool: update_ingress, ingress_name=", ingressName, ", namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 获取现有Ingress
	ingress, err := clientset.NetworkingV1().Ingresses(namespace).Get(ctx, ingressName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取Ingress %s 失败: %v", ingressName, err)), err
	}

	// 检查是否需要更新服务
	if serviceNameProvided && serviceName != "" {
		// 检查服务是否存在
		_, err = clientset.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("服务 %s 在命名空间 %s 中不存在: %v", serviceName, namespace, err)), err
		}
	}

	// 更新规则
	if len(ingress.Spec.Rules) > 0 && ingress.Spec.Rules[0].HTTP != nil && len(ingress.Spec.Rules[0].HTTP.Paths) > 0 {
		// 更新主机
		if hostProvided {
			ingress.Spec.Rules[0].Host = host
		}
		
		// 更新路径
		if pathProvided {
			ingress.Spec.Rules[0].HTTP.Paths[0].Path = path
		}
		
		// 更新路径类型
		if pathTypeProvided && pathType != "" {
			pathTypeValue := networkingv1.PathType(pathType)
			ingress.Spec.Rules[0].HTTP.Paths[0].PathType = &pathTypeValue
		}
		
		// 更新服务
		if serviceNameProvided && serviceName != "" {
			ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name = serviceName
		}
		
		// 更新服务端口
		if servicePortProvided && servicePort > 0 {
			ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number = int32(servicePort)
		}
	}
	
	// 更新Ingress Class
	if ingressClassNameProvided {
		if ingressClassName != "" {
			ingress.Spec.IngressClassName = &ingressClassName
		} else {
			ingress.Spec.IngressClassName = nil
		}
	}
	
	// 更新TLS
	if tlsEnabledProvided {
		if tlsEnabled {
			// 获取主机
			host := ""
			if len(ingress.Spec.Rules) > 0 {
				host = ingress.Spec.Rules[0].Host
			}
			
			if host != "" {
				secretName := ingressName + "-tls"
				if tlsSecretNameProvided && tlsSecretName != "" {
					secretName = tlsSecretName
				} else if len(ingress.Spec.TLS) > 0 && ingress.Spec.TLS[0].SecretName != "" {
					secretName = ingress.Spec.TLS[0].SecretName
				}
				
				ingress.Spec.TLS = []networkingv1.IngressTLS{
					{
						Hosts:      []string{host},
						SecretName: secretName,
					},
				}
			}
		} else {
			ingress.Spec.TLS = nil
		}
	} else if tlsSecretNameProvided && tlsSecretName != "" && len(ingress.Spec.TLS) > 0 {
		ingress.Spec.TLS[0].SecretName = tlsSecretName
	}

	// 更新Ingress
	updatedIngress, err := clientset.NetworkingV1().Ingresses(namespace).Update(ctx, ingress, metav1.UpdateOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("更新Ingress失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("Ingress %s 在命名空间 %s 中更新成功", updatedIngress.Name, updatedIngress.Namespace)), nil
}

// DeleteIngressTool 删除Ingress的工具函数
func DeleteIngressTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ingressName := request.Params.Arguments["ingress_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: delete_ingress, ingress_name=", ingressName, ", namespace=", namespace)

	// 创建K8s客户端
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// 删除Ingress
	err = clientset.NetworkingV1().Ingresses(namespace).Delete(ctx, ingressName, metav1.DeleteOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("删除Ingress失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("Ingress %s 在命名空间 %s 中已删除", ingressName, namespace)), nil
}

// 辅助函数：获取Ingress相关事件
func getEventsForIngress(ctx context.Context, clientset *kubernetes.Clientset, ing *networkingv1.Ingress) (*corev1.EventList, error) {
	fieldSelector := fmt.Sprintf("involvedObject.kind=Ingress,involvedObject.name=%s,involvedObject.namespace=%s",
		ing.Name, ing.Namespace)
	return clientset.CoreV1().Events(ing.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
}
