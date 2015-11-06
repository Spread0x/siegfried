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

package main

import (
	"bufio"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/richardlehane/siegfried"
	"github.com/richardlehane/siegfried/config"
	"github.com/richardlehane/siegfried/pkg/core"
)

type iterableID interface {
	next() core.Identification
}

type idChan chan core.Identification

func (ids idChan) next() core.Identification {
	id, ok := <-ids
	if !ok {
		return nil
	}
	lg.id(id)
	return id
}

type idSlice struct {
	idx int
	ids []core.Identification
}

func (is *idSlice) next() core.Identification {
	is.idx++
	if is.idx > len(is.ids) {
		return nil
	}
	return is.ids[is.idx-1]
}

func makeIdSlice(c iterableID) *idSlice {
	ids := make([]core.Identification, 0, 1)
	for id := c.next(); id != nil; id = c.next() {
		ids = append(ids, id)
	}
	return &idSlice{0, ids}
}

type writer interface {
	writeHead(s *siegfried.Siegfried)
	// if a directory give a negative sz
	writeFile(name string, sz int64, mod string, checksum []byte, err error, ids iterableID) config.Archive
	writeTail()
}

type csvWriter struct {
	rec []string
	w   *csv.Writer
}

func newCSV(w io.Writer) *csvWriter {
	l := 11
	if *hashf != "" {
		l = 12
	}
	return &csvWriter{make([]string, l), csv.NewWriter(os.Stdout)}
}

func (c *csvWriter) writeHead(s *siegfried.Siegfried) {
	if *hashf != "" {
		c.w.Write([]string{"filename", "filesize", "modified", "errors", hashHeader(false), "id", "puid", "format", "version", "mime", "basis", "warning"})
		return
	}
	c.w.Write([]string{"filename", "filesize", "modified", "errors", "id", "puid", "format", "version", "mime", "basis", "warning"})
}

func (c *csvWriter) writeFile(name string, sz int64, mod string, checksum []byte, err error, ids iterableID) config.Archive {
	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	rest := 4
	if checksum != nil {
		rest = 5
	}
	if ids == nil {
		empty := make([]string, 7)
		c.rec[0], c.rec[1], c.rec[2], c.rec[3] = name, strconv.Itoa(int(sz)), mod, errStr
		if checksum != nil {
			c.rec[4] = ""
		}
		copy(c.rec[rest:], empty)
		c.w.Write(c.rec)
		return 0
	}
	var arc config.Archive
	for id := ids.next(); id != nil; id = ids.next() {
		if id.Archive() > arc {
			arc = id.Archive()
		}
		c.rec[0], c.rec[1], c.rec[2], c.rec[3] = name, strconv.Itoa(int(sz)), mod, errStr
		if checksum != nil {
			c.rec[4] = hex.EncodeToString(checksum)
		}
		copy(c.rec[rest:], id.CSV())
		c.w.Write(c.rec)
	}
	return arc
}

func (c *csvWriter) writeTail() { c.w.Flush() }

type yamlWriter struct {
	replacer *strings.Replacer
	w        *bufio.Writer
}

func newYAML(w io.Writer) *yamlWriter {
	return &yamlWriter{strings.NewReplacer("'", "''"), bufio.NewWriter(w)}
}

func (y *yamlWriter) writeHead(s *siegfried.Siegfried) {
	y.w.WriteString(s.YAML())
}

func (y *yamlWriter) writeFile(name string, sz int64, mod string, checksum []byte, err error, ids iterableID) config.Archive {
	var errStr string
	if err != nil {
		errStr = fmt.Sprintf("'%s'", err.Error())
	}
	var h string
	if checksum != nil {
		h = fmt.Sprintf("%s   : %s\n", hashHeader(true), hex.EncodeToString(checksum))
	}
	fmt.Fprintf(y.w, "---\nfilename : '%s'\nfilesize : %d\nmodified : %s\nerrors   : %s\n%smatches  :\n", y.replacer.Replace(name), sz, mod, errStr, h)
	if ids == nil {
		return 0
	}
	var arc config.Archive
	for id := ids.next(); id != nil; id = ids.next() {
		if id.Archive() > arc {
			arc = id.Archive()
		}
		y.w.WriteString(id.YAML())
	}
	return arc
}

