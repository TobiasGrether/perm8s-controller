package controller

import (
	"context"
	v2 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v3 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	v1alpha2 "perm8s/pkg/apis/perm8s/v1alpha1"
	"perm8s/sync"
	"reflect"
	"regexp"
	"strings"
)

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

func (c *Controller) enqueueSyncSource(obj interface{}) {
	if objectRef, err := cache.ObjectToName(obj); err != nil {
		utilruntime.HandleError(err)
		return
	} else {
		c.syncSourceWorkqueue.Add(objectRef)
	}
}

func (c *Controller) runSyncSourceWorker(ctx context.Context) {
	for c.processNextSyncWorkItem(ctx) {
	}
}

func (c *Controller) processNextSyncWorkItem(ctx context.Context) bool {
	objRef, shutdown := c.syncSourceWorkqueue.Get()
	logger := klog.FromContext(ctx)

	if shutdown {
		return false
	}

	defer c.syncSourceWorkqueue.Done(objRef)

	// Run the syncHandler, passing it the structured reference to the object to be synced.
	err := c.syncSyncHandler(ctx, objRef.(cache.ObjectName))
	if err == nil {

		c.syncSourceWorkqueue.Forget(objRef)
		logger.Info("Successfully synced", "objectName", objRef)
		return true
	}

	logger.Error(err, "Error syncing; requeuing for later retry", "objectReference", objRef)

	c.syncSourceWorkqueue.AddRateLimited(objRef)
	return true
}

func (c *Controller) syncSyncHandler(ctx context.Context, objectRef cache.ObjectName) error {
	logger := klog.LoggerWithValues(klog.FromContext(ctx), "objectRef", objectRef)

	source, err := c.syncSourceLister.SynchronisationSources(objectRef.Namespace).Get(objectRef.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "SynchronisationSource in workqueue no longer exists")
			return nil
		}

		return err
	}

	computeFunc, ok := sync.SyncSources[source.Spec.Type]

	if !ok {
		logger.Error(err, "Cannot find synchronisation source type", "type", source.Spec.Type)
		c.recorder.Event(source, v2.EventTypeWarning, "Failed", "Cannot find synchronisation source type "+source.Spec.Type)
		return nil
	}

	logger = logger.WithValues("sourceType", source.Spec.Type)

	users, err := computeFunc(ctx, *source, c.apiClient)

	if err != nil {
		logger.Error(err, "Error while computing users", "type", source.Spec.Type)
		c.recorder.Event(source, v2.EventTypeWarning, "Failed", "Error while computing users")
		return nil
	}

	for _, user := range *users {
		identifier := GetIdentifier(user.Name)

		var groups []string

		for _, g := range user.Groups {
			if groupName, ok := source.Spec.GroupMappings[g]; ok {
				groups = append(groups, groupName)
			}
		}

		desiredUser := c.GetUserFromSyncUser(identifier, user.Name, source.Namespace, groups, source)

		currentUser, err := c.clientSet.Perm8sV1alpha1().Users(source.Namespace).Get(ctx, desiredUser.Name, v3.GetOptions{})

		if errors.IsNotFound(err) {
			logger.Info("User account does not exist for external identity user yet, creating new")
			_, err := c.clientSet.Perm8sV1alpha1().Users(source.Namespace).Create(ctx, desiredUser, v3.CreateOptions{})

			if err != nil {
				return err
			}
			c.recorder.Event(source, v2.EventTypeNormal, SuccessSynced, "User Account created for external users")
			continue
		}

		if err != nil {
			return err
		}

		if !reflect.DeepEqual(currentUser.Spec, desiredUser.Spec) {
			logger.Info("External User is out of sync, resynching")
			desiredUser.SetResourceVersion(currentUser.GetResourceVersion())
			_, err = c.clientSet.Perm8sV1alpha1().Users(source.Namespace).Update(ctx, desiredUser, v3.UpdateOptions{})

			if err != nil {
				return err
			}
		}
	}

	// finally, we need to make sure no users exist that are not part of the target group anymore
	result, err := c.clientSet.Perm8sV1alpha1().Users(source.Namespace).List(ctx, v3.ListOptions{})

	if err != nil {
		return err
	}

	for _, user := range result.Items {
		if user.Spec.AuthenticationSource != source.Name {
			continue
		}
		found := false
		for _, correctUser := range *users {
			if GetIdentifier(correctUser.Name) == user.Name {
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

	c.recorder.Event(source, v2.EventTypeNormal, SuccessSynced, "Synchronisation Source has been synced successfully")
	return nil
}

func GetIdentifier(accountName string) string {
	return nonAlphanumericRegex.ReplaceAllString(strings.ReplaceAll(strings.TrimSpace(strings.ToLower(accountName)), " ", "-"), "")
}

func (c *Controller) GetUserFromSyncUser(identifier string, accountName string, namespace string, memberships []string, source *v1alpha2.SynchronisationSource) *v1alpha2.User {
	return &v1alpha2.User{
		ObjectMeta: v3.ObjectMeta{
			Name:      identifier,
			Namespace: namespace,
			OwnerReferences: []v3.OwnerReference{
				*v3.NewControllerRef(source, v1alpha2.SchemeGroupVersion.WithKind("SynchronisationSource")),
			},
		},
		Spec: v1alpha2.UserSpec{
			AuthenticationSource: source.Name,
			GroupMemberships:     memberships,
			DisplayName:          accountName,
		},
	}
}
