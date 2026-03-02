package stream

import (
	"fmt"
	"io"
	"sync"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
)

// Ensure Stream Data Backend implements data.Backend interface
var _ data.Backend = (*Backend)(nil)

// Backend object
type Backend struct {
	store map[string]io.ReadSeekCloser
	mu    sync.Mutex
}

// PipeReadSeeker fills the gap so that stream looks like a regular backend
// and to not implement branched code pathes. Main code in get_file will ensure
// that seek is never called
type PipeReadSeeker struct {
	pipe *io.PipeReader
}

func (r *PipeReadSeeker) Read(data []byte) (n int, err error) {
	return r.pipe.Read(data)
}

func (r *PipeReadSeeker) Close() error {
	return r.pipe.Close()
}

func (r *PipeReadSeeker) Seek(int64, int) (int64, error) {
	return 0, fmt.Errorf("seek not supported on stream backend")
}

// NewBackend instantiate a new Stream Data Backend
// from configuration passed as argument
func NewBackend() (b *Backend) {
	b = new(Backend)
	b.store = make(map[string]io.ReadSeekCloser)
	return
}

// GetFile implementation for steam data backend will search
// on filesystem the requested steam and return its reading filehandle
func (b *Backend) GetFile(file *common.File) (stream io.ReadSeekCloser, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	storeID := file.UploadID + "/" + file.ID
	stream, ok := b.store[storeID]
	if !ok {
		return nil, fmt.Errorf("missing reader")
	}

	delete(b.store, storeID)

	return stream, err
}

// AddFile implementation for stream data backend will create a pipe and block until download begins.
// The store entry is consumed (deleted) by GetFile on first retrieval.
// The deferred delete here is a safety net for the case where GetFile is
// never called — it cleans up the entry after io.Copy completes.
func (b *Backend) AddFile(file *common.File, stream io.Reader) (err error) {
	storeID := file.UploadID + "/" + file.ID

	pipeReader, pipeWriter := io.Pipe()
	pipeReaderSeeker := &PipeReadSeeker{pipe: pipeReader}

	b.mu.Lock()

	b.store[storeID] = pipeReaderSeeker
	defer delete(b.store, storeID)

	b.mu.Unlock()

	// This will block until download begins
	_, err = io.Copy(pipeWriter, stream)
	_ = pipeWriter.CloseWithError(err)

	return err
}

// RemoveFile does not need to be implemented cleaning occurs in AddFile's defer delete
func (b *Backend) RemoveFile(file *common.File) (err error) {
	return nil
}
