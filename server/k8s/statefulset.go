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
	// Assuming CreateK8sClient and formatAge/formatLabels etc. are accessible
)

// ListStatefulSetsTool lists StatefulSets in a given namespace.
func ListStatefulSetsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: list_statefulsets, namespace=", namespace)

	// Create K8s client
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// Get StatefulSet list
	statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取StatefulSet列表失败: %v", err)), err
	}

	// Format output
	var result strings.Builder
	result.WriteString(fmt.Sprintf("命名空间: %s\n\n", namespace))
	result.WriteString("NAME\tREADY\tAGE\tCONTAINERS\tIMAGES\n") // Adjusted columns for StatefulSet

	for _, sts := range statefulsets.Items {
		// Calculate age
		age := formatAge(sts.CreationTimestamp.Time)

		// Get container and image info
		var containers []string
		var images []string
		for _, container := range sts.Spec.Template.Spec.Containers {
			containers = append(containers, container.Name)
			images = append(images, container.Image)
		}

		// StatefulSet status provides ReadyReplicas
		readyReplicas := sts.Status.ReadyReplicas
		desiredReplicas := *sts.Spec.Replicas // StatefulSet spec has replicas pointer

		result.WriteString(fmt.Sprintf("%s\t%d/%d\t%s\t%s\t%s\n",
			sts.Name,
			readyReplicas,
			desiredReplicas,
			age,
			strings.Join(containers, ","),
			strings.Join(images, ",")))
	}

	return mcp.NewToolResultText(result.String()), nil
}

