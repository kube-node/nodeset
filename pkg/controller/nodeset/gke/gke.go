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
	"context"
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

// Cluster contains information about a GKE cluster
type Cluster struct {
	ProjectID, Zone, ID string
}

// NewGKEService returns a client to interact with a GKE cluster
func NewGKEService(clusterID, zone, projectID string) (*gke.Service, *gce.Service, Cluster, error) {
	// Create Google Compute Engine token.
	var err error
	tokenSource := google.ComputeTokenSource("")
	if len(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")) > 0 {
		tokenSource, err = google.DefaultTokenSource(context.Background(), gce.ComputeScope, gke.CloudPlatformScope)
		if err != nil {
			return nil, nil, Cluster{}, err
		}
	}

	// In case its not specified, get it via the metadata api
	if len(projectID) == 0 || len(zone) == 0 {
		projectID, zone, err = getProjectAndZone()
		if err != nil {
			return nil, nil, Cluster{}, err
		}
	}
	glog.V(1).Infof("GCE projectID=%s Zone=%s", projectID, zone)

	// Create Google Compute Engine service.
	client := oauth2.NewClient(context.Background(), tokenSource)

	gceService, err := gce.New(client)
	if err != nil {
		return nil, nil, Cluster{}, err
	}

	gkeService, err := gke.New(client)
	if err != nil {
		return nil, nil, Cluster{}, err
	}

	return gkeService, gceService, Cluster{ProjectID: projectID, Zone: zone, ID: clusterID}, nil
}

// Code borrowed from gce cloud provider. Reuse the original as soon as it becomes public.
// getProjectAndZone loads the project & zone from the metadata api
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
