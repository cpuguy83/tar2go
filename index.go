package tar2go

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
)

var (
	// ErrDelete should be returned by an UpdaterFn when the file should be deleted.
	ErrDelete = errors.New("delete")
)

type Index struct {
	rdr *io.SectionReader
	tar *tar.Reader
	idx map[string]*indexReader
}

func NewIndex(rdr io.ReaderAt) *Index {
	ras, ok := rdr.(ReaderAtSized)
	var size int64
	if !ok {
		size = 1<<63 - 1
	} else {
		size = ras.Size()
	}
	return &Index{
		rdr: io.NewSectionReader(rdr, 0, size),
		idx: make(map[string]*indexReader),
	}
}

func (i *Index) index(name string) (*indexReader, error) {
	if rdr, ok := i.idx[name]; ok {
		return rdr, nil
	}

	if i.tar == nil {
		i.tar = tar.NewReader(i.rdr)
	}

	for {
		hdr, err := i.tar.Next()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			if err == io.EOF {
				return nil, fs.ErrNotExist
			}
			return nil, fmt.Errorf("error Indexing tar: %w", err)
		}
		fmt.Fprintln(os.Stderr, hdr.Name)

		pos, err := i.rdr.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, fmt.Errorf("error getting file offset: %w", err)
		}
		rdr := &indexReader{rdr: i.rdr, offset: pos, size: hdr.Size, hdr: hdr}
		i.idx[hdr.Name] = rdr

		if hdr.Name == name {
			return rdr, nil
		}
	}
}

func (i *Index) Reader() *io.SectionReader {
	return io.NewSectionReader(i.rdr, 0, i.rdr.Size())
}

func (i *Index) FS() fs.FS {
	return &filesystem{idx: i}
}

type ReaderAtSized interface {
	io.ReaderAt
	Size() int64
}

type UpdaterFn func(string, ReaderAtSized) (ReaderAtSized, bool, error)

// Update creates a new tar with the files updated by the passed in updater function.
// The output tar is written to the passed in io.Writer
func (i *Index) Update(w io.Writer, updater UpdaterFn) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	rdr := i.Reader()
	tr := tar.NewReader(rdr)

	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("error reading tar: %w", err)
		}

		offset, err := rdr.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("error getting file offset: %w", err)
		}

		ra, updated, err := updater(hdr.Name, io.NewSectionReader(i.rdr, offset, hdr.Size))
		if err != nil {
			if err == ErrDelete {
				continue
			}
			return fmt.Errorf("error updating file %s: %w", hdr.Name, err)
		}

		if updated {
			hdr.Size = ra.Size()
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("error writing tar header: %w", err)
		}

		if _, err := io.Copy(tw, io.NewSectionReader(ra, 0, ra.Size())); err != nil {
			return fmt.Errorf("error writing tar file: %w", err)
		}
	}
}

type indexReader struct {
	rdr    io.ReaderAt
	offset int64
	size   int64
	hdr    *tar.Header
}

func (r *indexReader) Reader() *io.SectionReader {
	return io.NewSectionReader(r.rdr, r.offset, r.size)
}
