// Package Bytematcher builds a matching engine from a set of signatures and performs concurrent matching against an input siegreader.Buffer.
package bytematcher

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/richardlehane/match/wac"
	"github.com/richardlehane/siegfried/pkg/core/bytematcher/frames"
	"github.com/richardlehane/siegfried/pkg/core/bytematcher/process"
	"github.com/richardlehane/siegfried/pkg/core/priority"
	"github.com/richardlehane/siegfried/pkg/core/siegreader"
)

type Matcher interface {
	Start()
	SetPriorities(priority.List)
	Identify(*siegreader.Buffer) chan Result
	String() string
	Save(io.Writer) (int, error)
}

// Bytematcher structure. Clients shouldn't need to get or set these fields directly, they are only exported so that this structure can be serialised and deserialised by encoding/gob.
type ByteMatcher struct {
	*process.Process
	Priorities priority.List
	bAho       *wac.Wac
	eAho       *wac.Wac
	started    bool
}

func New() *ByteMatcher {
	return &ByteMatcher{
		process.New(),
		nil,
		&wac.Wac{},
		&wac.Wac{},
		false,
	}
}

func Load(r io.Reader) (Matcher, error) {
	b := New()
	dec := gob.NewDecoder(r)
	err := dec.Decode(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (b *ByteMatcher) Save(w io.Writer) (int, error) {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	err := enc.Encode(b)
	if err != nil {
		return 0, err
	}
	sz := buf.Len()
	_, err = buf.WriteTo(w)
	if err != nil {
		return 0, err
	}
	return sz, nil
}

type sigErrors []error

func (se sigErrors) Error() string {
	str := "bytematcher.Signatures errors:"
	for _, v := range se {
		str += v.Error()
		str += "\n"
	}
	return str
}

// Create a new Bytematcher from a slice of signatures.
// Can give optional distance, range, choice, variable sequence length values to override the defaults of 8192, 2048, 64.
//   - the distance and range values dictate how signatures are segmented during processing
//   - the choices value controls how signature segments are converted into simple byte sequences during processing
//   - the varlen value controls what is the minimum length sequence acceptable for the variable Aho Corasick tree. The longer this length, the fewer false matches you will get during searching.
//
// Example:
//   bm, err := Signatures([]Signature{Signature{NewFrame(BOF, Sequence{'p','d','f'}, 0, 0)}})
func Signatures(sigs []frames.Signature, opts ...int) (*ByteMatcher, error) {
	b := New()
	b.SetOptions(opts...)
	se := make(sigErrors, 0)
	// process each of the sigs, adding them to b.Sigs and the various seq/frame/testTree sets
	for _, sig := range sigs {
		err := b.AddSignature(sig)
		if err != nil {
			se = append(se, err)
		}
	}
	if len(se) > 0 {
		return b, se
	}
	// set the maximum distances for this test tree so can properly size slices for matching
	for _, t := range b.Tests {
		t.MaxLeftDistance = process.MaxLength(t.Left)
		t.MaxRightDistance = process.MaxLength(t.Right)
	}
	return b, nil
}

func (b *ByteMatcher) SetPriorities(l priority.List) {
	b.Priorities = l
}

// Start initialises the aho corasick search trees after a Bytematcher has been loaded.
func (b *ByteMatcher) Start() {
	if b.started {
		return
	}
	b.bAho = wac.New(b.BOFSeq.Set)
	b.eAho = wac.New(b.EOFSeq.Set)
	b.started = true
}

type Result struct {
	Index int
	Basis string
}

// Identify matches a Bytematcher's signatures against the input siegreader.Buffer.
// Results are passed on the first returned int channel. These ints are the indexes of the matching signatures.
// The second and third int channels report on the Bytematcher's progress: returning offets from the beginning of the file and the end of the file.
//
// Example:
//   ret, bprog, eprog := bm.Identify(buf, q)
//   for v := range ret {
//     if v == 0 {
//       fmt.Print("Success! It is signature 0!")
//     }
//   }
func (b *ByteMatcher) Identify(sb *siegreader.Buffer) chan Result {
	quit, ret := make(chan struct{}), make(chan Result)
	go b.identify(sb, quit, ret)
	return ret
}

// Returns information about the Bytematcher including the number of BOF, VAR and EOF sequences, the number of BOF and EOF frames, and the total number of tests.
func (b *ByteMatcher) String() string {
	str := fmt.Sprintf("BOF seqs: %v\n", len(b.BOFSeq.Set))
	str += fmt.Sprintf("EOF seqs: %v\n", len(b.EOFSeq.Set))
	str += fmt.Sprintf("BOF frames: %v\n", len(b.BOFFrames.Set))
	str += fmt.Sprintf("EOF frames: %v\n", len(b.EOFFrames.Set))
	str += fmt.Sprintf("Total Tests: %v\n", len(b.Tests))
	var c, ic, l, r, ml, mr int
	for _, t := range b.Tests {
		c += len(t.Complete)
		ic += len(t.Incomplete)
		l += len(t.Left)
		if ml < t.MaxLeftDistance {
			ml = t.MaxLeftDistance
		}
		r += len(t.Right)
		if mr < t.MaxRightDistance {
			mr = t.MaxRightDistance
		}
	}
	str += fmt.Sprintf("Complete Tests: %v\n", c)
	str += fmt.Sprintf("Incomplete Tests: %v\n", ic)
	str += fmt.Sprintf("Left Tests: %v\n", l)
	str += fmt.Sprintf("Right Tests: %v\n", r)
	str += fmt.Sprintf("Maximum Left Distance: %v\n", ml)
	str += fmt.Sprintf("Maximum Right Distance: %v\n", mr)
	str += fmt.Sprintf("Maximum BOF Distance: %v\n", b.MaxBOF)
	str += fmt.Sprintf("Maximum EOF Distance: %v\n", b.MaxEOF)
	str += fmt.Sprintf("Priorities: %v\n", b.Priorities)
	return str
}
