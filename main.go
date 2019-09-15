/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	core "k8s.io/api/core/v1"
	extenstionsv1 "k8s.io/api/extensions/v1beta1"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	log.Info(fmt.Sprint("Init first line before adding scheme"))
	_ = clientgoscheme.AddToScheme(scheme)

	// +kubebuilder:scaffold:scheme
}

// InjectClient is called by the application.Builder
// to provide a client.Client
// This method is called when ctrl is initialized
// for reference - https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/runtime/inject/inject.go#L75
func (a *MyReconciler) InjectClient(c client.Client) error {
	log.Info(fmt.Sprint("Client Inject Method is Called"))
	a.Client = c
	return nil
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.Logger(true))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	err = builder.
		ControllerManagedBy(mgr).         // Create the ControllerManagedBy
		For(&extenstionsv1.Deployment{}). // Deployment is the Application API
		Owns(&core.Pod{}).                // Deployment owns Pods created by it
		Complete(&MyReconciler{})
	if err != nil {
		log.Error(err, "could not create controller")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// MyReconciler is a simple ControllerManagedBy example implementation.
type MyReconciler struct {
	client.Client
}

// Reconcile method
// Implement the business logic:
// This function will be called when there is a change to a ReplicaSet or a Pod with an OwnerReference
// to a ReplicaSet.
//
// * Read the ReplicaSet
// * Read the Pods
// * Set a Label on the ReplicaSet with the Pod count
func (a *MyReconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	// Read the ReplicaSet
	dep := &extenstionsv1.Deployment{}
	err := a.Get(context.TODO(), req.NamespacedName, dep)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Add Sidecar
	if val, found := dep.Labels["node-sidecar"]; val == "true" && found {
		isSidecarRunning := isSidecarRunning(dep)
		// don't inject if sidecar is already in the deployment
		if !isSidecarRunning {
			dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, sideCarContainer())
		}
	}

	err = a.Update(context.TODO(), dep)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func sideCarContainer() core.Container {
	return core.Container{
		Image: "aminmithil/node-demo:latest",
		Name:  "node-sidecar",
		Ports: []core.ContainerPort{
			core.ContainerPort{
				ContainerPort: 8081,
				Protocol:      "TCP",
			},
		},
	}
}

func isSidecarRunning(rs *extenstionsv1.Deployment) bool {
	for _, container := range rs.Spec.Template.Spec.Containers {
		if container.Name == "node-sidecar" {
			return true
		}
	}
	return false
}
