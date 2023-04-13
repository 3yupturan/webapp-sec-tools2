package customtemplates

import (
	"bytes"
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/projectdiscovery/gologger"
	"os"
	"path/filepath"
	"strings"
)

type customTemplateAzureBlob struct {
	azureBlobClient *azblob.Client
	containerName   string
}

func getAzureBlobClient(tenantID string, clientID string, clientSecret string, serviceURL string) (*azblob.Client, error) {
	// Create an Azure credential using the provided credentials
	credentials, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		gologger.Error().Msgf("Invalid Azure credentials: %v", err)
		return nil, err
	}

	// Create a client to manage Azure Blob Storage
	client, err := azblob.NewClient(serviceURL, credentials, nil)
	if err != nil {
		gologger.Error().Msgf("Error creating Azure Blob client: %v", err)
		return nil, err
	}

	return client, nil
}

func (bk *customTemplateAzureBlob) Download(location string, ctx context.Context) {
	// Define the local path to which the templates will be downloaded
	downloadPath := filepath.Join(location, CustomAzureTemplateDirectory, bk.containerName)

	// Get the list of all templates from the container
	pager := bk.azureBlobClient.NewListBlobsFlatPager(bk.containerName, &azblob.ListBlobsFlatOptions{
		// Don't include previous versions of the templates if versioning is enabled on the container
		Include: azblob.ListBlobsInclude{Snapshots: false, Versions: false},
	})

	// Loop through the list of blobs in the container and determine if they should be added to the list of templates
	// to be returned, and subsequently downloaded
	for pager.More() {
		resp, err := pager.NextPage(context.TODO())
		if err != nil {
			gologger.Error().Msgf("Error listing templates in Azure Blob container: %v", err)
			return
		}

		for _, blob := range resp.Segment.BlobItems {
			// If the blob is a .yaml, .yml, or .json file, download the file to the local filesystem
			if strings.HasSuffix(*blob.Name, ".yaml") || strings.HasSuffix(*blob.Name, ".yml") || strings.HasSuffix(*blob.Name, ".json") {
				// Download the template to the local filesystem at the downloadPath
				err := downloadTemplate(bk.azureBlobClient, bk.containerName, *blob.Name, filepath.Join(downloadPath, *blob.Name), ctx)
				if err != nil {
					gologger.Error().Msgf("Error downloading template: %v", err)
				}
			}
		}
	}
}

// Update updates the templates from the Azure Blob Storage container to the local filesystem. This is effectively a
// wrapper of the Download function which downloads of all templates from the container and doesn't manage a
// differential update.
func (bk *customTemplateAzureBlob) Update(location string, ctx context.Context) {
	// Treat the update as a download of all templates from the container
	bk.Download(location, ctx)
}

// downloadTemplate downloads a template from the Azure Blob Storage container to the local filesystem with the provided
// blob path and outputPath.
func downloadTemplate(client *azblob.Client, containerName string, path string, outputPath string, ctx context.Context) error {
	// Download the blob as a byte stream
	get, err := client.DownloadStream(ctx, containerName, path, nil)
	if err != nil {
		gologger.Error().Msgf("Error downloading template: %v", err)
		return err
	}

	downloadedData := bytes.Buffer{}
	retryReader := get.NewRetryReader(ctx, &azblob.RetryReaderOptions{})
	_, err = downloadedData.ReadFrom(retryReader)
	if err != nil {
		gologger.Error().Msgf("Error reading template: %v", err)
		return err
	}

	err = retryReader.Close()
	if err != nil {
		gologger.Error().Msgf("Error closing template filestream: %v", err)
		return err
	}

	// Write the downloaded template to the local filesystem at the outputPath with the filename of the blob name
	err = os.WriteFile(outputPath, downloadedData.Bytes(), 0644)

	return err
}
