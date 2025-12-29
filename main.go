package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/hellices/kubernetes-pfx-tls/pkg/controller"
	"github.com/hellices/kubernetes-pfx-tls/pkg/converter"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

func main() {
	var kubeconfig string
	var masterURL string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")

	klog.InitFlags(nil)
	flag.Parse()

	// Create the client config
	cfg, err := buildConfig(kubeconfig, masterURL)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %v", err)
	}

	// Create the Kubernetes client
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %v", err)
	}

	// Create informer factory
	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Minute*10)

	// Create the converter
	pfxConverter := converter.NewPFXConverter()

	// Create the controller
	ctrl := controller.NewSecretController(
		kubeClient,
		informerFactory.Core().V1().Secrets(),
		pfxConverter,
	)

	// Start informers
	informerFactory.Start(wait(context.Background()))

	// Run the controller
	if err := ctrl.Run(2, wait(context.Background())); err != nil {
		klog.Fatalf("Error running controller: %v", err)
	}
}

func buildConfig(kubeconfig, masterURL string) (*rest.Config, error) {
	if kubeconfig != "" {
		cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("error building config from flags: %w", err)
		}
		return cfg, nil
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("error building in-cluster config: %w", err)
	}
	return cfg, nil
}

func wait(ctx context.Context) <-chan struct{} {
	stopCh := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(stopCh)
	}()
	return stopCh
}
