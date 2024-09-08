package binaryencode

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func genBytes(l int) []byte {
	b := make([]byte, l)
	rand.Read(b) // dumb that this is deprecated :/
	return b
}

type testStruct struct {
	Int8  int8  `binary:"0"`
	Uint8 uint8 `binary:"1"`
	Byte  byte  `binary:"2"`

	Int16B  int16  `binary:"3,big"`
	Int16L  int16  `binary:"4,little"`
	Uint16B uint16 `binary:"5,big"`
	Uint16L uint16 `binary:"6,little"`

	Int32B  int32  `binary:"7,big"`
	Int32L  int32  `binary:"8,little"`
	Uint32B uint32 `binary:"9,big"`
	Uint32L uint32 `binary:"10,little"`

	Int64B  int64  `binary:"11,big"`
	Int64L  int64  `binary:"12,little"`
	Uint64B uint64 `binary:"13,big"`
	Uint64L uint64 `binary:"14,little"`

	Padding    Padding    `binary:"15,8"`
	NullString NullString `binary:"16"`

	LengthString8   LengthString   `binary:"17"`
	LengthString16B LengthString16 `binary:"18,big"`
	LengthString16L LengthString16 `binary:"19,little"`
	LengthString32B LengthString32 `binary:"20,big"`
	LengthString32L LengthString32 `binary:"21,little"`
	LengthString64B LengthString64 `binary:"22,big"`
	LengthString64L LengthString64 `binary:"23,little"`

	Unindexed uint
}

func (s *testStruct) gen(sl int) {
	s.Int8 = int8(rand.Int31())
	s.Uint8 = uint8(rand.Uint32())
	s.Byte = byte(rand.Uint32())

	s.Int16B = int16(rand.Int31())
	s.Int16L = int16(rand.Int31())
	s.Uint16B = uint16(rand.Uint32())
	s.Uint16L = uint16(rand.Uint32())

	s.Int32B = int32(rand.Int31())
	s.Int32L = int32(rand.Int31())
	s.Uint32B = uint32(rand.Uint32())
	s.Uint32L = uint32(rand.Uint32())

	s.Int64B = int64(rand.Int63())
	s.Int64L = int64(rand.Int63())
	s.Uint64B = uint64(rand.Uint64())
	s.Uint64L = uint64(rand.Uint64())

	s.Padding = 0
	s.NullString = NullString("testing testing")

	s.LengthString8 = LengthString(genBytes(sl))
	s.LengthString16B = LengthString16(genBytes(sl))
	s.LengthString16L = LengthString16(genBytes(sl))
	s.LengthString32B = LengthString32(genBytes(sl))
	s.LengthString32L = LengthString32(genBytes(sl))
	s.LengthString64B = LengthString64(genBytes(sl))
	s.LengthString64L = LengthString64(genBytes(sl))
}

func TestMarshal(t *testing.T) {
	var marshalable testStruct
	marshalable.gen(12)

	expected := make([]byte, 0)
	expected = append(expected, byte(marshalable.Int8), marshalable.Uint8, marshalable.Byte)

	expected = binary.BigEndian.AppendUint16(expected, uint16(marshalable.Int16B))
	expected = binary.LittleEndian.AppendUint16(expected, uint16(marshalable.Int16L))
	expected = binary.BigEndian.AppendUint16(expected, marshalable.Uint16B)
	expected = binary.LittleEndian.AppendUint16(expected, marshalable.Uint16L)

	expected = binary.BigEndian.AppendUint32(expected, uint32(marshalable.Int32B))
	expected = binary.LittleEndian.AppendUint32(expected, uint32(marshalable.Int32L))
	expected = binary.BigEndian.AppendUint32(expected, marshalable.Uint32B)
	expected = binary.LittleEndian.AppendUint32(expected, marshalable.Uint32L)

	expected = binary.BigEndian.AppendUint64(expected, uint64(marshalable.Int64B))
	expected = binary.LittleEndian.AppendUint64(expected, uint64(marshalable.Int64L))
	expected = binary.BigEndian.AppendUint64(expected, marshalable.Uint64B)
	expected = binary.LittleEndian.AppendUint64(expected, marshalable.Uint64L)

	expected = append(expected, bytes.Repeat([]byte{'\000'}, 8)...)
	expected = append(expected, append(marshalable.NullString, '\000')...)

	expected = append(expected, byte(len(marshalable.LengthString8)))
	expected = append(expected, marshalable.LengthString8...)
	expected = binary.BigEndian.AppendUint16(expected, uint16(len(marshalable.LengthString16B)))
	expected = append(expected, marshalable.LengthString16B...)
	expected = binary.LittleEndian.AppendUint16(expected, uint16(len(marshalable.LengthString16L)))
	expected = append(expected, marshalable.LengthString16L...)
	expected = binary.BigEndian.AppendUint32(expected, uint32(len(marshalable.LengthString32B)))
	expected = append(expected, marshalable.LengthString32B...)
	expected = binary.LittleEndian.AppendUint32(expected, uint32(len(marshalable.LengthString32L)))
	expected = append(expected, marshalable.LengthString32L...)
	expected = binary.BigEndian.AppendUint64(expected, uint64(len(marshalable.LengthString64B)))
	expected = append(expected, marshalable.LengthString64B...)
	expected = binary.LittleEndian.AppendUint64(expected, uint64(len(marshalable.LengthString64L)))
	expected = append(expected, marshalable.LengthString64L...)

	outcome := Marshal(marshalable, EncoderArgs{})

	if !reflect.DeepEqual(outcome, expected) {
		t.Log(spew.Sdump(outcome, expected))
		t.Fail()
		return
	}
}

func TestUnmarshal(t *testing.T) { // need to rewrite this so it doesnt always fail
	TestMarshal(t) // Marshal has to work for this test

	var expected testStruct
	expected.gen(12)
	unmarshalable := Marshal(expected, EncoderArgs{})
	var outcome testStruct
	err := Unmarshal(bytes.NewReader(unmarshalable), &outcome, EncoderArgs{})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(outcome, expected) {
		t.Log(spew.Sdump(outcome, expected))
		t.Fail()
		return
	}
}
