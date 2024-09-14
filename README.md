# Run
1. Create AKS Cluster
2. Create & Connect ACR to cluster
3. docker build . -t <your_acr_name>.azurecr.io/appcontroller
4. docker push <your_acr_name>.azurecr.io/appcontroller
5. update test/manifest/app-controler.yaml container image to point to your ACR image
6. kubectl apply -f ./test/manifests/app-controller.yaml
7. Update the test/manifests/application.yaml to reflect an app youd like to deploy
8. kubectl apply -f ./test/manifests/application.yaml