package azure

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"io"
	"net/url"
	"sync"
	"time"
)

const (
	defaultBlobEndpoint = "blob.core.windows.net"
	rotatingBufferSize  = 2 * 1024 * 1024 // rotating buffer for uploading
	maxRotatingBuffer   = 3               // number of rotating buffers for uploading
)

var (
	bytesWritten int
)

// BlobSizeRotateWriter
// endpoint defaults to blob.core.windows.net if nil
// we are not using AppendBlob because that has a cap of max 5000 appends
type BlobSizeRotateWriter struct {
	lock             sync.Mutex
	storageAccount   string
	storageAccessKey string
	containerName    string
	objectName       string
	rotationSize     int    // rotation size in bytes
	endpoint         string // defaults to blob.core.windows.net
	blockBlobURL     azblob.BlockBlobURL
	containerURL     azblob.ContainerURL
	azCredentials    *azblob.SharedKeyCredential
	pipeReader       *io.PipeReader
	pipeWriter       *io.PipeWriter
}

var waitGroup sync.WaitGroup

func NewAzureBlobRotateWriter(storageAccount, storageAccessKey, containerName, objectName string, rotationSize int, endpoint string) (*BlobSizeRotateWriter, error) {
	return newAzureBlobRotateWriter(storageAccount, storageAccessKey, containerName, objectName, endpoint, rotationSize)
}

func newAzureBlobRotateWriter(storageAccount, storageAccessKey, containerName, objectName, endpoint string, rotationSize int) (w *BlobSizeRotateWriter, err error) {
	w = &BlobSizeRotateWriter{
		storageAccount:   storageAccount,
		storageAccessKey: storageAccessKey,
		containerName:    containerName,
		objectName:       objectName,
		rotationSize:     rotationSize,
		endpoint:         endpoint,
	}

	var scheme = "http"
	var containerURI *url.URL

	if endpoint == "" {
		w.endpoint = defaultBlobEndpoint
		scheme = "https"
		// check the container url
		containerURI, _ = url.Parse(fmt.Sprintf("%s://%s.%s/%s", scheme, w.storageAccount, w.endpoint, w.containerName))
	} else {
		containerURI, _ = url.Parse(fmt.Sprintf("%s://%s/%s/%s", scheme, w.endpoint, w.storageAccount, w.containerName))
	}

	// construct the login
	w.azCredentials, err = azblob.NewSharedKeyCredential(storageAccount, storageAccessKey)
	if err != nil {
		return nil, err
	}

	// create the pipeline
	p := azblob.NewPipeline(w.azCredentials, azblob.PipelineOptions{})

	w.containerURL = azblob.NewContainerURL(*containerURI, p)
	ctxWithTimeout, ctxCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer ctxCancel()

	_, err = w.containerURL.Create(ctxWithTimeout, azblob.Metadata{}, azblob.PublicAccessNone)
	if err != nil {
		sErr := err.(azblob.StorageError)
		// do not return an error if the container already exists
		if sErr.ServiceCode() != azblob.ServiceCodeContainerAlreadyExists {
			return nil, err
		}
	}

	w.blockBlobURL = w.containerURL.NewBlockBlobURL(w.objectName)
	w.pipeReader, w.pipeWriter = io.Pipe()

	go func() {
		err = w.stream()
		if err != nil {
			return
		}
	}()

	return w, nil
}

func (w *BlobSizeRotateWriter) Write(output []byte) (int, error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	if len(output) > w.rotationSize {
		return 0, errors.New("error bytes to be written bigger then rotation size, increase rotation size")
	}

	// perform rotation if the bytesWritten en to be written bytes are bigger then
	// the rotation size or else we will miss writes
	if (bytesWritten + len(output)) >= w.rotationSize {
		err := w.Rotate()
		if err != nil {
			return 0, err
		}
	}

	bytesWritten = bytesWritten + len(output)

	return w.pipeWriter.Write(output)
}

func (w *BlobSizeRotateWriter) Close() (err error) {
	err = w.pipeWriter.Close()
	// wait until the pipe is closed before proceeding
	waitGroup.Wait()
	return err
}

func (w *BlobSizeRotateWriter) Rotate() (err error) {
	if w.pipeWriter != nil {
		_ = w.pipeWriter.Close()
		// wait until the pipe is closed before proceeding
		waitGroup.Wait()
	}

	rotatedBlockBlobUrl := w.containerURL.NewBlockBlobURL(fmt.Sprintf("%s-%d", w.objectName, time.Now().UnixNano()))
	startCopy, err := rotatedBlockBlobUrl.StartCopyFromURL(context.Background(),
		w.blockBlobURL.URL(),
		azblob.Metadata{},
		azblob.ModifiedAccessConditions{},
		azblob.BlobAccessConditions{},
		azblob.AccessTierHot,
		nil)
	if err != nil {
		return err
	}

	copyStatus := startCopy.CopyStatus()
	var copyError error

Loop:
	for {
		switch copyStatus {
		case azblob.CopyStatusPending:

			time.Sleep(1 * time.Second)
			copyStatus = startCopy.CopyStatus()
		case azblob.CopyStatusFailed:

			copyError = fmt.Errorf("copy error unable to rotate file")
			break Loop
		case azblob.CopyStatusSuccess:
			_, dErr := w.blockBlobURL.Delete(context.Background(), azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})

			if dErr != nil {
				copyError = fmt.Errorf("copy error unable to rename / remove file")
				break Loop
			}
			break Loop
		}
	}

	if copyError != nil {
		return copyError
	}

	w.pipeReader, w.pipeWriter = io.Pipe()
	go func() {
		err = w.stream()
		if err != nil {
			return
		}
	}()

	return nil
}

func (w *BlobSizeRotateWriter) stream() (err error) {
	waitGroup.Add(1)
	defer w.pipeReader.Close()
	_, err = azblob.UploadStreamToBlockBlob(context.Background(), w.pipeReader, w.blockBlobURL,
		azblob.UploadStreamToBlockBlobOptions{BufferSize: rotatingBufferSize, MaxBuffers: maxRotatingBuffer})
	if err != nil {
		return err
	}
	waitGroup.Done()
	return nil
}
