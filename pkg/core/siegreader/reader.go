package siegreader

import (
	"fmt"
	"io"
)

// Reader

type Reader struct {
	i, j    int
	scratch []byte
	end     bool // buffer adjoins the end of the file
	*Buffer
}

func (b *Buffer) NewReader() *Reader {
	// A BOF reader may not have been used, trigger a fill if necessary.
	r := &Reader{0, 0, nil, false, b}
	r.setBuf(0) // ignoring the error here is safe because we've successfully set the source
	return r
}

func (r *Reader) setBuf(o int) error {
	var err error
	r.scratch, err = r.Slice(o, readSz)
	if err == io.EOF {
		r.end = true
	}
	return err
}

func (r *Reader) Read(b []byte) (int, error) {
	var slc []byte
	var err error
	if len(b) > len(r.scratch)-r.j {
		slc, err = r.Slice(r.i, len(b))
		if err != nil {
			if err != io.EOF {
				return 0, err
			}
			r.end = true
		}
	} else {
		slc = r.scratch[r.j : r.j+len(b)]
	}
	n := copy(b, slc)
	r.i += n
	r.j += n
	return len(slc), err
}

func (r *Reader) ReadByte() (byte, error) {
	if r.j >= len(r.scratch) {
		if r.end {
			return 0, io.EOF
		}
		err := r.setBuf(r.i)
		if err != nil && err != io.EOF {
			return 0, err
		}
		r.j = 0
	}
	b := r.scratch[r.j]
	r.i++
	r.j++
	return b, nil
}

func (r *Reader) ReadAt(b []byte, off int64) (int, error) {
	var slc []byte
	var err error
	if r.i > int(off) && r.i-r.j <= int(off) && len(b)+int(off) <= r.i-r.j+readSz {
		s := int(off) - (r.i - r.j)
		slc = r.scratch[s : s+len(b)]
	} else {
		slc, err = r.Slice(int(off), len(b))
		if err != nil && err != io.EOF {
			return 0, err
		}
	}
	copy(b, slc)
	return len(slc), err
}

func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	var rev bool
	switch whence {
	case 0:
	case 1:
		offset = offset + int64(r.i)
	case 2:
		rev = true
	default:
		return 0, fmt.Errorf("Siegreader: Seek error, whence value must be one of 0,1,2 got %v", whence)
	}
	success, err := r.canSeek(offset, rev)
	if success {
		if rev {
			offset = r.sz - offset
		}
		r.i = int(offset)
		r.j = r.i % readSz
		return offset, err
	}
	return 0, err
}

// Reverse Reader

type ReverseReader struct {
	i, j    int
	scratch []byte
	end     bool // if buffer is adjacent to the BOF, i.e. we have scanned all the way back to the beginning
	*Buffer
}

func (b *Buffer) NewReverseReader() (*ReverseReader, error) {
	// fill the EOF now, if possible and not already done
	err := b.fillEof()
	return &ReverseReader{0, 0, nil, false, b}, err
}

func (r *ReverseReader) setBuf(o int) error {
	var err error
	r.scratch, err = r.eofSlice(o, readSz)
	if err == io.EOF {
		r.end = true
	}
	return err
}

func (r *ReverseReader) Read(b []byte) (int, error) {
	if r.i == 0 {
		r.setBuf(0)

	}
	var slc []byte
	var err error
	if len(b) > len(r.scratch)-r.j {
		slc, err = r.eofSlice(r.i, len(b))
		if err != nil {
			if err != io.EOF {
				return 0, err
			}
			r.end = true
		}
	} else {
		slc = r.scratch[len(r.scratch)-len(b) : len(r.scratch)-r.j]
	}
	n := copy(b, slc)
	r.i += n
	r.j += n
	return len(slc), err
}

func (r *ReverseReader) ReadByte() (byte, error) {
	var err error
	if r.i == 0 {
		r.setBuf(0)
	} else if r.j >= len(r.scratch) {
		if r.end {
			return 0, io.EOF
		}
		err = r.setBuf(r.i)
		if err != nil && err != io.EOF {
			return 0, err
		}
		r.j = 0
	}
	b := r.scratch[len(r.scratch)-r.j-1]
	r.i++
	r.j++
	return b, err
}
