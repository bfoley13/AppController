package main

import (
	"fmt"
	"os"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/bfoley13/appcontoller/pkg/controller"
)

func main() {
	mgr, err := controller.NewManager()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return mgr.Start(ctrl.SetupSignalHandler())
}
