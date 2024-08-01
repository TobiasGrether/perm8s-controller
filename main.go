package main

import (
    "flag"
    "net/http"
    _ "net/http/pprof"

    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    v1 "k8s.io/client-go/kubernetes/typed/core/v1"

    "k8s.io/klog/v2" // Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
    controller2 "perm8s/controller"
    clientset "perm8s/pkg/generated/clientset/versioned"
    informers "perm8s/pkg/generated/informers/externalversions"
    "perm8s/pkg/signals"
    "time"
)

var (
    masterURL  string
    kubeconfig string
    profiling  bool
)

func main() {
    klog.InitFlags(nil)
    flag.Parse()

    ctx := signals.SetupSignalHandler()
    logger := klog.FromContext(ctx)
    
    if profiling {
        logger.Info("Starting profiling (pprof) server on :8444")
        go func() {
			if err := http.ListenAndServe("0.0.0.0:8444", nil); err != nil {
				logger.Error(err,"failed to start profiling server on 0.0.0.0:8444", "err")
			}
            
		}()
    }

    cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
    if err != nil {
        logger.Error(err, "Error building kubeconfig")
        klog.FlushAndExit(klog.ExitFlushTimeout, 1)
    }

    client, err := kubernetes.NewForConfig(cfg)
    if err != nil {
        logger.Error(err, "Error building kubernetes clientset")
        klog.FlushAndExit(klog.ExitFlushTimeout, 1)
    }

    apiClient, err := v1.NewForConfig(cfg)
    if err != nil {
        logger.Error(err, "Error building kubernetes v1 client")
        klog.FlushAndExit(klog.ExitFlushTimeout, 1)
    }

    set, err := clientset.NewForConfig(cfg)
    if err != nil {
        logger.Error(err, "Error building kubernetes clientset")
        klog.FlushAndExit(klog.ExitFlushTimeout, 1)
    }

    informerFactory := informers.NewSharedInformerFactory(set, time.Second*30)

    controller := controller2.NewController(ctx, client, set, apiClient, informerFactory.Perm8s().V1alpha1())
    informerFactory.Start(ctx.Done())

    if err = controller.Run(ctx, 2); err != nil {
        logger.Error(err, "Error running user controller")
        klog.FlushAndExit(klog.ExitFlushTimeout, 1)
    }
}

func init() {
    flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
    flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
    flag.BoolVar(&profiling, "profiling", false, "Enable to turn on pprof performance profiling for this program")
}
