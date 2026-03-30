package pklloader

import (
	"bytes"
	"io/fs"
	"strings"
	"time"
)

// prefixFS presents an fs.FS whose contents appear under a single named
// directory. Reading "prefix/foo.pkl" reads "foo.pkl" from the inner FS.
type prefixFS struct {
	prefix string // e.g. "config"
	inner  fs.FS
}

func (p prefixFS) Open(name string) (fs.File, error) {
	if name == p.prefix || name == p.prefix+"/" {
		return p.inner.Open(".")
	}

	stripped, ok := strings.CutPrefix(name, p.prefix+"/")
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	if stripped == "" {
		stripped = "."
	}
	return p.inner.Open(stripped)
}

// overlayFS is an ordered list of prefixFS entries. Open tries each entry
// in order and returns the first successful match. This allows composing
// multiple fs.FS sources under different path prefixes into a single fs.FS.
type overlayFS []prefixFS

func (o overlayFS) Open(name string) (fs.File, error) {
	for _, p := range o {
		f, err := p.Open(name)
		if err == nil {
			return f, nil
		}
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

// staticFS serves a single file with fixed content.
type staticFS struct {
	name    string
	content []byte
}

func (s staticFS) Open(name string) (fs.File, error) {
	if name != s.name {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return &staticFile{
		Reader: bytes.NewReader(s.content),
		info: staticFileInfo{
			name: s.name,
			size: int64(len(s.content)),
		},
	}, nil
}

type staticFile struct {
	*bytes.Reader
	info staticFileInfo
}

func (f *staticFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *staticFile) Close() error               { return nil }

type staticFileInfo struct {
	name string
	size int64
}

func (fi staticFileInfo) Name() string      { return fi.name }
func (fi staticFileInfo) Size() int64       { return fi.size }
func (fi staticFileInfo) Mode() fs.FileMode { return 0o444 }
func (fi staticFileInfo) ModTime() time.Time { return time.Time{} }
func (fi staticFileInfo) IsDir() bool       { return false }
func (fi staticFileInfo) Sys() any          { return nil }

