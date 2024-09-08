package binaryencode

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// this will encode into n ignored/null bytes determined by the tag `binary:"i,n"`
type Padding interface{}

// a byte array that will be encoded as a null terminated string
type NullString []byte

type LengthString []byte   // a byte array that will be encoded as an 8 bit length prefixed string
type LengthString16 []byte // a byte array that will be encoded as a 16 bit length prefixed string
type LengthString32 []byte // a byte array that will be encoded as a 32 bit length prefixed string
type LengthString64 []byte // a byte array that will be encoded as a 64 bit length prefixed string. you probably dont want to use this because internally an int is used for length, this is just here for compatability

const (
	BigEndian    = true
	LittleEndian = false
)

var (
	ErrStringOverflow = errors.New("binary: string too large")
)

type EncoderArgs struct {
	DefaultEndianess bool
	MaxStringLength  int
}

type binaryField struct {
	v         reflect.Value
	index     uint64
	params    []string
	endianess bool
}

func sortFields(t reflect.Value, args EncoderArgs) []binaryField {
	fc := t.NumField()
	fields := make([]binaryField, 0, fc)
	for i := 0; i < fc; i++ {
		field := t.Type().Field(i)

		tag, ok := field.Tag.Lookup("binary")
		if !ok {
			continue
		}

		f := binaryField{
			v:         t.Field(i),
			endianess: args.DefaultEndianess,
		}

		f.params = strings.Split(tag, ",")

		var err error
		f.index, err = strconv.ParseUint(f.params[0], 10, 64)
		if err != nil {
			panic(err)
		}

		for _, p := range f.params {
			switch p {
			case "big":
				f.endianess = BigEndian
			case "little":
				f.endianess = LittleEndian
			}
		}

		fields = append(fields, f)
	}

	slices.SortFunc(fields, func(me binaryField, you binaryField) int {
		return int(me.index) - int(you.index)
	})

	return fields
}

