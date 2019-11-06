// Copyright 2018 Prosap sp. z o.o. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package plcconnector implements communication with PLC.
package plcconnector

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

// PLC .
type PLC struct {
	callback  func(service int, statut int, tag *Tag)
	closeI    bool
	closeMut  sync.RWMutex
	closeWMut sync.Mutex
	closeWait *sync.Cond
	eds       map[string]map[string]string
	port      uint16
	symbols   *Class
	tMut      sync.RWMutex
	tags      map[string]*Tag

	Class       map[int]*Class
	DumpNetwork bool // enables dumping network packets
	Verbose     bool // enables debugging output
	Timeout     time.Duration
}

// Init initialize library. Must be called first.
func Init(eds string, testTags bool) (*PLC, error) {
	var p PLC
	p.Class = make(map[int]*Class)
	p.tags = make(map[string]*Tag)
	p.Timeout = 60 * time.Second

	err := p.loadEDS(eds)
	if err != nil {
		return nil, err
	}

	if testTags {
		p.AddTag(Tag{Name: "testBOOL", Typ: TypeBOOL, Count: 4, data: []uint8{
			0x00, 0x01, 0xFF, 0x55}})

		p.AddTag(Tag{Name: "testSINT", Typ: TypeSINT, Count: 4, data: []uint8{
			0xFF, 0xFE, 0x00, 0x01}})

		p.AddTag(Tag{Name: "testINT", Typ: TypeINT, Count: 10, data: []uint8{
			0xFF, 0xFF, 0x00, 0x01, 0xFE, 0x00, 0xFC, 0x00, 0xCA, 0x00, 0xBD, 0x00, 0xB1, 0x00, 0xFF, 0x00, 127, 0x00, 128, 0x00}})

		p.AddTag(Tag{Name: "testDINT", Typ: TypeDINT, Count: 2, data: []uint8{
			0xFF, 0xFF, 0xFF, 0xFF,
			0x01, 0x00, 0x00, 0x00}})

		p.AddTag(Tag{Name: "testREAL", Typ: TypeREAL, Count: 2, data: []uint8{
			0xa4, 0x70, 0x9d, 0x3f,
			0xcd, 0xcc, 0x44, 0xc1}})

		p.AddTag(Tag{Name: "testDWORD", Typ: TypeDWORD, Count: 2, data: []uint8{
			0xFF, 0xFF, 0xFF, 0xFF,
			0x01, 0x00, 0x00, 0x00}})

		p.AddTag(Tag{Name: "testLINT", Typ: TypeLINT, Count: 2, data: []uint8{
			0xFF, 0xFF, 0xFF, 0xFF,
			0xFF, 0xFF, 0xFF, 0xFF,
			0x01, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00}})

		p.AddTag(Tag{Name: "testASCII", Typ: TypeSINT, Count: 17, data: []uint8{
			'H', 'e', 'l', 'l',
			'o', '!', 0x00, 0x01, 0x7F, 0xFE, 0xFC, 0xCA, 0xBD, 0xB1, 0xFF, 127, 128}})
	}

	return &p, nil
}

func (p *PLC) debug(args ...interface{}) {
	if p.Verbose {
		fmt.Println(args...)
	}
}

func (p *PLC) readTag(tag string, index int, count uint16) ([]uint8, uint16, bool) {
	p.tMut.RLock()
	tg, ok := p.tags[tag]
	var (
		tgtyp  uint16
		tgdata []uint8
	)
	if ok {
		tgtyp = uint16(tg.Typ)
		tl := typeLen(tgtyp)
		tgdata = make([]uint8, count*tl)
		if index+int(count) > tg.Count {
			ok = false
		} else {
			copy(tgdata, tg.data[index*int(tl):])
		}
	}
	p.tMut.RUnlock()
	if ok {
		if p.callback != nil {
			go p.callback(ReadTag, Success, &Tag{Name: tag, Typ: int(tgtyp), Index: index, Count: int(count), data: tgdata})
		}
		return tgdata, tgtyp, true
	}
	if p.callback != nil {
		go p.callback(ReadTag, PathSegmentError, nil)
	}
	return nil, 0, false
}

