LOCATION=westus2
RESOURCE_GROUP=brfole-test
CLUSTER_NAME=brfole-test-cluster
ACR_NAME=brfoletest
MSI_NAME=app-controller-msi
SUBSCRIPTION=26ad903f-2330-429d-8389-864ac35c4350
SERVICE_ACCOUNT_NAMESPACE=app-controller
SERVICE_ACCOUNT_NAME=app-controller-sa
FEDERATED_IDENTITY_CREDENTIAL_NAME=app-controller-fic


# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
CONTROLLER_TOOLS_VERSION ?= v0.14.0

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

crd: generate manifests ## Generates all associated files from CRD

manifests: controller-gen ## Generate CRD manifest
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./api/..." output:crd:artifacts:config=config/crd/bases

generate: $(CONTROLLER_GEN) ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object paths="./api/..."


controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

test-setup:
	az account set --subscription $(SUBSCRIPTION)
	az group create --name $(RESOURCE_GROUP) --location $(LOCATION)
	az acr create --name $(ACR_NAME) --resource-group $(RESOURCE_GROUP) --sku basic
	az aks create --resource-group "${RESOURCE_GROUP}" --name "${CLUSTER_NAME}" --enable-oidc-issuer --enable-workload-identity --generate-ssh-keys --attach-acr $(ACR_NAME)
	export AKS_OIDC_ISSUER="$(az aks show --name "${CLUSTER_NAME}" --resource-group "${RESOURCE_GROUP}" --query "oidcIssuerProfile.issuerUrl" --output tsv)"
	az identity create --name "${MSI_NAME}" --resource-group "${RESOURCE_GROUP}" --location "${LOCATION}" --subscription "${SUBSCRIPTION}"
	export USER_ASSIGNED_CLIENT_ID="$(az identity show --resource-group "${RESOURCE_GROUP}" --name "${MSI_NAME}" --query 'clientId' --output tsv)"
	az identity federated-credential create --name ${FEDERATED_IDENTITY_CREDENTIAL_NAME} --identity-name "${MSI_NAME}" --resource-group "${RESOURCE_GROUP}" --issuer "${AKS_OIDC_ISSUER}" --subject system:serviceaccount:"${SERVICE_ACCOUNT_NAMESPACE}":"${SERVICE_ACCOUNT_NAME}" --audience api://AzureADTokenExchange
	export IDENTITY_PRINCIPAL_ID=$(az identity show --name "${MSI_NAME}" --resource-group "${RESOURCE_GROUP}" --query principalId --output tsv)
	az role assignment create --assignee-object-id "${IDENTITY_PRINCIPAL_ID}" --role "Owner" --scope "/subscriptions/${SUBSCRIPTION}" --assignee-principal-type ServicePrincipal
	az aks get-credentials --resource-group $(RESOURCE_GROUP) --name ${CLUSTER_NAME} --overwrite-existing
	echo $(USER_ASSIGNED_CLIENT_ID)

test-image-push:
	az acr login --name $(ACR_NAME)
	docker build -t $(ACR_NAME).azurecr.io/appcontroller .
	docker push $(ACR_NAME).azurecr.io/appcontroller
