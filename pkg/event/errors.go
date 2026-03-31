package event

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrHandlerNotFound Error = "handler not found"
	ErrNilContext      Error = "context must not be nil"
	ErrNilEvent        Error = "event must not be nil"
)

type PanicError struct {
	Value any
	Stack []byte
}

func (e *PanicError) Error() string {
	return "panic recovered during handler execution"
}
