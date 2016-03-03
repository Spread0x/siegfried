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

package mimeinfo

import (
	"fmt"

	"github.com/richardlehane/siegfried/config"
	"github.com/richardlehane/siegfried/pkg/core"
	"github.com/richardlehane/siegfried/pkg/core/bytematcher"
	"github.com/richardlehane/siegfried/pkg/core/bytematcher/frames"
	"github.com/richardlehane/siegfried/pkg/core/mimematcher"
	"github.com/richardlehane/siegfried/pkg/core/namematcher"
	"github.com/richardlehane/siegfried/pkg/core/parseable"
	"github.com/richardlehane/siegfried/pkg/core/persist"
	"github.com/richardlehane/siegfried/pkg/core/textmatcher"
	"github.com/richardlehane/siegfried/pkg/core/xmlmatcher"
)

func init() {
	core.RegisterIdentifier(core.MIMEInfo, Load)
}

type Identifier struct {
	p          parseable.Parseable
	name       string
	details    string
	zipDefault bool
	infos      map[string]formatInfo
	gstart     int
	gids       []string
	mstart     int
	mids       []string
	xstart     int
	xids       []string
	bstart     int
	bids       []string
	tstart     int
}

func (i *Identifier) Save(ls *persist.LoadSaver) {
	ls.SaveByte(core.MIMEInfo)
	ls.SaveString(i.name)
	ls.SaveString(i.details)
	ls.SaveBool(i.zipDefault)
	ls.SaveSmallInt(len(i.infos))
	for k, v := range i.infos {
		ls.SaveString(k)
		ls.SaveString(v.comment)
		ls.SaveInts(v.globWeights)
		ls.SaveInts(v.magicWeights)
	}
	ls.SaveInt(i.gstart)
	ls.SaveStrings(i.gids)
	ls.SaveInt(i.mstart)
	ls.SaveStrings(i.mids)
	ls.SaveInt(i.xstart)
	ls.SaveStrings(i.xids)
	ls.SaveInt(i.bstart)
	ls.SaveStrings(i.bids)
	ls.SaveSmallInt(i.tstart)
}

func Load(ls *persist.LoadSaver) core.Identifier {
	i := &Identifier{}
	i.name = ls.LoadString()
	i.details = ls.LoadString()
	i.zipDefault = ls.LoadBool()
	i.infos = make(map[string]formatInfo)
	le := ls.LoadSmallInt()
	for j := 0; j < le; j++ {
		i.infos[ls.LoadString()] = formatInfo{
			ls.LoadString(),
			ls.LoadInts(),
			ls.LoadInts(),
		}
	}
	i.gstart = ls.LoadInt()
	i.gids = ls.LoadStrings()
	i.mstart = ls.LoadInt()
	i.mids = ls.LoadStrings()
	i.xstart = ls.LoadInt()
	i.xids = ls.LoadStrings()
	i.bstart = ls.LoadInt()
	i.bids = ls.LoadStrings()
	i.tstart = ls.LoadSmallInt()
	return i
}

func contains(ss []string, str string) bool {
	for _, s := range ss {
		if s == str {
			return true
		}
	}
	return false
}

func New(opts ...config.Option) (*Identifier, error) {
	for _, v := range opts {
		v()
	}
	mi, err := newMIMEInfo()
	if err != nil {
		return nil, err
	}
	id := &Identifier{
		p:       mi,
		name:    config.Name(),
		details: config.Details(),
		infos:   infos(mi.Infos()),
	}
	if contains(mi.IDs(), config.ZipMIME()) {
		id.zipDefault = true
	}
	return id, nil
}

