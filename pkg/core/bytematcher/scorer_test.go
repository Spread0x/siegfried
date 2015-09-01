package bytematcher

import (
	"bytes"
	"testing"

	"github.com/richardlehane/siegfried/pkg/core"
	"github.com/richardlehane/siegfried/pkg/core/bytematcher/frames/tests"
	"github.com/richardlehane/siegfried/pkg/core/siegreader"
)

func setup() (chan<- strike, <-chan core.Result) {
	bm := New()
	bm.Add(SignatureSet(tests.TestSignatures), nil)
	bufs := siegreader.New()
	buf, _ := bufs.Get(bytes.NewBuffer(TestSample1))
	res := make(chan core.Result)
	return bm.newScorer(buf, make(chan struct{}), res), res
}

func TestScorer(t *testing.T) {
	scorer, res := setup()
	scorer <- strike{0, 0, 0, 4, false, false, true}
	scorer <- strike{1, 0, 17, 9, true, false, false}
	scorer <- strike{1, 1, 30, 5, true, false, true}
	if r := <-res; r.Index() != 0 {
		t.Errorf("expecting result %d, got %d", 0, r.Index())
	}
}

// 1 Sept 15 BenchmarkScorer	   10000	   3260800 ns/op

func BenchmarkScorer(bench *testing.B) {
	scorer, res := setup()
	for i := 0; i < bench.N; i++ {
		scorer <- strike{0, 0, 0, 4, false, false, true}
		scorer <- strike{1, 0, 17, 9, true, false, false}
		scorer <- strike{1, 1, 30, 5, true, false, true}
		_ = <-res
	}
}
