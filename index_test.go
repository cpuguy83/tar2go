package tar2go

import (
	"archive/tar"
	"bytes"
	"testing"
)

func TestDirectoryPrefixFiltering(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	testWriteFile(t, tw, "./foo", []byte{})
	testWriteFile(t, tw, "/bar", []byte{})

	if err := tw.Flush(); err != nil {
		t.Fatal(err)
	}

	idx := NewIndex(bytes.NewReader(buf.Bytes()))
	if _, err := idx.index("foo"); err != nil {
		t.Error(err)
	}

	if _, err := idx.index("./foo"); err != nil {
		t.Error(err)
	}

	if _, err := idx.index("bar"); err != nil {
		t.Error(err)
	}
	if _, err := idx.index("/bar"); err != nil {
		t.Error(err)
	}
}

func testWriteFile(t *testing.T, tw *tar.Writer, name string, content []byte) {
	t.Helper()
	err := tw.WriteHeader(&tar.Header{
		Name: name,
		Size: int64(len(content)),
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
}
