/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clientgoutils

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// quantityStr: 10Gi, 10Mi ...
// storageClassName: nil to use default storageClassName
func (c *ClientGoUtils) CreatePVC(namespace string, name string, labels map[string]string, annotations map[string]string, quantityStr string, storageClassName *string) (*v1.PersistentVolumeClaim, error) {
	q, err := resource.ParseQuantity(quantityStr)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	var persistentVolumeClaim = &v1.PersistentVolumeClaim{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			Selector:    nil,
			Resources: v1.ResourceRequirements{
				Requests: map[v1.ResourceName]resource.Quantity{v1.ResourceStorage: q},
			},
			StorageClassName: storageClassName,
		},
	}
	persistentVolumeClaim.Name = name
	persistentVolumeClaim.Labels = labels
	persistentVolumeClaim.Annotations = annotations

	return c.ClientSet.CoreV1().PersistentVolumeClaims(namespace).Create(context.TODO(), persistentVolumeClaim, metav1.CreateOptions{})
}

func (c *ClientGoUtils) DeletePVC(namespace string, name string) error {
	return errors.Wrap(c.ClientSet.CoreV1().PersistentVolumeClaims(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{}), "")
}

func (c *ClientGoUtils) GetPvcByLabels(ctx context.Context, namespace string, labels map[string]string) ([]v1.PersistentVolumeClaim, error) {
	var labelSelector string
	if len(labels) > 0 {
		for key, val := range labels {
			labelSelector = fmt.Sprintf("%s,%s=%s", labelSelector, key, val)
		}
	}
	labelSelector = strings.TrimPrefix(labelSelector, ",")

	list, err := c.ClientSet.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return list.Items, nil
}

func (c *ClientGoUtils) GetPvcByName(ctx context.Context, namespace string, name string) (*v1.PersistentVolumeClaim, error) {
	pvc, err := c.ClientSet.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return pvc, nil
}
