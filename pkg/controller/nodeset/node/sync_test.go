package node

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
)

func TestController_syncNodeSetVersion(t *testing.T) {
	nodeset := &v1alpha1.NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "testnoteset1",
			Annotations: map[string]string{v1alpha1.NodeClassContentHashAnnotationKey: "old-hash"},
		},
		Spec: v1alpha1.NodeSetSpec{
			NodeClass: "testclass1",
		},
		Status: v1alpha1.NodeSetStatus{
			ObservedGeneration: 0,
		},
	}

	nodeclass := &v1alpha1.NodeClass{
		ObjectMeta:     metav1.ObjectMeta{Name: "testclass1"},
		NodeLabels:     map[string]string{"foo": "1", "bar": "2"},
		NodeController: "testcontroller",
		Config:         runtime.RawExtension{Raw: []byte("")},
	}

	c := newFakeController([]runtime.Object{nodeclass}, []runtime.Object{})
	changedNodeset, err := c.syncNodeSetVersion(nodeset)
	if err != nil {
		t.Fatal(err)
	}
	if changedNodeset == nil {
		t.Fatalf("no changed happened on the nodeset")
	}
	if changedNodeset.Status.ObservedGeneration != 1 {
		t.Errorf("nodeset observedGeneration was not incremented")
	}
	if changedNodeset.Annotations[v1alpha1.NodeClassContentHashAnnotationKey] == "old-hash" {
		t.Errorf("nodeset nodeclass hash annotation was not updated")
	}
}

func TestController_syncNodeSetVersionNotUpdated(t *testing.T) {
	nodeset := &v1alpha1.NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "testnoteset1",
			Annotations: map[string]string{v1alpha1.NodeClassContentHashAnnotationKey: "1f092539fdea9d0fd85c8cda3609ca2a"},
		},
		Spec: v1alpha1.NodeSetSpec{
			NodeClass: "testclass1",
		},
		Status: v1alpha1.NodeSetStatus{
			ObservedGeneration: 1,
		},
	}

	nodeclass := &v1alpha1.NodeClass{
		ObjectMeta:     metav1.ObjectMeta{Name: "testclass1"},
		NodeLabels:     map[string]string{"foo": "1", "bar": "2"},
		NodeController: "testcontroller",
		Config:         runtime.RawExtension{Raw: []byte("")},
	}

	c := newFakeController([]runtime.Object{nodeclass}, []runtime.Object{})
	changedNodeset, err := c.syncNodeSetVersion(nodeset)
	if err != nil {
		t.Fatal(err)
	}
	if changedNodeset != nil {
		t.Fatalf("got changed nodeset although nothing should have changed")
	}
}
