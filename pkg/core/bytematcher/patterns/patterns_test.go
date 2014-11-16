package patterns_test

import (
	"testing"

	. "github.com/richardlehane/siegfried/pkg/core/bytematcher/patterns/tests"
)

func TestSequence(t *testing.T) {
	if !TestSequences[0].Equals(TestSequences[1]) {
		t.Error("Seq fail: Equality")
	}
	if r, _ := TestSequences[0].Test([]byte{'t', 'o', 'o', 't'}); r {
		t.Error("Sequence fail: shouldn't match")
	}
	if _, l := TestSequences[2].Test([]byte{'t', 'e', 's', 't', 'y'}); l != 5 {
		t.Error("Sequence fail: should match")
	}
	reverseSeq := TestSequences[2].Reverse()
	if reverseSeq[1] != 't' || reverseSeq[2] != 's' || reverseSeq[3] != 'e' || reverseSeq[4] != 't' {
		t.Error("Sequence fail: Reverse")
	}
}

func TestChoice(t *testing.T) {
	if !TestChoices[0].Equals(TestChoices[1]) {
		t.Error("Choice fail: Equality")
	}
	if _, l := TestChoices[0].Test([]byte{'t', 'e', 's', 't'}); l != 4 {
		t.Error("Choice test fail: Test")
	}
	if TestChoices[0].NumSequences() != 2 {
		t.Error("Choice fail: NumSequences; expecting 2 got", TestChoices[0].NumSequences())
	}
	seqs := TestChoices[0].Sequences()
	if seqs[0][0] != 't' || seqs[1][0] != 't' {
		t.Error("Choice fail: Sequences; expecting t, t got ", seqs[0][0], seqs[1][0])
	}
}

func TestList(t *testing.T) {
	if TestLists[0].Equals(TestLists[1]) {
		t.Error("List fail: equality")
	}
	if _, l := TestLists[0].Test([]byte{'t', 'e', 's', 't', 't', 'e', 's', 't', 'y'}); l != 9 {
		t.Error("List test fail: Test")
	}
	if TestLists[0].NumSequences() != 1 {
		t.Error("List fail: NumSequences; expecting 1 got", TestLists[0].NumSequences())
	}
	seqs := TestLists[0].Sequences()
	if seqs[0][0] != 't' || seqs[0][8] != 'y' {
		t.Error("List fail: Sequences; expecting t, y got ", seqs[0][0], seqs[0][8])
	}
}