func (y *yamlWriter) writeTail() { y.w.Flush() }

type jsonWriter struct {
	subs     bool
	replacer *strings.Replacer
	w        *bufio.Writer
}

func newJSON(w io.Writer) *jsonWriter {
	return &jsonWriter{false, strings.NewReplacer(`"`, `\"`, `\\`, `\\`, `\`, `\\`), bufio.NewWriter(w)}
}

func (j *jsonWriter) writeHead(s *siegfried.Siegfried) {
	j.w.WriteString(s.JSON())
	j.w.WriteString("\"files\":[")
}

func (j *jsonWriter) writeFile(name string, sz int64, mod string, checksum []byte, err error, ids iterableID) config.Archive {
	if j.subs {
		j.w.WriteString(",")
	}
	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	var h string
	if checksum != nil {
		h = fmt.Sprintf("\"%s\":\"%s\",", hashHeader(false), hex.EncodeToString(checksum))
	}
	fmt.Fprintf(j.w, "{\"filename\":\"%s\",\"filesize\": %d,\"modified\":\"%s\",\"errors\": \"%s\",%s\"matches\": [", j.replacer.Replace(name), sz, mod, errStr, h)
	if ids == nil {
		return 0
	}
	var subs bool
	var arc config.Archive
	for id := ids.next(); id != nil; id = ids.next() {
		if id.Archive() > arc {
			arc = id.Archive()
		}
		if subs {
			j.w.WriteString(",")
		}
		j.w.WriteString(id.JSON())
		subs = true
	}
	j.w.WriteString("]}")
	j.subs = true
	return arc
}

func (j *jsonWriter) writeTail() {
	j.w.WriteString("]}\n")
	j.w.Flush()
}

type droidWriter struct {
	id      int
	parents map[string]parent
	rec     []string
	w       *csv.Writer
}

type parent struct {
	id      int
	uri     string
	archive string
}

func newDroid(w io.Writer) *droidWriter {
	return &droidWriter{
		parents: make(map[string]parent),
		rec:     make([]string, 18),
		w:       csv.NewWriter(os.Stdout),
	}
}

// "identifier", "id", "format name", "format version", "mimetype", "basis", "warning"

func (d *droidWriter) writeHead(s *siegfried.Siegfried) {
	d.w.Write([]string{
		"ID", "PARENT_ID", "URI", "FILE_PATH", "NAME",
		"METHOD", "STATUS", "SIZE", "TYPE", "EXT",
		"LAST_MODIFIED", "EXTENSION_MISMATCH", strings.ToUpper(hashHeader(false)) + "_HASH", "FORMAT_COUNT",
		"PUID", "MIME_TYPE", "FORMAT_NAME", "FORMAT_VERSION"})
}

func (d *droidWriter) writeFile(p string, sz int64, mod string, checksum []byte, err error, ids iterableID) config.Archive {
	d.id++
	d.rec[0], d.rec[6], d.rec[10] = strconv.Itoa(d.id), "Done", mod
	if err != nil {
		d.rec[6] = err.Error()
	}
	d.rec[1], d.rec[2], d.rec[3], d.rec[4], d.rec[9] = d.processPath(p)
	// if folder (has sz -1) or error
	if sz < 0 || ids == nil {
		d.rec[5], d.rec[7], d.rec[12], d.rec[13], d.rec[14], d.rec[15], d.rec[16], d.rec[17] = "", "", "", "", "", "", "", ""
		if sz < 0 {
			d.rec[8], d.rec[9], d.rec[11] = "Folder", "", "false"
			d.parents[d.rec[3]] = parent{d.id, d.rec[2], ""}
		} else {
			d.rec[8], d.rec[11] = "", ""
		}
		d.rec[3] = clearArchivePath(d.rec[2], d.rec[3])
		d.w.Write(d.rec)
		return 0
	}
	// size
	d.rec[7] = strconv.Itoa(int(sz))
	if checksum == nil {
		d.rec[12] = ""
	} else {
		d.rec[12] = hex.EncodeToString(checksum)
	}
	var arc config.Archive
	nids := makeIdSlice(ids)
	// leave early for unknowns
	if !nids.ids[0].Known() {
		d.rec[5], d.rec[8], d.rec[11], d.rec[13] = "", "File", "FALSE", "0"
		d.rec[14], d.rec[15], d.rec[16], d.rec[17] = "", "", "", ""
		d.rec[3] = clearArchivePath(d.rec[2], d.rec[3])
		d.w.Write(d.rec)
		return 0
	}
	d.rec[13] = strconv.Itoa(len(nids.ids))
	for id := nids.next(); id != nil; id = nids.next() {
		if id.Archive() > 0 {
			arc = id.Archive()
			d.rec[8] = "Container"
			d.parents[d.rec[3]] = parent{d.id, d.rec[2], arc.String()}
		} else {
			d.rec[8] = "File"
		}
		fields := id.CSV()
		d.rec[5], d.rec[11] = getMethod(fields[5]), mismatch(fields[6])
		d.rec[14], d.rec[15], d.rec[16], d.rec[17] = fields[1], fields[4], fields[2], fields[3]
		d.rec[3] = clearArchivePath(d.rec[2], d.rec[3])
		d.w.Write(d.rec)
	}
	return arc
}

func (d *droidWriter) writeTail() { d.w.Flush() }

func (d *droidWriter) processPath(p string) (parent, uri, path, name, ext string) {
	path, _ = filepath.Abs(p)
	path = strings.TrimSuffix(path, string(filepath.Separator))
	name = filepath.Base(path)
	dir := filepath.Dir(path)
	par, ok := d.parents[dir]
	if ok {
		parent = strconv.Itoa(par.id)
		uri = toUri(par.uri, par.archive, name)
	} else {
		puri := "file:/" + filepath.ToSlash(dir)
		uri = toUri(puri, "", name)
	}
	ext = strings.TrimPrefix(filepath.Ext(p), ".")
	return
}

func toUri(parenturi, parentarc, base string) string {
	if len(parentarc) > 0 {
		parenturi = parentarc + ":" + parenturi + "!"
	}
	return parenturi + "/" + base
}

func clearArchivePath(uri, path string) string {
	if strings.HasPrefix(uri, config.Zip.String()) ||
		strings.HasPrefix(uri, config.Tar.String()) ||
		strings.HasPrefix(uri, config.Gzip.String()) {
		path = ""
	}
	return path
}

func getMethod(basis string) string {
	switch {
	case strings.Contains(basis, "container"):
		return "Container"
	case strings.Contains(basis, "byte"):
		return "Signature"
	case strings.Contains(basis, "extension"):
		return "Extension"
	case strings.Contains(basis, "text"):
		return "Text"
	}
	return ""
}

func mismatch(warning string) string {
	if strings.Contains(warning, "extension mismatch") {
		return "TRUE"
	}
	return "FALSE"
}

type logWriter struct{}

func (l logWriter) writeHead(s *siegfried.Siegfried) {}
func (l logWriter) writeFile(name string, sz int64, mod string, cs []byte, err error, ids iterableID) config.Archive {
	var arc config.Archive
	for id := ids.next(); id != nil; id = ids.next() {
		if id.Archive() > 0 {
			arc = id.Archive()
		}
	}
	return arc
}
func (l logWriter) writeTail() {}
