package controller

import (
    "context"
    "fmt"
    "golang.org/x/time/rate"
    v2 "k8s.io/api/core/v1"
    utilruntime "k8s.io/apimachinery/pkg/util/runtime"
    "k8s.io/apimachinery/pkg/util/wait"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/kubernetes/scheme"
    v1 "k8s.io/client-go/kubernetes/typed/core/v1"
    "k8s.io/client-go/tools/cache"
    "k8s.io/client-go/tools/record"
    "k8s.io/client-go/util/workqueue"
    "k8s.io/klog/v2"
    clientset "perm8s/pkg/generated/clientset/versioned"
    permscheme "perm8s/pkg/generated/clientset/versioned/scheme"
    "perm8s/pkg/generated/informers/externalversions/perm8s/v1alpha1"
    listers "perm8s/pkg/generated/listers/perm8s/v1alpha1"
    "time"
)

type Controller struct {
    kubeclientset kubernetes.Interface
    clientSet     clientset.Interface
    apiCLient     *v1.CoreV1Client
    userLister        listers.UserLister
    groupLister listers.GroupLister
    authentikLister listers.AuthentikSynchronisationSourceLister
    usersSynced   cache.InformerSynced
    groupsSynced cache.InformerSynced
    authentikSourcesSynced cache.InformerSynced
    userWorkqueue workqueue.RateLimitingInterface
    groupWorkqueue workqueue.RateLimitingInterface
    authentikWorkqueue workqueue.RateLimitingInterface

    recorder record.EventRecorder
}

func NewController(
    ctx context.Context,
    kubeclientset kubernetes.Interface,
    clientSet clientset.Interface,
    apiClient *v1.CoreV1Client,
    version v1alpha1.Interface) *Controller {
    logger := klog.FromContext(ctx)
    
    utilruntime.Must(permscheme.AddToScheme(scheme.Scheme))
    logger.V(4).Info("Creating event broadcaster")

    eventBroadcaster := record.NewBroadcaster(record.WithContext(ctx))
    eventBroadcaster.StartStructuredLogging(0)
    eventBroadcaster.StartRecordingToSink(&v1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
    recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v2.EventSource{Component: controllerAgentName})
    ratelimiter := workqueue.NewMaxOfRateLimiter(
        workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
        &workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(50), 300)},
    )

    controller := &Controller{
        kubeclientset: kubeclientset,
        clientSet:     clientSet,
        apiCLient:     apiClient,
        userLister:        version.Users().Lister(),
        groupLister: version.Groups().Lister(),
        authentikLister : version.AuthentikSynchronisationSources().Lister(),
        usersSynced:   version.Users().Informer().HasSynced,
        groupsSynced: version.Groups().Informer().HasSynced,
        authentikSourcesSynced: version.AuthentikSynchronisationSources().Informer().HasSynced,
        userWorkqueue:     workqueue.NewRateLimitingQueue(ratelimiter),
        groupWorkqueue: workqueue.NewRateLimitingQueue(ratelimiter),
        authentikWorkqueue: workqueue.NewRateLimitingQueue(ratelimiter),
        recorder:      recorder,
    }

    logger.Info("Setting up event handlers")

    version.Users().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: controller.enqueueUser,
        UpdateFunc: func(old, new interface{}) {
            controller.enqueueUser(new)
        },
    })
    
    version.Groups().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: controller.enqueueGroup,
        UpdateFunc: func(old, new interface{}) {
            controller.enqueueGroup(new)
        },
    })
    
    version.AuthentikSynchronisationSources().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: controller.enqueueAuthentik,
        UpdateFunc: func(old, new interface{}) {
            controller.enqueueAuthentik(new)
        },
    })

    return controller
}

func (c *Controller) Run(ctx context.Context, workers int) error {
    defer utilruntime.HandleCrash()
    defer c.userWorkqueue.ShutDown()
    logger := klog.FromContext(ctx)

    logger.Info("Controller Started, waiting for informer caches to sync")

    if ok := cache.WaitForCacheSync(ctx.Done(), c.groupsSynced, c.usersSynced); !ok {
        return fmt.Errorf("failed to wait for caches to sync")
    }

    logger.Info("Starting user workers", "count", workers)
    for i := 0; i < workers; i++ {
        go wait.UntilWithContext(ctx, c.runUserWorker, time.Second)
    }
    
    logger.Info("Starting group workers", "count", workers)
    for i := 0; i < workers; i++ {
        go wait.UntilWithContext(ctx, c.runGroupWorker, time.Second)
    }
    
    logger.Info("Starting authentik workers", "count", workers)
    for i := 0; i < workers; i++ {
        go wait.UntilWithContext(ctx, c.runAuthentikWorker, time.Second)
    }

    logger.Info("Started workers")
    <-ctx.Done()
    logger.Info("Shutting down workers")

    return nil
}

