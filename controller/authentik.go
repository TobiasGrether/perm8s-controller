package controller

import (
    "context"
    "fmt"
    v2 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    v3 "k8s.io/apimachinery/pkg/apis/meta/v1"
    utilruntime "k8s.io/apimachinery/pkg/util/runtime"
    "k8s.io/client-go/tools/cache"
    "k8s.io/klog/v2"
    v1alpha2 "perm8s/pkg/apis/perm8s/v1alpha1"
    authentik "goauthentik.io/api/v3"
    "reflect"
    "regexp"
    "slices"
    "strings"
)

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

func (c *Controller) enqueueAuthentik(obj interface{}) {
    if objectRef, err := cache.ObjectToName(obj); err != nil {
        utilruntime.HandleError(err)
        return
    } else {
        c.authentikWorkqueue.Add(objectRef)
    }
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runAuthentikWorker(ctx context.Context) {
    for c.processNextAuthentikWorkItem(ctx) {
    }
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextAuthentikWorkItem(ctx context.Context) bool {
    objRef, shutdown := c.authentikWorkqueue.Get()
    logger := klog.FromContext(ctx)

    if shutdown {
        return false
    }

    // We call Done at the end of this func so the workqueue knows we have
    // finished processing this item. We also must remember to call Forget
    // if we do not want this work item being re-queued. For example, we do
    // not call Forget if a transient error occurs, instead the item is
    // put back on the workqueue and attempted again after a back-off
    // period.
    defer c.authentikWorkqueue.Done(objRef)

    // Run the syncHandler, passing it the structured reference to the object to be synced.
    err := c.syncAuthentikHandler(ctx, objRef.(cache.ObjectName))
    if err == nil {
        // If no error occurs then we Forget this item so it does not
        // get queued again until another change happens.
        c.authentikWorkqueue.Forget(objRef)
        logger.Info("Successfully synced", "objectName", objRef)
        return true
    }
    // there was a failure so be sure to report it.  This method allows for
    // pluggable error handling which can be used for things like
    // cluster-monitoring.
    logger.Error(err, "Error syncing; requeuing for later retry", "objectReference", objRef)
    // since we failed, we should requeue the item to work on later.  This
    // method will add a backoff to avoid hotlooping on particular items
    // (they're probably still not going to work right away) and overall
    // controller protection (everything I've done is broken, this controller
    // needs to calm down or it can starve other useful work) cases.
    c.authentikWorkqueue.AddRateLimited(objRef)
    return true
}

func (c *Controller) syncAuthentikHandler(ctx context.Context, objectRef cache.ObjectName) error {
    logger := klog.LoggerWithValues(klog.FromContext(ctx), "objectRef", objectRef)

    authentikConfig, err := c.authentikLister.AuthentikSynchronisationSources(objectRef.Namespace).Get(objectRef.Name)
    if err != nil {
        if errors.IsNotFound(err) {
            logger.Error(err, "AuthentikConfigurationSource in workqueue no longer exists")
            return nil
        }

        return err
    }

    secret, err := c.apiCLient.Secrets(authentikConfig.Namespace).Get(ctx, authentikConfig.Spec.SecretName, v3.GetOptions{})

    if errors.IsNotFound(err) {
        logger.Info("Cannot sync from Authentik source, secret cannot be found", "secretName", authentikConfig.Spec.SecretName, "namespace", authentikConfig.Namespace)
        c.recorder.Event(authentikConfig, v2.EventTypeWarning, "Sync failed", fmt.Sprintf("The secret '%v' cannot be found in namespace %v", authentikConfig.Spec.SecretName, authentikConfig.Namespace))
        return nil
    }

    if err != nil {
        logger.Error(err, "Error while doing sync from Authentik source")

        return err
    }

    config := authentik.NewConfiguration()
    config.Host = authentikConfig.Spec.URL
    config.Scheme = authentikConfig.Spec.Scheme
    config.AddDefaultHeader("Authorization", "Bearer "+string(secret.Data["token"]))
    client := authentik.NewAPIClient(config)
    list, _, err := client.CoreApi.CoreUsersList(context.Background()).PageSize(-1).Execute()

    if err != nil {
        logger.Error(err, "User list request failed for authentik instance")
        return err
    }

    var apiUsers []authentik.User

    for _, user := range list.Results {
        for _, allowedGroup := range authentikConfig.Spec.RequiredGroups {
            if slices.Contains(user.Groups, allowedGroup) {
                apiUsers = append(apiUsers, user)
                break
            }
        }
    }

    for _, user := range apiUsers {
        identifier := GetIdentifier(user.Username)

        var groups []string

        for _, g := range user.Groups {
            if groupName, ok := authentikConfig.Spec.GroupMappings[g]; ok {
                groups = append(groups, groupName)
            }
        }

        desiredUser := c.GetUserFromAuthentikUser(identifier, user.Username, authentikConfig.Namespace, groups, authentikConfig)

        currentUser, err := c.clientSet.Perm8sV1alpha1().Users(authentikConfig.Namespace).Get(ctx, desiredUser.Name, v3.GetOptions{})

        if errors.IsNotFound(err) {
            logger.Info("User account does not exist for external identity user yet, creating new")
            _, err := c.clientSet.Perm8sV1alpha1().Users(authentikConfig.Namespace).Create(ctx, desiredUser, v3.CreateOptions{})

            if err != nil {
                return err
            }
            c.recorder.Event(authentikConfig, v2.EventTypeNormal, SuccessSynced, "User Account created for external users")
            continue
        }

        if err != nil {
            return err
        }

        if !reflect.DeepEqual(currentUser.Spec, desiredUser.Spec) {
            logger.Info("External User is out of sync, resynching")
            _, err = c.clientSet.Perm8sV1alpha1().Users(authentikConfig.Namespace).Update(ctx, desiredUser, v3.UpdateOptions{})

            if err != nil {
                return err
            }
        }
    }

    // finally, we need to make sure no users exist that are not part of the target group anymore
    result, err := c.clientSet.Perm8sV1alpha1().Users(authentikConfig.Namespace).List(ctx, v3.ListOptions{})

    if err != nil {
        return err
    }

    for _, user := range result.Items {
        if user.Spec.AuthenticationSource != authentikConfig.Name {
            continue
        }
        found := false
        for _, correctUser := range apiUsers {
            if GetIdentifier(correctUser.Username) == user.Name {
                found = true
                break
            }
        }

        if !found {
            logger.Info("User is orphaned and will be deleted", "user", user.Name, "namespace", user.Namespace)

            err = c.clientSet.Perm8sV1alpha1().Users(user.Namespace).Delete(ctx, user.Name, v3.DeleteOptions{})

            if err != nil {
                logger.Error(err, "Error during deletion of orphaned User", "user", user.Name, "namespace", user.Namespace)
                return err
            }

            logger.Info("Orphaned user deleted successfully", "user", user.Name, "namespace", user.Namespace)
        }
    }

    c.recorder.Event(authentikConfig, v2.EventTypeNormal, SuccessSynced, "Authentik Source has been synced successfully")
    return nil
}

func GetIdentifier(accountName string) string {
    return nonAlphanumericRegex.ReplaceAllString(strings.TrimSpace(strings.ToLower(accountName)), "")
}

func (c *Controller) GetUserFromAuthentikUser(identifier string, accountName string, namespace string, memberships []string, source *v1alpha2.AuthentikSynchronisationSource) *v1alpha2.User {
    return &v1alpha2.User{
        ObjectMeta: v3.ObjectMeta{
            Name:      identifier,
            Namespace: namespace,
            OwnerReferences: []v3.OwnerReference{
                *v3.NewControllerRef(source, v1alpha2.SchemeGroupVersion.WithKind("AuthentikSynchronisationSource")),
            },
        },
        Spec: v1alpha2.UserSpec{
            AuthenticationSource: source.Name,
            GroupMemberships:     memberships,
            DisplayName:          accountName,
        },
    }
}
