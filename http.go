// Copyright 2018 Prosap sp. z o.o. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package plcconnector

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	tableStyle = "style='font-family:\"Courier New\", Courier, monospace; border-spacing: 20px 0;'"
)

func typeToString(t int) string {
	switch t {
	case TypeBOOL:
		return "BOOL"
	case TypeSINT:
		return "SINT"
	case TypeINT:
		return "INT"
	case TypeDINT:
		return "DINT"
	case TypeREAL:
		return "REAL"
	case TypeDWORD:
		return "DWORD"
	case TypeLINT:
		return "LINT"
	default:
		return "UNKNOWN"
	}
}

func asciiCode(x rune) (r string) {
	switch x {
	case 0:
		r = "NUL"
	case 1:
		r = "SOH"
	case 2:
		r = "STX"
	case 3:
		r = "ETX"
	case 4:
		r = "EOT"
	case 5:
		r = "ENQ"
	case 6:
		r = "ACK"
	case 7:
		r = "BEL"
	case 8:
		r = "BS"
	case 9:
		r = "HT"
	case 10:
		r = "LF"
	case 11:
		r = "VT"
	case 12:
		r = "FF"
	case 13:
		r = "CR"
	case 14:
		r = "SO"
	case 15:
		r = "SI"
	case 16:
		r = "DLE"
	case 17:
		r = "DC1"
	case 18:
		r = "DC2"
	case 19:
		r = "DC3"
	case 20:
		r = "DC4"
	case 21:
		r = "NAK"
	case 22:
		r = "SYN"
	case 23:
		r = "ETB"
	case 24:
		r = "CAN"
	case 25:
		r = "EM"
	case 26:
		r = "SUB"
	case 27:
		r = "ESC"
	case 28:
		r = "FS"
	case 29:
		r = "GS"
	case 30:
		r = "RS"
	case 31:
		r = "US"
	case 0x7F:
		r = "DEL"
	default:
		r = string(x)
	}
	return
}

func tagsIndexHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	var toSend strings.Builder

	toSend.WriteString("<!DOCTYPE html>\n<html><h3>PLC connector</h3><p>Wersja: 6</p>\n<table " + tableStyle + "><tr><th>Nazwa</th><th>Rozmiar</th><th>Typ</th><th>ASCII</th></tr>\n")

	tMut.RLock()
	arr := make([]string, 0, len(tags))

	for n := range tags {
		arr = append(arr, n)
	}

	sort.Strings(arr)

	for _, n := range arr {
		toSend.WriteString("<tr><td><a href=\"/" + n + "\">" + n + "</a></td><td>" + strconv.Itoa(tags[n].Count) + "</td><td>" + typeToString(tags[n].Typ) + "</td><td>")
		var ascii strings.Builder
		if tags[n].Typ != TypeREAL && tags[n].Typ != TypeBOOL {
			ascii.Grow(tags[n].Count)
			ln := int(typeLen(uint16(tags[n].Typ)))
			for i := 0; i < len(tags[n].data); i += ln {
				tmp := int64(tags[n].data[i])
				for j := 1; j < ln; j++ {
					tmp += int64(tags[n].data[i+j]) << uint(8*j)
				}
				switch tags[n].Typ {
				case TypeSINT:
					tmp = int64(int8(tmp))
				case TypeINT:
					tmp = int64(int16(tmp))
				case TypeDINT:
					tmp = int64(int32(tmp))
				case TypeDWORD:
					tmp = int64(int32(tmp))
				case TypeLINT:
					tmp = int64(int64(tmp))
				}
				if tmp < 256 && tmp >= 32 {
					ascii.WriteRune(rune(tmp))
				} else {
					break
				}
			}
			toSend.WriteString(ascii.String())
			toSend.WriteString("</td></tr>\n")
		}
	}
	tMut.RUnlock()

	toSend.WriteString("</table></html>")

	io.WriteString(w, toSend.String())
}

type tagJSON struct {
	Typ   string    `json:"type"`
	Count int       `json:"count"`
	Data  []float64 `json:"data"`
	ASCII []string  `json:"ascii,omitempty"`
}

func tagToJSON(t *Tag) string {
	var tj tagJSON
	tj.Count = int(t.Count)
	ln := int(typeLen(uint16(t.Typ)))
	for i := 0; i < len(t.data); i += ln {
		tmp := int64(t.data[i])
		for j := 1; j < ln; j++ {
			tmp += int64(t.data[i+j]) << uint(8*j)
		}
		switch t.Typ {
		case TypeBOOL:
			if tmp != 0 {
				tmp = 1
			}
		case TypeSINT:
			tmp = int64(int8(tmp))
		case TypeINT:
			tmp = int64(int16(tmp))
		case TypeDINT:
			tmp = int64(int32(tmp))
		case TypeDWORD:
			tmp = int64(int32(tmp))
		}
		if t.Typ == TypeREAL {
			tj.Data = append(tj.Data, float64(math.Float32frombits(uint32(tmp))))
		} else {
			tj.Data = append(tj.Data, float64(tmp))
		}
		if t.Typ != TypeREAL && t.Typ != TypeBOOL {
			if tmp < 256 && tmp >= 0 {
				tj.ASCII = append(tj.ASCII, asciiCode(rune(tmp)))
			} else {
				tj.ASCII = append(tj.ASCII, "")
			}
		}
	}
	tj.Typ = typeToString(t.Typ)

	b, err := json.Marshal(tj)
	if err != nil {
		fmt.Println(err)
		return "{}"
	}
	return string(b)
}

