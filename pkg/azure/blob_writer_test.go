package azure

import (
	"context"
	"fmt"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"net/url"
	"testing"
)

const (
	storageAccount   = "devstoreaccount1"
	storageAccessKey = "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
	containerName    = "testcontainer"
	endpoint         = "localhost:10000"
)

func TestNewAzureBlobRotateWriter(t *testing.T) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "mcr.microsoft.com/azure-storage/azurite",
		ExposedPorts: []string{"10000:10000"},
		Cmd:          []string{"azurite-blob", "--blobHost", "0.0.0.0"},
	}

	azuriteC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	assert.Nil(t, err)

	// nolint:errcheck
	defer azuriteC.Terminate(ctx)

	testString2 := "Hello World 2"
	testString1 := "Hello World 1"
	testString3 := "Hello World 3"

	rotateWriter, err := NewAzureBlobRotateWriter(
		storageAccount,
		storageAccessKey,
		containerName,
		"testfile",
		14,
		endpoint,
	)
	assert.Nil(t, err)

	t.Log("Writing string 1")
	_, err = rotateWriter.Write([]byte(testString1))
	assert.Nil(t, err)

	t.Log("Writing string 2")
	_, err = rotateWriter.Write([]byte(testString2))
	assert.Nil(t, err)

	t.Log("Writing string 3")
	_, err = rotateWriter.Write([]byte(testString3))
	assert.Nil(t, err)

	t.Log("Closing")
	err = rotateWriter.Close()
	assert.Nil(t, err)

	// construct the login
	azCredentials, err := azblob.NewSharedKeyCredential(storageAccount, storageAccessKey)
	assert.Nil(t, err)
	//construct the pipeline
	// create the pipeline
	p := azblob.NewPipeline(azCredentials, azblob.PipelineOptions{})

	testURI, _ := url.Parse(fmt.Sprintf("http://%s/%s/%s", endpoint, storageAccount, containerName))

	containerURL := azblob.NewContainerURL(*testURI, p)

	for marker := (azblob.Marker{}); marker.NotDone(); {
		listBlob, err := containerURL.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{})
		assert.Nil(t, err)
		marker = listBlob.NextMarker
		assert.Equal(t, 3, len(listBlob.Segment.BlobItems), "there need to be 3 blob items inside the container")
	}
}
