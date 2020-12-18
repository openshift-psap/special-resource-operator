package e2e

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift-psap/special-resource-operator/pkg/color"
	"github.com/openshift-psap/special-resource-operator/test/framework"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/pkg/errors"

	//srov1beta1 "github.com/openshift-psap/special-resource-operator/api/v1beta1"

	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	log = ctrl.Log.WithName(color.Print("deploy", color.Blue))
)

// WaitForClusterOperatorCondition blocks until the SRO ClusterOperator status
// condition 'conditionType' is equal to the value of 'conditionStatus'.
// The execution interval to check the value is 'interval' and retries last
// for at most the duration 'duration'.
func WaitForClusterOperatorCondition(cs *framework.ClientSet, interval, duration time.Duration,
	conditionType configv1.ClusterStatusConditionType, conditionStatus configv1.ConditionStatus) error {
	var explain error

	startTime := time.Now()
	if err := wait.PollImmediate(interval, duration, func() (bool, error) {
		co, err := cs.ClusterOperators().Get(context.TODO(), "special-resource-operator", metav1.GetOptions{})
		if err != nil {
			explain = err
			return false, nil
		}

		for _, cond := range co.Status.Conditions {
			if cond.Type == conditionType &&
				cond.Status == conditionStatus {
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		return errors.Wrapf(err, "failed to wait for ClusterOperator/special-resource-operator %s == %s (waited %s): %v",
			conditionType, conditionStatus, time.Since(startTime), explain)
	}
	return nil
}
