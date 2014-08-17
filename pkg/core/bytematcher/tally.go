package bytematcher

import (
	"sort"
	"sync"

	"github.com/richardlehane/siegfried/pkg/core/bytematcher/process"
)

type tally struct {
	*matcher
	results chan int
	quit    chan struct{}
	wait    chan []int

	once     *sync.Once
	bofQueue *sync.WaitGroup
	eofQueue *sync.WaitGroup
	stop     chan struct{}

	bofOff int
	eofOff int

	waitList []int
	waitM    *sync.RWMutex

	kfHits chan kfHit
	halt   chan bool
}

func newTally(r chan int, q chan struct{}, w chan []int, m *matcher) *tally {
	t := &tally{
		matcher:  m,
		results:  r,
		quit:     q,
		wait:     w,
		once:     &sync.Once{},
		bofQueue: &sync.WaitGroup{},
		eofQueue: &sync.WaitGroup{},
		stop:     make(chan struct{}),
		waitList: nil,
		waitM:    &sync.RWMutex{},
		kfHits:   make(chan kfHit),
		halt:     make(chan bool),
	}
	go t.filterHits()
	return t
}

func (t *tally) shutdown(eof bool) {
	go t.once.Do(func() { t.finalise(eof) })
}

func (t *tally) finalise(eof bool) {
	if eof {
		t.bofQueue.Wait()
		t.eofQueue.Wait()
	}
	close(t.quit)
	t.drain()
	if !eof {
		t.bofQueue.Wait()
		t.eofQueue.Wait()
	}
	close(t.results)
	close(t.stop)
}

func (t *tally) drain() {
	for {
		select {
		case _, ok := <-t.incoming:
			if !ok {
				t.incoming = nil
			}
		case _ = <-t.bofProgress:
		case _ = <-t.eofProgress:
		}
		if t.incoming == nil {
			return
		}
	}
}

type kfHit struct {
	id     process.KeyFrameID
	offset int
	length int
}

func (t *tally) filterHits() {
	var satisfied bool
	for {
		select {
		case _ = <-t.stop:
			return
		case hit := <-t.kfHits:
			if satisfied {
				t.halt <- true
				continue
			}
			// in case of a race
			if !t.checkWait(hit.id[0]) {
				t.halt <- false
				continue
			}
			success := t.applyKeyFrame(hit.id, hit.offset, hit.length)
			if success {
				if h := t.sendResult(hit.id[0]); h {
					t.halt <- true
					satisfied = true
					t.shutdown(false)
					continue
				}
			}
			t.halt <- false
		}
	}
}

func (t *tally) sendResult(res int) bool {
	t.results <- res
	w := <-t.wait // every result sent must result in a new priority list being returned & we need to drain this or it will block
	// nothing more to wait for
	if len(w) == 0 {
		return true
	}
	t.setWait(w)
	return false
}

func (t *tally) setWait(w []int) {
	t.waitM.Lock()
	t.waitList = w
	t.waitM.Unlock()
}

// check a signature ID against the priority list
func (t *tally) checkWait(i int) bool {
	t.waitM.RLock()
	defer t.waitM.RUnlock()
	if t.waitList == nil {
		return true
	}
	idx := sort.SearchInts(t.waitList, i)
	if idx == len(t.waitList) || t.waitList[idx] != i {
		return false
	}
	return true
}
