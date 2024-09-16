package app

import (
	"context"
	"fmt"

	appv1alpha1 "github.com/bfoley13/appcontroller/api/v1alpha1"
	"github.com/bfoley13/draft/pkg/template"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

type TemplateFiles struct {
	Files map[string][]byte
}

func (t TemplateFiles) EnsureDirectory(dir string) error {
	return nil
}

func (t TemplateFiles) WriteFile(fileName string, fileBytes []byte) error {
	if t.Files == nil {
		t.Files = map[string][]byte{}
	}
	t.Files[fileName] = fileBytes
	return nil
}

type appReconciler struct {
	client client.Client
	events record.EventRecorder
}

func NewReconciler(mgr ctrl.Manager) error {
	reconciler := &appReconciler{
		client: mgr.GetClient(),
		events: mgr.GetEventRecorderFor("aks-app-controller"),
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.Application{}).
		Owns(&appsv1.Deployment{}).
		Named("appcontroller").
		Complete(reconciler); err != nil {
		return err
	}

	return nil
}

func (ar *appReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	lgr := log.FromContext(ctx, "appcontroller", req.NamespacedName)
	ctx = log.IntoContext(ctx, lgr)
	lgr.Info("reconciling app")

	var app appv1alpha1.Application
	if err := ar.client.Get(ctx, req.NamespacedName, &app); err != nil {
		if apierrors.IsNotFound(err) {
			lgr.Info("app not found")
			return ctrl.Result{}, nil
		}

		lgr.Error(err, "unable to fetch app")
		return ctrl.Result{}, err
	}

	fileWriter := &TemplateFiles{
		Files: map[string][]byte{},
	}
	deploymentTemplate, err := template.GetTemplate("Deployment", "0.0.4", ".", fileWriter)
	if err != nil {
		lgr.Error(err, "unable to get deployment template")
		return ctrl.Result{}, err
	}

	deploymentTemplate.Config.SetVariable("GENERATORLABEL", "azure-devx-appcontroller")
	deploymentTemplate.Config.SetVariable("PORT", app.Spec.AppPort)
	deploymentTemplate.Config.SetVariable("APPNAME", app.Name)
	deploymentTemplate.Config.SetVariable("NAMESPACE", app.Spec.Namespace)
	deploymentTemplate.Config.SetVariable("IMAGENAME", app.Spec.DockerConfig.ImageName)
	deploymentTemplate.Config.SetVariable("IMAGETAG", app.Spec.DockerConfig.ImageTag)
	deploymentTemplate.Config.SetVariable("CPULIMIT", app.Spec.Resources.CPULimit)
	deploymentTemplate.Config.SetVariable("MEMLIMIT", app.Spec.Resources.MEMLimit)
	deploymentTemplate.Config.SetVariable("CPUREQ", app.Spec.Resources.CPUReq)
	deploymentTemplate.Config.SetVariable("MEMREQ", app.Spec.Resources.MEMReq)

	err = deploymentTemplate.CreateTemplates()
	if err != nil {
		lgr.Error(err, "unable to generate deployment templates")
		return ctrl.Result{}, err
	}

	lgr.Info(fmt.Sprintf("files len: %d", len(fileWriter.Files)))
	lgr.Info(fmt.Sprintf("files: %v", fileWriter.Files))

	for _, fileBytes := range fileWriter.Files {
		lgr.Info(fmt.Sprintf("file Bytes: %s", string(fileBytes)))

		deserialized, err := deserialize(fileBytes)
		if err != nil {
			lgr.Error(err, "unable to deserialize file")
			return ctrl.Result{}, err
		}

		_, err = createObject(NewRestConfig(), deserialized)
		if err != nil {
			lgr.Error(err, "unable to create object")
			return ctrl.Result{}, err
		}
		lgr.Info("created object")
	}

	return ctrl.Result{}, nil
}

func createObject(restConfig *rest.Config, obj runtime.Object) (runtime.Object, error) {
	kubeClientSet := kubernetes.NewForConfigOrDie(restConfig)
	// Create a REST mapper that tracks information about the available resources in the cluster.
	groupResources, err := restmapper.GetAPIGroupResources(kubeClientSet.Discovery())
	if err != nil {
		return nil, err
	}
	rm := restmapper.NewDiscoveryRESTMapper(groupResources)

	// Get some metadata needed to make the REST request.
	gvk := obj.GetObjectKind().GroupVersionKind()
	gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}
	mapping, err := rm.RESTMapping(gk, gvk.Version)
	if err != nil {
		return nil, err
	}

	namespace, err := meta.NewAccessor().Namespace(obj)
	if err != nil {
		return nil, err
	}

	// Create a client specifically for creating the object.
	restClient, err := newRestClient(restConfig, mapping.GroupVersionKind.GroupVersion())
	if err != nil {
		return nil, err
	}

	// Use the REST helper to create the object in the "default" namespace.
	restHelper := resource.NewHelper(restClient, mapping)
	return restHelper.Create(namespace, false, obj)
}

func newRestClient(restConfig *rest.Config, gv schema.GroupVersion) (rest.Interface, error) {
	restConfig.ContentConfig = resource.UnstructuredPlusDefaultContentConfig()
	restConfig.GroupVersion = &gv
	if len(gv.Group) == 0 {
		restConfig.APIPath = "/api"
	} else {
		restConfig.APIPath = "/apis"
	}

	return rest.RESTClientFor(restConfig)
}

func deserialize(data []byte) (runtime.Object, error) {
	apiextensionsv1.AddToScheme(scheme.Scheme)
	apiextensionsv1beta1.AddToScheme(scheme.Scheme)
	decoder := scheme.Codecs.UniversalDeserializer()

	runtimeObject, _, err := decoder.Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}

	return runtimeObject, nil
}

func NewRestConfig() *rest.Config {
	return ctrl.GetConfigOrDie()
}
