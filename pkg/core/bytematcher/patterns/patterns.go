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

// Package patterns describes the Pattern interface.
// Two standard patterns are also defined in this package, Sequence and Choice.
package patterns

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strconv"
	"unicode/utf8"

	"github.com/richardlehane/siegfried/pkg/core/persist"
)

func init() {
	Register(sequenceLoader, loadSequence)
	Register(choiceLoader, loadChoice)
	Register(listLoader, loadList)
	Register(notLoader, loadNot)
	Register(bmhLoader, loadBMH)
	Register(rbmhLoader, loadRBMH)
}

func Stringify(b []byte) string {
	if utf8.Valid(b) {
		return strconv.QuoteToASCII(string(b))
	}
	return hex.EncodeToString(b)
}

// Patterns are the smallest building blocks of a format signature.
// Exact byte sequence matches are a type of pattern, as are byte ranges, non-sequence matches etc.
// You can define custom patterns (e.g. for W3C date type) by implementing this interface.
type Pattern interface {
	Test([]byte) (bool, int)  // Returns boolean for match. For a positive match, the integer value represents the length of the match. For a negative match, the integer represents an offset jump before a subsequent test. That offset should be 0 if the remaining byte slice is smaller than the pattern.
	TestR([]byte) (bool, int) // Same as Test but for testing in reverse (from the right-most position of the byte slice).
	Equals(Pattern) bool      // Test equality with another pattern
	Length() (int, int)       // Minimum and maximum lengths of the pattern
	NumSequences() int        // Number of simple sequences represented by a pattern. Return 0 if the pattern cannot be represented by a defined number of simple sequence (e.g. for an indirect offset pattern) or, if in your opinion, the number of sequences is unreasonably large.
	Sequences() []Sequence    // Convert the pattern to a slice of sequences. Return an empty slice if the pattern cannot be represented by a defined number of simple sequences.
	String() string
	Save(*persist.LoadSaver) // encode the pattern into bytes for saving in a persist file
}

type Loader func(*persist.LoadSaver) Pattern

const (
	sequenceLoader byte = iota
	choiceLoader
	listLoader
	notLoader
	bmhLoader
	rbmhLoader
)

var loaders = [32]Loader{}

func Register(id byte, l Loader) {
	loaders[int(id)] = l
}

func Load(ls *persist.LoadSaver) Pattern {
	id := ls.LoadByte()
	l := loaders[int(id)]
	if l == nil {
		if ls.Err == nil {
			ls.Err = errors.New("bad pattern loader")
		}
		return nil
	}
	return l(ls)
}

// Sequence is a matching sequence of bytes.
type Sequence []byte

func (s Sequence) Test(b []byte) (bool, int) {
	if len(b) < len(s) {
		return false, 0
	}
	if bytes.Equal(s, b[:len(s)]) {
		return true, len(s)
	}
	return false, 1
}

func (s Sequence) TestR(b []byte) (bool, int) {
	if len(b) < len(s) {
		return false, 0
	}
	if bytes.Equal(s, b[len(b)-len(s):]) {
		return true, len(s)
	}
	return false, 1
}

func (s Sequence) Equals(pat Pattern) bool {
	seq2, ok := pat.(Sequence)
	if ok {
		return bytes.Equal(s, seq2)
	}
	return false
}

func (s Sequence) Length() (int, int) {
	return len(s), len(s)
}

func (s Sequence) NumSequences() int {
	return 1
}

func (s Sequence) Sequences() []Sequence {
	return []Sequence{s}
}

func (s Sequence) String() string {
	return "seq " + Stringify(s)
}

// The Reverse method is unique to this pattern. It is used for the EOF byte sequence set
func (s Sequence) Reverse() Sequence {
	p := make(Sequence, len(s))
	for i, j := 0, len(s)-1; j > -1; i, j = i+1, j-1 {
		p[i] = s[j]
	}
	return p
}

func (s Sequence) Save(ls *persist.LoadSaver) {
	ls.SaveByte(sequenceLoader)
	ls.SaveBytes(s)
}

func loadSequence(ls *persist.LoadSaver) Pattern {
	return Sequence(ls.LoadBytes())
}

// Choice is a slice of patterns, any of which can test true for the pattern to succeed. Returns the longest matching pattern
type Choice []Pattern

func (c Choice) test(b []byte, f func(Pattern, []byte) (bool, int)) (bool, int) {
	var r, res bool
	var tl, fl, lgth int
	for _, pat := range c {
		res, lgth = f(pat, b)
		if res {
			r = true
			if lgth > tl {
				tl = lgth
			}
		} else if lgth > fl {
			fl = lgth
		}
	}
	if r {
		return r, tl
	}
	return r, fl
}

func (c Choice) Test(b []byte) (bool, int) {
	return c.test(b, Pattern.Test)
}

func (c Choice) TestR(b []byte) (bool, int) {
	return c.test(b, Pattern.TestR)
}

func (c Choice) Equals(pat Pattern) bool {
	c2, ok := pat.(Choice)
	if ok {
		if len(c) == len(c2) {
			for _, p := range c {
				ident := false
				for _, p2 := range c2 {
					if p.Equals(p2) {
						ident = true
					}
				}
				if !ident {
					return false
				}
			}
			return true
		}
	}
	return false
}

func (c Choice) Length() (int, int) {
	var min, max int
	if len(c) > 0 {
		min, max = c[0].Length()
	}
	for _, pat := range c {
		min2, max2 := pat.Length()
		if min2 < min {
			min = min2
		}
		if max2 > max {
			max = max2
		}
	}
	return min, max
}

func (c Choice) NumSequences() int {
	var s int
	for _, pat := range c {
		num := pat.NumSequences()
		if num == 0 { // if any of the patterns can't be converted to sequences, don't return any
			return 0
		}
		s += num
	}
	return s
}

