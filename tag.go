package plcconnector

import (
	"encoding/binary"
	"math"
	"reflect"
)

// Tag .
type Tag struct {
	Name  string
	Type  int
	Index int
	Count int

	data []uint8
	td   []Tag
	tn   string
}

// Len .
func (t Tag) Len() int {
	return int(typeLen(uint16(t.Type)))
}

// TagBOOL .
func TagBOOL(v bool, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeBOOL
	if v {
		a.data = []byte{0xFF}
	} else {
		a.data = []byte{0}
	}
	return &a
}

// TagArrayBool .
func TagArrayBool(v []bool, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeBOOL
	a.data = make([]byte, c)
	for i, x := range v {
		if x {
			a.data[i] = 0xFF
		}
	}
	return &a
}

// TagSINT .
func TagSINT(v int8, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeSINT
	a.data = []byte{uint8(v)}
	return &a
}

// TagArraySINT .
func TagArraySINT(v []int8, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeSINT
	a.data = make([]byte, c)
	for i := 0; i < c; i++ {
		a.data[i] = uint8(v[i])
	}
	return &a
}

// TagUSINT .
func TagUSINT(v uint8, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeUSINT
	a.data = []byte{v}
	return &a
}

// TagArrayUSINT .
func TagArrayUSINT(v []uint8, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeUSINT
	a.data = make([]byte, c)
	for i := 0; i < c; i++ {
		a.data[i] = v[i]
	}
	return &a
}

// TagINT .
func TagINT(v int16, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeINT
	a.data = make([]byte, 2)
	binary.LittleEndian.PutUint16(a.data, uint16(v))
	return &a
}

// TagArrayINT .
func TagArrayINT(v []int16, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeINT
	a.data = make([]byte, 2*c)
	for i := 0; i < c; i++ {
		binary.LittleEndian.PutUint16(a.data[2*i:], uint16(v[i]))
	}
	return &a
}

// TagUINT .
func TagUINT(v uint16, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeUINT
	a.data = make([]byte, 2)
	binary.LittleEndian.PutUint16(a.data, v)
	return &a
}

// TagArrayUINT .
func TagArrayUINT(v []uint16, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeUINT
	a.data = make([]byte, 2*c)
	for i := 0; i < c; i++ {
		binary.LittleEndian.PutUint16(a.data[2*i:], v[i])
	}
	return &a
}

// TagDINT .
func TagDINT(v int32, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeDINT
	a.data = make([]byte, 4)
	binary.LittleEndian.PutUint32(a.data, uint32(v))
	return &a
}

// TagArrayDINT .
func TagArrayDINT(v []int32, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeDINT
	a.data = make([]byte, 4*c)
	for i := 0; i < c; i++ {
		binary.LittleEndian.PutUint32(a.data[4*i:], uint32(v[i]))
	}
	return &a
}

// TagUDINT .
func TagUDINT(v uint32, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeUDINT
	a.data = make([]byte, 4)
	binary.LittleEndian.PutUint32(a.data, v)
	return &a
}

// TagArrayUDINT .
func TagArrayUDINT(v []uint32, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeUDINT
	a.data = make([]byte, 4*c)
	for i := 0; i < c; i++ {
		binary.LittleEndian.PutUint32(a.data[4*i:], v[i])
	}
	return &a
}

// TagLINT .
func TagLINT(v int64, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeLINT
	a.data = make([]byte, 8)
	binary.LittleEndian.PutUint64(a.data, uint64(v))
	return &a
}

// TagArrayLINT .
func TagArrayLINT(v []int64, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeLINT
	a.data = make([]byte, 8*c)
	for i := 0; i < c; i++ {
		binary.LittleEndian.PutUint64(a.data[8*i:], uint64(v[i]))
	}
	return &a
}

// TagULINT .
func TagULINT(v uint64, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeULINT
	a.data = make([]byte, 8)
	binary.LittleEndian.PutUint64(a.data, v)
	return &a
}

// TagArrayULINT .
func TagArrayULINT(v []uint64, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeULINT
	a.data = make([]byte, 8*c)
	for i := 0; i < c; i++ {
		binary.LittleEndian.PutUint64(a.data[8*i:], v[i])
	}
	return &a
}

// TagREAL .
func TagREAL(v float32, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeREAL
	a.data = make([]byte, 4)
	binary.LittleEndian.PutUint32(a.data, math.Float32bits(v))
	return &a
}

