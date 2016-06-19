// Copyright 2016 Richard Lehane. All rights reserved.
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

package loc

import (
	"fmt"
	"sort"
	"strings"

	"github.com/richardlehane/siegfried/config"
	"github.com/richardlehane/siegfried/pkg/core"
	"github.com/richardlehane/siegfried/pkg/core/identifier"
	"github.com/richardlehane/siegfried/pkg/core/persist"
)

func init() {
	core.RegisterIdentifier(core.LOC, Load)
}

type Identifier struct {
	infos map[string]formatInfo
	*identifier.Base
}

func (i *Identifier) Save(ls *persist.LoadSaver) {
	ls.SaveByte(core.LOC)
	ls.SaveSmallInt(len(i.infos))
	for k, v := range i.infos {
		ls.SaveString(k)
		ls.SaveString(v.name)
		ls.SaveString(v.longName)
		ls.SaveString(v.mimeType)
	}
	i.Base.Save(ls)
}

func Load(ls *persist.LoadSaver) core.Identifier {
	i := &Identifier{}
	i.infos = make(map[string]formatInfo)
	le := ls.LoadSmallInt()
	for j := 0; j < le; j++ {
		i.infos[ls.LoadString()] = formatInfo{
			ls.LoadString(),
			ls.LoadString(),
			ls.LoadString(),
		}
	}
	i.Base = identifier.Load(ls)
	return i
}

func New(opts ...config.Option) (core.Identifier, error) {
	for _, v := range opts {
		v()
	}
	loc, err := newLOC(config.LOC())
	if err != nil {
		return nil, err
	}
	// if we are inspecting...
	if config.Inspect() {
		loc = identifier.Filter(config.Limit(loc.IDs()), loc)
		is := infos(loc.Infos())
		sigs, ids, err := loc.Signatures()
		if err != nil {
			return nil, fmt.Errorf("LOC: parsing signatures; got %s", err)
		}
		var id string
		for i, sig := range sigs {
			if ids[i] != id {
				id = ids[i]
				fmt.Printf("%s: \n", is[id].name)
			}
			fmt.Println(sig)
		}
		return nil, nil
	}
	// set updated
	updated := loc.(fdds).Updated().Format(dateFmt)
	// add extensions
	for _, v := range config.Extend() {
		e, err := newLOC(v)
		if err != nil {
			return nil, fmt.Errorf("LOC: error loading extension file %s; got %s", v, err)
		}
		loc = identifier.Join(loc, e)
	}
	// apply config
	loc = identifier.ApplyConfig(loc)
	// return identifier
	return &Identifier{
		infos: infos(loc.Infos()),
		Base:  identifier.New(loc, config.ZipLOC(), updated),
	}, nil
}

func (i *Identifier) Fields() []string {
	return []string{"namespace", "id", "format", "full", "mime", "basis", "warning"}
}

func (i *Identifier) Recorder() core.Recorder {
	return &Recorder{i, make(pids, 0, 10), 0, false, false, false, false}
}

type Recorder struct {
	*Identifier
	ids        pids
	cscore     int
	satisfied  bool
	extActive  bool
	mimeActive bool
	textActive bool
}

const (
	extScore = 1 << iota
	mimeScore
	textScore
	incScore
)

func (r *Recorder) Active(m core.MatcherType) {
	if r.Identifier.Active(m) {
		switch m {
		case core.NameMatcher:
			r.extActive = true
		case core.MIMEMatcher:
			r.mimeActive = true
		case core.TextMatcher:
			r.textActive = true
		}
	}
}

