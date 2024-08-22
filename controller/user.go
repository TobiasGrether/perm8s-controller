package controller

import (
	"context"
	"fmt"
	v2 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v3 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	v1alpha2 "perm8s/pkg/apis/perm8s/v1alpha1"
	"reflect"
)

func (c *Controller) enqueueUser(obj interface{}) {
	if objectRef, err := cache.ObjectToName(obj); err != nil {
		utilruntime.HandleError(err)
		return
	} else {
		c.userWorkqueue.Add(objectRef)
	}
}

func (c *Controller) runUserWorker(ctx context.Context) {
	for c.processNextUserWorkItem(ctx) {
	}
}

func (c *Controller) processNextUserWorkItem(ctx context.Context) bool {
	objRef, shutdown := c.userWorkqueue.Get()
	logger := klog.FromContext(ctx)

	if shutdown {
		return false
	}

	defer c.userWorkqueue.Done(objRef)

	err := c.syncUserHandler(ctx, objRef.(cache.ObjectName))
	if err == nil {

		c.userWorkqueue.Forget(objRef)
		logger.Info("Successfully synced", "objectName", objRef)
		return true
	}

	logger.Error(err, "Error syncing; requeuing for later retry", "objectReference", objRef)

	c.userWorkqueue.AddRateLimited(objRef)
	return true
}

func (c *Controller) syncUserHandler(ctx context.Context, objectRef cache.ObjectName) error {
	logger := klog.LoggerWithValues(klog.FromContext(ctx), "objectRef", objectRef)

	user, err := c.userLister.Users(objectRef.Namespace).Get(objectRef.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "User in workqueue no longer exists")
			return nil
		}

		return err
	}

	serviceAccount, err := c.apiClient.ServiceAccounts(user.Namespace).Get(ctx, user.Name, v3.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Service account does not exist yet, creating new", "accountName", user.Name, "namespace", user.Namespace)
			serviceAccount, err = c.apiClient.ServiceAccounts(user.Namespace).Create(ctx, c.ServiceAccountFromUser(user), v3.CreateOptions{})

			if err != nil {
				logger.Error(err, "Error while creating serviceaccount", "user", user.Name)
				return nil
			}

			logger.Info("Service account created successfully")

			c.recorder.Event(user, v2.EventTypeNormal, SuccessCreated, MessageUserCreated)
		}
	}

	_, err = c.apiClient.Secrets(user.Namespace).Get(ctx, fmt.Sprintf("%v-usertoken", serviceAccount.Name), v3.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("No token secret exists for user, creating secret", "user", user.Name, "serviceAccount", serviceAccount.Name)

			_, err = c.apiClient.Secrets(serviceAccount.Namespace).Create(ctx, c.AuthenticationSecretFromServiceAccount(serviceAccount, user), v3.CreateOptions{})

			if err != nil {
				logger.Error(err, "Error while creating authentication secret", "user", user.Name, "serviceAccount", serviceAccount.Name, "namespace", serviceAccount.Namespace)
				return nil
			}
		}
	}

	for _, group := range user.Spec.GroupMemberships {
		// we need to ensure that both the cluster group, the regular groups for each affected namespace, as well as the group object itself and everything else exists
		group, err := c.groupLister.Groups(user.Namespace).Get(group)

		if errors.IsNotFound(err) {
			logger.Info("User has group which does not exist. No UserGroup sync will be done for this group.")
			continue
		}

		if err != nil {
			logger.Error(err, "Error while retrieving Group for UserGroup sync", "user", user.Name)
			return err
		}

		// Cluster groups are groups that have their permissions assigned to the entire cluster. Permissions assigned to these roles will be available throughout every namespace
		if group.Spec.ClusterGroup {
			// check for clusterrolebinding
			clusterRoleBinding, err := c.kubeclientset.RbacV1().ClusterRoleBindings().Get(ctx, fmt.Sprintf("%v-membership-%v", user.Name, group.Name), v3.GetOptions{})
			if errors.IsNotFound(err) {
				logger.Info("ClusterRoleBinding does not exist yet for user, creating")
				clusterRoleBinding, err = c.kubeclientset.RbacV1().ClusterRoleBindings().Create(ctx, c.ClusterRoleBindingForUserMembership(user, group), v3.CreateOptions{})

				if err != nil {
					logger.Error(err, "Error while creating ClusterRoleBinding for UserGroup sync", "user", user.Name, "group", group.Name)
					return err
				}
			}

			desiredClusterRoleBinding := c.ClusterRoleBindingForUserMembership(user, group)
			if !reflect.DeepEqual(clusterRoleBinding.Subjects, desiredClusterRoleBinding.Subjects) || !reflect.DeepEqual(clusterRoleBinding.RoleRef, desiredClusterRoleBinding.RoleRef) {
				logger.Info("ClusterRoleBinding for UserGroup is out of sync, resyncing", "user", user.Name, "group", group.Name)

				_, err = c.kubeclientset.RbacV1().ClusterRoleBindings().Update(ctx, desiredClusterRoleBinding, v3.UpdateOptions{})

				if err != nil {
					logger.Error(err, "Error while updating ClusterRoleBinding for UserGroup sync", "user", user.Name, "group", group.Name)
					return err
				}
			}
		} else {
			for _, namespace := range group.Spec.Namespaces {
				roleBinding, err := c.kubeclientset.RbacV1().RoleBindings(namespace).Get(ctx, fmt.Sprintf("%v-membership-%v", user.Name, group.Name), v3.GetOptions{})
				if errors.IsNotFound(err) {
					logger.Info("RoleBinding does not exist yet for user, creating", "user", user.Name, "namespace", namespace, "group", group.Name)
					roleBinding, err = c.kubeclientset.RbacV1().RoleBindings(namespace).Create(ctx, c.RoleBindingForUserMembership(user, group, namespace), v3.CreateOptions{})

					if err != nil {
						logger.Error(err, "Error while creating RoleBinding for UserGroup sync", "user", user.Name, "group", group.Name, "namespace", namespace)
						return err
					}

					c.recorder.Event(user, v2.EventTypeNormal, SuccessSynced, "Created RoleBinding for user in namespace "+namespace)

				}

				desiredRoleBinding := c.RoleBindingForUserMembership(user, group, namespace)
				if !reflect.DeepEqual(roleBinding.Subjects, desiredRoleBinding.Subjects) || !reflect.DeepEqual(roleBinding.RoleRef, desiredRoleBinding.RoleRef) {
					logger.Info("RoleBinding for UserGroup is out of sync, resyncing", "user", user.Name, "group", group.Name, "namespace", namespace)

					_, err = c.kubeclientset.RbacV1().RoleBindings(namespace).Update(ctx, desiredRoleBinding, v3.UpdateOptions{})

					if err != nil {
						logger.Error(err, "Error while updating RoleBinding for UserGroup sync", "user", user.Name, "group", group.Name, "namespace", namespace)
						return err
					}
				}
			}
		}
	}

	c.recorder.Event(user, v2.EventTypeNormal, SuccessSynced, MessageUserSynced)
	return nil
}