// TagArrayREAL .
func TagArrayREAL(v []float32, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeREAL
	a.data = make([]byte, 4*c)
	for i := 0; i < c; i++ {
		binary.LittleEndian.PutUint32(a.data[4*i:], math.Float32bits(v[i]))
	}
	return &a
}

// TagLREAL .
func TagLREAL(v float64, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeLREAL
	a.data = make([]byte, 8)
	binary.LittleEndian.PutUint64(a.data, math.Float64bits(v))
	return &a
}

// TagArrayLREAL .
func TagArrayLREAL(v []float64, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeLREAL
	a.data = make([]byte, 8*c)
	for i := 0; i < c; i++ {
		binary.LittleEndian.PutUint64(a.data[8*i:], math.Float64bits(v[i]))
	}
	return &a
}

// TagString .
func TagString(v string, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeSTRING
	a.data = []byte{byte(len(v)), byte(len(v) >> 8)}
	a.data = append(a.data, []byte(v)...)
	return &a
}

// TagShortString .
func TagShortString(v string, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeSHORTSTRING
	a.data = []byte{byte(len(v))}
	a.data = append(a.data, []byte(v)...)
	return &a
}

// TagStringI . TODO len>255
func TagStringI(v string, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeSTRINGI
	a.data = []byte{1, 'e', 'n', 'g', 0xDA, 4, 0, byte(len(v))}
	a.data = append(a.data, []byte(v)...)
	return &a
}

// NewTag .
func (p *PLC) NewTag(i interface{}, n string) {
	var a *Tag
	r := reflect.TypeOf(i)
	v := reflect.ValueOf(i)
	switch r.Kind() {
	case reflect.Bool:
		a = TagBOOL(v.Bool(), n)
	case reflect.Int8:
		a = TagSINT(int8(v.Int()), n)
	case reflect.Int16:
		a = TagINT(int16(v.Int()), n)
	case reflect.Int32:
		a = TagDINT(int32(v.Int()), n)
	case reflect.Int64:
		a = TagLINT(v.Int(), n)
	case reflect.Uint8:
		a = TagUSINT(uint8(v.Uint()), n)
	case reflect.Uint16:
		a = TagUINT(uint16(v.Uint()), n)
	case reflect.Uint32:
		a = TagUDINT(uint32(v.Int()), n)
	case reflect.Uint64:
		a = TagULINT(v.Uint(), n)
	case reflect.Float32:
		a = TagREAL(float32(v.Float()), n)
	case reflect.Float64:
		a = TagLREAL(v.Float(), n)
	case reflect.Array:
		e := r.Elem()
		l := r.Len()
		switch e.Kind() {
		case reflect.Bool:
			bytes := make([]bool, l)
			for i := range bytes {
				bytes[i] = v.Index(i).Bool()
			}
			a = TagArrayBool(bytes, l, n)
		case reflect.Int8:
			bytes := make([]int8, l)
			for i := range bytes {
				bytes[i] = int8(v.Index(i).Int())
			}
			a = TagArraySINT(bytes, l, n)
		case reflect.Int16:
			bytes := make([]int16, l)
			for i := range bytes {
				bytes[i] = int16(v.Index(i).Int())
			}
			a = TagArrayINT(bytes, l, n)
		case reflect.Int32:
			bytes := make([]int32, l)
			for i := range bytes {
				bytes[i] = int32(v.Index(i).Int())
			}
			a = TagArrayDINT(bytes, l, n)
		case reflect.Int64:
			bytes := make([]int64, l)
			for i := range bytes {
				bytes[i] = v.Index(i).Int()
			}
			a = TagArrayLINT(bytes, l, n)
		case reflect.Uint8:
			bytes := make([]uint8, l)
			for i := range bytes {
				bytes[i] = uint8(v.Index(i).Uint())
			}
			a = TagArrayUSINT(bytes, l, n)
		case reflect.Uint16:
			bytes := make([]uint16, l)
			for i := range bytes {
				bytes[i] = uint16(v.Index(i).Uint())
			}
			a = TagArrayUINT(bytes, l, n)
		case reflect.Uint32:
			bytes := make([]uint32, l)
			for i := range bytes {
				bytes[i] = uint32(v.Index(i).Uint())
			}
			a = TagArrayUDINT(bytes, l, n)
		case reflect.Uint64:
			bytes := make([]uint64, l)
			for i := range bytes {
				bytes[i] = v.Index(i).Uint()
			}
			a = TagArrayULINT(bytes, l, n)
		case reflect.Float32:
			bytes := make([]float32, l)
			for i := range bytes {
				bytes[i] = float32(v.Index(i).Float())
			}
			a = TagArrayREAL(bytes, l, n)
		case reflect.Float64:
			bytes := make([]float64, l)
			for i := range bytes {
				bytes[i] = v.Index(i).Float()
			}
			a = TagArrayLREAL(bytes, l, n)
		case reflect.Struct:
			a = new(Tag)
			a.Name = n
			a.Count = l
			a.Type = TypeStruct + 1 // FIXME type id
			fs := e.NumField()
			a.tn = e.Name()
			a.td = make([]Tag, fs)
			for i := 0; i < fs; i++ {
				a.td[i].Name = e.Field(i).Name
				e2 := e.Field(i).Type
				a.td[i].Count = 1
				switch e2.Kind() {
				case reflect.Uint16:
					a.td[i].Type = TypeUINT
				case reflect.Float32:
					a.td[i].Type = TypeREAL
				default:
					panic("unsupported struct type " + e2.String())
				}
			}
		default:
			panic("unsupported embedded type " + e.String())
		}
	case reflect.Slice:
		panic("use array not slice")
	case reflect.String:
		a = TagString(v.String(), n)
	case reflect.Struct:
		a = new(Tag)
		a.Name = n
		a.Count = 1
		a.Type = TypeStruct + 1 // FIXME type id
		fs := v.NumField()
		a.tn = r.Name()
		a.td = make([]Tag, fs)
		for i := 0; i < fs; i++ {
			a.td[i].Name = r.Field(i).Name
			e := r.Field(i).Type
			a.td[i].Count = 1
			switch e.Kind() {
			case reflect.Uint16:
				a.td[i].Type = TypeUINT
			case reflect.Float32:
				a.td[i].Type = TypeREAL
			default:
				panic("unsupported struct type " + e.String())
			}
		}
	default:
		panic("unknown type " + r.String())
	}
	p.AddTag(*a)
}

