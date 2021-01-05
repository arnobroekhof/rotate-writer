package files

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	errCreatingFile = errors.New("error creating newFileRotateWriter file")
	errClosingFile  = errors.New("error closing File")
	errRenamingFile = errors.New("error renaming File")
	bytesWritten    int
)

type FileSizeRotateWriter struct {
	lock         sync.Mutex
	filename     string   // should be set to the actual filename
	rotationSize int      // rotation size in bytes
	fp           *os.File // file handler
}

// NewFileSizeRotateWriter makes a newFileRotateWriter FileSizeRotateWriter. Return nil if error occurs during setup.
func NewFileSizeRotateWriter(filename string, rotationSize int) (*FileSizeRotateWriter, error) {
	// Check file before we initialize.
	return newFileRotateWriter(filename, rotationSize)
}

func newFileRotateWriter(filename string, rotationSize int) (w *FileSizeRotateWriter, err error) {
	w = &FileSizeRotateWriter{filename: filename, rotationSize: rotationSize}

	// Create a file.
	w.fp, err = os.Create(w.filename)
	if err != nil {
		return nil, errCreatingFile
	}

	return w, nil
}

// Write satisfies the io.Writer interface.
func (w *FileSizeRotateWriter) Write(output []byte) (int, error) {
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

	// perform the actual write
	write, err := w.fp.Write(output)
	if err != nil {
		return 0, err
	}
	// increment bytes seen with the length of the byte array
	bytesWritten = bytesWritten + len(output)

	return write, nil
}

func (w *FileSizeRotateWriter) Close() (err error) {
	return w.fp.Close()
}

// Perform the actual act of rotating and re-opening / re-creating the file file.
func (w *FileSizeRotateWriter) Rotate() (err error) {

	// close the current file
	if w.fp != nil {
		err = w.fp.Close()
		w.fp = nil
		if err != nil {
			return errClosingFile
		}
	}

	// rename the file if exists
	_, err = os.Stat(w.filename)
	if err == nil {
		// file exists
		err = os.Rename(w.filename, fmt.Sprintf("%s-%d", w.filename, time.Now().UnixNano()))
		if err != nil {
			return errRenamingFile
		}
	}

	// create a newFileRotateWriter file
	w.fp, err = os.Create(w.filename)
	if err != nil {
		return errCreatingFile
	}

	// reset bytes seen
	bytesWritten = 0

	return nil
}

//nolint:unused
func (w *FileSizeRotateWriter) stream() (err error) {
	// does nothing yes
	return nil
}