func (p *PLC) saveTag(tag string, typ uint16, index int, count uint16, data []uint8) bool {
	p.tMut.Lock()
	tg, ok := p.tags[tag]
	if ok && tg.Typ == int(typ) && tg.Count >= index+int(count) {
		copy(tg.data[index*int(typeLen(typ)):], data)
	} else {
		p.tags[tag] = &Tag{Name: tag, Typ: int(typ), Count: int(count), data: data} // FIXME Symbols
	}
	p.tMut.Unlock()
	if p.callback != nil {
		go p.callback(WriteTag, Success, &Tag{Name: tag, Typ: int(typ), Index: index, Count: int(count), data: data})
	}
	return true
}

// AddTag adds tag.
func (p *PLC) AddTag(t Tag) {
	if t.data == nil {
		size := typeLen(uint16(t.Typ)) * uint16(t.Count)
		t.data = make([]uint8, size)
	}
	in := NewInstance(8)
	in.Attr[1] = AttrString(t.Name, "SymbolName")
	typ := uint16(t.Typ)
	if t.Count > 1 {
		typ |= 0x2000 // 1D Array
	}
	in.Attr[2] = AttrUINT(typ, "SymbolType")
	in.Attr[7] = AttrUINT(typeLen(uint16(t.Typ)), "BaseTypeSize")
	in.Attr[8] = &Attribute{Name: "Dimensions", data: []uint8{uint8(t.Count), uint8(t.Count >> 8), uint8(t.Count >> 16), uint8(t.Count >> 24), 0, 0, 0, 0, 0, 0, 0, 0}}
	p.tMut.Lock()
	p.symbols.SetInstance(p.symbols.lastInst+1, in)
	p.tags[t.Name] = &t
	p.tMut.Unlock()
}

// UpdateTag sets data to the tag
func (p *PLC) UpdateTag(name string, offset int, data []uint8) bool {
	p.tMut.Lock()
	defer p.tMut.Unlock()
	t, ok := p.tags[name]
	if !ok {
		fmt.Println("plcconnector UpdateTag: no tag named ", name)
		return false
	}
	offset *= int(typeLen(uint16(t.Typ)))
	to := offset + len(data)
	if to > len(t.data) {
		fmt.Println("plcconnector UpdateTag: to large data ", name)
		return false
	}
	for i := offset; i < to; i++ {
		t.data[i] = data[i-offset]
	}
	return true
}

// Callback registers function called at receiving communication with PLC.
// tag may be nil in event of error or reset.
func (p *PLC) Callback(function func(service int, status int, tag *Tag)) {
	p.callback = function
}

// Serve listens on the TCP network address host.
func (p *PLC) Serve(host string) error {
	rand.Seed(time.Now().UnixNano())

	p.closeMut.Lock()
	p.closeI = false
	p.closeMut.Unlock()

	p.closeWait = sync.NewCond(&p.closeWMut)

	sock := net.ListenConfig{}
	sock.Control = sockControl
	serv2, err := sock.Listen(context.Background(), "tcp", host)
	if err != nil {
		fmt.Println("plcconnector Serve: ", err)
		return err
	}
	p.port = getPort(host)
	serv := serv2.(*net.TCPListener)
	for {
		serv.SetDeadline(time.Now().Add(time.Second))
		conn, err := serv.AcceptTCP()
		if e, ok := err.(net.Error); ok && e.Timeout() {
			p.closeMut.RLock()
			endP := p.closeI
			p.closeMut.RUnlock()
			if endP {
				break
			}
		} else if err != nil {
			fmt.Println("plcconnector Serve: ", err)
			return err
		} else {
			go p.handleRequest(conn)
		}
	}
	serv.Close()
	p.debug("Serve shutdown")
	p.closeWait.Signal()
	return nil
}

// Close shutdowns server
func (p *PLC) Close() {
	p.closeMut.Lock()
	p.closeI = true
	p.closeMut.Unlock()
	p.closeWait.L.Lock()
	p.closeWait.Wait()
	p.closeWait.L.Unlock()
}