// DescribeStatefulSetTool gets details of a specific StatefulSet.
func DescribeStatefulSetTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	statefulSetName := request.Params.Arguments["statefulset_name"].(string) // Use statefulset_name
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: describe_statefulset, statefulset_name=", statefulSetName, ", namespace=", namespace)

	// Create K8s client
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// Get StatefulSet details
	sts, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取StatefulSet详情失败: %v", err)), err
	}

	// Format output
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Name:               %s\n", sts.Name))
	result.WriteString(fmt.Sprintf("Namespace:          %s\n", sts.Namespace))
	result.WriteString(fmt.Sprintf("CreationTimestamp:  %s\n", sts.CreationTimestamp.Format(time.RFC3339)))
	result.WriteString(fmt.Sprintf("Labels:             %s\n", formatLabels(sts.Labels)))
	result.WriteString(fmt.Sprintf("Annotations:        %s\n", formatLabels(sts.Annotations)))
	result.WriteString(fmt.Sprintf("Selector:           %s\n", formatSelector(sts.Spec.Selector)))
	result.WriteString(fmt.Sprintf("Replicas:           %d desired | %d current | %d ready | %d updated\n", // StatefulSet status fields differ
		*sts.Spec.Replicas,
		sts.Status.Replicas,
		sts.Status.ReadyReplicas,
		sts.Status.UpdatedReplicas))
	result.WriteString(fmt.Sprintf("UpdateStrategy:     %s\n", sts.Spec.UpdateStrategy.Type))
	if sts.Spec.UpdateStrategy.RollingUpdate != nil && sts.Spec.UpdateStrategy.Type == appsv1.RollingUpdateStatefulSetStrategyType {
		partition := int32(0)
		if sts.Spec.UpdateStrategy.RollingUpdate.Partition != nil {
			partition = *sts.Spec.UpdateStrategy.RollingUpdate.Partition
		}
		result.WriteString(fmt.Sprintf("RollingUpdateStrategy: Partition: %d\n", partition))
	}
	result.WriteString(fmt.Sprintf("PodManagementPolicy: %s\n", sts.Spec.PodManagementPolicy))
	result.WriteString(fmt.Sprintf("ServiceName:        %s\n", sts.Spec.ServiceName))

	// Pod Template
	result.WriteString("\nPod Template:\n")
	result.WriteString(fmt.Sprintf("  Labels:       %s\n", formatLabels(sts.Spec.Template.Labels)))
	// Add other Pod Template details if needed (e.g., ServiceAccount, NodeSelector)

	// Containers
	result.WriteString("  Containers:\n")
	for _, container := range sts.Spec.Template.Spec.Containers {
		result.WriteString(fmt.Sprintf("   %s:\n", container.Name))
		result.WriteString(fmt.Sprintf("    Image:      %s\n", container.Image))
		result.WriteString(fmt.Sprintf("    Ports:      %s\n", formatContainerPorts(container.Ports))) // Assuming helper exists
		result.WriteString(fmt.Sprintf("    Host Ports: %s\n", formatContainerHostPorts(container.Ports))) // Assuming helper exists

		if len(container.Command) > 0 {
			result.WriteString(fmt.Sprintf("    Command:    %s\n", strings.Join(container.Command, " ")))
		}
		if len(container.Args) > 0 {
			result.WriteString(fmt.Sprintf("    Args:       %s\n", strings.Join(container.Args, " ")))
		}

		result.WriteString(fmt.Sprintf("    Environment: %s\n", formatEnvVars(container.Env)))         // Assuming helper exists
		result.WriteString(fmt.Sprintf("    Mounts:      %s\n", formatVolumeMounts(container.VolumeMounts))) // Assuming helper exists

		// Resources (similar to Deployment)
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
		// Probes (similar to Deployment)
		if container.LivenessProbe != nil {
			result.WriteString("    Liveness Probe:  Defined\n") // Simplified, could add details
		}
		if container.ReadinessProbe != nil {
			result.WriteString("    Readiness Probe: Defined\n") // Simplified, could add details
		}
	}

	// Volumes (similar to Deployment, but VolumeClaimTemplates are specific to STS)
	if len(sts.Spec.Template.Spec.Volumes) > 0 {
		result.WriteString("  Volumes:\n")
		// Format volumes similar to Deployment's DescribeDeploymentTool
		for _, volume := range sts.Spec.Template.Spec.Volumes {
			result.WriteString(fmt.Sprintf("   %s: <Volume details based on type>\n", volume.Name)) // Add detailed formatting later if needed
		}
	}

	// Volume Claim Templates
	if len(sts.Spec.VolumeClaimTemplates) > 0 {
		result.WriteString("\nVolume Claim Templates:\n")
		for _, pvc := range sts.Spec.VolumeClaimTemplates {
			result.WriteString(fmt.Sprintf("  Name: %s\n", pvc.Name))
			result.WriteString(fmt.Sprintf("    Labels: %s\n", formatLabels(pvc.Labels)))
			result.WriteString(fmt.Sprintf("    Annotations: %s\n", formatLabels(pvc.Annotations)))
			result.WriteString(fmt.Sprintf("    StorageClass: %s\n", *pvc.Spec.StorageClassName))
			result.WriteString(fmt.Sprintf("    AccessModes: %v\n", pvc.Spec.AccessModes))
			if storage, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
				result.WriteString(fmt.Sprintf("    Capacity: %s\n", storage.String()))
			}
		}
	}

	// Get related events
	events, err := getEventsForStatefulSet(ctx, clientset, sts) // Use STS specific helper
	if err == nil && len(events.Items) > 0 {
		result.WriteString("\nEvents:\n")
		result.WriteString("LAST SEEN\tTYPE\tREASON\tOBJECT\tMESSAGE\n")
		for _, event := range events.Items {
			age := formatAge(event.LastTimestamp.Time) // Assuming helper exists
			result.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s/%s\t%s\n", // Adjusted object format
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

// ScaleStatefulSetTool scales a StatefulSet.
func ScaleStatefulSetTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	statefulSetName := request.Params.Arguments["statefulset_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	replicas, ok := request.Params.Arguments["replicas"].(float64)
	if !ok {
		return mcp.NewToolResultText("缺少必要的参数: replicas"), fmt.Errorf("缺少replicas参数")
	}
	replicasInt := int32(replicas)

	fmt.Println("ai 正在调用mcp server的tool: scale_statefulset, statefulset_name=", statefulSetName, ", namespace=", namespace, ", replicas=", replicasInt)

	// Create K8s client
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// Get current StatefulSet
	sts, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取StatefulSet失败: %v", err)), err
	}

	// Record original replicas
	oldReplicas := *sts.Spec.Replicas

	// Update replicas
	sts.Spec.Replicas = &replicasInt

	// Apply update
	_, err = clientset.AppsV1().StatefulSets(namespace).Update(ctx, sts, metav1.UpdateOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("扩缩StatefulSet失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("已将StatefulSet %s 在命名空间 %s 中的副本数从 %d 扩缩到 %d",
		statefulSetName, namespace, oldReplicas, replicasInt)), nil
}

// RestartStatefulSetTool restarts a StatefulSet by patching the pod template annotation.
func RestartStatefulSetTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	statefulSetName := request.Params.Arguments["statefulset_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: restart_statefulset, statefulset_name=", statefulSetName, ", namespace=", namespace)

	// Create K8s client
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// Get current StatefulSet
	sts, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取StatefulSet失败: %v", err)), err
	}

	// Add or update restart annotation in the Pod template
	if sts.Spec.Template.Annotations == nil {
		sts.Spec.Template.Annotations = make(map[string]string)
	}
	sts.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	// Apply update
	// Note: For StatefulSets, patching might be preferred over Update to avoid conflicts,
	// but Update works for this annotation change. Consider using Patch for more complex updates.
	_, err = clientset.AppsV1().StatefulSets(namespace).Update(ctx, sts, metav1.UpdateOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("重启StatefulSet失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("StatefulSet %s 在命名空间 %s 中已开始重启", statefulSetName, namespace)), nil
}

// DeleteStatefulSetTool deletes a StatefulSet.
func DeleteStatefulSetTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	statefulSetName := request.Params.Arguments["statefulset_name"].(string)
	namespace, _ := request.Params.Arguments["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	fmt.Println("ai 正在调用mcp server的tool: delete_statefulset, statefulset_name=", statefulSetName, ", namespace=", namespace)

	// Create K8s client
	clientset, err := CreateK8sClient()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("创建Kubernetes客户端失败: %v", err)), err
	}

	// Delete StatefulSet
	err = clientset.AppsV1().StatefulSets(namespace).Delete(ctx, statefulSetName, metav1.DeleteOptions{})
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("删除StatefulSet失败: %v", err)), err
	}

	return mcp.NewToolResultText(fmt.Sprintf("StatefulSet %s 在命名空间 %s 中已删除", statefulSetName, namespace)), nil
}

// --- Helper functions ---

// getEventsForStatefulSet fetches events related to a specific StatefulSet.
func getEventsForStatefulSet(ctx context.Context, clientset *kubernetes.Clientset, sts *appsv1.StatefulSet) (*corev1.EventList, error) {
	fieldSelector := fmt.Sprintf("involvedObject.kind=StatefulSet,involvedObject.name=%s,involvedObject.namespace=%s",
		sts.Name, sts.Namespace)
	listOptions := metav1.ListOptions{
		FieldSelector: fieldSelector,
	}
	return clientset.CoreV1().Events(sts.Namespace).List(ctx, listOptions)
}


// Note: Assuming formatAge, formatLabels, formatContainerPorts, formatContainerHostPorts,
// formatEnvVars, formatVolumeMounts, CreateK8sClient are defined elsewhere (e.g., pod.go, client.go, or a shared util file)