func (i *Identifier) Add(m core.Matcher, t core.MatcherType) error {
	switch t {
	default:
		return fmt.Errorf("MIMEInfo: unknown matcher type %d", t)
	case core.NameMatcher:
		if !config.NoName() {
			var globs []string
			globs, i.gids = i.p.Globs()
			l, err := m.Add(namematcher.SignatureSet(globs), nil)
			if err != nil {
				return err
			}
			i.gstart = l - len(i.gids)
			return nil
		}
	case core.MIMEMatcher:
		if !config.NoMIME() {
			var mimes []string
			mimes, i.mids = i.p.MIMEs()
			l, err := m.Add(mimematcher.SignatureSet(mimes), nil)
			if err != nil {
				return err
			}
			i.mstart = l - len(i.mids)
			return nil
		}
	case core.XMLMatcher:
		if !config.NoXML() {
			var xmls [][2]string
			xmls, i.xids = i.p.XMLs()
			l, err := m.Add(xmlmatcher.SignatureSet(xmls), nil)
			if err != nil {
				return err
			}
			i.xstart = l - len(i.xids)
			return nil
		}
	case core.ContainerMatcher:
		return nil
	case core.ByteMatcher:
		var sigs []frames.Signature
		var err error
		sigs, i.bids, err = i.p.Signatures()
		if err != nil {
			return err
		}
		l, err := m.Add(bytematcher.SignatureSet(sigs), nil)
		if err != nil {
			return err
		}
		i.bstart = l - len(i.bids)
	case core.TextMatcher:
		if !config.NoText() && contains(i.p.IDs(), config.TextMIME()) {
			l, _ := m.Add(textmatcher.SignatureSet{}, nil)
			i.tstart = l
		}
	}
	return nil
}

func (i *Identifier) Name() string {
	return i.name
}

func (i *Identifier) Details() string {
	return i.details
}

func (i *Identifier) String() string {
	str := fmt.Sprintf("Name: %s\nDetails: %s\n", i.name, i.details)
	str += fmt.Sprintf("Number of filename signatures: %d \n", len(i.gids))
	str += fmt.Sprintf("Number of MIME signatures: %d \n", len(i.mids))
	str += fmt.Sprintf("Number of XML signatures: %d \n", len(i.xids))
	str += fmt.Sprintf("Number of byte signatures: %d \n", len(i.bids))
	return str
}

func (i *Identifier) Recognise(m core.MatcherType, idx int) (bool, string) {
	switch m {
	default:
		return false, ""
	case core.NameMatcher:
		if idx >= i.gstart && idx < i.gstart+len(i.gids) {
			idx = idx - i.gstart
			return true, i.name + ": " + i.gids[idx]
		}
		return false, ""
	case core.MIMEMatcher:
		if idx >= i.mstart && idx < i.mstart+len(i.mids) {
			idx = idx - i.mstart
			return true, i.name + ": " + i.mids[idx]
		}
		return false, ""
	case core.XMLMatcher:
		if idx >= i.xstart && idx < i.xstart+len(i.xids) {
			idx = idx - i.xstart
			return true, i.name + ": " + i.xids[idx]
		}
		return false, ""
	case core.ContainerMatcher:
		return false, ""
	case core.ByteMatcher:
		if idx >= i.bstart && idx < i.bstart+len(i.bids) {
			return true, i.name + ": " + i.bids[idx]
		}
		return false, ""
	case core.TextMatcher:
		if idx == i.tstart {
			return true, i.name + ": " + config.TextPuid()
		}
		return false, ""
	}
}

func (i *Identifier) Recorder() core.Recorder {
	return nil
}

type Recorder struct {
	*Identifier
	ids        mids
	cscore     int
	satisfied  bool
	globActive bool
	mimeActive bool
	xmlActive  bool
	textActive bool
}

func (r *Recorder) Active(m core.MatcherType) {
	switch m {
	case core.NameMatcher:
		if len(r.ePuids) > 0 {
			r.extActive = true
		}
	case core.MIMEMatcher:
		if len(r.mPuids) > 0 {
			r.mimeActive = true
		}
	case core.TextMatcher:
		if r.tStart > 0 {
			r.textActive = true
		}
	}
}

