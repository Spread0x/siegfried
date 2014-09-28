package containermatcher

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/richardlehane/siegfried/pkg/core"
	"github.com/richardlehane/siegfried/pkg/core/siegreader"
)

func (m Matcher) Identify(n string, b *siegreader.Buffer) chan core.Result {
	// check trigger
	buf, err := b.Slice(0, 8)
	if err != nil {
		return nil
	}
	var res chan core.Result
	for _, c := range m {
		if c.trigger(buf) {
			res = make(chan core.Result)
			if q, i := c.defaultMatch(n); q {
				go func() {
					res <- defaultHit(i)
					close(res)
				}()
				return res
			}
			rdr, err := c.rdr(b)
			if err != nil {
				close(res)
				return res
			}
			go c.identify(rdr, res)
			return res
		}
	}
	return nil
}

func (c *ContainerMatcher) defaultMatch(n string) (bool, int) {
	if c.Default == "" {
		return false, 0
	}
	ext := filepath.Ext(n)
	if len(ext) > 0 && strings.TrimPrefix(ext, ".") == c.Default {
		// the default is always the last container sig
		return true, len(c.Parts) - 1
	}
	return false, 0
}

func (c *ContainerMatcher) identify(rdr Reader, res chan core.Result) {
	// safe to call on a nil matcher
	if c == nil {
		close(res)
		return
	}
	// reset
	if c.started {
		c.waitList = nil
		for i := range c.partsMatched {
			c.partsMatched[i] = c.partsMatched[i][:0]
			c.ruledOut[i] = false
		}
	} else {
		c.entryBuf = siegreader.New()
		c.partsMatched = make([][]hit, len(c.Parts))
		c.ruledOut = make([]bool, len(c.Parts))
		c.hits = make([]hit, 0, 20) // shared hits buffer to avoid allocs
		c.started = true
	}
	for err := rdr.Next(); err == nil; err = rdr.Next() {
		ct, ok := c.NameCTest[rdr.Name()]
		if !ok {
			continue
		}
		// name has matched, lets test the CTests
		if c.processHits(ct.identify(c, rdr, rdr.Name()), ct, rdr.Name(), res) {
			break
		}
	}
	close(res)
}

func (ct *CTest) identify(c *ContainerMatcher, rdr Reader, name string) []hit {
	// reset hits
	c.hits = c.hits[:0]
	for _, h := range ct.Satisfied {
		if c.checkWait(h) {
			c.hits = append(c.hits, hit{h, name, "name only"})
		}
	}
	if ct.Unsatisfied != nil {
		rdr.SetSource(c.entryBuf)
		for r := range ct.BM.Identify("", c.entryBuf) {
			h := ct.Unsatisfied[r.Index()]
			if c.checkWait(h) && c.checkHits(h) {
				c.hits = append(c.hits, hit{h, name, r.Basis()})
			}
		}
		rdr.Close()
	}
	return c.hits
}

// process the hits from the ctest: adding hits to the parts matched, checking priorities
// return true if satisfied and can quit
func (c *ContainerMatcher) processHits(hits []hit, ct *CTest, name string, res chan core.Result) bool {
	// if there are no hits, rule out any sigs in the ctest
	if len(hits) == 0 {
		for _, v := range ct.Satisfied {
			c.ruledOut[v] = true
		}
		for _, v := range ct.Unsatisfied {
			c.ruledOut[v] = true
		}
		return false
	}

	for _, h := range hits {
		c.partsMatched[h.id] = append(c.partsMatched[h.id], h)
		if len(c.partsMatched[h.id]) == c.Parts[h.id] {
			if c.checkWait(h.id) {
				res <- result(c.partsMatched[h.id]) // send a Result here
				if c.Priorities != nil {
					if len(c.Priorities[h.id]) == 0 {
						return true
					}
					c.waitList = c.Priorities[h.id]
				}
			}
		}
	}
	// if nothing ruled out but this test, then we must continue
	if len(hits) == len(ct.Satisfied)+len(ct.Unsatisfied) {
		return false
	}
	// we can rule some possible matches out...
	for _, v := range ct.Satisfied {
		if len(c.partsMatched[v]) == 0 || c.partsMatched[v][len(c.partsMatched[v])-1].name != name {
			c.ruledOut[v] = true
		}
	}
	for _, v := range ct.Unsatisfied {
		if len(c.partsMatched[v]) == 0 || c.partsMatched[v][len(c.partsMatched[v])-1].name != name {
			c.ruledOut[v] = true
		}
	}
	// loop over the wait list, seeing if they are all ruled out
	var satisfied = true
	// assume we're satisfied & look for a living priority
	for _, v := range c.waitList {
		if !c.ruledOut[v] {
			satisfied = false
			break
		}
	}
	return satisfied
}

// eliminate duplicate hits - must do this since rely on number of matches for each sig as test for full match
func (c *ContainerMatcher) checkHits(i int) bool {
	for _, h := range c.hits {
		if i == h.id {
			return false
		}
	}
	return true
}

// is this match one we are waiting for?
func (c *ContainerMatcher) checkWait(i int) bool {
	// if there are no priorities set, we let it through
	if c.waitList == nil {
		return true
	}
	idx := sort.SearchInts(c.waitList, i)
	// if we have a wait list, and we aren't waiting for this match, we should filter it out
	if idx == len(c.waitList) || c.waitList[idx] != i {
		return false
	}
	// yes, this match is on the list
	return true
}

type result []hit

func (r result) Index() int {
	if len(r) == 0 {
		return -1
	}
	return r[0].id
}

func (r result) Basis() string {
	var basis string
	for i, v := range r {
		if i > 0 {
			basis += "; "
		}
		basis += "container name " + v.name
		if len(v.basis) > 0 {
			basis += " with " + v.basis
		}
	}
	return basis
}

type hit struct {
	id    int
	name  string
	basis string
}

type defaultHit int

func (d defaultHit) Index() int {
	return int(d)
}

func (d defaultHit) Basis() string {
	return "container match with trigger and default extension"
}
