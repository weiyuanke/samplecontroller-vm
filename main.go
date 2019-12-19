package main

import (
	"context"
	"flag"
	"github.com/google/uuid"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/sample-controller/pkg/generated/clientset/versioned"
	informers "k8s.io/sample-controller/pkg/generated/informers/externalversions"
)

var (
	masterURL          string
	kubeConfig         string
	leaseLockName      string
	leaseLockNamespace string
	leaseLockId        string
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	if leaseLockName == "" {
		klog.Fatal("unable to get lease lock resource name (missing lease-lock-name flag).")
	}
	if leaseLockNamespace == "" {
		klog.Fatal("unable to get lease lock resource namespace (missing lease-lock-namespace flag).")
	}

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeConfig)
	if err != nil {
		klog.Fatalf("Error building kubeConfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	exampleClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building example clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	exampleInformerFactory := informers.NewSharedInformerFactory(exampleClient, time.Second*30)

	controller := NewController(
		kubeClient, exampleClient, kubeInformerFactory.Apps().V1().Deployments(),
		exampleInformerFactory.Samplecontroller().V1alpha1().VMs())

	// use a Go context so we can tell the leaderelection code when we
	// want to step down
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      leaseLockName,
			Namespace: leaseLockNamespace,
		},
		Client: kubeClient.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: leaseLockId,
		},
	}

	var stopCh chan struct{}
	stopCh = make(chan struct{})

	// start the leader election loop
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock: lock,
		ReleaseOnCancel: true,
		LeaseDuration:   30 * time.Second,
		RenewDeadline:   15 * time.Second,
		RetryPeriod:     5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
			},
			OnStoppedLeading: func() {
				klog.Infof("leader lost: %s", leaseLockId)
				stopCh <- struct{}{}
			},
			OnNewLeader: func(identity string) {
				if identity == leaseLockId {
					klog.Infof("we are the new leader, lockId:%s", identity)
					stopCh = make(chan struct{})
					kubeInformerFactory.Start(stopCh)
					exampleInformerFactory.Start(stopCh)
					go controller.Run(2, stopCh)
				} else {
					klog.Infof("we(%s) are not the leader: %s", leaseLockId, identity)
					stopCh <- struct{}{}
				}
			},
		},
	})
}

func init() {
	flag.StringVar(&kubeConfig, "kubeConfig", "", "Path to a kubeConfig.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. ")
	flag.StringVar(&leaseLockId, "id",  uuid.New().String(), "id")
	flag.StringVar(&leaseLockName, "lease-lock-name", "defname", "the lock name")
	flag.StringVar(&leaseLockNamespace, "lease-lock-namespace", "default", "the lock namespace")
}
