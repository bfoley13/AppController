package v1alpha1

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}

type ApplicationSpec struct {
	ApplicationName string        `json:"appName"`
	Namespace       string        `json:"namespace"`
	Repository      *Repository   `json:"repository,omitempty"`
	DockerConfig    *DockerConfig `json:"DockerConfig,omitempty"`
	Acr             *Acr          `json:"ACR,omitempty"`
}

type Repository struct {
	Owner      string `json:"owner"`
	Name       string `json:"name"`
	BranchName string `json:"branchName"`
}

type DockerConfig struct {
	Dockerfile   string `json:"dockerfile"`
	BuildContext string `json:"buildContext"`
}

type Acr struct {
	Id string `json:"id"`
}

type ManagedObjectReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
	APIGroup  string `json:"apiGroup"`
}

type ApplicationStatus struct {
	Conditions                    []metav1.Condition       `json:"conditions"`
	ControllerReplicas            int32                    `json:"controllerReplicas"`
	ControllerReadyReplicas       int32                    `json:"controllerReadyReplicas"`
	ControllerAvailableReplicas   int32                    `json:"controllerAvailableReplicas"`
	ControllerUnavailableReplicas int32                    `json:"controllerUnavailableReplicas"`
	CollisionCount                int32                    `json:"collisionCount"`
	ManagedResourceRefs           []ManagedObjectReference `json:"managedResourceRefs,omitempty"`
}

type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ApplicationSpec   `json:"spec"`
	Status            ApplicationStatus `json:"status,omitempty"`
}

func (n *Application) GetCondition(t string) *metav1.Condition {
	return meta.FindStatusCondition(n.Status.Conditions, t)
}

func (n *Application) SetCondition(c metav1.Condition) {
	current := n.GetCondition(c.Type)

	if current != nil && current.Status == c.Status && current.Message == c.Message && current.Reason == c.Reason {
		current.ObservedGeneration = n.Generation
		return
	}

	c.ObservedGeneration = n.Generation
	c.LastTransitionTime = metav1.Now()
	meta.SetStatusCondition(&n.Status.Conditions, c)
}

func (n *Application) Collides(ctx context.Context, cl client.Client) (bool, string, error) {
	lgr := logr.FromContextOrDiscard(ctx).WithValues("name", n.Name, "namespace", n.Namespace)
	lgr.Info("checking for Application collisions")

	var appList ApplicationList
	if err := cl.List(ctx, &appList); err != nil {
		lgr.Error(err, "listing Applications")
		return false, "", fmt.Errorf("listing Applications: %w", err)
	}

	// TODO: likely need to check ACR + other things later
	for _, app := range appList.Items {
		if app.Name == n.Name && app.Namespace == n.Namespace {
			lgr.Info("Application collision found")
			return true, fmt.Sprintf("app.Name \"%s\" is invalid because Application \"%s\" already in use", app.Name, app.Name), nil
		}
	}

	return false, "", nil
}

type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}
