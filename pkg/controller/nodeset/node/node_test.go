package node

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/golang/glog"
	nodesetfake "github.com/kube-node/nodeset/pkg/client/clientset/versioned/fake"
	nodesetinformers "github.com/kube-node/nodeset/pkg/client/informers/externalversions"
	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
)

func newFakeController(crdObjects []runtime.Object, coreObjects []runtime.Object) *Controller {
	nodesetClient := nodesetfake.NewSimpleClientset(crdObjects...)
	kubeClient := kubefake.NewSimpleClientset(coreObjects...)

	nodesetInformers := nodesetinformers.NewSharedInformerFactory(nodesetClient, 1*time.Minute)
	coreInformers := coreinformers.NewSharedInformerFactory(kubeClient, 1*time.Minute)

	c, err := New("test", kubeClient, nodesetClient, nodesetInformers.Nodeset().V1alpha1().NodeSets(), nodesetInformers.Nodeset().V1alpha1().NodeClasses(), coreInformers.Core().V1().Nodes())
	if err != nil {
		glog.Fatal(err)
	}

	nodesetInformers.Start(wait.NeverStop)
	coreInformers.Start(wait.NeverStop)

	cache.WaitForCacheSync(wait.NeverStop, c.nodesetSynced, c.nodeSynced, c.nodeClassSynced)
	return c
}

func TestController_CreateNode(t *testing.T) {
	tests := []struct {
		name              string
		existingNodeclass *v1alpha1.NodeClass
		wantNodeClass     *v1alpha1.NodeClass
		nodeset           *v1alpha1.NodeSet
	}{
		{
			name: "successfully create simple node",
			existingNodeclass: &v1alpha1.NodeClass{
				ObjectMeta:     metav1.ObjectMeta{Name: "testclass1"},
				NodeLabels:     map[string]string{"foo": "1", "bar": "2"},
				NodeController: "testcontroller",
				Config:         runtime.RawExtension{Raw: []byte("{\"Flags\":{\"--flag1\":\"flag1\",\"--flag2\":\"flag2\"},\"Name\":\"testconfigmap\",\"OS\":{\"Commands\":[\"echo 'foo'\"]}}")},
			},
			wantNodeClass: &v1alpha1.NodeClass{
				ObjectMeta:     metav1.ObjectMeta{Name: mergedNodeSetNamePrefix + "testnoteset1"},
				NodeLabels:     map[string]string{"foo": "1", "bar": "2"},
				NodeController: "testcontroller",
				Config:         runtime.RawExtension{Raw: []byte("{\"Flags\":{\"--flag1\":\"flag1\",\"--flag2\":\"flag2\"},\"Name\":\"testconfigmap\",\"OS\":{\"Commands\":[\"echo 'foo'\"]}}")},
			},
			nodeset: &v1alpha1.NodeSet{
				ObjectMeta: metav1.ObjectMeta{Name: "testnoteset1"},
				Spec: v1alpha1.NodeSetSpec{
					NodeClass: "testclass1",
				},
			},
		},
		{
			name: "successfully create node with config override",
			existingNodeclass: &v1alpha1.NodeClass{
				ObjectMeta:     metav1.ObjectMeta{Name: "testclass1"},
				NodeLabels:     map[string]string{"foo": "1", "bar": "2"},
				NodeController: "testcontroller",
				Config:         runtime.RawExtension{Raw: []byte("{\"Flags\":{\"--flag1\":\"flag1\",\"--flag2\":\"flag2\"},\"Name\":\"testconfigmap\",\"OS\":{\"Commands\":[\"echo 'foo'\"]}}")},
			},
			wantNodeClass: &v1alpha1.NodeClass{
				ObjectMeta:     metav1.ObjectMeta{Name: mergedNodeSetNamePrefix + "testnoteset1"},
				NodeLabels:     map[string]string{"foo": "1", "bar": "2"},
				NodeController: "testcontroller",
				Config:         runtime.RawExtension{Raw: []byte("{\"Flags\":{\"--flag1\":\"override\",\"--flag2\":\"override\"},\"Name\":\"testconfigmap\",\"OS\":{\"Commands\":[\"echo 'foo'\"]}}")},
			},
			nodeset: &v1alpha1.NodeSet{
				ObjectMeta: metav1.ObjectMeta{Name: "testnoteset1"},
				Spec: v1alpha1.NodeSetSpec{
					NodeClass: "testclass1",
					Config:    runtime.RawExtension{Raw: []byte("{\"Flags\":{\"--flag1\":\"override\",\"--flag2\":\"override\"}}")},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := newFakeController([]runtime.Object{test.existingNodeclass}, []runtime.Object{})
			node, err := c.createNode(test.nodeset)
			if err != nil {
				t.Fatal(err)
			}

			for k, v := range test.existingNodeclass.NodeLabels {
				if node.Labels[k] != v {
					t.Errorf("node is missing label %s=%s", k, v)
				}
			}
			if node.Labels[kubeHostnameLabelKey] != node.Name {
				t.Errorf("node is missing hostname label %s=%s", kubeHostnameLabelKey, node.Name)
			}

			if !strings.Contains(node.Name, test.nodeset.Name) {
				t.Error("node name does not contain nodeset name")
			}

			if node.Annotations[v1alpha1.NodeClassContentAnnotationKey] == "" {
				t.Error("node does not contain nodeclass in annotation")
			}

			if node.Annotations[v1alpha1.NodeSetGenerationAnnotationKey] == "" {
				t.Error("node does not contain nodeset generation in annotation")
			}

			if len(node.ObjectMeta.OwnerReferences) == 1 {
				if node.ObjectMeta.OwnerReferences[0].Name != test.nodeset.Name {
					t.Error("node has a invalid owner reference")
				}
			} else {
				t.Error("node is missing owner reference")
			}

			haveNodeclassContent, err := base64.StdEncoding.DecodeString(node.Annotations[v1alpha1.NodeClassContentAnnotationKey])
			if err != nil {
				t.Fatal(err)
			}

			haveNodeclass := v1alpha1.NodeClass{}
			err = json.Unmarshal(haveNodeclassContent, &haveNodeclass)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(haveNodeclass.NodeLabels, test.wantNodeClass.NodeLabels) {
				t.Errorf("nodeclass from node annotation has invalid labels")
			}
			if string(haveNodeclass.Config.Raw) != string(test.wantNodeClass.Config.Raw) {
				t.Errorf("nodeclass from node annotation has invalid config")
			}
			if haveNodeclass.NodeController != test.wantNodeClass.NodeController {
				t.Errorf("nodeclass from node annotation has invalid nodecontroller")
			}
		})
	}
}
