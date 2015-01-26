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

// Package siegreader implements multiple independent Readers (and ReverseReaders) from a single Buffer.
//
// Example:
//   buffers := siegreader.Buffers()
//   buffer, err := buffers.Get(io.Reader)
//   if err != nil {
//     log.Fatal(err)
//   }
//   rdr := siegreader.ReaderFrom(buffer)
//	 second_rdr := siegreader.ReaderFrom(buffer)
//   brdr := siegreader.LimitReaderFrom(buffer, -1)
//   rrdr, err := siegreader.LimitReverseReaderFrom(buffer, 16000)
//   i, err := rdr.Read(slc)
//   i2, err := second_rdr.Read(slc2)
//   i3, err := rrdr.ReadByte()
package siegreader

import "errors"

var (
	ErrQuit      = errors.New("siegreader: quit chan closed while awaiting EOF")
	ErrNilBuffer = errors.New("siegreader: attempt to SetSource on a nil buffer")
)

const (
	readSz      int = 4096
	initialRead     = readSz * 2
	wheelSz         = readSz * 2
	eofSz           = readSz * 2
	smallFileSz     = readSz * 16
)

type Buffer interface {
	Slice(off int64, l int) ([]byte, error)
	EofSlice(off int64, l int) ([]byte, error)
	SetQuit(chan struct{})
	hasQuit() bool
	Size() int64
	SizeNow() int64
	canSeek(off int64, rev bool) (bool, error)
	setLimit()
	waitLimit()
	reachedLimit()
}