package main

import (
	"flag"
	"github.com/refat75/apiServerController/pkg/signals"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	"path/filepath"
	"time"

	clientset "github.com/refat75/apiServerController/pkg/generated/clientset/versioned"
	informers "github.com/refat75/apiServerController/pkg/generated/informers/externalversions"
)

func main() {
	klog.InitFlags(nil)
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	ctx := signals.SetupSignalHandler()
	logger := klog.FromContext(ctx)

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		logger.Error(err, "Error building kubeconfig")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Error building kubernetes clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	exampleclient, err := clientset.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Error building example clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	exampleInformerFactory := informers.NewSharedInformerFactory(exampleclient, time.Second*30)

	controller := NewController(ctx, kubeClient, exampleclient,
		kubeInformerFactory.Apps().V1().Deployments(),
		exampleInformerFactory.Appscode().V1().Apiservers())

	kubeInformerFactory.Start(ctx.Done())
	exampleInformerFactory.Start(ctx.Done())

	if err = controller.Run(ctx, 1); err != nil {
		logger.Error(err, "Error running controller")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
}
