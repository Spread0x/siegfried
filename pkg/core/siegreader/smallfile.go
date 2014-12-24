// Copyright 2014 Richard Lehane. All rights reserved.
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

package siegreader

import (
	"io"
	"log"
	"os"
	"sync"
)

type sfprotected struct {
	sync.Mutex
	val     int
	eofRead bool
}

// Buffer wraps an io.Reader, buffering its contents in byte slices that will keep growing until IO.EOF.
// It supports multiple concurrent Readers, including Readers reading from the end of the stream (ReverseReaders)
type SmallFile struct {
	quit      chan struct{} // allows quittting - otherwise will block forever while awaiting EOF
	src       io.Reader
	buf, eof  []byte
	completec chan struct{} // signals when the file has been completely read, allows EOF scanning beyond the small buffer
	complete  bool          // marks that the file has been completely read
	sz        int64
	w         sfprotected // index of latest write
}

// New instatatiates a new Buffer with a buf size of 4096*3, and an end-of-file buf size of 4096
func NewSF() *SmallFile {
	b := &SmallFile{}
	b.buf, b.eof = make([]byte, initialRead), make([]byte, readSz)
	return b
}

func (b *SmallFile) reset() {
	b.completec = make(chan struct{})
	b.complete = false
	b.sz = 0
	b.w.Lock()
	b.w.val = 0
	b.w.eofRead = false
	b.w.Unlock()
}

// SetSource sets the buffer's source.
// Can be any io.Reader. If it is an os.File, will load EOF buffer early. Otherwise waits for a complete read.
// The source can be reset to recycle an existing Buffer.
// Siegreader blocks on EOF reads or Size() calls when the reader isn't a file or the stream isn't completely read. The quit channel overrides this block.
func (b *SmallFile) SetSource(r io.Reader) error {
	if b == nil {
		return ErrNilBuffer
	}
	b.reset()
	b.src = r
	file := r.(*os.File)
	info, err := file.Stat()
	if err != nil {
		return err
	}
	b.sz = info.Size()
	if b.sz > int64(initialRead) {
		b.eof = b.eof[:cap(b.eof)]
	} else {
		b.eof = b.eof[:0]
	}
	_, err = b.fill() // initial fill
	return err
}

func (sf *SmallFile) SetQuit(q chan struct{}) {
	sf.quit = q
}

// Size returns the buffer's size, which is available immediately for files. Must wait for full read for streams.
func (sf *SmallFile) Size() int64 {
	return sf.sz
}

// non-blocking Size(), for use with zip reader
func (sf *SmallFile) SizeNow() int64 {
	return sf.sz
}

func (sf *SmallFile) grow() {
	// Rules for growing:
	// - if we need to grow, we have passed the initial read and can assume we will need whole file so, if we have a sz grow to it straight away
	// - otherwise, double capacity each time
	buf := make([]byte, int(sf.sz))
	copy(buf, sf.buf[:sf.w.val]) // don't care about unlocking as grow() is only called by fill()
	sf.buf = buf
}

// Rules for filling:
// - if we have a sz greater than 0, if there is stuff in the eof buffer, and if we are less than readSz from the end, copy across from the eof buffer
// - read readsz * 2 at a time
func (b *SmallFile) fill() (int, error) {
	// if we've run out of room, grow the buffer
	if len(b.buf)-readSz < b.w.val {
		b.grow()
	}
	// if we have an eof buffer, and we are near the end of the file, avoid an extra read by copying straight into the main buffer
	if len(b.eof) > 0 && b.w.eofRead && b.w.val+readSz >= int(b.sz) {
		close(b.completec)
		b.complete = true
		lr := int(b.sz) - b.w.val
		b.w.val += copy(b.buf[b.w.val:b.w.val+lr], b.eof[readSz-lr:])
		return b.w.val, io.EOF
	}
	// otherwise, let's read
	e := b.w.val + readSz
	if e > len(b.buf) {
		e = len(b.buf)
	}
	i, err := b.src.Read(b.buf[b.w.val:e])
	if i < readSz {
		err = io.EOF // Readers can give EOF or nil here
	}
	if err != nil {
		close(b.completec)
		b.complete = true
		if err == io.EOF {
			b.w.val += i
			// if we haven't got an eof buf already
			if len(b.eof) < readSz {
				b.sz = int64(b.w.val)
			}
		}
		return b.w.val, err
	}
	b.w.val += i
	return b.w.val, nil
}