func (r *Recorder) Record(m core.MatcherType, res core.Result) bool {
	switch m {
	default:
		return false
	case core.NameMatcher:
		if res.Index() >= r.eStart && res.Index() < r.eStart+len(r.ePuids) {
			idx := res.Index() - r.eStart
			r.ids = add(r.ids, r.name, r.ePuids[idx], r.infos[r.ePuids[idx]], res.Basis(), extScore)
			return true
		} else {
			return false
		}
	case core.MIMEMatcher:
		if res.Index() >= r.mStart && res.Index() < r.mStart+len(r.mPuids) {
			idx := res.Index() - r.mStart
			r.ids = add(r.ids, r.name, r.mPuids[idx], r.infos[r.mPuids[idx]], res.Basis(), mimeScore)
			return true
		} else {
			return false
		}
	case core.ContainerMatcher:
		// add zip default
		if res.Index() < 0 {
			if r.zipDefault {
				r.cscore += incScore
				r.ids = add(r.ids, r.name, config.ZipPuid(), r.infos[config.ZipPuid()], res.Basis(), r.cscore)
			}
			return false
		}
		if res.Index() >= r.cStart && res.Index() < r.cStart+len(r.cPuids) {
			idx := res.Index() - r.cStart
			r.cscore += incScore
			basis := res.Basis()
			p, t := place(idx, r.cPuids)
			if t > 1 {
				basis = basis + fmt.Sprintf(" (signature %d/%d)", p, t)
			}
			r.ids = add(r.ids, r.name, r.cPuids[idx], r.infos[r.cPuids[idx]], basis, r.cscore)
			return true
		} else {
			return false
		}
	case core.ByteMatcher:
		if res.Index() >= r.bStart && res.Index() < r.bStart+len(r.bPuids) {
			if r.satisfied {
				return true
			}
			idx := res.Index() - r.bStart
			r.cscore += incScore
			basis := res.Basis()
			p, t := place(idx, r.bPuids)
			if t > 1 {
				basis = basis + fmt.Sprintf(" (signature %d/%d)", p, t)
			}
			r.ids = add(r.ids, r.name, r.bPuids[idx], r.infos[r.bPuids[idx]], basis, r.cscore)
			return true
		} else {
			return false
		}
	case core.TextMatcher:
		if res.Index() == r.tStart {
			if r.satisfied {
				return true
			}
			r.ids = add(r.ids, r.name, config.TextPuid(), r.infos[config.TextPuid()], res.Basis(), textScore)
			return true
		} else {
			return false
		}
	}
}

func place(idx int, ids []string) (int, int) {
	puid := ids[idx]
	var prev, post int
	for i := idx - 1; i > -1 && ids[i] == puid; i-- {
		prev++
	}
	for i := idx + 1; i < len(ids) && ids[i] == puid; i++ {
		post++
	}
	return prev + 1, prev + post + 1
}

func (r *Recorder) Satisfied(mt core.MatcherType) bool {
	if r.cscore < incScore {
		if mt == core.ByteMatcher {
			return false
		}
		if len(r.ids) == 0 {
			return false
		}
		for _, res := range r.ids {
			if res.ID == config.TextPuid() {
				return false
			}
		}
	}
	r.satisfied = true
	return true
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
				nids = []Identification{Identification{r.name, "UNKNOWN", "", "", "", nil, fmt.Sprintf("no match; possibilities based on %v are %v", lowConfidence(conf), strings.Join(poss, ", ")), 0, 0}}
			}
			r.ids = nids
		}
		res <- r.checkActive(r.ids[0])
		if len(r.ids) > 1 {
			for i, v := range r.ids[1:] {
				if v.confidence == conf || (r.noPriority && v.confidence >= incScore) {
					res <- r.checkActive(r.ids[i+1])
				} else {
					break
				}
			}
		}
	} else {
		res <- Identification{r.name, "UNKNOWN", "", "", "", nil, "no match", 0, 0}
	}
}

func (r *Recorder) checkActive(i Identification) Identification {
	if r.extActive && (i.confidence&extScore != extScore) {
		for _, v := range r.ePuids {
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
		for _, v := range r.mPuids {
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
	for _, v := range r.cPuids {
		if puid == v {
			return true
		}
	}
	for _, v := range r.bPuids {
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
	return fmt.Sprintf("  - ns      : %v\n      id    : %v\n    format  : %v\n    version : %v\n    mime    : %v\n    basis   : %v\n    warning : %v\n",
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