func (c Choice) Sequences() []Sequence {
	num := c.NumSequences()
	seqs := make([]Sequence, 0, num)
	for _, pat := range c {
		seqs = append(seqs, pat.Sequences()...)
	}
	return seqs
}

func (c Choice) String() string {
	s := "c["
	for i, pat := range c {
		s += pat.String()
		if i < len(c)-1 {
			s += ","
		}
	}
	return s + "]"
}

func (c Choice) Save(ls *persist.LoadSaver) {
	ls.SaveByte(choiceLoader)
	ls.SaveSmallInt(len(c))
	for _, pat := range c {
		pat.Save(ls)
	}
}

func loadChoice(ls *persist.LoadSaver) Pattern {
	l := ls.LoadSmallInt()
	choices := make(Choice, l)
	for i := range choices {
		choices[i] = Load(ls)
	}
	return choices
}

// List is a slice of patterns, all of which must test true sequentially in order for the pattern to succeed.
type List []Pattern

func (l List) Test(b []byte) (bool, int) {
	if len(l) < 1 {
		return false, 0
	}
	success, first := l[0].Test(b)
	if !success {
		return false, first
	}
	total := first
	if len(l) > 1 {
		for _, p := range l[1:] {
			if len(b) <= total {
				return false, 0
			}
			success, le := p.Test(b[total:])
			if !success {
				return false, first
			}
			total += le
		}
	}
	return true, total
}

func (l List) TestR(b []byte) (bool, int) {
	if len(l) < 1 {
		return false, 0
	}
	success, first := l[len(l)-1].TestR(b)
	if !success {
		return false, first
	}
	total := first
	if len(l) > 1 {
		for i := len(l) - 2; i >= 0; i-- {
			if len(b) <= total {
				return false, 0
			}
			success, le := l[i].TestR(b[:len(b)-total])
			if !success {
				return false, first
			}
			total += le
		}
	}
	return true, total
}

func (l List) Equals(pat Pattern) bool {
	l2, ok := pat.(List)
	if ok {
		if len(l) == len(l2) {
			for i, p := range l {
				if !p.Equals(l2[i]) {
					return false
				}
			}
		}
	}
	return true
}

func (l List) Length() (int, int) {
	var min, max int
	for _, pat := range l {
		pmin, pmax := pat.Length()
		min += pmin
		max += pmax
	}
	return min, max
}

func (l List) NumSequences() int {
	s := 1
	for _, pat := range l {
		num := pat.NumSequences()
		if num == 0 { // if any of the patterns can't be converted to sequences, don't return any
			return 0
		}
		s *= num
	}
	return s
}

func (l List) Sequences() []Sequence {
	total := l.NumSequences()
	seqs := make([]Sequence, total)
	for _, pat := range l {
		num := pat.NumSequences()
		times := total / num
		idx := 0
		for _, seq := range pat.Sequences() {
			for i := 0; i < times; i++ {
				seqs[idx] = append(seqs[idx], seq...)
				idx++
			}
		}
	}
	return seqs
}

func (l List) String() string {
	s := "l["
	for i, pat := range l {
		s += pat.String()
		if i < len(l)-1 {
			s += ","
		}
	}
	return s + "]"
}

func (l List) Save(ls *persist.LoadSaver) {
	ls.SaveByte(listLoader)
	ls.SaveSmallInt(len(l))
	for _, pat := range l {
		pat.Save(ls)
	}
}

func loadList(ls *persist.LoadSaver) Pattern {
	le := ls.LoadSmallInt()
	list := make(List, le)
	for i := range list {
		list[i] = Load(ls)
	}
	return list
}

type Not struct{ Pattern }

func (n Not) Test(b []byte) (bool, int) {
	min, _ := n.Pattern.Length()
	if len(b) < min {
		return false, 0
	}
	ok, _ := n.Pattern.Test(b)
	if !ok {
		return true, min
	}
	return false, 1
}

func (n Not) TestR(b []byte) (bool, int) {
	min, _ := n.Pattern.Length()
	if len(b) < min {
		return false, 0
	}
	ok, _ := n.Pattern.TestR(b)
	if !ok {
		return true, min
	}
	return false, 1
}

func (n Not) Equals(pat Pattern) bool {
	n2, ok := pat.(Not)
	if ok {
		return n.Pattern.Equals(n2.Pattern)
	}
	return false
}

func (n Not) Length() (int, int) {
	min, _ := n.Pattern.Length()
	return min, min
}

func (n Not) NumSequences() int {
	_, max := n.Pattern.Length()
	if max > 1 {
		return 0
	}
	num := n.Pattern.NumSequences()
	if num == 0 {
		return 0
	}
	return 256 - num
}

func (n Not) Sequences() []Sequence {
	num := n.NumSequences()
	if num < 1 {
		return nil
	}
	seqs := make([]Sequence, 0, num)
	pseqs := n.Pattern.Sequences()
	allBytes := make([]Sequence, 256)
	for i := 0; i < 256; i++ {
		allBytes[i] = Sequence{byte(i)}
	}
	for _, v := range allBytes {
		eq := false
		for _, w := range pseqs {
			if v.Equals(w) {
				eq = true
				break
			}
		}
		if eq {
			continue
		}
		seqs = append(seqs, v)
	}
	return seqs
}

func (n Not) String() string {
	return "not[" + n.Pattern.String() + "]"
}

func (n Not) Save(ls *persist.LoadSaver) {
	ls.SaveByte(notLoader)
	n.Pattern.Save(ls)
}

func loadNot(ls *persist.LoadSaver) Pattern {
	return Not{Load(ls)}
}
