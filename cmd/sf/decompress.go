// Copyright 2015 Richard Lehane. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"io"
	"path/filepath"

	"archive/tar"
	"archive/zip"
	"compress/gzip"

	"github.com/richardlehane/siegfried/pkg/core/siegreader"
)

type decompressor interface {
	next() error // when finished, should return io.EOF
	reader() io.Reader
	path() string
	size() int64
	mod() string
}

type zipD struct {
	idx int
	p   string
	rdr *zip.Reader
	rc  io.ReadCloser
}

func newZip(ra io.ReaderAt, path string, sz int64) (decompressor, error) {
	zr, err := zip.NewReader(ra, sz)
	return &zipD{idx: -1, p: path, rdr: zr}, err
}

func (z *zipD) close() {
	if z.rc == nil {
		return
	}
	z.rc.Close()
}

func (z *zipD) next() error {
	z.close() // close the previous entry, if any
	// proceed
	z.idx++
	// scan past directories
	for ; z.idx < len(z.rdr.File) && z.rdr.File[z.idx].FileInfo().IsDir(); z.idx++ {
	}
	if z.idx >= len(z.rdr.File) {
		return io.EOF
	}
	var err error
	z.rc, err = z.rdr.File[z.idx].Open()
	return err
}

func (z *zipD) reader() io.Reader {
	return z.rc
}

func (z *zipD) path() string {
	return z.p + string(filepath.Separator) + filepath.FromSlash(z.rdr.File[z.idx].Name)
}

func (z *zipD) size() int64 {
	return int64(z.rdr.File[z.idx].UncompressedSize64)
}

func (z *zipD) mod() string {
	return z.rdr.File[z.idx].ModTime().String()
}

type tarD struct {
	p   string
	hdr *tar.Header
	rdr *tar.Reader
}

func newTar(r io.Reader, path string) (decompressor, error) {
	return &tarD{p: path, rdr: tar.NewReader(r)}, nil
}

func (t *tarD) next() error {
	var err error
	// scan past directories
	for t.hdr, err = t.rdr.Next(); err == nil && t.hdr.FileInfo().IsDir(); t.hdr, err = t.rdr.Next() {
	}
	return err
}

func (t *tarD) reader() io.Reader {
	return t.rdr
}

func (t *tarD) path() string {
	return t.p + string(filepath.Separator) + filepath.FromSlash(t.hdr.Name)
}

func (t *tarD) size() int64 {
	return t.hdr.Size
}

func (t *tarD) mod() string {
	return t.hdr.ModTime.String()
}

type gzipD struct {
	sz   int64
	p    string
	read bool
	rdr  *gzip.Reader
}

func newGzip(b siegreader.Buffer, path string) (decompressor, error) {
	_ = b.SizeNow()              // in case of a stream, force full read
	buf, err := b.EofSlice(0, 4) // gzip stores uncompressed size in last 4 bytes of the stream
	if err != nil {
		return nil, err
	}
	sz := int64(uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24)
	g, err := gzip.NewReader(siegreader.ReaderFrom(b))
	return &gzipD{sz: sz, p: path, rdr: g}, err
}

func (g *gzipD) next() error {
	if g.read {
		g.rdr.Close()
		return io.EOF
	}
	g.read = true
	return nil
}

func (g *gzipD) reader() io.Reader {
	return g.rdr
}

func (g *gzipD) path() string {
	return g.p + string(filepath.Separator) + g.rdr.Name
}

func (g *gzipD) size() int64 {
	return g.sz
}

func (g *gzipD) mod() string {
	return g.rdr.ModTime.String()
}