func Marshal(in interface{}, args EncoderArgs) []byte {
	t := reflect.ValueOf(in)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	fields := sortFields(t, args)

	var b []byte
	for _, field := range fields {
		switch field.v.Type() {
		case reflect.TypeFor[byte](): // 8 bit
			b = append(b, field.v.Interface().(byte))
		case reflect.TypeFor[uint8]():
			b = append(b, field.v.Interface().(uint8))
		case reflect.TypeFor[int8]():
			b = append(b, byte(field.v.Interface().(int8)))
		case reflect.TypeFor[uint16](): // 16 bit
			v := field.v.Interface().(uint16)
			if field.endianess {
				b = binary.BigEndian.AppendUint16(b, v)
			} else {
				b = binary.LittleEndian.AppendUint16(b, v)
			}
		case reflect.TypeFor[int16]():
			v := uint16(field.v.Interface().(int16))
			if field.endianess {
				b = binary.BigEndian.AppendUint16(b, v)
			} else {
				b = binary.LittleEndian.AppendUint16(b, v)
			}
		case reflect.TypeFor[uint32](): // 32 bit
			v := field.v.Interface().(uint32)
			if field.endianess {
				b = binary.BigEndian.AppendUint32(b, v)
			} else {
				b = binary.LittleEndian.AppendUint32(b, v)
			}
		case reflect.TypeFor[int32]():
			v := uint32(field.v.Interface().(int32))
			if field.endianess {
				b = binary.BigEndian.AppendUint32(b, v)
			} else {
				b = binary.LittleEndian.AppendUint32(b, v)
			}
		case reflect.TypeFor[uint](), reflect.TypeFor[uintptr](), reflect.TypeFor[uint64](): // 64 bit (native ints should be treated as 64 bit)
			v := field.v.Uint()
			if field.endianess {
				b = binary.BigEndian.AppendUint64(b, v)
			} else {
				b = binary.LittleEndian.AppendUint64(b, v)
			}
		case reflect.TypeFor[int](), reflect.TypeFor[int64]():
			v := uint64(field.v.Int())
			if field.endianess {
				b = binary.BigEndian.AppendUint64(b, v)
			} else {
				b = binary.LittleEndian.AppendUint64(b, v)
			}
		case reflect.TypeFor[Padding](): // variable length
			count, err := strconv.ParseInt(field.params[1], 10, 64)
			if err != nil {
				panic(err)
			}
			b = append(b, bytes.Repeat([]byte{'\000'}, int(count))...)
		case reflect.TypeFor[NullString]():
			b = append(b, append(field.v.Interface().(NullString), '\000')...)
		case reflect.TypeFor[LengthString]():
			ls := field.v.Interface().(LengthString)
			b = append(b, append([]byte{byte(len(ls))}, ls...)...)
		case reflect.TypeFor[LengthString16]():
			ls := field.v.Interface().(LengthString16)
			if field.endianess {
				b = append(b, append(binary.BigEndian.AppendUint16(nil, uint16(len(ls))), ls...)...)
			} else {
				b = append(b, append(binary.LittleEndian.AppendUint16(nil, uint16(len(ls))), ls...)...)
			}
		case reflect.TypeFor[LengthString32]():
			ls := field.v.Interface().(LengthString32)
			if field.endianess {
				b = append(b, append(binary.BigEndian.AppendUint32(nil, uint32(len(ls))), ls...)...)
			} else {
				b = append(b, append(binary.LittleEndian.AppendUint32(nil, uint32(len(ls))), ls...)...)
			}
		case reflect.TypeFor[LengthString64]():
			ls := field.v.Interface().(LengthString64)
			if field.endianess {
				b = append(b, append(binary.BigEndian.AppendUint64(nil, uint64(len(ls))), ls...)...)
			} else {
				b = append(b, append(binary.LittleEndian.AppendUint64(nil, uint64(len(ls))), ls...)...)
			}
		case reflect.TypeFor[string]():
			b = append(b, []byte(field.v.Interface().(string))...)
		case reflect.TypeFor[[]byte]():
			b = append(b, field.v.Interface().([]byte)...)
		default:
			panic("cannot binary encode: unsupported type")
		}
	}
	return b
}

