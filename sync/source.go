package sync

import (
    "context"
    v1 "k8s.io/client-go/kubernetes/typed/core/v1"
    "perm8s/pkg/apis/perm8s/v1alpha1"
)

// ComputeUserFunc should return a list of all users, uniformly returned through the SyncUser struct.
// It is called whenever the owning SynchronisationSource CRD is synched in the cluster to compute a list
// of all users that should have access to the cluster in this configuration.
// Anyone who is not returned by this function but still has a User linked to them will have their User deleted.
// The Corev1 Client is passed to the function to allow access to secrets or configmaps for configuration
type ComputeUserFunc func(ctx context.Context, source v1alpha1.SynchronisationSource, coreClient *v1.CoreV1Client) (*[]SyncUser, error)

type SyncUser struct {
    Name   string   `json:"name"`
    Groups []string `json:"groups"`
}
