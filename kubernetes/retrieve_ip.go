package kubernetes

import (
	"context"
	"fmt"
	"log"
	"net"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var lastIP net.IP

func SetupIPRetriever(namespace string, node string) (<-chan net.IP, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve client config: %w", err)
	}

	client := kubernetes.NewForConfigOrDie(config)

	NodeTweakListOptions := func(opts *metav1.ListOptions) {
		opts.FieldSelector = fields.OneTermEqualSelector("spec.nodeName", node).String()
	}

	factory := informers.NewSharedInformerFactoryWithOptions(client, 0,
		informers.WithNamespace(namespace), informers.WithTweakListOptions(NodeTweakListOptions))

	ipch := make(chan net.IP, 1)
	informer := factory.Core().V1().Pods().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(object interface{}) {
			handlePodUpdate(object.(*corev1.Pod), ipch)
		},
		UpdateFunc: func(_, object interface{}) {
			handlePodUpdate(object.(*corev1.Pod), ipch)
		},
	})

	log.Printf("Starting informer for pod in namespace %q and running on node %q", namespace, node)
	factory.Start(context.Background().Done())
	factory.WaitForCacheSync(context.Background().Done())
	log.Printf("Informer correctly started")
	return ipch, nil
}

func handlePodUpdate(pod *corev1.Pod, ipch chan<- net.IP) {
	log.Printf("Received update for pod %q, with IP %q", pod.GetName(), pod.Status.PodIP)
	if !pod.GetDeletionTimestamp().IsZero() {
		return
	}

	ip := net.ParseIP(pod.Status.PodIP)
	if ip == nil {
		return
	}

	if ip.Equal(lastIP) {
		return
	}

	if lastIP != nil {
		log.Fatal("Detected observed IP change, restarting")
	}

	ipch <- ip
}
