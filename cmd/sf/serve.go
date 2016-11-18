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
	"encoding/base64"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/richardlehane/siegfried"
)

func handleErr(w http.ResponseWriter, status int, e error) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, e.Error())
}

func decodePath(s string) (string, error) {
	if len(s) < 11 {
		return "", fmt.Errorf("Path too short, expecting 11 characters got %d", len(s))
	}
	data, err := base64.URLEncoding.DecodeString(s[10:])
	if err != nil {
		return "", fmt.Errorf("Error base64 decoding file path, error message %v", err)
	}
	return string(data), nil
}

func parseRequest(w http.ResponseWriter, r *http.Request) (
	mime string, wr writer, norec bool, z bool, cs hash.Hash, sf *siegfried.Siegfried) {
	vals := r.URL.Query()
	// json, csv, droid or yaml
	var fmt int
	switch {
	case *jsono:
		fmt = 1
	case *csvo:
		fmt = 2
	case *droido:
		fmt = 3
	}
	if v, ok := vals["format"]; ok && len(v) > 0 {
		switch v[0] {
		case "yaml":
			fmt = 0
		case "json":
			fmt = 1
		case "csv":
			fmt = 2
		case "droid":
			fmt = 3
		}
	}
	if accept := r.Header.Get("Accept"); accept != "" {
		switch accept {
		case "application/x-yaml":
			fmt = 0
		case "application/json":
			fmt = 1
		case "text/csv", "application/csv":
			fmt = 2
		case "application/x-droid":
			fmt = 3
		}
	}
	switch fmt {
	case 0:
		wr = newYAML(w)
		mime = "application/x-yaml"
	case 1:
		wr = newJSON(w)
		mime = "application/json"
	case 2:
		wr = newCSV(w)
		mime = "text/csv"
	case 3:
		wr = newDroid(w)
		mime = "application/x-droid"
	}
	// no recurse
	norec = *nr
	if v, ok := vals["nr"]; ok && len(v) > 0 {
		if v[0] == "true" {
			norec = true
		} else {
			norec = false
		}
	}
	// archive
	z = *archive
	if v, ok := vals["z"]; ok && len(v) > 0 {
		if v[0] == "true" {
			z = true
		} else {
			z = false
		}
	}
	// checksum
	h := *hashf
	if v, ok := vals["hash"]; ok && len(v) > 0 {
		h = v[0]
	}
	cs = getHash(h)
	// sig
	if v, ok := vals["sig"]; ok && len(v) > 0 {
		path, err := base64.URLEncoding.DecodeString(v[0])
		if err == nil {
			sf, _ = siegfried.Load(string(path))
		}
	}
	return
}

func handleIdentify(s *siegfried.Siegfried, ctxts chan *context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		mime, wr, nr, _, _, _ := parseRequest(w, r)
		wg := &sync.WaitGroup{}
		if r.Method == "POST" {
			f, h, err := r.FormFile("file")
			if err != nil {
				handleErr(w, http.StatusNotFound, err)
				return
			}
			defer f.Close()
			var sz int64
			var mod string
			osf, ok := f.(*os.File)
			if ok {
				info, err := osf.Stat()
				if err != nil {
					handleErr(w, http.StatusInternalServerError, err)
				}
				sz = info.Size()
				mod = info.ModTime().String()
			} else {
				sz = r.ContentLength
			}
			w.Header().Set("Content-Type", mime)
			wr.writeHead(s, "")
			ctx := getCtx(h.Filename, "", mod, sz)
			wg.Add(1)
			ctx.wg = wg
			ctxts <- ctx
			identifyRdr(f, ctx, ctxts, getCtx)
			wg.Wait()
			wr.writeTail()
			return
		} else {
			path, err := decodePath(r.URL.Path)
			if err != nil {
				handleErr(w, http.StatusNotFound, err)
				return
			}
			w.Header().Set("Content-Type", mime)
			wr.writeHead(s, "")
			err = identify(ctxts, path, "", nr, getCtx)
			wg.Wait()
			wr.writeTail()
			if err != nil {
				handleErr(w, http.StatusNotFound, err)
			}
			return
		}
	}
}

const usage = `
	<html>
		<head>
			<title>Siegfried server</title>
		</head>
		<body>
			<h1>Siegfried server usage</h1>
			<p>The siegfried server has two modes of identification: GET request, where a file or directory path is given in the URL and the server retrieves the file(s); or POST request, where the file is sent over the network as form-data.</p> 
			<h2>GET request</h2>
			<p><strong>GET</strong> <i>/identify/[<a href="https://tools.ietf.org/html/rfc4648#section-5">URL-safe base64 encoded</a> file name or folder name](?nr=true&format=csv|yaml|json)</i></p>
			<p>E.g. http://localhost:5138/identify/YzpcTXkgRG9jdW1lbnRzXGhlbGxvX3dvcmxkLmRvYw==</p>
			<h3>Parameters</h3>
			<p><i>nr</i> (optional) - this parameter can be used to stop sub-directory recursion when a directory path is given.</p>
			<p><i>format</i> (optional) - this parameter can be used to select the output format (csv, yaml, json). Default is json. Alternatively, HTTP content negotiation can be used.</p>
			
			<p><i>hash</i></p>
			<p><i>z</i></p>
			<p><i>sig</i> (optional)</p>
			<h2>POST request</h2>
			<p><strong>POST</strong> <i>/identify(?format=csv|yaml|json)</i> Attach a file as form-data with the key "file".</p>
			<p>E.g. curl localhost:5138/identify -F file=@myfile.doc</i>
			<h3>Parameters</h3>
			<p><i>format</i> (optional) - this parameter can be used to select the output format (csv, yaml, json). Default is json. Alternatively, HTTP content negotiation can be used.</p>
		</body>
	</html>
`

func handleMain(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" || r.URL.Path != "/" {
		handleErr(w, http.StatusNotFound, fmt.Errorf("Not a valid path"))
		return
	}
	w.Header().Set("Content-Type", "text/html")
	io.WriteString(w, usage)
}

func listen(port string, s *siegfried.Siegfried, ctxts chan *context) {
	http.HandleFunc("/", handleMain)
	http.HandleFunc("/identify", handleIdentify(s, ctxts))
	http.HandleFunc("/identify/", handleIdentify(s, ctxts))
	http.ListenAndServe(port, nil)
}
