package app

import (
	"context"
	"fmt"
	"time"

	az "github.com/Azure/go-autorest/autorest/azure"
	appv1alpha1 "github.com/bfoley13/appcontroller/api/v1alpha1"
	"github.com/bfoley13/appcontroller/pkg/azure"
	"github.com/bfoley13/draft/pkg/template"
	"github.com/go-logr/logr"
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
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func init() {
	ctrl.SetLogger(getLogger())
	// need to set klog logger to same logger to get consistent logging format for all logs.
	// without this things like leader election that use klog will not have the same format.
	// https://github.com/kubernetes/client-go/blob/560efb3b8995da3adcec09865ca78c1ddc917cc9/tools/leaderelection/leaderelection.go#L250
	klog.SetLogger(getLogger())
}

func getLogger(opts ...zap.Opts) logr.Logger {

	// zap is the default recommended logger for controller-runtime when wanting json structured output
	return zap.New(opts...)
}

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

<<<<<<< HEAD
=======
	runResp, err := RunAcrBuild(ctx, app)
	if err != nil {
		lgr.Error(err, "unable to run acr build")
		return ctrl.Result{}, err
	}

	if runResp == nil || runResp.Properties == nil {
		lgr.Info("run respoinse or priperties is nil")
		return ctrl.Result{}, err
	}

>>>>>>> c44d3d3 (update readme)
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
	deploymentTemplate.Config.SetVariable("IMAGENAME", fmt.Sprintf("%s/%s", *runResp.Properties.OutputImages[0].Registry, *runResp.Properties.OutputImages[0].Repository))
	deploymentTemplate.Config.SetVariable("IMAGETAG", *runResp.Properties.OutputImages[0].Tag)
	deploymentTemplate.Config.SetVariable("CPULIMIT", app.Spec.Resources.CPULimit)
	deploymentTemplate.Config.SetVariable("MEMLIMIT", app.Spec.Resources.MEMLimit)
	deploymentTemplate.Config.SetVariable("CPUREQ", app.Spec.Resources.CPUReq)
	deploymentTemplate.Config.SetVariable("MEMREQ", app.Spec.Resources.MEMReq)

	err = deploymentTemplate.CreateTemplates()
	if err != nil {
		lgr.Error(err, "unable to generate deployment templates")
		return ctrl.Result{}, err
	}

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

// func addDockerfileToRepo(ctx context.Context) error {
// 	s := github.NewGitHubService("<GH_TOKEN>")
// 	s.CreateBranch(ctx, "bfoley13", "go_echo", "test-branch")
// 	return nil
// }

func RunAcrBuild(ctx context.Context, app appv1alpha1.Application) (*armcontainerregistry.RunsClientGetResponse, error) {
	lgr := log.FromContext(ctx, "appcontroller-acrbuild")
	acrClient, err := azure.NewACRClient(ctx)
	if err != nil {
		lgr.Error(err, "unable to create devhub client")
		return nil, err
	}

	// resp, err := acrClient.GetBuildSourceUploadURL(ctx, "bfoley-test", "appcontrollertest", nil)
	// if err != nil {
	// 	lgr.Error(err, "unable to get build source url")
	// 	return err
	// }

	// uploadURL := resp.UploadURL
	// relativePath := resp.RelativePath

	// if uploadURL == nil || relativePath == nil || *uploadURL == "" || *relativePath == "" {
	// 	lgr.Error(err, "upload url or relative path is nil or empty")
	// 	return fmt.Errorf("upload url or relative path is nil or empty")
	// }

	// lgr.Info(fmt.Sprintf("upload url: %s, relative path: %s", *uploadURL, *relativePath))

	// blobURL, err := url.Parse(*uploadURL)
	// if err != nil {
	// 	lgr.Error(err, "unable to parse blob url")
	// 	return err
	// }

	// blobClient, err := azure.NewBlobClientFromUrl(ctx, *uploadURL)
	// if err != nil {
	// 	lgr.Error(err, "unable to create blob client")
	// 	return err
	// }

	// s := github.NewGitHubService("<GH_TOKEN>")
	// fileBytes, err := s.DownloadRepo(ctx, "bfoley13", "go_echo", "main")
	// if err != nil {
	// 	lgr.Error(err, "unable to download repo")
	// 	return err
	// }

	// os.MkdirAll(filepath.Dir(*relativePath), os.ModePerm)
	// os.WriteFile(*relativePath, fileBytes, os.ModePerm)

	// newFile, _ := os.Open(*relativePath)

	// blobResp, err := blobClient.UploadFile(ctx, newFile, nil)
	// if err != nil {
	// 	lgr.Error(err, "unable to upload tar file")
	// 	return err
	// }
	// lgr.Info(fmt.Sprintf("blob upload response: %v", blobResp))

	resource, err := az.ParseResourceID(app.Spec.Acr.Id)
	if err != nil {
		lgr.Error(err, "unable to parse resource id")
		return nil, err
	}

	poller, err := acrClient.BeginScheduleRun(ctx, resource.ResourceGroup, resource.ResourceName, &armcontainerregistry.DockerBuildRequest{
		DockerFilePath: toPtr(app.Spec.DockerConfig.Dockerfile),
		ImageNames:     []*string{toPtr(fmt.Sprintf("%s:%s", app.Spec.DockerConfig.ImageName, app.Spec.DockerConfig.ImageTag))},
		Type:           toPtr("DockerBuildRequest"),
		IsPushEnabled:  toPtr(true),
		SourceLocation: toPtr(fmt.Sprintf("https://github.com/%s/%s.git#%s:%s", app.Spec.Repository.Owner, app.Spec.Repository.Name, app.Spec.Repository.BranchName, app.Spec.DockerConfig.BuildContext)),
		Platform: &armcontainerregistry.PlatformProperties{
			OS:           toPtr(armcontainerregistry.OSLinux),
			Architecture: toPtr(armcontainerregistry.ArchitectureAmd64),
		},
	}, nil)
	if err != nil {
		lgr.Error(err, "unable schedule docker build run")
		return nil, err
	}

	acrRes, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		lgr.Error(err, "failed polling for acr build")
		return nil, err
	}
	runsClient, err := azure.NewACRRunsClient(ctx, resource.SubscriptionID)
	if err != nil {
		lgr.Error(err, "failed to get acr runs client")
		return nil, err
	}

	isTerminalState := false
	var acrRunResp *armcontainerregistry.RunsClientGetResponse
	for !isTerminalState {
		runsResp, err := runsClient.Get(ctx, resource.ResourceGroup, resource.ResourceName, *acrRes.Properties.RunID, nil)
		if err != nil {
			lgr.Error(err, "failed to get acr run")
			return nil, err
		}

		if runsResp.Properties != nil && runsResp.Properties.Status != nil {
			if *runsResp.Properties.Status == "Succeeded" {
				isTerminalState = true
				lgr.Info("acr build succeeded")
				acrRunResp = &runsResp
				continue
			} else if *runsResp.Properties.Status == "Failed" {
				err = fmt.Errorf("failed acr build: %s", string(*runsResp.Properties.RunErrorMessage))
				lgr.Error(err, "acr build failed")
				return nil, err
			} else {
				lgr.Info(fmt.Sprintf("acr build in state: %s", *runsResp.Properties.Status))
			}
		}

		lgr.Info("waiting for acr build to complete")
		time.Sleep(5 * time.Second)
	}

	return acrRunResp, nil
}

func toPtr[T any](s T) *T {
	v := s
	return &v
}
