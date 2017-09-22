/*
Copyright 2016 The Kubernetes Authors.

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

package gke

import (
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"github.com/golang/glog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gce "google.golang.org/api/compute/v1"
	gke "google.golang.org/api/container/v1"
)

type Cluster struct {
	Project, Zone, Name string
}

func NewGKEService(clusterName string) (*gke.Service, *gce.Service, Cluster, error) {
	// Create Google Compute Engine token.
	var err error
	tokenSource := google.ComputeTokenSource("")
	if len(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")) > 0 {
		tokenSource, err = google.DefaultTokenSource(oauth2.NoContext, gce.ComputeScope)
		if err != nil {
			return nil, nil, Cluster{}, err
		}
	}
	var projectId, zone string
	if len(projectId) == 0 || len(zone) == 0 {
		projectId, zone, err = getProjectAndZone()
		if err != nil {
			return nil, nil, Cluster{}, err
		}
	}
	glog.V(1).Infof("GCE projectId=%s Zone=%s", projectId, zone)

	// Create Google Compute Engine service.
	client := oauth2.NewClient(oauth2.NoContext, tokenSource)

	gceService, err := gce.New(client)
	if err != nil {
		return nil, nil, Cluster{}, err
	}

	gkeService, err := gke.New(client)
	if err != nil {
		return nil, nil, Cluster{}, err
	}

	return gkeService, gceService, Cluster{projectId, zone, clusterName}, nil
}

/*
// Gets all registered node pools
func (m *gceManagerImpl) fetchAllNodePools() error {
	m.assertGKE()

	nodePoolsResponse, err := m.service.Projects.Zones.Clusters.NodePools.List(m.projectId, m.Zone, m.clusterName).Do()
	if err != nil {
		return err
	}

	existingMigs := map[GceRef]struct{}{}

	for _, nodePool := range nodePoolsResponse.NodePools {
		autoprovisioned := strings.Contains(nodePool.Name, nodeAutoprovisioningPrefix)
		autoscaled := nodePool.Autoscaling != nil && nodePool.Autoscaling.Enabled
		if !autoprovisioned && !autoscaled {
			continue
		}
		// format is
		// "https://www.googleapis.com/compute/v1/projects/mwielgus-proj/zones/europe-west1-b/instanceGroupManagers/gke-cluster-1-default-pool-ba78a787-grp"
		for _, igurl := range nodePool.InstanceGroupUrls {
			Project, Zone, name, err := parseGceUrl(igurl, "instanceGroupManagers")
			if err != nil {
				return err
			}
			mig := &Mig{
				GceRef: GceRef{
					Name:    name,
					Zone:    Zone,
					Project: Project,
				},
				gceManager:      m,
				exist:           true,
				autoprovisioned: autoprovisioned,
				nodePoolName:    nodePool.Name,
			}
			existingMigs[mig.GceRef] = struct{}{}

			if autoscaled {
				mig.minSize = int(nodePool.Autoscaling.MinNodeCount)
				mig.maxSize = int(nodePool.Autoscaling.MaxNodeCount)
			} else if autoprovisioned {
				mig.minSize = minAutoprovisionedSize
				mig.maxSize = maxAutoprovisionedSize
			}
			m.RegisterMig(mig)
		}
	}
	for _, mig := range m.getMigs() {
		if _, found := existingMigs[mig.config.GceRef]; !found {
			m.UnregisterMig(mig.config)
		}
	}
	return nil
}

*/

// Code borrowed from gce cloud provider. Reuse the original as soon as it becomes public.
func getProjectAndZone() (string, string, error) {
	result, err := metadata.Get("instance/Zone")
	if err != nil {
		return "", "", err
	}
	parts := strings.Split(result, "/")
	if len(parts) != 4 {
		return "", "", fmt.Errorf("unexpected response: %s", result)
	}
	zone := parts[3]
	projectID, err := metadata.ProjectID()
	if err != nil {
		return "", "", err
	}
	return projectID, zone, nil
}
