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
	"fmt"
	"io"
	"os"
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

type writer interface {
	writeHead(s *siegfried.Siegfried)
	writeFile(name string, sz int64, err error, ids iterableID) config.Archive
	writeTail()
}

type debugWriter struct{}

func (d debugWriter) writeHead(s *siegfried.Siegfried) {}
func (d debugWriter) writeFile(name string, sz int64, err error, ids iterableID) config.Archive {
	return 0
}
func (d debugWriter) writeTail() {}

type csvWriter struct {
	rec []string
	w   *csv.Writer
}

func newCSV(w io.Writer) *csvWriter {
	return &csvWriter{make([]string, 10), csv.NewWriter(os.Stdout)}
}

func (c *csvWriter) writeHead(s *siegfried.Siegfried) {
	c.w.Write([]string{"filename", "filesize", "errors", "identifier", "id", "format name", "format version", "mimetype", "basis", "warning"})
}

func (c *csvWriter) writeFile(name string, sz int64, err error, ids iterableID) config.Archive {
	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	if ids == nil {
		empty := make([]string, 7)
		c.rec[0], c.rec[1], c.rec[2] = name, strconv.Itoa(int(sz)), errStr
		copy(c.rec[3:], empty)
		c.w.Write(c.rec)
		return 0
	}
	var archive config.Archive
	for id := ids.next(); id != nil; id = ids.next() {
		if id.Archive() > archive {
			archive = id.Archive()
		}
		c.rec[0], c.rec[1], c.rec[2] = name, strconv.Itoa(int(sz)), errStr
		copy(c.rec[3:], id.CSV())
		c.w.Write(c.rec)
	}
	return archive
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

func (y *yamlWriter) writeFile(name string, sz int64, err error, ids iterableID) config.Archive {
	var errStr string
	if err != nil {
		errStr = fmt.Sprintf("'%s'", err.Error())
	}
	fmt.Fprintf(y.w, "---\nfilename : '%s'\nfilesize : %d\nerrors   : %s\nmatches  :\n", y.replacer.Replace(name), sz, errStr)
	if ids == nil {
		return 0
	}
	var archive config.Archive
	for id := ids.next(); id != nil; id = ids.next() {
		if id.Archive() > archive {
			archive = id.Archive()
		}
		y.w.WriteString(id.YAML())
	}
	return archive
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

func (j *jsonWriter) writeFile(name string, sz int64, err error, ids iterableID) config.Archive {
	if j.subs {
		j.w.WriteString(",")
	}
	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	fmt.Fprintf(j.w, "{\"filename\":\"%s\",\"filesize\": %d,\"errors\": \"%s\",\"matches\": [", j.replacer.Replace(name), sz, errStr)
	if ids == nil {
		return 0
	}
	var subs bool
	var archive config.Archive
	for id := ids.next(); id != nil; id = ids.next() {
		if id.Archive() > archive {
			archive = id.Archive()
		}
		if subs {
			j.w.WriteString(",")
		}
		j.w.WriteString(id.JSON())
		subs = true
	}
	j.w.WriteString("]}")
	j.subs = true
	return archive
}

func (j *jsonWriter) writeTail() {
	j.w.WriteString("]}\n")
	j.w.Flush()
}
