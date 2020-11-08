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
	"flag"
	"net/http"
	"os"
	"time"

	api "github.com/dsyer/spring-boot-operator/api/v1"
	"github.com/dsyer/spring-boot-operator/controllers"
	"github.com/go-logr/logr"
	"github.com/vmware-labs/reconciler-runtime/reconcilers"
	"github.com/vmware-labs/reconciler-runtime/tracker"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme     = runtime.NewScheme()
	setupLog   = ctrl.Log.WithName("setup")
	syncPeriod = 10 * time.Hour
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = api.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = apps.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var probesAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probesAddr, "probes-addr", ":8081", "The address health probes bind to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.Logger(true))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probesAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "controller-leader-election-helper-spring",
		SyncPeriod:             &syncPeriod,
		Port:                   9443,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = controllers.MicroserviceReconciler(
		reconcilers.Config{
			Client:    mgr.GetClient(),
			APIReader: mgr.GetAPIReader(),
			Recorder:  mgr.GetEventRecorderFor("Microservice"),
			Log:       ctrl.Log.WithName("controllers").WithName("Microservice"),
			Scheme:    mgr.GetScheme(),
			Tracker:   tracker.New(syncPeriod, ctrl.Log.WithName("controllers").WithName("Microservice").WithName("tracker")),
		},
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Microservice")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder
	if err = controllers.ServiceBindingReconciler(
		reconcilers.Config{
			Client:    mgr.GetClient(),
			APIReader: mgr.GetAPIReader(),
			Recorder:  mgr.GetEventRecorderFor("ServiceBinding"),
			Log:       ctrl.Log.WithName("controllers").WithName("ServiceBinding"),
			Scheme:    mgr.GetScheme(),
			Tracker:   tracker.New(syncPeriod, ctrl.Log.WithName("controllers").WithName("ServiceBinding").WithName("tracker")),
		},
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ServiceBinding")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("default", func(_ *http.Request) error { return nil }); err != nil {
		setupLog.Error(err, "unable to create health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("default", func(_ *http.Request) error { return nil }); err != nil {
		setupLog.Error(err, "unable to create ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func createRecorder(log logr.Logger, mgr ctrl.Manager) record.EventRecorder {
	config := mgr.GetConfig()
	kubeclientset, _ := kubernetes.NewForConfig(config)
	eventBroadcaster := record.NewBroadcaster()
	// eventBroadcaster.StartLogging(log.Info)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(clientgoscheme.Scheme, corev1.EventSource{Component: "spring-boot-operator"})
	return recorder
}
