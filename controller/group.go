package controller

import (
    "context"
    v2 "k8s.io/api/core/v1"
    v4 "k8s.io/api/rbac/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    v3 "k8s.io/apimachinery/pkg/apis/meta/v1"
    utilruntime "k8s.io/apimachinery/pkg/util/runtime"
    "k8s.io/client-go/tools/cache"
    "k8s.io/klog/v2"
    v1alpha2 "perm8s/pkg/apis/perm8s/v1alpha1"
    "reflect"
)

func (c *Controller) enqueueGroup(obj interface{}) {
    if objectRef, err := cache.ObjectToName(obj); err != nil {
        utilruntime.HandleError(err)
        return
    } else {
        c.groupWorkqueue.Add(objectRef)
    }
}

func (c *Controller) runGroupWorker(ctx context.Context) {
    for c.processNextGroupWorkItem(ctx) {
    }
}

func (c *Controller) processNextGroupWorkItem(ctx context.Context) bool {
    objRef, shutdown := c.groupWorkqueue.Get()
    logger := klog.FromContext(ctx)

    if shutdown {
        return false
    }

    defer c.groupWorkqueue.Done(objRef)

    err := c.syncGroupHandler(ctx, objRef.(cache.ObjectName))
    if err == nil {
        c.groupWorkqueue.Forget(objRef)
        logger.Info("Successfully synced", "objectName", objRef)
        return true
    }
    logger.Error(err, "Error syncing; requeuing for later retry", "objectReference", objRef)

    c.groupWorkqueue.AddRateLimited(objRef)
    return true
}

func (c *Controller) syncGroupHandler(ctx context.Context, objectRef cache.ObjectName) error {
    logger := klog.LoggerWithValues(klog.FromContext(ctx), "objectRef", objectRef)

    logger.Info("Syncing object")

    group, err := c.groupLister.Groups(objectRef.Namespace).Get(objectRef.Name)
    if err != nil {
        if errors.IsNotFound(err) {
            logger.Error(err, "Group in workqueue no longer exists")
            return nil
        }

        return err
    }

    clusterRole, err := c.kubeclientset.RbacV1().ClusterRoles().Get(ctx, group.Name, v3.GetOptions{})

    if errors.IsNotFound(err) {
        logger.Info("Cluster Role does not exist yet, creating new", "groupName", group.Name)
        clusterRole, err = c.kubeclientset.RbacV1().ClusterRoles().Create(ctx, c.ClusterRoleFromGroup(group), v3.CreateOptions{})

        if err != nil {
            logger.Error(err, "Error while creating ClusterRole", "group", group.Name)
            return nil
        }

        logger.Info("ClusterRole created successfully")

        c.recorder.Event(group, v2.EventTypeNormal, SuccessCreated, "ClusterRole created successfully")
    }
    
    if err != nil {
        return err
    }
    
    desiredClusterRoleState := c.ClusterRoleFromGroup(group)
    
    if !reflect.DeepEqual(clusterRole.Rules,  desiredClusterRoleState.Rules) {
        logger.Info("Cluster role is out of sync, resyncing", "clusterRoleName", clusterRole.Name)
        _, err = c.kubeclientset.RbacV1().ClusterRoles().Update(ctx, desiredClusterRoleState, v3.UpdateOptions{})
        
        if err != nil {
            logger.Error(err, "Error while syncing ClusterRole", "clusterRoleName", clusterRole.Name)
            return err
        }
        
        c.recorder.Event(group, v2.EventTypeNormal, SuccessSynced, "ClusterRole synchronised successfully")
    }

    // todo update cluster role where necessary

    c.recorder.Event(group, v2.EventTypeNormal, SuccessSynced, MessageGroupSynced)
    return nil
}

func (c *Controller) ClusterRoleFromGroup(group *v1alpha2.Group) *v4.ClusterRole {
    return &v4.ClusterRole{
        ObjectMeta: v3.ObjectMeta{
            Name:      group.Name,
            Namespace: group.Namespace,
            OwnerReferences: []v3.OwnerReference{
                *v3.NewControllerRef(group, v1alpha2.SchemeGroupVersion.WithKind("Group")),
            },
        },

        Rules: group.Spec.Permissions,
    }
}
