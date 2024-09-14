package main

import (
	"fmt"
	"os"

	"github.com/bfoley13/appcontroller/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	mgr, err := controller.NewManager()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
