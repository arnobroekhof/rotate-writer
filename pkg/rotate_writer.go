package pkg

type RotateWriter interface {
	Write(b []byte) (int, error)
	Close() (err error)
	Rotate() (err error)
	stream() (err error)
}
