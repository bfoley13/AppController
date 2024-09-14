package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/devhub/armdevhub"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
)

func NewDevHubClient(ctx context.Context) (*armdevhub.DeveloperHubServiceClient, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	factory, err := armdevhub.NewClientFactory("26ad903f-2330-429d-8389-864ac35c4350", cred, nil)
	if err != nil {
		return nil, err
	}

	return factory.NewDeveloperHubServiceClient(), nil
}

func NewACRClient(ctx context.Context) (*armcontainerregistry.RegistriesClient, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	factory, err := armcontainerregistry.NewClientFactory("26ad903f-2330-429d-8389-864ac35c4350", cred, nil)
	if err != nil {
		return nil, err
	}

	return factory.NewRegistriesClient(), nil
}

func NewBlobClientFromUrl(ctx context.Context, url string) (*blockblob.Client, error) {
	// cred, err := azidentity.NewDefaultAzureCredential(nil)
	// if err != nil {
	// 	return nil, err
	// }

	return blockblob.NewClientWithNoCredential(url, nil)
}

func NewACRRunsClient(ctx context.Context, subscription string) (*armcontainerregistry.RunsClient, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	return armcontainerregistry.NewRunsClient(subscription, cred, nil)
}
