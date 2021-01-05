package s3

import (
	"context"
	"github.com/minio/minio-go/v6"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
)

func TestNewS3RotateWriter(t *testing.T) {

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio",
		ExposedPorts: []string{"9000:9000"},
		Cmd:          []string{"server", "/data"},
	}

	minioC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		t.Fatal(err)
	}

	// nolint:errcheck
	defer minioC.Terminate(ctx)

	testString2 := "Hello World 2"
	testString1 := "Hello World 1"
	testString3 := "Hello World 3"

	rotateWriter, err := NewS3RotateWriter("localhost:9000",
		"testbucket", "testfile", ".txt",
		"eu-central-1", "minioadmin",
		"minioadmin",
		false,
		14,
	)
	assert.Nil(t, err)

	_, err = rotateWriter.Write([]byte(testString1))
	assert.Nil(t, err)

	_, err = rotateWriter.Write([]byte(testString2))
	assert.Nil(t, err)

	_, err = rotateWriter.Write([]byte(testString3))
	assert.Nil(t, err)

	err = rotateWriter.Close()
	assert.Nil(t, err)

	minioClient, err := minio.New("localhost:9000", "minioadmin", "minioadmin", false)
	if err != nil {
		t.Fatal(err)
	}

	// Create a done channel to control 'ListObjects' go routine.
	doneCh := make(chan struct{})
	defer close(doneCh)

	var itemCount = 0
	for object := range minioClient.ListObjects("testbucket", "", false, doneCh) {
		assert.Equal(t, 13, int(object.Size), "object is not the expected size")
		itemCount++
	}
	assert.Equal(t, 3, itemCount)

}
