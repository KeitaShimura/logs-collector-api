package mock

import "io"

// ErrorCloser は Close 時にフラグを立てるモック
type ErrorCloser struct {
	io.Reader
	Closed bool
}

func (e *ErrorCloser) Close() error {
	e.Closed = true

	return nil
}
