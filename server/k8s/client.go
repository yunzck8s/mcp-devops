package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// CreateK8sClient 创建Kubernetes客户端
func CreateK8sClient() (*kubernetes.Clientset, error) {
	// 尝试获取集群内部配置
	config, err := rest.InClusterConfig()
	if err == nil {
		// 成功获取集群内配置
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, fmt.Errorf("从集群内配置创建客户端失败: %v", err)
		}
		return clientset, nil
	}

	// 如果不在集群内，尝试使用kubeconfig
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		// 使用默认位置
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("获取用户home目录失败: %v", err)
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// 检查kubeconfig文件是否存在
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		return nil, fmt.Errorf("kubeconfig文件 %s 不存在", kubeconfig)
	}

	// 使用kubeconfig创建配置
	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("从kubeconfig构建配置失败: %v", err)
	}

	// 创建客户端
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("创建客户端失败: %v", err)
	}

	return clientset, nil
}
