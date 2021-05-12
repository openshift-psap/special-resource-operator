package dependencies

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	"github.com/openshift-psap/special-resource-operator/pkg/clients"
	"github.com/openshift-psap/special-resource-operator/pkg/color"
	"github.com/openshift-psap/special-resource-operator/pkg/exit"
	"github.com/openshift-psap/special-resource-operator/pkg/helmer"
	"helm.sh/helm/v3/pkg/chart"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	log logr.Logger
)

func init() {
	log = zap.New(zap.UseDevMode(true)).WithName(color.Print("dependencies", color.Brown))
}

func getConfigMap(namespace string, name string) *unstructured.Unstructured {

	cm := &unstructured.Unstructured{}
	cm.SetAPIVersion("v1")
	cm.SetKind("ConfigMap")

	dep := types.NamespacedName{Namespace: namespace, Name: name}

	err := clients.Interface.Get(context.TODO(), dep, cm)

	if apierrors.IsNotFound(err) {
		exit.OnError(err)
	}

	return cm
}

func CheckConfigMap(child string) string {

	cm := getConfigMap(os.Getenv("OPERATOR_NAMESPACE"), "special-resource-depedencies")

	data, found, err := unstructured.NestedMap(cm.Object, "data")
	exit.OnError(err)
	// No parent found for depedency just return
	if !found {
		return ""
	}
	// We have a dependency return the parent
	if parent, found := data[child]; found {
		return parent.(string)
	}

	return ""
}

func UpdateConfigMap(parent string, child string) {

	cm := getConfigMap(os.Getenv("OPERATOR_NAMESPACE"), "special-resource-depedencies")

	data, found, err := unstructured.NestedMap(cm.Object, "data")
	exit.OnError(err)

	dependencies := make(map[string]interface{})
	dependencies[child] = parent

	if !found {
		data = make(map[string]interface{})
		data["data"] = dependencies
		err := unstructured.SetNestedMap(cm.Object, dependencies, "data")
		exit.OnError(err)
	}

	err = unstructured.SetNestedMap(cm.Object, dependencies, "data")
	exit.OnError(err)

	err = clients.Interface.Update(context.TODO(), cm)
	exit.OnError(err)
}

func CheckOverride(chartDeps []*chart.Dependency, crDeps []helmer.HelmDependency) []*chart.Dependency {

	// If there are no overrides in the CR just ignore
	// and use the Chart.yaml depdendencies
	if len(crDeps) == 0 {
		log.Info("No overrides using Chart.yaml dependencies")
		return chartDeps
	}

	var override []*chart.Dependency

	for _, dep := range crDeps {
		override = append(override, &chart.Dependency{
			Name:       dep.Name,
			Version:    dep.Version,
			Repository: dep.Repository,
		})

	}
	return override
}