func (b *SmallFile) fillEof() error {
	// return nil for a non-file or small file reader
	if len(b.eof) < readSz {
		return nil
	}
	b.w.Lock()
	defer b.w.Unlock()
	if b.w.eofRead {
		return nil // another reverse reader has already filled the buffer
	}
	rs := b.src.(io.ReadSeeker)
	_, err := rs.Seek(0-int64(readSz), 2)
	if err != nil {
		return err
	}
	_, err = rs.Read(b.eof)
	if err != nil {
		return err
	}
	_, err = rs.Seek(int64(b.w.val), 0)
	if err != nil {
		return err
	}
	b.w.eofRead = true
	return nil
}

// Return a slice from the buffer that begins at offset s and has length l
func (b *SmallFile) Slice(s, l int) ([]byte, error) {
	b.w.Lock()
	defer b.w.Unlock()
	var err error
	var bound int
	if s+l > b.w.val && !b.complete {
		for bound, err = b.fill(); s+l > bound && err == nil; bound, err = b.fill() {
		}
	}
	if err == nil && !b.complete {
		return b.buf[s : s+l], nil
	}
	if err == io.EOF || b.complete {
		if s+l > b.w.val {
			if s > b.w.val {
				return nil, io.EOF
			}
			// in the case of an empty file
			if b.Size() == 0 {
				return nil, io.EOF
			}
			return b.buf[s:b.w.val], io.EOF
		} else {
			return b.buf[s : s+l], nil
		}
	}
	return nil, err
}

// Return a slice from the end of the buffer that begins at offset s and has length l.
// This will block until the slice is available (which may be until the full stream is read).
func (b *SmallFile) EofSlice(s, l int) ([]byte, error) {
	// block until the EOF is available or we quit
	select {
	case <-b.quit:
		return nil, ErrQuit
	}
	var buf []byte
	if len(b.eof) > 0 && s+l <= len(b.eof) {
		buf = b.eof
	} else {
		select {
		case <-b.quit:
			return nil, ErrQuit
		case <-b.completec:
		}
		buf = b.buf[:int(b.sz)]
		if s+l == len(buf) {
			return buf[:len(buf)-s], io.EOF
		}
	}
	if s+l > len(buf) {
		if s > len(buf) {
			return nil, io.EOF
		}
		return buf[:len(buf)-s], io.EOF
	}
	return buf[len(buf)-(s+l) : len(buf)-s], nil
}

// SafeSlice calls Slice or EofSlice (which one depends on the rev argument: true for EofSlice)
func (b *SmallFile) SafeSlice(s, l int, rev bool) ([]byte, error) {
	if rev {
		return b.EofSlice(s, l)
	} else {
		return b.Slice(s, l)
	}
}

// MustSlice calls Slice or EofSlice (which one depends on the rev argument: true for EofSlice) and suppresses the error.
// If a non io.EOF error is encountered, it will be logged as a warning.
func (b *SmallFile) MustSlice(s, l int, rev bool) []byte {
	var slc []byte
	var err error
	if rev {
		slc, err = b.EofSlice(s, l)
	} else {
		slc, err = b.Slice(s, l)
	}
	if err != nil && err != io.EOF {
		log.Printf("Siegreader warning: failed to slice %d for length %d; slice length is is %d; reverse is %b", s, l, len(slc), rev)
	}
	return slc
}

// fill until a seek to a particular offset is possible, then return true, if it is impossible return false
func (b *SmallFile) canSeek(o int64, rev bool) (bool, error) {
	if rev {
		if b.sz > 0 {
			o = b.sz - o
			if o < 0 {
				return false, nil
			}
			// continue on to fill below
		} else {
			var err error
			for _, err = b.fill(); err == nil; _, err = b.fill() {
			}
			if err != io.EOF {
				return false, err
			}
			if b.sz-o < 0 {
				return false, nil
			}
			return true, nil
		}
	}
	b.w.Lock()
	defer b.w.Unlock()
	var err error
	var bound int
	if o > int64(b.w.val) {
		for bound, err = b.fill(); o > int64(bound) && err == nil; bound, err = b.fill() {
		}
	}
	if err == nil {
		return true, nil
	}
	if err == io.EOF {
		if o > int64(b.w.val) {
			return false, err
		}
		return true, nil
	}
	return false, err
}
