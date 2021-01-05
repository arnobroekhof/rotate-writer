package s3

import (
	"errors"
	"fmt"
	"github.com/minio/minio-go/v6"
	"io"
	"sync"
	"time"
)

var bytesWritten int

type ObjectSizeRotateWriter struct {
	lock            sync.Mutex
	endpoint        string
	objectName      string
	objectExtension string
	bucketName      string
	location        string
	useSSL          bool
	accessKeyID     string
	secretAccessKey string
	rotationSize    int
	minioClient     *minio.Client
	pipeReader      *io.PipeReader
	pipeWriter      *io.PipeWriter
}

var waitGroup sync.WaitGroup

func NewS3RotateWriter(endpoint, bucketName, objectName, objectExtension, location, accessKeyID, secretAcessKey string, useSSL bool, rotationSize int) (*ObjectSizeRotateWriter, error) {
	return newS3RotateWriter(endpoint, objectName, objectExtension, bucketName, location, accessKeyID, secretAcessKey, useSSL, rotationSize)
}

func newS3RotateWriter(endpoint, objectName, objectExtension, bucketName, location, accessKeyID, secretAcessKey string, useSSL bool, rotationSize int) (w *ObjectSizeRotateWriter, err error) {
	w = &ObjectSizeRotateWriter{
		endpoint:        endpoint,
		objectName:      objectName,
		bucketName:      bucketName,
		location:        location,
		useSSL:          useSSL,
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAcessKey,
		rotationSize:    rotationSize,
		objectExtension: objectExtension,
	}

	// init the client
	w.minioClient, err = minio.New(endpoint, accessKeyID, secretAcessKey, useSSL)
	if err != nil {
		return nil, err
	}

	// check if the bucket exists and create if not
	err = w.minioClient.MakeBucket(bucketName, location)
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := w.minioClient.BucketExists(bucketName)
		if errBucketExists != nil && !exists {
			return nil, errBucketExists
		}
	}

	// construct pipe reader and writer
	w.pipeReader, w.pipeWriter = io.Pipe()
    go func() {
        err = w.stream()
        if err != nil {
            return
        }
    }()
	return w, nil
}

func (w *ObjectSizeRotateWriter) Write(output []byte) (int, error) {
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

func (w *ObjectSizeRotateWriter) stream() (err error) {
	waitGroup.Add(1)
	defer w.pipeReader.Close()
	_, err = w.minioClient.PutObject(w.bucketName, fmt.Sprintf("%s%s", w.objectName, w.objectExtension), w.pipeReader, -1, minio.PutObjectOptions{
		ContentType: "application/octet-stream"})
	if err != nil {
		return err
	}
	waitGroup.Done()
	return err
}

func (w *ObjectSizeRotateWriter) Close() (err error) {
	err = w.pipeWriter.Close()
	// wait until the pipe is closed before proceeding
	waitGroup.Wait()
	return err
}

func (w *ObjectSizeRotateWriter) Rotate() (err error) {

	if w.pipeWriter != nil {
		_ = w.pipeWriter.Close()
		// wait until the pipe is closed before proceeding
		waitGroup.Wait()
	}

	// reset bytes written to 0
	bytesWritten = 0

	// source object
	src := minio.NewSourceInfo(w.bucketName, fmt.Sprintf("%s%s", w.objectName, w.objectExtension), nil)
	// destination
	dst, err := minio.NewDestinationInfo(w.bucketName, fmt.Sprintf("%s-%d%s", w.objectName, time.Now().UnixNano(), w.objectExtension), nil, nil)
	if err != nil {
		return err
	}
	err = w.minioClient.CopyObject(dst, src)
	if err != nil {
		return err
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
