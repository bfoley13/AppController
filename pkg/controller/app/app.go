package app

import (
	"context"

	appv1alpha1 "github.com/bfoley13/appcontroller/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

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
}
