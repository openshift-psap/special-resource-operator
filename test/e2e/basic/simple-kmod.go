package e2e

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/openshift-psap/special-resource-operator/test/framework"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = ginkgo.Describe("[basic][simple-kmod] create and deploy simple-kmod", func() {
	const (
		pollInterval = 5 * time.Second
		waitDuration = 5 * time.Minute
	)

	cs := framework.NewClientSet()
	cl := framework.NewControllerRuntimeClient()

	var explain string

	// Check that operator deployment has 1 available pod
	ginkgo.It(fmt.Sprintf("Can deploy simple-kmod"), func() {

		buffer, err := ioutil.ReadFile("../../../config/recipes/simple-kmod/0000-simple-kmod-cr.yaml")
		if err != nil {
			panic(err)
		}
		framework.CreateFromYAML(buffer, cl)

		ginkgo.By(fmt.Sprintf("driver-container-base is completed"))
		err = wait.PollImmediate(pollInterval, waitDuration, func() (bool, error) {
			driverContainerBase, err := cs.("simple-kmod").List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return false, fmt.Errorf("Couldn't get simple-kmod DaemonSet: %v", err)
			}

			// TODO TEMPORARY HACK
			if len(skDaemonSets.Items) == 1 {
				return true, nil
			}

			return false, nil
		})

		ginkgo.By(fmt.Sprintf("simple-kmod is ready"))

		//How can we check if the module is actually loaded... or check logs of simple-kmod??

		err = wait.PollImmediate(pollInterval, waitDuration, func() (bool, error) {
			skDaemonSets, err := cs.DaemonSets("simple-kmod").List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return false, fmt.Errorf("Couldn't get simple-kmod DaemonSet: %v", err)
			}

			//TEMPORARY HACK
			if len(skDaemonSets.Items) == 1 {
				return true, nil
			}

			return false, nil
		})
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), explain)
	})

})
