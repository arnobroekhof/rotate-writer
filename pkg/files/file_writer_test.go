package files

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogWriter(t *testing.T) {

	testString2 := "Hello World 2"
	testString1 := "Hello World 1"
	testString3 := "Hello World 3"

	// create tmp directory
	tmpDir, err := ioutil.TempDir("", "rotate_writer_*_test")
	defer os.RemoveAll(tmpDir)
	assert.Nil(t, err)

	testFile, err := ioutil.TempFile(tmpDir, "testfile")
	assert.Nil(t, err)

	rotateWriter, err := NewFileSizeRotateWriter(testFile.Name(), 14)
	assert.Nil(t, err)

	_, err = rotateWriter.Write([]byte(testString1))
	assert.Nil(t, err)

	_, err = rotateWriter.Write([]byte(testString2))
	assert.Nil(t, err)

	_, err = rotateWriter.Write([]byte(testString3))
	assert.Nil(t, err)

	err = rotateWriter.Close()
	assert.Nil(t, err)

	files, _ := ioutil.ReadDir(tmpDir)
	var fileCount int
	for _, f := range files {
		assert.Equal(t, int64(13), f.Size(), "file is not the size we expect")
		fileCount++
	}

	assert.Equal(t, 3, fileCount)
}
