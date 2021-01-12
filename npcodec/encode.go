package npcodec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"time"

	np "ulambda/ninep"
)

// Adopted from https://github.com/docker/go-p9p/encoding.go and Go's codecs

func Unmarshal(data []byte, v interface{}) error {
	dec := &decoder{bytes.NewReader(data)}
	return dec.decode(v)
}

func Marshal(v interface{}) ([]byte, error) {
	var b bytes.Buffer
	enc := &encoder{&b}
	if err := enc.encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

type encoder struct {
	wr io.Writer
}

func (e *encoder) encode(vs ...interface{}) error {
	for _, v := range vs {
		switch v := v.(type) {
		case uint8, uint16, uint32, uint64, np.Tfcall, np.Ttag, np.Tfid, np.Tmode, np.Qtype, np.Tsize, np.Tpath, np.TQversion, np.Tperm, np.Tiounit, np.Toffset, np.Tlength, np.Tgid,
			*uint8, *uint16, *uint32, *uint64, *np.Tfcall, *np.Ttag, *np.Tfid, *np.Tmode, *np.Qtype, *np.Tsize, *np.Tpath, *np.TQversion, *np.Tperm, *np.Tiounit, *np.Toffset, *np.Tlength, *np.Tgid:
			if err := binary.Write(e.wr, binary.LittleEndian, v); err != nil {
				return err
			}
		case []byte:
			if err := e.encode(uint32(len(v))); err != nil {
				return err
			}

			if err := binary.Write(e.wr, binary.LittleEndian, v); err != nil {
				return err
			}

		case *[]byte:
			if err := e.encode(*v); err != nil {
				return err
			}
		case string:
			if err := binary.Write(e.wr, binary.LittleEndian, uint16(len(v))); err != nil {
				return err
			}

			_, err := io.WriteString(e.wr, v)
			if err != nil {
				return err
			}
		case *string:
			if err := e.encode(*v); err != nil {
				return err
			}

		case []string:
			if err := e.encode(uint16(len(v))); err != nil {
				return err
			}

			for _, m := range v {
				if err := e.encode(m); err != nil {
					return err
				}
			}
		case *[]string:
			if err := e.encode(*v); err != nil {
				return err
			}
		case time.Time:
			if err := e.encode(uint32(v.Unix())); err != nil {
				return err
			}
		case *time.Time:
			if err := e.encode(*v); err != nil {
				return err
			}
		case np.Tqid:
			if err := e.encode(v.Type, v.Version, v.Path); err != nil {
				return err
			}
		case *np.Tqid:
			if err := e.encode(*v); err != nil {
				return err
			}
		case []np.Tqid:
			if err := e.encode(uint16(len(v))); err != nil {
				return err
			}

			elements := make([]interface{}, len(v))
			for i := range v {
				elements[i] = &v[i]
			}

			if err := e.encode(elements...); err != nil {
				return err
			}
		case *[]np.Tqid:
			if err := e.encode(*v); err != nil {
				return err
			}
		case np.Stat:
			elements, err := fields9p(v)
			if err != nil {
				return err
			}
			sz := uint16(SizeNp(elements...)) // Stat sz
			if err := e.encode(sz); err != nil {
				return err
			}

			if err := e.encode(elements...); err != nil {
				return err
			}
		case *np.Stat:
			if err := e.encode(*v); err != nil {
				return err
			}
		case []np.Stat:
			elements := make([]interface{}, len(v))
			for i := range v {
				elements[i] = &v[i]
			}

			if err := e.encode(elements...); err != nil {
				return err
			}
		case *[]np.Stat:
			if err := e.encode(*v); err != nil {
				return err
			}
		case np.Fcall:
			if err := e.encode(v.Type, v.Tag, v.Msg); err != nil {
				return err
			}
		case *np.Fcall:
			if err := e.encode(*v); err != nil {
				return err
			}
		case np.Tmsg:
			elements, err := fields9p(v)
			if err != nil {
				return err
			}
			if err := e.encode(elements...); err != nil {
				return err
			}
		default:
			log.Fatal("Unknown type")
		}
	}

	return nil
}

type decoder struct {
	rd io.Reader
}

func (d *decoder) decode(vs ...interface{}) error {
	for _, v := range vs {
		switch v := v.(type) {
		case *uint8, *uint16, *uint32, *uint64, *np.Tfcall, *np.Ttag, *np.Tfid, *np.Tmode, *np.Qtype, *np.Tsize, *np.Tpath, *np.TQversion, *np.Tperm, *np.Tiounit, *np.Toffset, *np.Tlength, *np.Tgid:
			if err := binary.Read(d.rd, binary.LittleEndian, v); err != nil {
				return err
			}
		case *[]byte:
			var l uint32

			if err := d.decode(&l); err != nil {
				return err
			}

			if l > 0 {
				*v = make([]byte, int(l))
			}

			if err := binary.Read(d.rd, binary.LittleEndian, v); err != nil {
				return err
			}
		case *string:
			var l uint16

			// implement string[s] encoding
			if err := d.decode(&l); err != nil {
				return err
			}

			b := make([]byte, l)

			n, err := io.ReadFull(d.rd, b)
			if err != nil {
				return err
			}

			if n != int(l) {
				return fmt.Errorf("unexpected string length")
			}
			*v = string(b)
		case *[]string:
			var l uint16

			if err := d.decode(&l); err != nil {
				return err
			}

			elements := make([]interface{}, int(l))
			*v = make([]string, int(l))
			for i := range elements {
				elements[i] = &(*v)[i]
			}

			if err := d.decode(elements...); err != nil {
				return err
			}
		case *time.Time:
			var epoch uint32
			if err := d.decode(&epoch); err != nil {
				return err
			}

			*v = time.Unix(int64(epoch), 0).UTC()
		case *np.Tqid:
			if err := d.decode(&v.Type, &v.Version, &v.Path); err != nil {
				return err
			}
		case *[]np.Tqid:
			var l uint16

			if err := d.decode(&l); err != nil {
				return err
			}

			elements := make([]interface{}, int(l))
			*v = make([]np.Tqid, int(l))
			for i := range elements {
				elements[i] = &(*v)[i]
			}

			if err := d.decode(elements...); err != nil {
				return err
			}
		case *np.Stat:
			var l uint16

			if err := d.decode(&l); err != nil {
				return err
			}

			b := make([]byte, l)
			if _, err := io.ReadFull(d.rd, b); err != nil {
				return err
			}

			elements, err := fields9p(v)
			if err != nil {
				return err
			}

			dec := &decoder{bytes.NewReader(b)}

			if err := dec.decode(elements...); err != nil {
				return err
			}
		case *np.Fcall:
			if err := d.decode(&v.Type, &v.Tag); err != nil {
				return err
			}
			msg, err := newMsg(v.Type)
			if err != nil {
				return err
			}

			// allocate msg
			rv := reflect.New(reflect.TypeOf(msg))
			if err := d.decode(rv.Interface()); err != nil {
				return err
			}

			v.Msg = rv.Elem().Interface().(np.Tmsg)

		case np.Tmsg:
			elements, err := fields9p(v)
			if err != nil {
				return err
			}

			if err := d.decode(elements...); err != nil {
				return err
			}
		default:
			log.Fatal("Decode: unknown type")
		}
	}

	return nil
}

// SizeNp calculates the projected size of the values in vs when encoded into
// 9p binary protocol. If an element or elements are not valid for 9p encoded,
// the value 0 will be used for the size. The error will be detected when
// encoding.
func SizeNp(vs ...interface{}) uint32 {
	var s uint32
	for _, v := range vs {
		if v == nil {
			continue
		}

		switch v := v.(type) {
		case uint8, uint16, uint32, uint64, np.Tfcall, np.Ttag, np.Tfid, np.Tmode, np.Qtype, np.Tsize, np.Tpath, np.TQversion, np.Tperm, np.Tiounit, np.Toffset, np.Tlength, np.Tgid,
			*uint8, *uint16, *uint32, *uint64, *np.Tfcall, *np.Ttag, *np.Tfid, *np.Tmode, *np.Qtype, *np.Tsize, *np.Tpath, *np.TQversion, *np.Tperm, *np.Tiounit, *np.Toffset, *np.Tlength, *np.Tgid:
			s += uint32(binary.Size(v))
		case []byte:
			s += uint32(binary.Size(uint32(0)) + len(v))
		case *[]byte:
			s += SizeNp(uint32(0), *v)
		case string:
			s += uint32(binary.Size(uint16(0)) + len(v))
		case *string:
			s += SizeNp(*v)
		case []string:
			s += SizeNp(uint16(0))

			for _, sv := range v {
				s += SizeNp(sv)
			}
		case *[]string:
			s += SizeNp(*v)
		case np.Tqid:
			s += SizeNp(v.Type, v.Version, v.Path)
		case *np.Tqid:
			s += SizeNp(*v)
		case []np.Tqid:
			s += SizeNp(uint16(0))
			elements := make([]interface{}, len(v))
			for i := range elements {
				elements[i] = &v[i]
			}
			s += SizeNp(elements...)
		case *[]np.Tqid:
			s += SizeNp(*v)
		case np.Stat:
			elements, err := fields9p(v)
			if err != nil {
				log.Fatal("Stat ", err)
			}
			s += SizeNp(elements...) + SizeNp(uint16(0))
		case *np.Stat:
			s += SizeNp(*v)
		case []np.Stat:
			elements := make([]interface{}, len(v))
			for i := range elements {
				elements[i] = &v[i]
			}
			s += SizeNp(elements...)
		case *[]np.Stat:
			s += SizeNp(*v)
		case np.Fcall:
			s += SizeNp(v.Type, v.Tag, v.Msg)
		case *np.Fcall:
			s += SizeNp(*v)
		case np.Tmsg:
			// walk the fields of the message to get the total size. we just
			// use the field order from the message struct. We may add tag
			// ignoring if needed.
			elements, err := fields9p(v)
			if err != nil {
				log.Fatal("Tmsg ", err)
			}

			s += SizeNp(elements...)
		default:
			log.Fatal("Unknown type")
		}
	}

	return s
}

// fields9p lists the settable fields from a struct type for reading and
// writing. We are using a lot of reflection here for fairly static
// serialization but we can replace this in the future with generated code if
// performance is an issue.
func fields9p(v interface{}) ([]interface{}, error) {
	rv := reflect.Indirect(reflect.ValueOf(v))

	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("cannot extract fields from non-struct: %v", rv)
	}

	var elements []interface{}
	for i := 0; i < rv.NumField(); i++ {
		f := rv.Field(i)

		if !f.CanInterface() {
			// unexported field, skip it.
			continue
		}

		if f.CanAddr() {
			f = f.Addr()
		}

		elements = append(elements, f.Interface())
	}

	return elements, nil
}

func string9p(v interface{}) string {
	if v == nil {
		return "nil"
	}

	rv := reflect.Indirect(reflect.ValueOf(v))

	if rv.Kind() != reflect.Struct {
		log.Fatal("not a struct")
	}

	var s string

	for i := 0; i < rv.NumField(); i++ {
		f := rv.Field(i)
		s += fmt.Sprintf(" %v=%v", strings.ToLower(rv.Type().Field(i).Name), f.Interface())
	}

	return s
}
