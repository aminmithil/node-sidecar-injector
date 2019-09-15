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
	_ = clientgoscheme.AddToScheme(scheme)

	// +kubebuilder:scaffold:scheme
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
		For(&extenstionsv1.Deployment{}). // ReplicaSet is the Application API
		Owns(&core.Pod{}).                // ReplicaSet owns Pods created by it
		Complete(&DeploymentReconciler{})
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

// DeploymentReconciler is a simple ControllerManagedBy example implementation.
type DeploymentReconciler struct {
	client.Client
}

// InjectClient is called by the application.Builder
// to provide a client.Client
func (a *DeploymentReconciler) InjectClient(c client.Client) error {
	log.Info(fmt.Sprint("Client Inject Method is Called"))
	a.Client = c
	return nil
}

// Reconcile method
// Implement the business logic:
// This function will be called when there is a change to a ReplicaSet or a Pod with an OwnerReference
// to a ReplicaSet.
//
// * Read the ReplicaSet
// * Read the Pods
// * Set a Label on the ReplicaSet with the Pod count
func (a *DeploymentReconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	// Read the ReplicaSet
	rs := &extenstionsv1.Deployment{}
	err := a.Get(context.TODO(), req.NamespacedName, rs)
	if err != nil {
		return reconcile.Result{}, err
	}

	// List the Pods matching the PodTemplate Labels
	pods := &core.PodList{}
	err = a.List(context.TODO(), pods, client.InNamespace(req.Namespace),
		client.MatchingLabels(rs.Spec.Template.Labels))
	if err != nil {
		return reconcile.Result{}, err
	}

	// Add Sidecar
	if val, found := rs.Labels["node-sidecar"]; val == "true" && found {
		isSidecarRunning := isSidecarRunning(rs)
		if !isSidecarRunning {
			rs.Spec.Template.Spec.Containers = append(rs.Spec.Template.Spec.Containers, sideCarContainer())
		}
	}
	rs.Labels["pod-count"] = fmt.Sprintf("%v", len(pods.Items))
	err = a.Update(context.TODO(), rs)
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
