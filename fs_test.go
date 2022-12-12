package tar2go

import (
	"archive/tar"
	"bytes"
	"io"
	"testing"
)

func tarFrom(t *testing.T, files map[string][]byte) io.ReaderAt {
	t.Helper()

	buf := bytes.NewBuffer(nil)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	for name, data := range files {
		err := tw.WriteHeader(&tar.Header{
			Name: name,
			Size: int64(len(data)),
		})
		if err != nil {
			t.Fatal(err)
		}

		_, err = tw.Write(data)
		if err != nil {
			t.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

func TestFS(t *testing.T) {
	testFiles := map[string][]byte{
		"foo":     []byte("foo"),
		"bar/baz": []byte("baz"),
	}
	ra := tarFrom(t, testFiles)

	idx := NewIndex(ra)
	fs := idx.FS()

	for name, data := range testFiles {
		f, err := fs.Open(name)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		fi, err := f.Stat()
		if err != nil {
			t.Fatal(err)
		}

		if fi.Size() != int64(len(data)) {
			t.Fatalf("file size mismatch: %d != %d", fi.Size(), len(data))
		}
		if fi.IsDir() {
			t.Fatalf("file is a directory")
		}

		if sys, ok := fi.Sys().(*tar.Header); !ok {
			t.Fatalf("exepcted Sys() to be a *tar.Header, got: %T", sys)
		}

		if _, ok := f.(io.ReaderAt); !ok {
			t.Fatalf("file should be a ReaderAt: %T", f)
		}

		buf := make([]byte, len(data))
		n, err := io.ReadFull(f, buf)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(data) {
			t.Fatalf("read %d bytes, expected %d", n, len(data))
		}

		if !bytes.Equal(buf, data) {
			t.Fatalf("read data mismatch: %q != %q", string(buf), string(data))
		}
	}

}