func (r *Recorder) Record(m core.MatcherType, res core.Result) bool {
	switch m {
	default:
		return false
	case core.NameMatcher:
		if hit, id := r.Hit(m, res.Index()); hit {
			r.ids = add(r.ids, r.Name(), id, r.infos[id], res.Basis(), extScore)
			return true
		} else {
			return false
		}
	case core.MIMEMatcher:
		if hit, id := r.Hit(m, res.Index()); hit {
			r.ids = add(r.ids, r.Name(), id, r.infos[id], res.Basis(), mimeScore)
			return true
		} else {
			return false
		}
	case core.RIFFMatcher:
		if hit, id := r.Hit(m, res.Index()); hit {
			if r.satisfied {
				return true
			}
			r.cscore += incScore
			r.ids = add(r.ids, r.Name(), id, r.infos[id], res.Basis(), r.cscore)
			return true
		} else {
			return false
		}
	case core.ByteMatcher:
		if hit, id := r.Hit(m, res.Index()); hit {
			if r.satisfied {
				return true
			}
			r.cscore += incScore
			basis := res.Basis()
			p, t := r.Place(core.ByteMatcher, res.Index())
			if t > 1 {
				basis = basis + fmt.Sprintf(" (signature %d/%d)", p, t)
			}
			r.ids = add(r.ids, r.Name(), id, r.infos[id], basis, r.cscore)
			return true
		} else {
			return false
		}
	}
}

func (r *Recorder) Satisfied(mt core.MatcherType) (bool, int) {
	if r.NoPriority() {
		return false, 0
	}
	if r.cscore < incScore {
		if mt == core.ByteMatcher || mt == core.XMLMatcher || mt == core.RIFFMatcher {
			return false, 0
		}
		if len(r.ids) == 0 {
			return false, 0
		}
	}
	r.satisfied = true
	if mt == core.ByteMatcher {
		return true, r.Start(mt)
	}
	return true, 0
}

func lowConfidence(conf int) string {
	var ls = make([]string, 0, 1)
	if conf&extScore == extScore {
		ls = append(ls, "extension")
	}
	if conf&mimeScore == mimeScore {
		ls = append(ls, "MIME")
	}
	if conf&textScore == textScore {
		ls = append(ls, "text")
	}
	switch len(ls) {
	case 0:
		return ""
	case 1:
		return ls[0]
	case 2:
		return ls[0] + " and " + ls[1]
	default:
		return strings.Join(ls[:len(ls)-1], ", ") + " and " + ls[len(ls)-1]
	}
}

func (r *Recorder) Report(res chan core.Identification) {
	// no results
	if len(r.ids) == 0 {
		res <- Identification{
			Namespace: r.Name(),
			ID:        "UNKNOWN",
			Warning:   "no match",
		}
		return
	}
	sort.Sort(r.ids)
	// exhaustive
	if r.Multi() == config.Exhaustive {
		for _, v := range r.ids {
			res <- r.updateWarning(v)
		}
		return
	}
	conf := r.ids[0].confidence
	// if we've only got extension / mime matches, check if those matches are ruled out by lack of byte match
	// only permit a single extension or mime only match
	// add warnings too
	if conf <= textScore {
		nids := make([]Identification, 0, 1)
		for _, v := range r.ids {
			// if overall confidence is greater than mime or ext only, then rule out any lesser confident matches
			if conf > mimeScore && v.confidence != conf {
				break
			}
			// if the match has no corresponding byte or RIFF signature...
			if ok := r.HasSig(v.ID, core.RIFFMatcher, core.ByteMatcher); !ok {
				// break immediately if more than one match
				if len(nids) > 0 {
					nids = nids[:0]
					break
				}
				nids = append(nids, v)
			}
		}
		if len(nids) != 1 {
			poss := make([]string, len(r.ids))
			for i, v := range r.ids {
				poss[i] = v.ID
				conf = conf | v.confidence
			}
			res <- Identification{
				Namespace: r.Name(),
				ID:        "UNKNOWN",
				Warning:   fmt.Sprintf("no match; possibilities based on %v are %v", lowConfidence(conf), strings.Join(poss, ", ")),
			}
			return
		}
		r.ids = nids
	}
	// handle single result only
	if r.Multi() == config.Single && len(r.ids) > 1 && r.ids[0].confidence == r.ids[1].confidence {
		poss := make([]string, 0, len(r.ids))
		for _, v := range r.ids {
			if v.confidence < conf {
				break
			}
			poss = append(poss, v.ID)
		}
		res <- Identification{
			Namespace: r.Name(),
			ID:        "UNKNOWN",
			Warning:   fmt.Sprintf("multiple matches %v", strings.Join(poss, ", ")),
		}
		return
	}
	for i, v := range r.ids {
		if i > 0 {
			switch r.Multi() {
			case config.Single:
				return
			case config.Conclusive:
				if v.confidence < conf {
					return
				}
			default:
				if v.confidence < incScore {
					return
				}
			}
		}
		res <- r.updateWarning(v)
	}
	return
}

