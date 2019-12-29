// Package dpkg decodes debian package files.
package dpkg

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/blakesmith/ar"
	"github.com/ulikunitz/xz"
)

// Deb represents a parsed debian package.
type Deb struct {
	files map[string]DataFile
}

// Files returns all the data files within the archive.
func (d *Deb) Files() map[string]DataFile {
	return d.files
}

func (d *Deb) loadFiles(tr *tar.Reader) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}
		var b bytes.Buffer
		if _, err := io.Copy(&b, tr); err != nil {
			return fmt.Errorf("reading %q: %v", hdr.Name, err)
		}
		d.files[hdr.Name] = DataFile{
			Hdr:  *hdr,
			Data: b.Bytes(),
		}
	}
	return nil
}

// DataFile represents a file within a deb.
type DataFile struct {
	Hdr  tar.Header
	Data []byte
}

// Open parses a debian package file.
func Open(r io.Reader) (*Deb, error) {
	a := ar.NewReader(r)
	out := Deb{
		files: map[string]DataFile{},
	}

	for {
		hdr, err := a.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, err
		}

		switch hdr.Name {
		case "debian-binary":
		case "control.tar.xz":
		case "data.tar.xz":
			tr, err := xzTar(a)
			if err != nil {
				return nil, fmt.Errorf("deb data: %v", err)
			}
			if err := out.loadFiles(tr); err != nil {
				return nil, fmt.Errorf("failed loading files in deb: %v", err)
			}
		default:
			return nil, fmt.Errorf("unrecognised file in deb: %v", hdr.Name)
		}
	}

	if len(out.files) == 0 {
		return nil, errors.New("no files in package")
	}

	return &out, nil
}

func xzTar(r io.Reader) (*tar.Reader, error) {
	r, err := xz.NewReader(r)
	if err != nil {
		return nil, err
	}
	return tar.NewReader(r), nil
}
