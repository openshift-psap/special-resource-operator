package controllers

import (
	"context"
	"fmt"

	srov1beta1 "github.com/openshift-psap/special-resource-operator/api/v1beta1"
	"github.com/openshift-psap/special-resource-operator/pkg/assets"
	"github.com/openshift-psap/special-resource-operator/pkg/color"
	"github.com/openshift-psap/special-resource-operator/pkg/exit"
	"github.com/openshift-psap/special-resource-operator/pkg/metrics"
	errs "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// GetName of the special resource operator
func (r *SpecialResourceReconciler) GetName() string {
	return "special-resource-operator"
}

// +kubebuilder:rbac:groups=sro.openshift.io,resources=specialresources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sro.openshift.io,resources=specialresources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sro.openshift.io,resources=specialresources/finalizers,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/log,verbs=get
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get
// +kubebuilder:rbac:groups=config.openshift.io,resources=proxies,verbs=get;list
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=use;get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=image.openshift.io,resources=imagestreams,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=image.openshift.io,resources=imagestreams/finalizers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=image.openshift.io,resources=imagestreams/layers,verbs=get
// +kubebuilder:rbac:groups=core,resources=imagestreams/layers,verbs=get
// +kubebuilder:rbac:groups=build.openshift.io,resources=buildconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=build.openshift.io,resources=builds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=list;watch;create;update;patch;delete;get
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;update;
// +kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=storage.k8s.io,resources=csinodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=watch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=csidrivers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusteroperators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusteroperators/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=issuers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=create;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/finalizers,verbs=create;delete;get;list;update;patch;delete;watch
// +kubebuilder:rbac:groups=apps,resources=deployments/finalizers,resourceNames=shipwright-build,verbs=update
// +kubebuilder:rbac:groups=apps,resources=replicasets,verbs=create;delete;get;list;patch;update;watch;get
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=shipwright.io,resources=*,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=shipwright.io,resources=buildruns,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=shipwright.io,resources=buildstrategies,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=shipwright.io,resources=clusterbuildstrategies,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=tekton.dev,resources=taskruns,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=tekton.dev,resources=tasks,verbs=create;delete;get;list;patch;update;watch

// SpecialResourcesReconcile Takes care of all specialresources in the cluster
func SpecialResourcesReconcile(r *SpecialResourceReconciler, req ctrl.Request) (ctrl.Result, error) {

	log = r.Log.WithName(color.Print("preamble", color.Purple))

	log.Info("Reconciling SpecialResource(s) in all Namespaces")
	specialresources := &srov1beta1.SpecialResourceList{}

	opts := []client.ListOption{}
	err := r.List(context.TODO(), specialresources, opts...)
	if err != nil {
		if errors.IsNotFound(err) {
			// Requested object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// set specialResourcesCreated metric to the number of specialresources
	metrics.SetSpecialResourcesCreated(len(specialresources.Items))

	for _, r.parent = range specialresources.Items {

		//log = r.Log.WithValues("specialresource", r.parent.Name)
		log = r.Log.WithName(color.Print(r.parent.Name, color.Green))
		log.Info("Resolving Dependencies")

		if r.parent.Name == "special-resource-preamble" {
			log.Info("Preamble done, waiting for driver-container requests")
			continue
		}

		// Execute finalization logic if CR is being deleted
		isMarkedToBeDeleted := r.parent.GetDeletionTimestamp() != nil
		if isMarkedToBeDeleted {
			r.specialresource = r.parent
			log.Info("Marked to be deleted, reconciling finalizer")
			err = reconcileFinalizers(r)
			return reconcile.Result{}, err
		}

		// Only one level dependency support for now
		for _, r.dependency = range r.parent.Spec.DependsOn {

			//log = r.Log.WithValues("specialresource", r.dependency.Name)
			log = r.Log.WithName(color.Print(r.dependency.Name, color.Purple))
			log.Info("Getting Dependency")

			// Assign the specialresource to the reconciler object
			if r.specialresource, err = getDependencyFrom(specialresources, r.dependency.Name); err != nil {
				log.Info("Could not get SpecialResource dependency", "error", fmt.Sprintf("%v", err))
				if r.specialresource, err = createSpecialResourceFrom(r, r.dependency.Name); err != nil {
					//return reconcile.Result{}, errs.New("Dependency creation failed")
					log.Info("Dependency creation failed", "error", fmt.Sprintf("%v", err))
					return reconcile.Result{Requeue: true}, nil
				}
				// We need to fetch the newly created SpecialResources, reconciling
				return reconcile.Result{}, nil
			}

			log.Info("Reconciling Dependency")
			if err := ReconcileHardwareConfigurations(r); err != nil {
				// We do not want a stacktrace here, errs.Wrap already created
				// breadcrumb of errors to follow. Just sprintf with %v rather than %+v
				log.Info("Could not reconcile hardware configurations", "error", fmt.Sprintf("%v", err))
				//return reconcile.Result{}, errs.New("Reconciling failed")
				return reconcile.Result{Requeue: true}, nil
			}
		}

		r.specialresource = r.parent
		log = r.Log.WithName(color.Print(r.specialresource.Name, color.Green))
		log.Info("Reconciling")

		// Add a finalizer to CR if it does not already have one
		if !contains(r.specialresource.GetFinalizers(), specialresourceFinalizer) {
			if err := addFinalizer(r); err != nil {
				log.Info("Failed to add finalizer", "error", fmt.Sprintf("%v", err))
				return reconcile.Result{}, err
			}
		}

		// Reconcile the special resource recipe
		if err := ReconcileHardwareConfigurations(r); err != nil {
			// We do not want a stacktrace here, errs.Wrap already created
			// breadcrumb of errors to follow. Just sprintf with %v rather than %+v
			log.Info("Could not reconcile hardware configurations", "error", fmt.Sprintf("%v", err))
			//return reconcile.Result{}, errs.New("Reconciling failed")
			return reconcile.Result{Requeue: true}, nil
		}

	}

	return reconcile.Result{}, nil

}

func getDependencyFrom(specialresources *srov1beta1.SpecialResourceList, name string) (srov1beta1.SpecialResource, error) {

	log.Info("Looking for")

	for _, specialresource := range specialresources.Items {
		if specialresource.Name == name {
			return specialresource, nil
		}
	}

	return srov1beta1.SpecialResource{}, errs.New("Not found")
}

func createSpecialResourceFrom(r *SpecialResourceReconciler, name string) (srov1beta1.SpecialResource, error) {

	specialresource := srov1beta1.SpecialResource{}

	crpath := "/opt/sro/recipes/" + name
	crfile := assets.GetFrom(crpath)

	if len(crfile) == 0 {
		exit.OnError(errs.New("Could not read CR " + name + "from local path"))
	}

	if len(crfile) > 1 {
		log.Info("More than one default CR provided, taking the first one")
	}

	// Only one CR creation if they are more ignore all others
	// makes no sense to create multiple CRs for the same specialresource
	cryaml := crfile[0:1][0]

	log.Info("Creating SpecialResource: " + cryaml.Name)

	if err := createFromYAML(cryaml.Content, r, r.specialresource.Spec.Namespace); err != nil {
		log.Info("Cannot create, something went horribly wrong")
		exit.OnError(err)
	}

	return specialresource, errs.New("Created new SpecialResource we need to Reconcile")
}
