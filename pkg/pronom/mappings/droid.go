// This file contains struct mappings to unmarshal three different PRONOM XML formats: the signature file format, the report format, and the container format
package mappings

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// Droid Signature File

type Droid struct {
	XMLName     xml.Name     `xml:"FFSignatureFile"`
	Version     int          `xml:",attr"`
	FileFormats []FileFormat `xml:"FileFormatCollection>FileFormat"`
}

func (d Droid) String() string {
	buf := new(bytes.Buffer)
	for _, v := range d.FileFormats {
		fmt.Fprintln(buf, v)
	}
	return buf.String()
}

type FileFormat struct {
	ID         int      `xml:",attr"`
	Puid       string   `xml:"PUID,attr"`
	Name       string   `xml:",attr"`
	Version    string   `xml:",attr"`
	MIMEType   string   `xml:",attr"`
	Extensions []string `xml:"Extension"`
	Priorities []int    `xml:"HasPriorityOverFileFormatID"`
	*Report
}

func (f FileFormat) String() string {
	null := func(s string) string {
		if strings.TrimSpace(s) == "" {
			return "NULL"
		}
		return s
	}
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "Puid: %s; Name: %s; Version: %s; Ext(s): %s\n", f.Puid, f.Name, f.Version, strings.Join(f.Extensions, ", "))
	fmt.Fprintln(buf, f.Description)
	for _, v := range f.Signatures {
		fmt.Fprint(buf, "Signature\n")
		for _, v1 := range v.ByteSequences {
			fmt.Fprintf(buf, "Position: %s, Offset: %s, MaxOffset: %s, IndirectLoc: %s, IndirectLen: %s, Endianness: %s\n",
				null(v1.Position), null(v1.Offset), null(v1.MaxOffset), null(v1.IndirectLoc), null(v1.IndirectLen), null(v1.Endianness))
			fmt.Fprintln(buf, v1.Hex)
		}
	}
	return buf.String()
}