func (c *Controller) ServiceAccountFromUser(user *v1alpha2.User) *v2.ServiceAccount {
	automount := true
	return &v2.ServiceAccount{
		ObjectMeta: v3.ObjectMeta{
			Name:      user.Name,
			Namespace: user.GetNamespace(),
			OwnerReferences: []v3.OwnerReference{
				*v3.NewControllerRef(user, v1alpha2.SchemeGroupVersion.WithKind("User")),
			},
		},
		AutomountServiceAccountToken: &automount,
	}
}

func (c *Controller) ClusterRoleBindingForUserMembership(user *v1alpha2.User, group *v1alpha2.Group) *v1.ClusterRoleBinding {
	return &v1.ClusterRoleBinding{
		ObjectMeta: v3.ObjectMeta{
			Name: fmt.Sprintf("%v-membership-%v", user.Name, group.Name),
			OwnerReferences: []v3.OwnerReference{
				*v3.NewControllerRef(user, v1alpha2.SchemeGroupVersion.WithKind("User")),
			},
		},
		Subjects: []v1.Subject{
			{
				Name:      user.Name,
				Kind:      "ServiceAccount",
				Namespace: user.Namespace,
			},
		},
		RoleRef: v1.RoleRef{
			Kind:     "ClusterRole",
			Name:     group.Name,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func (c *Controller) RoleBindingForUserMembership(user *v1alpha2.User, group *v1alpha2.Group, namespace string) *v1.RoleBinding {
	return &v1.RoleBinding{
		ObjectMeta: v3.ObjectMeta{
			Name: fmt.Sprintf("%v-membership-%v", user.Name, group.Name),
			OwnerReferences: []v3.OwnerReference{
				*v3.NewControllerRef(user, v1alpha2.SchemeGroupVersion.WithKind("User")),
			},
			Namespace: namespace,
		},
		Subjects: []v1.Subject{
			{
				Name:      user.Name,
				Kind:      "ServiceAccount",
				Namespace: user.Namespace,
			},
		},
		RoleRef: v1.RoleRef{
			Kind:     "ClusterRole",
			Name:     group.Name,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func (c *Controller) AuthenticationSecretFromServiceAccount(serviceAccount *v2.ServiceAccount, user *v1alpha2.User) *v2.Secret {
	return &v2.Secret{
		Type: v2.SecretTypeServiceAccountToken,
		ObjectMeta: v3.ObjectMeta{
			Name:      fmt.Sprintf("%v-usertoken", serviceAccount.Name),
			Namespace: serviceAccount.Namespace,
			OwnerReferences: []v3.OwnerReference{
				*v3.NewControllerRef(user, v1alpha2.SchemeGroupVersion.WithKind("User")),
			},
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": serviceAccount.Name,
			},
		},
	}
}
