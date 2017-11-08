/*
Copyright 2014 The Kubernetes Authors.

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

// Package options provides the flags used for the controller manager.
//
package options

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Options is the main context object for the controller manager.
type Options struct {
	Kubeconfig string

	// minResyncPeriod is the resync period in reflectors; will be random between
	// minResyncPeriod and 2*minResyncPeriod.
	MinResyncPeriod metav1.Duration
	// kubeAPIQPS is the QPS to use while talking with kubernetes apiserver.
	KubeAPIQPS float32
	// kubeAPIBurst is the QPS burst to use while talking with kubernetes apiserver.
	KubeAPIBurst int32
	// ControllerName is name of the NodeSet controller, used to select which NodeSets
	// will be processed by this controller, based on pod's "spec.ControllerName".
	ControllerName string

	// BackendName is the name of the backend to use.
	BackendName string

	// GKEClusterID is the cluster ID for GKE.
	GKEClusterID string

	// GKEClusterZone is the zone of the cluster in GKE
	GKEClusterZone string

	// GKEProjectID is the google project ID
	GKEProjectID string
}

var validBackends = []string{"node", "gke"}

// New creates a new NodeSetControllerServer with a default config.
func New() *Options {
	s := Options{
		MinResyncPeriod: metav1.Duration{Duration: 5 * time.Minute},
		ControllerName:  "default",
	}
	return &s
}

// AddFlags adds flags for a specific NodeSetControllerServer to the specified FlagSet.
func (s *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.Kubeconfig, "kubeconfig", s.Kubeconfig, "Path to kubeconfig file with authorization and master location information.")
	fs.DurationVar(&s.MinResyncPeriod.Duration, "min-resync-period", s.MinResyncPeriod.Duration, "The resync period in reflectors will be random between MinResyncPeriod and 2*MinResyncPeriod")
	fs.Float32Var(&s.KubeAPIQPS, "kube-api-qps", s.KubeAPIQPS, "QPS to use while talking with kubernetes apiserver")
	fs.Int32Var(&s.KubeAPIBurst, "kube-api-burst", s.KubeAPIBurst, "Burst to use while talking with kubernetes apiserver")
	fs.StringVar(&s.ControllerName, "controller-name", s.ControllerName, "Name of the NodeSet controller, used to select which pods will be processed by this controller, based on pod's \"spec.ControllerName\".")
	fs.StringVar(&s.BackendName, "backend", s.ControllerName, fmt.Sprintf("The backend to use (supported: %s).", strings.Join(validBackends, ", ")))
	fs.StringVar(&s.GKEClusterID, "gke-cluster-id", s.GKEClusterID, "The cluster name in GKE if that backend is enabled.")
	fs.StringVar(&s.GKEClusterZone, "gke-zone", s.GKEClusterZone, "The zone of the cluster in GKE if that backend is enabled.")
	fs.StringVar(&s.GKEProjectID, "gke-project-id", s.GKEProjectID, "The project ID in google if the GKE backend is enabled.")
}

// Validate is used to validate the options and config before launching.
func (s *Options) Validate() error {
	var errs []error

	backends := sets.NewString(validBackends...)
	if !backends.Has(s.BackendName) {
		errs = append(errs, fmt.Errorf("invalid backend name %q, allowed: %s", s.BackendName, strings.Join(validBackends, ", ")))
	}

	if s.BackendName == "gke" {
		if s.GKEClusterID == "" {
			errs = append(errs, errors.New("GKE cluster name is unset"))
		}
	}

	return utilerrors.NewAggregate(errs)
}
