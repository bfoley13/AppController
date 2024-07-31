package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	appv1apha1 "github.com/bfoley13/appcontroller/api/v1alpha1"
	"github.com/go-logr/logr"
	cfgv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	ubzap "go.uber.org/zap"
	"gopkg.in/yaml.v2"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	secv1 "sigs.k8s.io/secrets-store-csi-driver/apis/v1"
)

const (
	crdPath = "/crd"
)

var scheme = runtime.NewScheme()

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
	utilruntime.Must(appv1apha1.AddToScheme(s))
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

	setupLog := mgr.GetLogger().WithName("setup")
	// create non-caching clients, non-caching for use before manager has started
	cl, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "unable to create non-caching client")
		return nil, fmt.Errorf("creating non-caching client: %w", err)
	}

	if err = loadCRDs(cl, setupLog); err != nil {
		setupLog.Error(err, "unable to load crds")
		return nil, fmt.Errorf("loading crds: %w", err)
	}

	return mgr, nil
}

// loadCRDs loads the CRDs from the specified path into the cluster
func loadCRDs(c client.Client, log logr.Logger) error {
	log = log.WithValues("crdPath", crdPath)
	log.Info("reading crd directory")
	files, err := os.ReadDir(crdPath)
	if err != nil {
		return fmt.Errorf("reading crd directory %s: %w", crdPath, err)
	}

	log.Info("applying crds")
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		path := filepath.Join(crdPath, file.Name())
		log := log.WithValues("path", path)
		log.Info("reading crd file")
		var content []byte
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading crd file %s: %w", path, err)
		}

		log.Info("unmarshalling crd file")
		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := yaml.UnmarshalStrict(content, crd); err != nil {
			return fmt.Errorf("unmarshalling crd file %s: %w", path, err)
		}

		log.Info("upserting crd")
		ctx := context.Background()
		err = c.Patch(ctx, crd, client.Merge, client.FieldOwner("aks-app-controller"), client.ForceOwnership)
		if k8serr.IsNotFound(err) {
			err = c.Create(ctx, crd)
		}

		if err != nil {
			return fmt.Errorf("path/create crds: %w", err)
		}
	}

	log.Info("crds loaded")
	return nil
}
