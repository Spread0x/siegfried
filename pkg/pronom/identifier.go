package pronom

import (
	"sort"
	"sync"

	"github.com/richardlehane/siegfried/pkg/core"
	"github.com/richardlehane/siegfried/pkg/core/bytematcher"
	"github.com/richardlehane/siegfried/pkg/core/namematcher"
	"github.com/richardlehane/siegfried/pkg/core/siegreader"
)

type PronomIdentifier struct {
	Bm         *bytematcher.ByteMatcher
	BPuids     []string
	Em         namematcher.ExtensionMatcher
	EPuids     []string
	Priorities map[string][]int
	ids        pids
}

type PronomIdentification struct {
	puid       string
	confidence float64
}

func (pid PronomIdentification) String() string {
	return pid.puid
}

func (pid PronomIdentification) Confidence() float64 {
	return pid.confidence
}

func (pid PronomIdentification) Basis() string {
	return "because I said so" //obviously this needs to be changed!
}

func (pi *PronomIdentifier) Identify(b *siegreader.Buffer, n string, c chan core.Identification, wg *sync.WaitGroup) {
	pi.ids = pi.ids[:0]
	if len(n) > 0 {
		for _, v := range pi.Em.Identify(n) {
			pi.ids = add(pi.ids, pi.EPuids[v], 0.1)
		}
	}

	var currLimit []int
	var cscore float64 = 0.1

	ids, limit := pi.Bm.Identify(b)

	for i := range ids {
		if !checkLimit(i, currLimit) {
			continue
		}
		cscore *= 1.1
		puid := pi.BPuids[i]
		pi.ids = add(pi.ids, puid, cscore)

		l, ok := pi.Priorities[puid]
		if !ok {
			close(limit)
			break
		} else {
			limit <- l
			currLimit = l
		}
	}

	if len(pi.ids) > 0 {
		sort.Sort(pi.ids)
		conf := pi.ids[0].confidence
		c <- pi.ids[0]
		if len(pi.ids) > 1 {
			for i, v := range pi.ids[1:] {
				if v.confidence == conf {
					c <- pi.ids[i+1]
				} else {
					break
				}
			}
		}
	}
	wg.Done()
}

// Messy duplicating this function here (it is also in bytematcher) - this is to get around a particular race condition
func checkLimit(i int, l []int) bool {
	if l == nil {
		return true
	}
	idx := sort.SearchInts(l, i)
	if idx == len(l) || l[idx] != i {
		return false
	}
	return true
}

type pids []PronomIdentification

func (p pids) Len() int { return len(p) }

func (p pids) Less(i, j int) bool { return p[j].confidence < p[i].confidence }

func (p pids) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func add(p pids, f string, c float64) pids {
	for i, v := range p {
		if v.puid == f {
			p[i].confidence += c
			return p
		}
	}
	return append(p, PronomIdentification{f, c})
}