func bytesToBinString(bs []byte) string {
	var buf strings.Builder
	for _, b := range bs {
		buf.WriteString(fmt.Sprintf("%.8b ", b))
	}
	return buf.String()
}

func hexTr(ln int) string {
	var r strings.Builder
	r.WriteString("<pre style='font-family:\"Courier New\", Courier, monospace; margin: 0px;'>")
	for i := ln; i > 0; i-- {
		r.WriteString(fmt.Sprintf("%v       ", i*8))
	}
	r.WriteString("</pre>")
	return r.String()
}

func floatToString(f uint32) string {
	s := f >> 31
	e := (f & 0x7f800000) >> 23
	m := f & 0x007fffff
	return fmt.Sprintf("<td>%v</td><td>%v</td><td>%.8b</td><td>%.23b</td></tr>\n", math.Float32frombits(f), s, e, m)
}

func tagToHTML(t *Tag) string {
	var toSend strings.Builder

	ln := int(typeLen(uint16(t.Typ)))

	toSend.WriteString("<!DOCTYPE html>\n<html><h3>" + t.Name + "</h3>")
	toSend.WriteString("<table " + tableStyle + "><tr><th>N</th><th>" + typeToString(t.Typ) + "</th>")
	if t.Typ == TypeBOOL {
		toSend.WriteString("</tr>\n")
	} else if t.Typ == TypeREAL {
		toSend.WriteString("<th>SIGN</th><th>EXPONENT</th><th>MANTISSA</th></tr>\n")
	} else {
		toSend.WriteString("<th>HEX</th><th>ASCII</th><th>BIN</th></tr>\n")
		toSend.WriteString(fmt.Sprintf("<td></td><td></td><td></td><td></td><td>%s</td></tr>\n", hexTr(ln)))
	}

	n := 0
	for i := 0; i < len(t.data); i += ln {
		tmp := int64(t.data[i])
		for j := 1; j < ln; j++ {
			tmp += int64(t.data[i+j]) << uint(8*j)
		}
		hx := ""
		var err error
		buf := new(bytes.Buffer)

		switch t.Typ {
		case TypeBOOL:
			if tmp != 0 {
				tmp = 1
			}
		case TypeSINT:
			err = binary.Write(buf, binary.BigEndian, int8(tmp))
			tmp = int64(int8(tmp))
		case TypeINT:
			err = binary.Write(buf, binary.BigEndian, int16(tmp))
			tmp = int64(int16(tmp))
		case TypeDINT:
			err = binary.Write(buf, binary.BigEndian, int32(tmp))
			tmp = int64(int32(tmp))
		case TypeDWORD:
			err = binary.Write(buf, binary.BigEndian, int32(tmp))
			tmp = int64(int32(tmp))
		case TypeLINT:
			err = binary.Write(buf, binary.BigEndian, tmp)
		}
		if t.Typ != TypeREAL && t.Typ != TypeBOOL {
			ascii := ""
			if err == nil {
				hx = hex.EncodeToString(buf.Bytes())
			}
			bin := bytesToBinString(buf.Bytes())
			if tmp < 256 && tmp >= 0 {
				ascii = asciiCode(rune(tmp))
			}
			toSend.WriteString(fmt.Sprintf("<td>%d</td><td>%v</td><td>%v</td><td>%v</td><td>%v</td></tr>\n", n, tmp, hx, ascii, bin))
		} else if t.Typ == TypeBOOL {
			toSend.WriteString(fmt.Sprintf("<td>%d</td><td>%v</td></tr>\n", n, tmp))
		} else {
			toSend.WriteString(fmt.Sprintf("<td>%d</td>%s</tr>\n", n, floatToString(uint32(tmp))))
		}
		n++
	}

	toSend.WriteString("</table></html>")

	return toSend.String()
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		tagsIndexHTML(w, r)
	} else {
		tMut.RLock()
		t, ok := tags[path.Base(r.URL.Path)]
		if ok {
			_, json := r.URL.Query()["json"]
			if json {
				str := tagToJSON(t)
				tMut.RUnlock()
				w.Header().Set("Cache-Control", "no-store")
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				io.WriteString(w, str)
			} else {
				str := tagToHTML(t)
				tMut.RUnlock()
				w.Header().Set("Cache-Control", "no-store")
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				io.WriteString(w, str)
			}
		} else {
			tMut.RUnlock()
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, "not found")
		}
	}
}

var server *http.Server

// ServeHTTP listens on the TCP network address host.
func ServeHTTP(host string) *http.Server {
	server = &http.Server{Addr: host, Handler: http.HandlerFunc(handler)}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			fmt.Println("plcconnector ServeHTTP: ", err)
		}
	}()
	return server
}

// CloseHTTP shutdowns the HTTP server
func CloseHTTP() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var err error
	if err = server.Shutdown(ctx); err != nil {
		fmt.Println("plcconnector CloseHTTP: ", err)
	}
	debug("server.Shutdown")
	return err
}