type req struct {
	c        net.Conn
	connID   uint32
	rrdata   sendData
	encHead  encapsulationHeader
	p        *PLC
	readBuf  *bufio.Reader
	writeBuf *bytes.Buffer
	wrCIPBuf *bytes.Buffer
}

func (r *req) read(data interface{}) error {
	err := binary.Read(r.readBuf, binary.LittleEndian, data)
	if err != nil {
		fmt.Println(err)
	}
	if r.p.DumpNetwork {
		fmt.Printf("%#v\n", data)
	}
	return err
}

func (r *req) write(data interface{}) {
	err := binary.Write(r.writeBuf, binary.LittleEndian, data)
	if err != nil {
		fmt.Println(err)
	}
}

func (r *req) writeCIP(data interface{}) {
	err := binary.Write(r.wrCIPBuf, binary.LittleEndian, data)
	if err != nil {
		fmt.Println(err)
	}
}

func (r *req) reset() {
	r.readBuf.Reset(r.c)
	r.writeBuf.Reset()
	r.wrCIPBuf.Reset()
}

func bwrite(buf *bytes.Buffer, data interface{}) {
	err := binary.Write(buf, binary.LittleEndian, data)
	if err != nil {
		fmt.Println(err)
	}
}

func (p *PLC) handleRequest(conn net.Conn) {
	r := req{}
	r.connID = uint32(0)
	r.c = conn
	r.p = p
	r.readBuf = bufio.NewReader(conn)
	r.writeBuf = new(bytes.Buffer)
	r.wrCIPBuf = new(bytes.Buffer)

loop:
	for {
		r.reset()

		p.closeMut.RLock()
		endP := p.closeI
		p.closeMut.RUnlock()
		if endP {
			break loop
		}

		timeout := time.Now().Add(p.Timeout)
		err := conn.SetReadDeadline(timeout)
		if err != nil {
			fmt.Println(err)
			break loop
		}

		p.debug()
		err = r.read(&r.encHead)
		if err != nil {
			break loop
		}

	command:
		switch r.encHead.Command {
		case ecNOP:
			if r.eipNOP() != nil {
				break loop
			}
			continue loop

		case ecRegisterSession:
			if r.eipRegisterSession() != nil {
				break loop
			}

		case ecUnRegisterSession:
			p.debug("UnregisterSession")
			break loop

		case ecListIdentity: // TODO: UDP
			if r.eipListIdentity() != nil {
				break loop
			}

		case ecListServices: // TODO: UDP
			if r.eipListServices() != nil {
				break loop
			}

		case ecListInterfaces: // TODO: UDP
			p.debug("ListInterfaces")
			r.write(uint16(0)) // ItemCount

		case ecSendRRData, ecSendUnitData:
			p.debug("SendRRData/SendUnitData")

			var (
				item         itemType
				protd        protocolData
				protSeqCount uint16
				resp         response
			)
			err = r.read(&r.rrdata)
			if err != nil {
				break loop
			}

			if r.rrdata.Timeout != 0 && r.encHead.Command == ecSendRRData {
				timeout = time.Now().Add(time.Duration(r.rrdata.Timeout) * time.Second)
				err = conn.SetReadDeadline(timeout)
				if err != nil {
					fmt.Println(err)
					break loop
				}
			}

			r.rrdata.Timeout = 0
			cidok := false
			mayCon := false
			itemserror := false

			if r.rrdata.ItemCount != 2 {
				p.debug("itemCount != 2")
				r.encHead.Status = eipIncorrectData
				break command
			}

			// address item
			err = r.read(&item)
			if err != nil {
				break loop
			}
			if item.Type == itConnAddress { // TODO itemdata to connID
				itemdata := make([]uint8, item.Length)
				err = r.read(&itemdata)
				if err != nil {
					break loop
				}
				cidok = true
			} else if item.Type != itNullAddress {
				p.debug("unkown address item:", item.Type)
				itemserror = true
				itemdata := make([]uint8, item.Length)
				err = r.read(&itemdata)
				if err != nil {
					break loop
				}
			}

			// data item
			err = r.read(&item)
			if err != nil {
				break loop
			}
			if item.Type == itConnData {
				err = r.read(&protSeqCount)
				if err != nil {
					break loop
				}
				cidok = true
			} else if item.Type != itUnconnData {
				p.debug("unkown data item:", item.Type)
				itemserror = true
				itemdata := make([]uint8, item.Length)
				err = r.read(&itemdata)
				if err != nil {
					break loop
				}
			}

			if itemserror {
				r.encHead.Status = eipIncorrectData
				break command
			}

			// CIP
			err = r.read(&protd)
			if err != nil {
				break loop
			}

			resp.Service = protd.Service + 128
			resp.Status = Success

			ePath := make([]uint8, protd.PathSize*2)
			err = r.read(&ePath)
			if err != nil {
				break loop
			}

			class, instance, attr, path, err := r.parsePath(ePath)
			p.debug("class", class, "instance", instance, "attr", attr, "path", path)
			// if err != nil {
			// 	resp.Status = PathSegmentError
			// 	resp.AddStatusSize = 1

			// 	r.write(resp)
			// 	r.write(uint16(0))
			// 	break command // FIXME
			// }

			switch protd.Service {
			case GetAttrAll:
				p.debug("GetAttributesAll")
				mayCon = true

				c, in := p.GetClassInstance(class, instance)
				if c != nil {
					p.debug(c.Name, instance)
					r.write(resp)
					r.write(in.getAttrAll())
				} else {
					p.debug("path unknown", path)
					resp.Status = PathUnknown
					r.write(resp)
				}

			case GetAttrList:
				p.debug("GetAttributesList")
				mayCon = true
				var (
					count uint16
					buf   bytes.Buffer
					st    uint16
				)

				err = r.read(&count)
				if err != nil {
					break loop
				}
				attr := make([]uint16, count)
				err = r.read(&attr)
				if err != nil {
					break loop
				}

				c, in := p.GetClassInstance(class, instance)
				if c != nil {
					ln := len(in.Attr)
					for _, i := range attr {
						bwrite(&buf, i)
						if int(i) < ln && in.Attr[i] != nil {
							p.debug(in.Attr[i].Name)
							st = Success
							bwrite(&buf, st)
							bwrite(&buf, in.Attr[i].data)
						} else {
							resp.Status = AttrListError
							st = AttrNotSup
							bwrite(&buf, st)
						}
					}

					r.write(resp)
					r.write(count)
					r.write(buf.Bytes())
				} else {
					p.debug("path unknown", path)
					resp.Status = PathUnknown
					r.write(resp)
				}

			case GetInstAttrList: // TODO status 6 too much data (504 unconnected)
				p.debug("GetInstanceAttributesList")
				mayCon = true
				var (
					count uint16
					buf   bytes.Buffer
				)

				err = r.read(&count)
				if err != nil {
					break loop
				}
				attr := make([]uint16, count)
				err = r.read(&attr)
				if err != nil {
					break loop
				}

				c, li := p.GetClassInstancesList(class, instance)
				if c != nil {
					for _, x := range li {
						in := c.Inst[x]
						ln := len(in.Attr)
						bwrite(&buf, uint32(x))
						for _, i := range attr {
							if int(i) < ln && in.Attr[i] != nil {
								bwrite(&buf, in.Attr[i].data)
							} else { // FIXME break
								resp.Status = AttrListError
							}
						}
					}

					r.write(resp)
					r.write(buf.Bytes())
				} else {
					p.debug("path unknown", path)
					resp.Status = PathUnknown
					r.write(resp)
				}

			case GetAttr:
				p.debug("GetAttributesSingle")
				mayCon = true

				var (
					aok bool
					at  *Attribute
				)
				c, in := p.GetClassInstance(class, instance)
				if c != nil && attr < len(in.Attr) {
					at = in.Attr[attr]
					if at != nil {
						aok = true
					}
				}
				resp.Service = protd.Service + 128

				if c != nil && aok {
					p.debug(c.Name, instance, at.Name)
					r.write(resp)
					r.write(at.data)
				} else {
					p.debug("path unknown", path)
					resp.Status = PathUnknown
					r.write(resp)
				}

			case InititateUpload: // TODO only File class?
				p.debug("InititateUpload")
				mayCon = true
				var maxSize uint8

				err = r.read(&maxSize)
				if err != nil {
					break loop
				}

				c, in := p.GetClassInstance(class, instance)
				if c != nil {
					p.debug(c.Name, instance, maxSize)

					var sr initUploadResponse
					sr.FileSize = uint32(len(in.data))
					sr.TransferSize = maxSize
					in.argUint8[0] = maxSize // TransferSize
					in.argUint8[1] = 0       // TransferNumber
					in.argUint8[2] = 0       // TransferNumber rollover

					r.write(resp)
					r.write(sr)
				} else {
					p.debug("path unknown", path)
					resp.Status = PathUnknown
					r.write(resp)
				}

			case UploadTransfer: // TODO only File class?
				p.debug("UploadTransfer")
				mayCon = true
				var transferNo uint8

				err = r.read(&transferNo)
				if err != nil {
					break loop
				}

				c, in := p.GetClassInstance(class, instance)
				if c != nil {
					if transferNo == in.argUint8[1] || transferNo == in.argUint8[1]+1 || (transferNo == 0 && in.argUint8[1] == 255) {
						p.debug(c.Name, instance, transferNo)

						if transferNo == 0 && in.argUint8[1] == 255 { // rollover
							p.debug("rollover")
							in.argUint8[2]++ // FIXME retry!
						}

						var sr uploadTransferResponse
						addcksum := false
						dtlen := len(in.data)
						pos := (int(in.argUint8[2]) + 1) * int(transferNo) * int(in.argUint8[0])
						posto := pos + int(in.argUint8[0])
						if posto > dtlen {
							posto = dtlen
						}
						dt := in.data[pos:posto]
						sr.TransferNumber = transferNo
						if transferNo == 0 && dtlen <= int(in.argUint8[0]) {
							sr.TranferPacketType = tptFirstLast
							addcksum = true
						} else if transferNo == 0 && in.argUint8[2] == 0 {
							sr.TranferPacketType = tptFirst
						} else if pos+int(in.argUint8[0]) >= dtlen {
							sr.TranferPacketType = tptLast
							addcksum = true
						} else {
							sr.TranferPacketType = tptMiddle
						}
						in.argUint8[1] = transferNo

						ln := uint16(binary.Size(resp) + binary.Size(sr) + len(dt))
						if addcksum {
							ln += uint16(binary.Size(in.Attr[7].data))
						}

						p.debug(sr)
						p.debug(len(dt), pos, posto)

						r.write(resp)
						r.write(sr)
						r.write(dt)
						if addcksum {
							r.write(in.Attr[7].data)
						}
					} else {
						p.debug("transfer number error", transferNo)

						resp.Status = InvalidPar
						resp.AddStatusSize = 1

						r.write(resp)
						r.write(uint16(0))
					}
				} else {
					p.debug("path unknown", path)

					resp.Status = PathUnknown
					r.write(resp)
				}

			case ForwardOpen:
				p.debug("ForwardOpen")

				var (
					fodata forwardOpenData
					sr     forwardOpenResponse
				)
				err = r.read(&fodata)
				if err != nil {
					break loop
				}
				connPath := make([]uint8, fodata.ConnPathSize*2)
				err = r.read(&connPath)
				if err != nil {
					break loop
				}

				sr.OTConnectionID = rand.Uint32()
				sr.TOConnectionID = fodata.TOConnectionID
				sr.ConnSerialNumber = fodata.ConnSerialNumber
				sr.VendorID = fodata.VendorID
				sr.OriginatorSerialNumber = fodata.OriginatorSerialNumber
				sr.OTAPI = fodata.OTRPI
				sr.TOAPI = fodata.TORPI
				sr.AppReplySize = 0

				r.connID = fodata.TOConnectionID

				r.write(resp)
				r.write(sr)

			case ForwardClose:
				p.debug("ForwardClose")

				var (
					fcdata forwardCloseData
					sr     forwardCloseResponse
				)
				err = r.read(&fcdata)
				if err != nil {
					break loop
				}
				connPath := make([]uint8, fcdata.ConnPathSize*2)
				err = r.read(&connPath)
				if err != nil {
					break loop
				}

				sr.ConnSerialNumber = fcdata.ConnSerialNumber
				sr.VendorID = fcdata.VendorID
				sr.OriginatorSerialNumber = fcdata.OriginatorSerialNumber
				sr.AppReplySize = 0

				r.connID = 0

				r.write(resp)
				r.write(sr)

			case ReadTag:
				p.debug("ReadTag")
				mayCon = true

				var (
					tagName  string
					tagIndex int
					tagCount uint16
				)

				if len(path) > 0 && path[0].typ == ansiExtended {
					tagName = path[0].txt
					if len(path) > 1 && path[1].typ == pathElement {
						tagIndex = path[1].val
					}
				}
				err = r.read(&tagCount)
				if err != nil {
					break loop
				}
				p.debug(tagName, tagIndex, tagCount)

				if rtData, tagType, ok := p.readTag(tagName, tagIndex, tagCount); ok {
					r.write(resp)
					r.write(tagType)
					r.write(rtData)
				} else {
					resp.Status = PathSegmentError
					resp.AddStatusSize = 1

					r.write(resp)
					r.write(uint16(0))
				}

			case WriteTag:
				p.debug("WriteTag")
				mayCon = true

				var (
					tagName  string
					tagType  uint16
					tagIndex int
					tagCount uint16
				)

				if len(path) > 0 && path[0].typ == ansiExtended {
					tagName = path[0].txt
					if len(path) > 1 && path[1].typ == pathElement {
						tagIndex = path[1].val
					}
				}
				err = r.read(&tagType)
				if err != nil {
					break loop
				}
				err = r.read(&tagCount)
				if err != nil {
					break loop
				}
				p.debug(tagName, tagType, tagIndex, tagCount)

				wrData := make([]uint8, typeLen(tagType)*tagCount)
				err = r.read(wrData)
				if err != nil {
					break loop
				}

				if p.saveTag(tagName, tagType, tagIndex, tagCount, wrData) {
					r.write(resp)
				} else {
					resp.Status = PathSegmentError
					resp.AddStatusSize = 1

					r.write(resp)
					r.write(uint16(0))
				}

			case Reset:
				p.debug("Reset")

				if p.callback != nil {
					go p.callback(Reset, Success, nil)
				}
				r.write(resp)

			default:
				p.debug("unknown service:", protd.Service)

				resp.Status = ServNotSup
				r.write(resp)
			}

			r.writeCIP(r.rrdata)
			if mayCon && cidok && r.connID != 0 {
				r.writeCIP(itemType{Type: itConnAddress, Length: uint16(binary.Size(r.connID))})
				r.writeCIP(r.connID)
				r.writeCIP(itemType{Type: itConnData, Length: uint16(binary.Size(protSeqCount) + r.writeBuf.Len())})
				r.writeCIP(protSeqCount)
			} else {
				r.writeCIP(itemType{Type: itNullAddress, Length: 0})
				r.writeCIP(itemType{Type: itUnconnData, Length: uint16(r.writeBuf.Len())})
			}

		default:
			p.debug("unknown command:", r.encHead.Command)

			data := make([]uint8, r.encHead.Length)
			err = r.read(&data)
			if err != nil {
				break loop
			}
			r.encHead.Status = eipInvalid

			r.write(data)
		}

		err = conn.SetWriteDeadline(timeout)
		if err != nil {
			fmt.Println(err)
			break loop
		}

		r.encHead.Length = uint16(r.wrCIPBuf.Len() + r.writeBuf.Len())
		var buf bytes.Buffer

		err = binary.Write(&buf, binary.LittleEndian, r.encHead)
		if err != nil {
			fmt.Println(err)
			break loop
		}
		buf.Write(r.wrCIPBuf.Bytes())
		buf.Write(r.writeBuf.Bytes())

		_, err = conn.Write(buf.Bytes())
		if err != nil {
			fmt.Println(err)
			break loop
		}
	}
	err := conn.Close()
	if err != nil {
		fmt.Println(err)
	}
}
