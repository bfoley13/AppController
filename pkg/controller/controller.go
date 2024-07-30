package controller

import (
	"os"

	"github.com/go-logr/logr"
	cfgv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	ubzap "go.uber.org/zap"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	secv1 "sigs.k8s.io/secrets-store-csi-driver/apis/v1"
)

var scheme = runtime.NewScheme()

const (
	nicIngressClassIndex = "spec.ingressClassName"
)

func init() {
	registerSchemes(scheme)
	// need to set klog logger to same logger to get consistent logging format for all logs.
	// without this things like leader election that use klog will not have the same format.
	// https://github.com/kubernetes/client-go/blob/560efb3b8995da3adcec09865ca78c1ddc917cc9/tools/leaderelection/leaderelection.go#L250
	klog.SetLogger(getLogger())
}

func getLogger(opts ...zap.Opts) logr.Logger {
	// use raw opts to add caller info to logs
	rawOpts := zap.RawZapOpts(ubzap.AddCaller())

	// zap is the default recommended logger for controller-runtime when wanting json structured output
	return zap.New(append(opts, rawOpts)...)
}

func registerSchemes(s *runtime.Scheme) {
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(appv1.AddToScheme(s))
	utilruntime.Must(secv1.Install(s))
	utilruntime.Must(cfgv1alpha2.AddToScheme(s))
	utilruntime.Must(policyv1alpha1.AddToScheme(s))
	utilruntime.Must(apiextensionsv1.AddToScheme(s))
}

func NewManager() (manager.Manager, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		os.Exit(1)
	}

	mgr, err := manager.New(cfg, manager.Options{
		Scheme: scheme,
	})
	if err != nil {
		os.Exit(1)
	}

	return mgr, nil
}