func Unmarshal(in io.Reader, out interface{}, args EncoderArgs) error {
	t := reflect.ValueOf(out)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	fields := sortFields(t, args)

	var rc int
	for _, field := range fields {
		if !field.v.CanSet() {
			continue
		}
		switch field.v.Type() {
		case reflect.TypeFor[byte](), reflect.TypeFor[uint8](): // 8 bit
			b := make([]byte, 1)
			n, err := in.Read(b)
			if err != nil {
				return err
			}
			rc += n
			field.v.SetUint(uint64(b[0]))
		case reflect.TypeFor[int8]():
			b := make([]byte, 1)
			n, err := in.Read(b)
			if err != nil {
				return err
			}
			rc += n
			field.v.SetInt(int64(b[0]))
		case reflect.TypeFor[uint16](): // 16 bit
			b := make([]byte, 2)
			n, err := in.Read(b)
			if err != nil {
				return err
			}
			rc += n
			if field.endianess {
				field.v.SetUint(uint64(binary.BigEndian.Uint16(b)))
			} else {
				field.v.SetUint(uint64(binary.LittleEndian.Uint16(b)))
			}
		case reflect.TypeFor[int16]():
			b := make([]byte, 2)
			n, err := in.Read(b)
			if err != nil {
				return err
			}
			rc += n
			if field.endianess {
				field.v.SetInt(int64(binary.BigEndian.Uint16(b)))
			} else {
				field.v.SetInt(int64(binary.LittleEndian.Uint16(b)))
			}
		case reflect.TypeFor[uint32](): // 32 bit
			b := make([]byte, 4)
			n, err := in.Read(b)
			if err != nil {
				return err
			}
			rc += n
			if field.endianess {
				field.v.SetUint(uint64(binary.BigEndian.Uint32(b)))
			} else {
				field.v.SetUint(uint64(binary.LittleEndian.Uint32(b)))
			}
		case reflect.TypeFor[int32]():
			b := make([]byte, 4)
			n, err := in.Read(b)
			if err != nil {
				return err
			}
			rc += n
			if field.endianess {
				field.v.SetInt(int64(binary.BigEndian.Uint32(b)))
			} else {
				field.v.SetInt(int64(binary.LittleEndian.Uint32(b)))
			}
		case reflect.TypeFor[uint](), reflect.TypeFor[uintptr](), reflect.TypeFor[uint64](): // 64 bit (native ints should be treated as 64 bit)
			b := make([]byte, 8)
			n, err := in.Read(b)
			if err != nil {
				return err
			}
			rc += n
			if field.endianess {
				field.v.SetUint(binary.BigEndian.Uint64(b))
			} else {
				field.v.SetUint(binary.LittleEndian.Uint64(b))
			}
		case reflect.TypeFor[int](), reflect.TypeFor[int64]():
			b := make([]byte, 8)
			n, err := in.Read(b)
			if err != nil {
				return err
			}
			rc += n
			if field.endianess {
				field.v.SetInt(int64(binary.BigEndian.Uint64(b)))
			} else {
				field.v.SetInt(int64(binary.LittleEndian.Uint64(b)))
			}
		case reflect.TypeFor[Padding](): // variable length
			count, err := strconv.ParseInt(field.params[1], 10, 64)
			if err != nil {
				panic(err)
			}
			n, err := in.Read(make([]byte, count))
			rc += n
			if err != nil {
				return err
			}
		case reflect.TypeFor[NullString]():
			var str NullString
			b := make([]byte, 1)
			for {
				n, err := in.Read(b)
				if err != nil {
					return err
				}
				rc += n
				if b[0] == '\000' {
					break
				}
				str = append(str, b[0])
			}
			field.v.Set(reflect.ValueOf(str))
		case reflect.TypeFor[LengthString]():
			b := make([]byte, 1)
			n, err := in.Read(b)
			if err != nil {
				return err
			}
			rc += n
			length := uint8(b[0])
			b = make([]byte, length)
			n, err = in.Read(b)
			if err != nil {
				return err
			}
			rc += n
			field.v.SetBytes(b)
		case reflect.TypeFor[LengthString16]():
			b := make([]byte, 2)
			n, err := in.Read(b)
			if err != nil {
				return err
			}
			rc += n
			var length uint16
			if field.endianess {
				length = binary.BigEndian.Uint16(b)
			} else {
				length = binary.LittleEndian.Uint16(b)
			}
			b = make([]byte, length)
			n, err = in.Read(b)
			if err != nil {
				return err
			}
			rc += n
			field.v.SetBytes(b)
		case reflect.TypeFor[LengthString32]():
			b := make([]byte, 4)
			n, err := in.Read(b)
			if err != nil {
				return err
			}
			rc += n
			var length uint32
			if field.endianess {
				length = binary.BigEndian.Uint32(b)
			} else {
				length = binary.LittleEndian.Uint32(b)
			}
			b = make([]byte, length)
			n, err = in.Read(b)
			if err != nil {
				return err
			}
			rc += n
			field.v.SetBytes(b)
		case reflect.TypeFor[LengthString64]():
			b := make([]byte, 8)
			n, err := in.Read(b)
			if err != nil {
				return err
			}
			rc += n
			var length uint64
			if field.endianess {
				length = binary.BigEndian.Uint64(b)
			} else {
				length = binary.LittleEndian.Uint64(b)
			}
			b = make([]byte, length)
			n, err = in.Read(b)
			if err != nil {
				return err
			}
			rc += n
			field.v.SetBytes(b)
		case reflect.TypeFor[string](), reflect.TypeFor[[]byte]():
			panic("cannot binary encode: types string and []byte are unsuppored for unmarshal")
		default:
			panic("cannot binary encode: unsupported type")
		}
	}
	return nil
}
