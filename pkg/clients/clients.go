package clients

import (
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/openshift-psap/special-resource-operator/pkg/exit"
	clientconfigv1 "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	Interface  *ClientsInterface
	RestConfig *rest.Config
	Namespace  string
	config     genericclioptions.ConfigFlags
)

type ClientsInterface struct {
	client.Client
	kubernetes.Clientset
	clientconfigv1.ConfigV1Client
	record.EventRecorder
	authn.Keychain
	discovery.CachedDiscoveryInterface
}

// GetKubeClientSetOrDie Add a native non-caching client for advanced CRUD operations
func GetKubeClientSetOrDie() kubernetes.Clientset {

	clientSet, err := kubernetes.NewForConfig(RestConfig)
	exit.OnError(err)
	return *clientSet
}

// GetConfigClientOrDie Add a configv1 client to the reconciler
func GetConfigClientOrDie() clientconfigv1.ConfigV1Client {

	client, err := clientconfigv1.NewForConfig(RestConfig)
	exit.OnError(err)
	return *client
}

func GetCachedDiscoveryClientOrDie() discovery.CachedDiscoveryInterface {

	client, err := config.ToDiscoveryClient()
	exit.OnError(err)
	return client
}
