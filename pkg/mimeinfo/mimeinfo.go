// Copyright 2015 Richard Lehane. All rights reserved.
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
	"encoding/binary"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/richardlehane/siegfried/config"
	"github.com/richardlehane/siegfried/pkg/core/bytematcher/frames"
	"github.com/richardlehane/siegfried/pkg/core/bytematcher/patterns"
	"github.com/richardlehane/siegfried/pkg/core/parseable"
	"github.com/richardlehane/siegfried/pkg/core/priority"
	"github.com/richardlehane/siegfried/pkg/mimeinfo/mappings"
)

type mimeinfo []mappings.MIMEType

func newMIMEInfo() (mimeinfo, error) {
	buf, err := ioutil.ReadFile(config.MIMEInfo())
	if err != nil {
		return nil, err
	}
	mi := &mappings.MIMEInfo{}
	err = xml.Unmarshal(buf, mi)
	if err != nil {
		return nil, err
	}
	index := make(map[string]int)
	for i, v := range mi.MIMETypes {
		index[v.MIME] = i
	}
	for i, v := range mi.MIMETypes {
		if len(v.SuperiorClasses) == 1 {
			sup := index[v.SuperiorClasses[0].SubClassOf]
			if len(mi.MIMETypes[sup].XMLPattern) > 0 {
				mi.MIMETypes[i].XMLPattern = append(mi.MIMETypes[i].XMLPattern, mi.MIMETypes[sup].XMLPattern...)
			}
			if len(mi.MIMETypes[sup].Magic) > 0 {
				nm := make([]mappings.Magic, len(mi.MIMETypes[sup].Magic))
				copy(nm, mi.MIMETypes[sup].Magic)
				for i, w := range nm {
					if len(w.Priority) > 0 {
						num, err := strconv.Atoi(w.Priority)
						if err == nil {
							nm[i].Priority = strconv.Itoa(num - 1)
							continue
						}
					}
					nm[i].Priority = "49"
				}
				mi.MIMETypes[i].Magic = append(mi.MIMETypes[i].Magic, nm...)
			}
		}
	}
	return mi.MIMETypes, nil
}

func (mi mimeinfo) IDs() []string {
	ids := make([]string, len(mi))
	for i, v := range mi {
		ids[i] = v.MIME
	}
	return ids
}

type formatInfo struct {
	comment      string
	globWeights  []int
	magicWeights []int
}

// turn generic FormatInfo into mimeinfo formatInfo
func infos(m map[string]parseable.FormatInfo) map[string]formatInfo {
	i := make(map[string]formatInfo, len(m))
	for k, v := range m {
		i[k] = v.(formatInfo)
	}
	return i
}

func (mi mimeinfo) Infos() map[string]parseable.FormatInfo {
	fmap := make(map[string]parseable.FormatInfo, len(mi))
	for _, v := range mi {
		fi := formatInfo{}
		if len(v.Comment) > 0 {
			fi.comment = v.Comment[0]
		} else if len(v.Comments) > 0 {
			fi.comment = v.Comments[0]
		}
		var magicWeight int
		for _, mg := range v.Magic {
			magicWeight += len(mg.Matches)
		}
		fi.globWeights, fi.magicWeights = make([]int, len(v.Globs)), make([]int, 0, magicWeight)
		for i, w := range v.Globs {
			if len(w.Weight) > 0 {
				num, err := strconv.Atoi(w.Weight)
				if err == nil {
					fi.globWeights[i] = num
					continue
				}
			}
			fi.globWeights[i] = 50
		}
		for _, w := range v.Magic {
			weight := 50
			if len(w.Priority) > 0 {
				if num, err := strconv.Atoi(w.Priority); err == nil {
					weight = num
				}
			}
			for _, s := range w.Matches {
				ss, _ := toSigs(s)
				for _, sig := range ss {
					if sig != nil {
						fi.magicWeights = append(fi.magicWeights, weight)
					}
				}
			}
		}
		fmap[v.MIME] = fi
	}
	return fmap
}

func (mi mimeinfo) Globs() ([]string, []string) {
	globs, ids := make([]string, 0, len(mi)), make([]string, 0, len(mi))
	for _, v := range mi {
		for _, w := range v.Globs {
			globs, ids = append(globs, w.Pattern), append(ids, v.MIME)
		}
	}
	return globs, ids
}

func (mi mimeinfo) MIMEs() ([]string, []string) {
	mimes, ids := make([]string, 0, len(mi)), make([]string, 0, len(mi))
	for _, v := range mi {
		mimes, ids = append(mimes, v.MIME), append(ids, v.MIME)
		for _, w := range v.Aliases {
			mimes, ids = append(mimes, w.Alias), append(ids, v.MIME)
		}
	}
	return mimes, ids
}

// slice of root/NS
func (mi mimeinfo) XMLs() ([][2]string, []string) {
	xmls, ids := make([][2]string, 0, len(mi)), make([]string, 0, len(mi))
	for _, v := range mi {
		for _, w := range v.XMLPattern {
			xmls, ids = append(xmls, [2]string{w.Local, w.NS}), append(ids, v.MIME)
		}
	}
	return xmls, ids
}

