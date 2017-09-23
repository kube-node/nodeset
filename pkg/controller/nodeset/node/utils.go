/*
Copyright 2017 The Kubernetes Authors.

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

package node

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientcorev1 "k8s.io/client-go/listers/core/v1"

	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"github.com/peterbourgon/mergemap"
)

const (
	nodeCacheCheckInterval = 100 * time.Millisecond
	nodeCacheCheckTimeout  = 10 * time.Second
)

func nodeIsRunning(node *corev1.Node) bool {
	var readyConditionStatus = corev1.ConditionFalse
	for _, c := range node.Status.Conditions {
		if c.Type == corev1.NodeReady {
			readyConditionStatus = c.Status
			break
		}
	}
	return readyConditionStatus == corev1.ConditionTrue && node.Status.NodeInfo.KubeletVersion != ""
}

func waitForNodeInCache(name string, lister clientcorev1.NodeLister) error {
	return wait.PollImmediate(nodeCacheCheckInterval, nodeCacheCheckTimeout, func() (done bool, err error) {
		if _, err := lister.Get(name); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

func waitForNodeDeletedInCache(name string, lister clientcorev1.NodeLister) error {
	return wait.PollImmediate(nodeCacheCheckInterval, nodeCacheCheckTimeout, func() (done bool, err error) {
		if _, err := lister.Get(name); err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
}

func convertNodeClassToString(class *v1alpha1.NodeClass) (content, hash string, err error) {
	ncb, err := json.Marshal(class)
	if err != nil {
		return content, hash, err
	}
	content = base64.StdEncoding.EncodeToString(ncb)

	h := md5.New()
	_, err = io.WriteString(h, content)
	if err != nil {
		return content, hash, err
	}
	hash = fmt.Sprintf("%x", h.Sum(nil))

	return content, hash, err
}

func mergeNodeClass(nodeClass *v1alpha1.NodeClass, overrideConfig runtime.RawExtension) (*v1alpha1.NodeClass, error) {
	var err error
	mergedNodeClass := &v1alpha1.NodeClass{
		ObjectMeta:     metav1.ObjectMeta{Name: mergedNodeSetNamePrefix + nodeClass.Name},
		NodeController: nodeClass.NodeController,
		Resources:      nodeClass.Resources,
		NodeLabels:     nodeClass.NodeLabels,
	}

	//Merge config
	var original, overrides, merged map[string]interface{}
	if len(nodeClass.Config.Raw) > 0 {
		err = json.Unmarshal(nodeClass.Config.Raw, &original)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal original nodeclass config: %v", err)
		}
	}
	if len(overrideConfig.Raw) > 0 {
		err = json.Unmarshal(overrideConfig.Raw, &overrides)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal override nodeclass config: %v", err)
		}
	}
	merged = mergemap.Merge(original, overrides)
	mergedNodeClass.Config.Raw, err = json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged nodeclass config: %v", err)
	}

	return mergedNodeClass, nil
}