// DataBytes returns array of bytes.
func (t *Tag) DataBytes() []byte {
	return t.data
}

// DataBOOL returns array of BOOL.
func (t *Tag) DataBOOL() []bool {
	ret := make([]bool, 0, t.Count)
	for i := 0; i < len(t.data); i++ {
		tmp := false
		if t.data[i] != 0 {
			tmp = true
		}
		ret = append(ret, tmp)
	}
	return ret
}

// DataSINT returns array of int8.
func (t *Tag) DataSINT() []int8 {
	ret := make([]int8, 0, t.Count)
	for i := 0; i < len(t.data); i++ {
		ret = append(ret, int8(t.data[i]))
	}
	return ret
}

// DataINT returns array of int16.
func (t *Tag) DataINT() []int16 {
	ret := make([]int16, 0, t.Count)
	for i := 0; i < len(t.data); i += 2 {
		tmp := int16(t.data[i])
		tmp += int16(t.data[i+1]) << 8
		ret = append(ret, tmp)
	}
	return ret
}

// DataDINT returns array of int32.
func (t *Tag) DataDINT() []int32 {
	ret := make([]int32, 0, t.Count)
	for i := 0; i < len(t.data); i += 4 {
		tmp := int32(t.data[i])
		tmp += int32(t.data[i+1]) << 8
		tmp += int32(t.data[i+2]) << 16
		tmp += int32(t.data[i+3]) << 24
		ret = append(ret, tmp)
	}
	return ret
}

// DataREAL returns array of float32.
func (t *Tag) DataREAL() []float32 {
	ret := make([]float32, 0, t.Count)
	for i := 0; i < len(t.data); i += 4 {
		tmp := uint32(t.data[i])
		tmp += uint32(t.data[i+1]) << 8
		tmp += uint32(t.data[i+2]) << 16
		tmp += uint32(t.data[i+3]) << 24
		ret = append(ret, math.Float32frombits(tmp))
	}
	return ret
}

// DataDWORD returns array of int32.
func (t *Tag) DataDWORD() []int32 {
	return t.DataDINT()
}

// DataLINT returns array of int64.
func (t *Tag) DataLINT() []int64 {
	ret := make([]int64, 0, t.Count)
	for i := 0; i < len(t.data); i += 8 {
		tmp := int64(t.data[i])
		tmp += int64(t.data[i+1]) << 8
		tmp += int64(t.data[i+2]) << 16
		tmp += int64(t.data[i+3]) << 24
		tmp += int64(t.data[i+4]) << 32
		tmp += int64(t.data[i+5]) << 40
		tmp += int64(t.data[i+6]) << 48
		tmp += int64(t.data[i+7]) << 56
		ret = append(ret, tmp)
	}
	return ret
}
