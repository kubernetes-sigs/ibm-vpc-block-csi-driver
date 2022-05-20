package k8s_utils

import (
	"context"

	"github.com/IBM/secret-utils-lib/pkg/utils"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetConfigMapData ...
func GetConfigMapData(kc KubernetesClient, configMapName, dataName string) (string, error) {
	kc.logger.Info("Fetching config map")

	clientset := kc.GetClientSet()
	namespace := kc.GetNameSpace()

	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		kc.logger.Error("Error fetching cluster-info configmap", zap.Error(err))
		return "", utils.Error{Description: utils.ErrFetchingClusterConfig, BackendError: err.Error()}
	}

	data, ok := cm.Data[dataName]
	if !ok {
		kc.logger.Error("cluster-config.json is not present")
		return "", utils.Error{Description: utils.ErrEmptyClusterConfig}
	}

	kc.logger.Info("Fetched config map data")
	return data, nil
}