func (mi mimeinfo) Signatures() ([]frames.Signature, []string, error) {
	var errs []error
	sigs, ids := make([]frames.Signature, 0, len(mi)), make([]string, 0, len(mi))
	for _, v := range mi {
		for _, w := range v.Magic {
			for _, s := range w.Matches {
				ss, err := toSigs(s)
				for _, sig := range ss {
					if sig != nil {
						sigs, ids = append(sigs, sig), append(ids, v.MIME)
					}
				}
				if err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
	var err error
	if len(errs) > 0 {
		errStrs := make([]string, len(errs))
		for i, e := range errs {
			errStrs[i] = e.Error()
		}
		err = errors.New(strings.Join(errStrs, "; "))
	}
	return sigs, ids, err
}

func toSigs(m mappings.Match) ([]frames.Signature, error) {
	f, err := toFrames(m)
	if err != nil || f == nil {
		return nil, err
	}
	if len(m.Matches) == 0 {
		return []frames.Signature{frames.Signature(f)}, nil
	}
	subs := make([][]frames.Signature, 0, len(m.Matches))
	for _, m2 := range m.Matches {
		frs, err := toSigs(m2)
		if err != nil {
			return nil, err
		}
		if frs != nil {
			subs = append(subs, frs)
		}
	}
	var l, idx int
	for _, v := range subs {
		l += len(v)
	}
	ss := make([]frames.Signature, l)
	for _, v := range subs {
		for _, w := range v {
			ss[idx] = append(frames.Signature(f), w...)
			idx++
		}
	}
	return ss, nil
}

func toFrames(m mappings.Match) ([]frames.Frame, error) {
	pat, min, max, err := toPattern(m)
	if err != nil || pat == nil {
		return nil, err
	}
	mask, ok := pat.(Mask)
	if !ok {
		return []frames.Frame{frames.NewFrame(frames.BOF, pat, min, max)}, nil
	}
	pats, ints := unmask(mask)
	f := []frames.Frame{frames.NewFrame(frames.BOF, pats[0], min+ints[0], max+ints[0])}
	if len(pats) > 1 {
		for i, p := range pats[1:] {
			f = append(f, frames.NewFrame(frames.PREV, p, ints[i+1], ints[i+1]))
		}
	}
	return f, nil
}

func toPattern(m mappings.Match) (patterns.Pattern, int, int, error) {
	min, max, err := toOffset(m.Offset)
	if err != nil {
		return nil, min, max, err
	}
	var pat patterns.Pattern
	switch m.Typ {
	case "byte":
		i, err := strconv.ParseInt(m.Value, 0, 16)
		if err != nil {
			return nil, min, max, err
		}
		pat = Int8(i)
	case "big16":
		i, err := strconv.ParseInt(m.Value, 0, 32)
		if err != nil {
			return nil, min, max, err
		}
		pat = Big16(i)
	case "little16":
		i, err := strconv.ParseInt(m.Value, 0, 32)
		if err != nil {
			return nil, min, max, err
		}
		pat = Little16(i)
	case "host16":
		i, err := strconv.ParseInt(m.Value, 0, 32)
		if err != nil {
			return nil, min, max, err
		}
		pat = Host16(i)
	case "big32":
		i, err := strconv.ParseInt(m.Value, 0, 64)
		if err != nil {
			return nil, min, max, err
		}
		pat = Big32(i)
	case "little32":
		i, err := strconv.ParseInt(m.Value, 0, 64)
		if err != nil {
			return nil, min, max, err
		}
		pat = Little32(i)
	case "host32":
		i, err := strconv.ParseInt(m.Value, 0, 64)
		if err != nil {
			return nil, min, max, err
		}
		pat = Host32(i)
	case "string":
		pat = patterns.Sequence(unquote(m.Value))
	case "stringignorecase":
		pat = IgnoreCase(unquote(m.Value))
	case "unicodeLE":
		uints := utf16.Encode([]rune(string(unquote(m.Value))))
		buf := make([]byte, len(uints)*2)
		for i, u := range uints {
			binary.LittleEndian.PutUint16(buf[i*2:], u)
		}
		pat = patterns.Sequence(buf)
	case "regex":
		return nil, min, max, nil // ignore regex magic
	default:
		return nil, min, max, errors.New("unknown magic type " + m.Typ)
	}
	if len(m.Mask) > 0 {
		pat = Mask{pat, unquote(m.Mask)}
	}
	return pat, min, max, err
}

func toOffset(off string) (int, int, error) {
	var min, max int
	var err error
	if off == "" {
		return min, max, err
	}
	idx := strings.IndexByte(off, ':')
	switch {
	case idx < 0:
		min, err = strconv.Atoi(off)
		max = min
	case idx == 0:
		max, err = strconv.Atoi(off[1:])
	default:
		min, err = strconv.Atoi(off[:idx])
		if err == nil {
			max, err = strconv.Atoi(off[idx+1:])
		}
	}
	return min, max, err
}

var (
	rpl = strings.NewReplacer("\\ ", " ", "\\n", "\n", "\\t", "\t", "\\r", "\r", "\\b", "\b", "\\f", "\f", "\\v", "\v")
	rgx = regexp.MustCompile(`\\([0-9]{1,3}|x[0-9A-Fa-f]{1,2})`)
)

func numReplace(b []byte) []byte {
	var i uint64
	var err error
	if b[1] == 'x' {
		i, err = strconv.ParseUint(string(b[2:]), 16, 8)
	} else {
		// octal
		if len(b) == 4 {
			i, err = strconv.ParseUint(string(b[1:]), 8, 8)
		} else { // decimal
			i, err = strconv.ParseUint(string(b[1:]), 10, 8)
		}
	}
	if err != nil {
		panic(b)
	}
	return []byte{byte(i)}
}

func unquote(input string) []byte {
	// deal with hex first
	if len(input) > 2 && input[:2] == "0x" {
		h, err := hex.DecodeString(input[2:])
		if err == nil {
			return h
		} else {
			panic(input + " " + err.Error())
		}
	}
	return rgx.ReplaceAllFunc([]byte(rpl.Replace(input)), numReplace)
}

// we don't create a priority map for mimeinfo
func (mi mimeinfo) Priorities() priority.Map { return nil }
