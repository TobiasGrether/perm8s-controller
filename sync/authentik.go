package sync

import (
    "context"
    "errors"
    "fmt"
    authentik "goauthentik.io/api/v3"
    errors2 "k8s.io/apimachinery/pkg/api/errors"
    v3 "k8s.io/apimachinery/pkg/apis/meta/v1"
    v1 "k8s.io/client-go/kubernetes/typed/core/v1"
    "k8s.io/klog/v2"
    "perm8s/pkg/apis/perm8s/v1alpha1"
    "slices"
)

func ComputeAuthentikUsers(ctx context.Context, source v1alpha1.SynchronisationSource, coreClient *v1.CoreV1Client) (*[]SyncUser, error) {
	sourceConfig := source.Spec.Authentik
    logger := klog.FromContext(ctx).WithValues("provider", "authentik")
	
	if sourceConfig == nil {
		return nil, errors.New("cannot sync from authentik: No Authentik configuration provided")
	}
	

	secret, err := coreClient.Secrets(source.Namespace).Get(ctx, sourceConfig.SecretName, v3.GetOptions{})

    if errors2.IsNotFound(err) {
        logger.Error(err, "Cannot sync from Authentik source, secret cannot be found", "secretName", sourceConfig.SecretName, "namespace", source.Namespace)
        return nil, errors.New(fmt.Sprintf("Cannot sync from Authentik source, secret %v cannot be found in namespace %v", sourceConfig.SecretName, source.Namespace))
    }
    

    if err != nil {
        logger.Error(err, "Error while doing sync from Authentik source")

        return nil, err
    }

    config := authentik.NewConfiguration()
    config.Host = sourceConfig.URL
    config.Scheme = sourceConfig.Scheme
    config.AddDefaultHeader("Authorization", "Bearer "+string(secret.Data["token"]))
    client := authentik.NewAPIClient(config)
    list, _, err := client.CoreApi.CoreUsersList(context.Background()).PageSize(-1).Execute()

    if err != nil {
        logger.Error(err, "User list request failed for authentik instance")
        return nil, err
    }

    var allowedUsers []SyncUser

    for _, user := range list.Results {
        for _, allowedGroup := range sourceConfig.RequiredGroups {
            if slices.Contains(user.Groups, allowedGroup) {
                allowedUsers = append(allowedUsers, SyncUser{
                    Name: user.Name,
                    Groups: user.Groups,
                })
                break
            }
        }
    }
    
    return &allowedUsers, nil
}