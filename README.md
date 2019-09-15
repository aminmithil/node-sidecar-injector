# Kubernetes sidecar controller 
Kubernetes sidecar controller is simple example to show the usecase of the controller. This example is using [go-client](https://github.com/kubernetes/client-go)
and [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder). This controller keep monitoring any changes in the [Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) and if deployment match the label it will add a sidecar.

## Code breakdown
### Main
When run the controller it will run ```func main()``` from ```main.go```.
* Parse flags and initialize logging
```var metricsAddr string
var enableLeaderElection bool
flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
    "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
flag.Parse()

ctrl.SetLogger(zap.Logger(true))
```
* Initialize [Manager](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/manager). Manager provides shared dependencies like client, schemes, caches, etc.
```
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    Scheme:             scheme,
    MetricsBindAddress: metricsAddr,
    LeaderElection:     enableLeaderElection,
})
if err != nil {
    setupLog.Error(err, "unable to start manager")
    os.Exit(1)
}
```
* Controller runs a Reconsile method when something changed in the object that it is monitoring. Define which object to monitor and which object this controller owns.
```
err = builder.
    ControllerManagedBy(mgr).         // Create the ControllerManagedBy
    For(&extenstionsv1.Deployment{}). // Deployment is the Application API
    Owns(&core.Pod{}).                // Deployment owns Pods created by it
    Complete(&MyReconciler{})
if err != nil {
    log.Error(err, "could not create controller")
    os.Exit(1)
}
```
* In the end start the manager
```
setupLog.Info("starting manager")
if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
    setupLog.Error(err, "problem running manager")
    os.Exit(1)
}
```

### InjectClient
This method is responsible for adding client to the `MyReconsile` object so that can be used in the `Reconsile` method. For detailed information please visit [here](https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/runtime/inject/inject.go#L75).
```
func (a *MyReconciler) InjectClient(c client.Client) error {
	log.Info(fmt.Sprint("Client Inject Method is Called"))
	a.Client = c
	return nil
}
```

### Reconcile 
* Get all the deployment running inside the cluster
```
dep := &extenstionsv1.Deployment{}
err := a.Get(context.TODO(), req.NamespacedName, dep)
if err != nil {
    return reconcile.Result{}, err
}
```

* Add sidecar in the deployment object if it match the label ```node-sidecar: "true"```. This method is using [isSidecarRunning](https://github.com/aminmithil/node-sidecar-injector/blob/master/main.go#L150) and [sideCarContainer](https://github.com/aminmithil/node-sidecar-injector/blob/master/main.go#L137)
```
if val, found := dep.Labels["node-sidecar"]; val == "true" && found {
    isSidecarRunning := isSidecarRunning(dep)
    // don't inject if sidecar is already in the deployment
    if !isSidecarRunning {
        dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, sideCarContainer())
    }
}
```

* Update the running deployment
```
err = a.Update(context.TODO(), dep)
if err != nil {
    return reconcile.Result{}, err
}
```