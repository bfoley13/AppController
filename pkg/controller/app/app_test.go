package app

import (
	"context"
	"testing"

	appv1alpha1 "github.com/bfoley13/appcontroller/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestRunACR(t *testing.T) {
	t.Run("RunACR", func(t *testing.T) {

		app := appv1alpha1.Application{
			Spec: appv1alpha1.ApplicationSpec{
				Repository: &appv1alpha1.Repository{
					Owner:      "bfoley13",
					Name:       "go_echo",
					BranchName: "main",
				},
				DockerConfig: &appv1alpha1.DockerConfig{
					Dockerfile:   "Dockerfile",
					BuildContext: ".",
					ImageName:    "go_echo",
					ImageTag:     "latest",
				},
				Acr: &appv1alpha1.Acr{
					Id: "/subscriptions/26ad903f-2330-429d-8389-864ac35c4350/resourceGroups/bfoley-test/providers/Microsoft.ContainerRegistry/registries/appcontrollertest",
				},
			},
		}

		_, err := RunAcrBuild(context.Background(), app)
		assert.Nil(t, err)
	})
}
