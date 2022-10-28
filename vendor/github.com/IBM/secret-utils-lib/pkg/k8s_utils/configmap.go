package k8s_utils

import (
	"context"
	"fmt"

	"github.com/IBM/secret-utils-lib/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetConfigMapData ...
func GetConfigMapData(kc KubernetesClient, configMapName, dataName string) (string, error) {

	clientset := kc.GetClientSet()
	namespace := kc.GetNameSpace()

	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	data, ok := cm.Data[dataName]
	if !ok {
		return "", utils.Error{Description: fmt.Sprintf(utils.ErrEmptyConfigMapData, dataName, configMapName)}
	}

	return data, nil
}
