# rotate-writer

rotate-writer is a golang library that implements the io.Writer interface and that can be used to implement 
some sort of rotation for files based on the given bytes so they cannot grow without bounds.

currently rotate-writer supports 3 different writer targets

* File
* S3
* Azure Blob Storage

## Usage

### File

```go
package examples

import (
    "fmt"
    "github.com/arnobroekhof/rotate-writer/file"
)

func someFunction() {
    d1 := []byte("hello\ngo\n")
    w, err := file.NewFileRotateWriter("/some/destination", 10240000) // set rotation size to 10MB

    if err != nil {
        panic(err)
    }

    defer w.Close()

    _, err = w.Write(d1)
    if err != nil {
        fmt.Println(err)
    }

}
```

### S3

```go
package examples

import (
    "fmt"
    "github.com/arnobroekhof/rotate-writer/s3"
)

func filewriting() {
    d1 := []byte("hello\ngo\n")
    w, err := s3.NewS3RotateWriter("s3.amazonaws.com","somebucket", "/some.object.name", ".txt", "eu-west-1", "some-access-key", "some-secret-key",true ,10240000)
    
    if err != nil {
        panic(err)
    }

    defer w.Close()

    _, err = w.Write(d1)
    if err != nil {
        fmt.Println(err)
    }

}
```

### Azure Blob Storage

```go
package examples

import (
    "fmt"
    "github.com/arnobroekhof/rotate-writer/azure"
)

func filewriting() {
    d1 := []byte("hello\ngo\n")

    w, err := azure.NewAzureBlobRotateWriter("storageAccount", "storageAccessKey", "containerName", "objectName", 10240000, "")

    if err != nil {
        panic(err)
    }

    defer w.Close()

    _, err = w.Write(d1)
    if err != nil {
        fmt.Println(err)
    }

}
```

## TODO

- [x] File Writer
- [x] S3 Writer
- [x] Azure Blob Storage Writer
- [ ] Google Cloud Storage Writer
- [ ] Implement compression
- [ ] Implement custom rotation name instead of date
- [ ] Create Examples
- [ ] Use github actions for releases and linting
- [ ] Add Go Docs
