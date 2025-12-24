package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/hellices/kubernetes-pfx-tls/pkg/converter"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// SecretController watches Secrets and converts PFX certificates to PEM
type SecretController struct {
	kubeClient    kubernetes.Interface
	secretLister  corelisters.SecretLister
	secretsSynced cache.InformerSynced
	workqueue     workqueue.RateLimitingInterface
	converter     *converter.PFXConverter
	ctx           context.Context
}

// NewSecretController creates a new SecretController
func NewSecretController(
	kubeClient kubernetes.Interface,
	secretInformer coreinformers.SecretInformer,
	converter *converter.PFXConverter,
) *SecretController {
	controller := &SecretController{
		kubeClient:    kubeClient,
		secretLister:  secretInformer.Lister(),
		secretsSynced: secretInformer.Informer().HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Secrets"),
		converter:     converter,
	}

	klog.Info("Setting up event handlers")
	secretInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueSecret,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueSecret(new)
		},
	})

	return controller
}

// Run starts the controller
func (c *SecretController) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Create context from stopCh
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.ctx = ctx

	go func() {
		<-stopCh
		cancel()
	}()

	klog.Info("Starting Secret controller")

	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.secretsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

func (c *SecretController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *SecretController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *SecretController) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	secret, err := c.secretLister.Secrets(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("secret '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	// Check if the secret should be processed
	if !c.shouldProcessSecret(secret) {
		return nil
	}

	// Check if already converted
	if secret.Annotations[converter.AnnotationConverted] == "true" {
		klog.V(4).Infof("Secret %s/%s already converted, skipping", namespace, name)
		return nil
	}

	// Process the secret
	return c.processSecret(secret)
}

func (c *SecretController) shouldProcessSecret(secret *corev1.Secret) bool {
	if secret.Annotations == nil {
		return false
	}

	convertValue, exists := secret.Annotations[converter.AnnotationPFXConvert]
	return exists && convertValue == "true"
}

func (c *SecretController) processSecret(secret *corev1.Secret) error {
	klog.Infof("Processing secret %s/%s for PFX to PEM conversion", secret.Namespace, secret.Name)

	// Get PFX data key from annotation, default to "pfx"
	pfxKey := secret.Annotations[converter.AnnotationPFXDataKey]
	if pfxKey == "" {
		pfxKey = "pfx"
	}

	pfxData, exists := secret.Data[pfxKey]
	if !exists {
		return fmt.Errorf("PFX data not found in secret at key '%s'", pfxKey)
	}

	// Get password
	password, err := c.getPassword(secret)
	if err != nil {
		return fmt.Errorf("failed to get password: %w", err)
	}

	// Convert PFX to PEM
	certPEM, keyPEM, caPEM, err := c.converter.ConvertPFXToPEM(pfxData, password)
	if err != nil {
		return fmt.Errorf("failed to convert PFX to PEM: %w", err)
	}

	// Update secret with PEM data
	secretCopy := secret.DeepCopy()
	if secretCopy.Data == nil {
		secretCopy.Data = make(map[string][]byte)
	}

	// Store PEM data
	secretCopy.Data["tls.crt"] = certPEM
	secretCopy.Data["tls.key"] = keyPEM
	if len(caPEM) > 0 {
		secretCopy.Data["ca.crt"] = caPEM
	}

	// Change type to kubernetes.io/tls
	secretCopy.Type = corev1.SecretTypeTLS

	// Mark as converted
	if secretCopy.Annotations == nil {
		secretCopy.Annotations = make(map[string]string)
	}
	secretCopy.Annotations[converter.AnnotationConverted] = "true"

	// Update the secret
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	
	_, err = c.kubeClient.CoreV1().Secrets(secret.Namespace).Update(ctx, secretCopy, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	klog.Infof("Successfully converted PFX to PEM for secret %s/%s", secret.Namespace, secret.Name)
	return nil
}

func (c *SecretController) getPassword(secret *corev1.Secret) (string, error) {
	// Check if password is in annotation
	if password, exists := secret.Annotations[converter.AnnotationPFXPassword]; exists {
		return password, nil
	}

	// Check if password is in another secret
	secretName := secret.Annotations[converter.AnnotationPFXPasswordSecretName]
	secretKey := secret.Annotations[converter.AnnotationPFXPasswordSecretKey]

	if secretName != "" && secretKey != "" {
		ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
		defer cancel()
		
		passwordSecret, err := c.kubeClient.CoreV1().Secrets(secret.Namespace).Get(ctx, secretName, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to get password secret: %w", err)
		}

		passwordData, exists := passwordSecret.Data[secretKey]
		if !exists {
			return "", fmt.Errorf("password key '%s' not found in secret '%s'", secretKey, secretName)
		}

		return string(passwordData), nil
	}

	// No password specified, try empty password
	return "", nil
}

func (c *SecretController) enqueueSecret(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}
