package readwriter

import "io"

type chanWriter struct {
	ch chan byte
}

func NewChanWriter() *chanWriter {
	return &chanWriter{make(chan byte, 1024)}
}

func (w *chanWriter) Chan() <-chan byte {
	return w.ch
}
func (w *chanWriter) Write(p []byte) (int, error) {
	n := 0
	for _, b := range p {
		w.ch <- b
		n++
	}
	return n, nil
}

func (w *chanWriter) Close() error {
	close(w.ch)
	return nil
}

func (w *chanWriter) Read(p []byte) (int, error) {
	n := 0
	ch := w.Chan()
	for ; n < len(p); n++ {

		b, ok := <-ch
		if ok {
			p[n] = b
		} else {
			return n, io.EOF
		}
	}
	return n, nil
}