func (r *Recorder) updateWarning(i Identification) Identification {
	// apply low confidence
	if i.confidence <= textScore {
		if len(i.Warning) > 0 {
			i.Warning += "; " + "match on " + lowConfidence(i.confidence) + " only"
		} else {
			i.Warning = "match on " + lowConfidence(i.confidence) + " only"
		}
	}
	// apply mismatches
	if r.extActive && (i.confidence&extScore != extScore) {
		for _, v := range r.IDs(core.NameMatcher) {
			if i.ID == v {
				if len(i.Warning) > 0 {
					i.Warning += "; extension mismatch"
				} else {
					i.Warning = "extension mismatch"
				}
				break
			}
		}
	}
	if r.mimeActive && (i.confidence&mimeScore != mimeScore) {
		for _, v := range r.IDs(core.MIMEMatcher) {
			if i.ID == v {
				if len(i.Warning) > 0 {
					i.Warning += "; MIME mismatch"
				} else {
					i.Warning = "MIME mismatch"
				}
				break
			}
		}
	}
	return i
}

type Identification struct {
	Namespace  string
	ID         string
	Name       string
	LongName   string
	Mime       string
	Basis      []string
	Warning    string
	archive    config.Archive
	confidence int
}

func (id Identification) String() string {
	return id.ID
}

func (id Identification) Known() bool {
	return id.ID != "UNKNOWN"
}

func (id Identification) Warn() string {
	return id.Warning
}

func quoteText(s string) string {
	if len(s) == 0 {
		return s
	}
	return "'" + s + "'"
}

func (id Identification) YAML() string {
	var basis string
	if len(id.Basis) > 0 {
		basis = quoteText(strings.Join(id.Basis, "; "))
	}
	return fmt.Sprintf("  - ns      : %v\n    id      : %v\n    format  : %v\n    full     : %v\n    mime    : %v\n    basis   : %v\n    warning : %v\n",
		id.Namespace, id.ID, quoteText(id.Name), quoteText(id.LongName), quoteText(id.Mime), basis, quoteText(id.Warning))
}

func (id Identification) JSON() string {
	var basis string
	if len(id.Basis) > 0 {
		basis = strings.Join(id.Basis, "; ")
	}
	return fmt.Sprintf("{\"ns\":\"%s\",\"id\":\"%s\",\"format\":\"%s\",\"full\":\"%s\",\"mime\":\"%s\",\"basis\":\"%s\",\"warning\":\"%s\"}",
		id.Namespace, id.ID, id.Name, id.LongName, id.Mime, basis, id.Warning)
}

func (id Identification) CSV() []string {
	var basis string
	if len(id.Basis) > 0 {
		basis = strings.Join(id.Basis, "; ")
	}
	return []string{
		id.Namespace,
		id.ID,
		id.Name,
		id.LongName,
		id.Mime,
		basis,
		id.Warning,
	}
}

func (id Identification) Archive() config.Archive {
	return id.archive
}

type pids []Identification

func (p pids) Len() int { return len(p) }

func (p pids) Less(i, j int) bool { return p[j].confidence < p[i].confidence }

func (p pids) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func add(p pids, id string, f string, info formatInfo, basis string, c int) pids {
	for i, v := range p {
		if v.ID == f {
			p[i].confidence += c
			p[i].Basis = append(p[i].Basis, basis)
			return p
		}
	}
	return append(p, Identification{id, f, info.name, info.longName, info.mimeType, []string{basis}, "", config.IsArchive(f), c})
}
