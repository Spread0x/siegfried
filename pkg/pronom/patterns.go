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

// Define custom patterns (implementing the siegfried.Pattern interface) for the different patterns allowed by the PRONOM spec.
package pronom

import (
	"bytes"
	"fmt"

	"github.com/richardlehane/siegfried/pkg/core/bytematcher/patterns"
	"github.com/richardlehane/siegfried/pkg/core/persist"
)

func init() {
	patterns.Register(rangeLoader, loadRange)
	patterns.Register(maskLoader, loadMask)
	patterns.Register(anyMaskLoader, loadAnyMask)
}

const (
	rangeLoader byte = iota + 8
	maskLoader
	anyMaskLoader
)

type Range struct {
	From, To []byte
}

func (r Range) Test(b []byte) (bool, int) {
	if len(b) < len(r.From) || len(b) < len(r.To) {
		return false, 0
	}
	if bytes.Compare(r.From, b[:len(r.From)]) < 1 {
		if bytes.Compare(r.To, b[:len(r.To)]) > -1 {
			return true, len(r.From)
		}
	}
	return false, 1
}

func (r Range) TestR(b []byte) (bool, int) {
	if len(b) < len(r.From) || len(b) < len(r.To) {
		return false, 0
	}
	if bytes.Compare(r.From, b[len(b)-len(r.From):]) < 1 {
		if bytes.Compare(r.To, b[len(b)-len(r.To):]) > -1 {
			return true, len(r.From)
		}
	}
	return false, 1
}

func (r Range) Equals(pat patterns.Pattern) bool {
	rng, ok := pat.(Range)
	if ok {
		if bytes.Equal(rng.From, r.From) {
			if bytes.Equal(rng.To, r.To) {
				return true
			}
		}
	}
	return false
}

func (r Range) Length() (int, int) {
	return len(r.From), len(r.From)
}

func (r Range) NumSequences() int {
	l := len(r.From)
	if l > 2 || l < 1 {
		return 0
	}
	if l == 2 {
		if r.To[0]-r.From[0] > 1 {
			return 0
		}
		return 256*int(r.To[0]-r.From[0]) + int(r.To[1]) - int(r.From[1]) + 1
	}
	return int(r.To[0]-r.From[0]) + 1
}

func (r Range) Sequences() []patterns.Sequence {
	num := r.NumSequences()
	seqs := make([]patterns.Sequence, num)
	if num < 1 {
		return seqs
	}
	if len(r.From) == 2 {
		if r.From[0] == r.To[0] {
			for i := 0; i < num; i++ {
				seqs[i] = patterns.Sequence{r.From[0], r.From[1] + byte(i)}
			}
			return seqs
		}
		max := 256 - int(r.From[1])
		for i := 0; i < max; i++ {
			seqs[i] = patterns.Sequence{r.From[0], r.From[1] + byte(i)}
		}
		for i := 0; max < num; max++ {
			seqs[max] = patterns.Sequence{r.To[0], byte(0 + i)}
			i++
		}
		return seqs
	}
	for i := 0; i < num; i++ {
		seqs[i] = patterns.Sequence{r.From[0] + byte(i)}
	}
	return seqs
}

func (r Range) String() string {
	return "r " + patterns.Stringify(r.From) + " - " + patterns.Stringify(r.To)
}

func (r Range) Save(ls *persist.LoadSaver) {
	ls.SaveByte(rangeLoader)
	ls.SaveBytes(r.From)
	ls.SaveBytes(r.To)
}

func loadRange(ls *persist.LoadSaver) patterns.Pattern {
	return Range{
		ls.LoadBytes(),
		ls.LoadBytes(),
	}
}

type Mask byte

func (m Mask) Test(b []byte) (bool, int) {
	if len(b) == 0 {
		return false, 0
	}
	if byte(m)&b[0] == byte(m) {
		return true, 1
	}
	return false, 1
}

func (m Mask) TestR(b []byte) (bool, int) {
	if len(b) == 0 {
		return false, 0
	}
	if byte(m)&b[len(b)-1] == byte(m) {
		return true, 1
	}
	return false, 1
}

func (m Mask) Equals(pat patterns.Pattern) bool {
	msk, ok := pat.(Mask)
	if ok {
		if m == msk {
			return true
		}
	}
	return false
}

func (m Mask) Length() (int, int) {
	return 1, 1
}

func countBits(b byte) int {
	var count uint
	for b > 0 {
		b &= b - 1
		count++
	}
	return 256 / (1 << count)
}

func allBytes() []byte {
	all := make([]byte, 256)
	for i := range all {
		all[i] = byte(i)
	}
	return all
}

func (m Mask) NumSequences() int {
	return countBits(byte(m))
}

func (m Mask) Sequences() []patterns.Sequence {
	seqs := make([]patterns.Sequence, 0, m.NumSequences())
	for _, b := range allBytes() {
		if byte(m)&b == byte(m) {
			seqs = append(seqs, patterns.Sequence{b})
		}
	}
	return seqs
}

func (m Mask) String() string {
	return fmt.Sprintf("m %#x", byte(m))
}

func (m Mask) Save(ls *persist.LoadSaver) {
	ls.SaveByte(maskLoader)
	ls.SaveByte(byte(m))
}

func loadMask(ls *persist.LoadSaver) patterns.Pattern {
	return Mask(ls.LoadByte())
}

type AnyMask byte

func (am AnyMask) Test(b []byte) (bool, int) {
	if len(b) == 0 {
		return false, 0
	}
	if byte(am)&b[0] != 0 {
		return true, 1
	}
	return false, 1
}

func (am AnyMask) TestR(b []byte) (bool, int) {
	if len(b) == 0 {
		return false, 0
	}
	if byte(am)&b[len(b)-1] != 0 {
		return true, 1
	}
	return false, 1
}

func (am AnyMask) Equals(pat patterns.Pattern) bool {
	amsk, ok := pat.(AnyMask)
	if ok {
		if am == amsk {
			return true
		}
	}
	return false
}

func (am AnyMask) Length() (int, int) {
	return 1, 1
}

func (am AnyMask) NumSequences() int {
	return 256 - countBits(byte(am))
}

func (am AnyMask) Sequences() []patterns.Sequence {
	seqs := make([]patterns.Sequence, 0, am.NumSequences())
	for _, b := range allBytes() {
		if byte(am)&b != 0 {
			seqs = append(seqs, patterns.Sequence{b})
		}
	}
	return seqs
}

func (am AnyMask) String() string {
	return fmt.Sprintf("am %#x", byte(am))
}

func (am AnyMask) Save(ls *persist.LoadSaver) {
	ls.SaveByte(anyMaskLoader)
	ls.SaveByte(byte(am))
}

func loadAnyMask(ls *persist.LoadSaver) patterns.Pattern {
	return AnyMask(ls.LoadByte())
}
