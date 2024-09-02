package app

import (
	"context"

	appv1alpha1 "github.com/bfoley13/appcontroller/api/v1alpha1"
	"github.com/bfoley13/draft/pkg/template"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

	fileWriter := TemplateFiles{}
	deploymentTemplate, err := template.GetTemplate("Deployment", "0.0.5", "", fileWriter)
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

	//now you have yaml bytes in the fileWrite but need to apply them
	//https://stackoverflow.com/questions/58783939/using-client-go-to-kubectl-apply-against-the-kubernetes-api-directly-with-mult

	return ctrl.Result{}, nil
}
