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

package pronom

import (
	"fmt"
	"sort"
	"strings"

	"github.com/richardlehane/siegfried/config"
	"github.com/richardlehane/siegfried/pkg/core"
	"github.com/richardlehane/siegfried/pkg/core/identifier"
	"github.com/richardlehane/siegfried/pkg/core/parseable"
	"github.com/richardlehane/siegfried/pkg/core/persist"
)

func init() {
	core.RegisterIdentifier(core.Pronom, Load)
}

type Identifier struct {
	p     parseable.Parseable
	infos map[string]formatInfo
	*identifier.Base
}

func (i *Identifier) Save(ls *persist.LoadSaver) {
	ls.SaveByte(core.Pronom)
	ls.SaveSmallInt(len(i.infos))
	for k, v := range i.infos {
		ls.SaveString(k)
		ls.SaveString(v.name)
		ls.SaveString(v.version)
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
	pronom, err := newPronom()
	if err != nil {
		return nil, err
	}
	return &Identifier{
		p:     pronom,
		infos: infos(p.Infos()),
		Base:  identifier.New(p, contains(p.IDs(), config.ZipPuid())),
	}, nil
}

func (i *Identifier) Fields() []string {
	return []string{"namespace", "id", "format", "version", "mime", "basis", "warning"}
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
	switch m {
	case core.NameMatcher:
		if r.Identifier.Active(m) {
			r.extActive = true
		}
	case core.MIMEMatcher:
		if r.Identifier.Active(m) {
			r.mimeActive = true
		}
	case core.TextMatcher:
		if r.Identifier.Active(m) {
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
	case core.ContainerMatcher:
		// add zip default
		if res.Index() < 0 {
			if r.ZipDefault() {
				r.cscore += incScore
				r.ids = add(r.ids, r.Name(), config.ZipPuid(), r.infos[config.ZipPuid()], res.Basis(), r.cscore)
			}
			return false
		}
		if hit, id := r.Hit(m, res.Index()); hit {
			r.cscore += incScore
			basis := res.Basis()
			p, t := r.Place(core.ContainerMatcher, res.Index())
			if t > 1 {
				basis = basis + fmt.Sprintf(" (signature %d/%d)", p, t)
			}
			r.ids = add(r.ids, r.Name(), id, r.infos[id], basis, r.cscore)
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
	case core.TextMatcher:
		if hit, id := r.Hit(m, res.Index()); hit {
			if r.satisfied {
				return true
			}
			r.ids = add(r.ids, r.Name(), id, r.infos[id], res.Basis(), textScore)
			return true
		} else {
			return false
		}
	}
}

func (r *Recorder) Satisfied(mt core.MatcherType) (bool, int) {
	if r.cscore < incScore {
		if mt == core.ByteMatcher || mt == core.XMLMatcher {
			return false, 0
		}
		if len(r.ids) == 0 {
			return false, 0
		}
		for _, res := range r.ids {
			if res.ID == config.TextPuid() {
				return false, 0
			}
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
	if len(r.ids) > 0 {
		sort.Sort(r.ids)
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
				// if we have plain text result that is based on ext or mime only,
				// and not on a text match, and if text matcher is on for this identifier,
				// then don't report a text match
				if v.ID == config.TextPuid() && conf < textScore && r.textActive {
					continue
				}
				// if the match has no corresponding byte or container signature...
				if ok := r.hasSig(v.ID); !ok {
					// break immediately if more than one match
					if len(nids) > 0 {
						nids = nids[:0]
						break
					}
					if len(v.Warning) > 0 {
						v.Warning += "; " + "match on " + lowConfidence(v.confidence) + " only"
					} else {
						v.Warning = "match on " + lowConfidence(v.confidence) + " only"
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
				nids = []Identification{Identification{r.Name(), "UNKNOWN", "", "", "", nil, fmt.Sprintf("no match; possibilities based on %v are %v", lowConfidence(conf), strings.Join(poss, ", ")), 0, 0}}
			}
			r.ids = nids
		}
		res <- r.checkActive(r.ids[0])
		if len(r.ids) > 1 {
			for i, v := range r.ids[1:] {
				if v.confidence == conf || (r.NoPriority() && v.confidence >= incScore) {
					res <- r.checkActive(r.ids[i+1])
				} else {
					break
				}
			}
		}
	} else {
		res <- Identification{r.Name(), "UNKNOWN", "", "", "", nil, "no match", 0, 0}
	}
}

func (r *Recorder) checkActive(i Identification) Identification {
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

func (r *Recorder) hasSig(puid string) bool {
	for _, v := range r.IDs(core.ContainerMatcher) {
		if puid == v {
			return true
		}
	}
	for _, v := range r.IDs(core.ByteMatcher) {
		if puid == v {
			return true
		}
	}
	return false
}

type Identification struct {
	Namespace  string
	ID         string
	Name       string
	Version    string
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
	return fmt.Sprintf("  - ns      : %v\n    id      : %v\n    format  : %v\n    version : %v\n    mime    : %v\n    basis   : %v\n    warning : %v\n",
		id.Namespace, id.ID, quoteText(id.Name), quoteText(id.Version), quoteText(id.Mime), basis, quoteText(id.Warning))
}

func (id Identification) JSON() string {
	var basis string
	if len(id.Basis) > 0 {
		basis = strings.Join(id.Basis, "; ")
	}
	return fmt.Sprintf("{\"ns\":\"%s\",\"id\":\"%s\",\"format\":\"%s\",\"version\":\"%s\",\"mime\":\"%s\",\"basis\":\"%s\",\"warning\":\"%s\"}",
		id.Namespace, id.ID, id.Name, id.Version, id.Mime, basis, id.Warning)
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
		id.Version,
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
	return append(p, Identification{id, f, info.name, info.version, info.mimeType, []string{basis}, "", config.IsArchive(f), c})
}
