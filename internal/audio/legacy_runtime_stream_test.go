package audio

import (
	"errors"
	"fmt"
	"io"
	"sync/atomic"
)

type streamFailure struct{ err error }

type ownedRuntimeStream struct {
	io.ReadSeeker
	owner io.ReadSeeker
}

func (stream *ownedRuntimeStream) Close() error { return closeRuntimeSource(stream.owner) }

type runtimeStream struct {
	source  io.ReadSeeker
	failure atomic.Pointer[streamFailure]
}

func (stream *runtimeStream) Failure() error {
	if failure := stream.failure.Load(); failure != nil {
		return failure.err
	}
	return nil
}

func (stream *runtimeStream) Read(buffer []byte) (int, error) {
	if stream.Failure() != nil {
		return 0, io.EOF
	}
	count, err := stream.source.Read(buffer)
	if errors.Is(err, io.EOF) {
		return count, io.EOF
	}
	if err != nil {
		stream.failure.CompareAndSwap(nil, &streamFailure{err: fmt.Errorf("runtime audio read: %w", err)})
		return count, io.EOF
	}
	return count, err
}

func (stream *runtimeStream) Seek(offset int64, whence int) (int64, error) {
	if err := stream.Failure(); err != nil {
		return 0, err
	}
	position, err := stream.source.Seek(offset, whence)
	if err != nil {
		failure := fmt.Errorf("runtime audio seek: %w", err)
		stream.failure.CompareAndSwap(nil, &streamFailure{err: failure})
		return position, failure
	}
	return position, nil
}
