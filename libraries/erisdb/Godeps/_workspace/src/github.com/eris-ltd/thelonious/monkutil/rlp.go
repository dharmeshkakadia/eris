package monkutil

import (
	"bytes"
	"fmt"
	"math/big"
)

var COMPRESS = false

type RlpEncode interface {
	RlpEncode() []byte
}

type RlpEncodeDecode interface {
	RlpEncode
	RlpValue() []interface{}
}

func Rlp(encoder RlpEncode) []byte {
	return encoder.RlpEncode()
}

type RlpEncoder struct {
	rlpData []byte
}

func NewRlpEncoder() *RlpEncoder {
	encoder := &RlpEncoder{}

	return encoder
}
func (coder *RlpEncoder) EncodeData(rlpData interface{}) []byte {
	return Encode(rlpData)
}

const (
	RlpEmptyList = 0x80
	RlpEmptyStr  = 0x40
)

const rlpEof = -1

func Char(c []byte) int {
	if len(c) > 0 {
		return int(c[0])
	}

	return rlpEof
}

func DecodeWithReader(reader *bytes.Buffer) interface{} {
	var slice []interface{}

	// Read the next byte
	char := Char(reader.Next(1))
	switch {
	case char <= 0x7f:
		return char

	case char <= 0xb7:
		return reader.Next(int(char - 0x80))

	case char <= 0xbf:
		length := ReadVarInt(reader.Next(int(char - 0xb7)))

		return reader.Next(int(length))

	case char <= 0xf7:
		length := int(char - 0xc0)
		for i := 0; i < length; i++ {
			obj := DecodeWithReader(reader)
			slice = append(slice, obj)
		}

		return slice
	case char <= 0xff:
		length := ReadVarInt(reader.Next(int(char - 0xf7)))
		for i := uint64(0); i < length; i++ {
			obj := DecodeWithReader(reader)
			slice = append(slice, obj)
		}

		return slice
	default:
		panic(fmt.Sprintf("byte not supported: %q", char))
	}

	return slice
}

var (
	directRlp = big.NewInt(0x7f)
	numberRlp = big.NewInt(0xb7)
	zeroRlp   = big.NewInt(0x0)
)

func Encode(object interface{}) []byte {
	return _Encode(object)
}

func _Encode(object interface{}) []byte {
	var buff bytes.Buffer

	if object != nil {
		switch t := object.(type) {
		case *Value:
			buff.Write(_Encode(t.Raw()))
		// Code dup :-/
		case int:
			buff.Write(_Encode(big.NewInt(int64(t))))
		case uint:
			buff.Write(_Encode(big.NewInt(int64(t))))
		case int8:
			buff.Write(_Encode(big.NewInt(int64(t))))
		case int16:
			buff.Write(_Encode(big.NewInt(int64(t))))
		case int32:
			buff.Write(_Encode(big.NewInt(int64(t))))
		case int64:
			buff.Write(_Encode(big.NewInt(t)))
		case uint16:
			buff.Write(_Encode(big.NewInt(int64(t))))
		case uint32:
			buff.Write(_Encode(big.NewInt(int64(t))))
		case uint64:
			buff.Write(_Encode(big.NewInt(int64(t))))
		case byte:
			buff.Write(_Encode(big.NewInt(int64(t))))
		case *big.Int:
			// Not sure how this is possible while we check for
			if t == nil {
				buff.WriteByte(0xc0)
			} else {
				buff.Write(_Encode(t.Bytes()))
			}
		case Bytes:
			buff.Write(_Encode([]byte(t)))
		case []byte:
			// compress repeated 0s before rlp encode
			if COMPRESS {
				t = compress0(t)
				//fmt.Println("compressed:", t)
			}

			if len(t) == 1 && t[0] <= 0x7f {
				buff.Write(t)
			} else if len(t) < 56 {
				buff.WriteByte(byte(len(t) + 0x80))
				buff.Write(t)
			} else {
				b := big.NewInt(int64(len(t)))
				buff.WriteByte(byte(len(b.Bytes()) + 0xb7))
				buff.Write(b.Bytes())
				buff.Write(t)
			}
		case string:
			buff.Write(_Encode([]byte(t)))
		case []interface{}:
			// Inline function for writing the slice header
			WriteSliceHeader := func(length int) {
				if length < 56 {
					buff.WriteByte(byte(length + 0xc0))
				} else {
					b := big.NewInt(int64(length))
					buff.WriteByte(byte(len(b.Bytes()) + 0xf7))
					buff.Write(b.Bytes())
				}
			}

			var b bytes.Buffer
			for _, val := range t {
				b.Write(_Encode(val))
			}
			WriteSliceHeader(len(b.Bytes()))
			buff.Write(b.Bytes())
		}
	} else {
		// Empty list for nil
		buff.WriteByte(0xc0)
	}

	return buff.Bytes()
}

// TODO Use a bytes.Buffer instead of a raw byte slice.
// Cleaner code, and use draining instead of seeking the next bytes to read
func Decode_(data []byte, pos uint64) (interface{}, uint64) {
	var slice []interface{}
	char := int(data[pos])
	switch {
	case char <= 0x7f:
		return data[pos], pos + 1

	case char <= 0xb7:
		b := uint64(data[pos]) - 0x80

		return data[pos+1 : pos+1+b], pos + 1 + b

	case char <= 0xbf:
		b := uint64(data[pos]) - 0xb7

		b2 := ReadVarInt(data[pos+1 : pos+1+b])

		return data[pos+1+b : pos+1+b+b2], pos + 1 + b + b2

	case char <= 0xf7:
		b := uint64(data[pos]) - 0xc0
		prevPos := pos
		pos++
		for i := uint64(0); i < b; {
			var obj interface{}

			// Get the next item in the data list and append it
			obj, prevPos = Decode(data, pos)
			slice = append(slice, obj)

			// Increment i by the amount bytes read in the previous
			// read
			i += (prevPos - pos)
			pos = prevPos
		}
		return slice, pos

	case char <= 0xff:
		l := uint64(data[pos]) - 0xf7
		b := ReadVarInt(data[pos+1 : pos+1+l])

		pos = pos + l + 1

		prevPos := b
		for i := uint64(0); i < uint64(b); {
			var obj interface{}

			obj, prevPos = Decode(data, pos)
			slice = append(slice, obj)

			i += (prevPos - pos)
			pos = prevPos
		}
		return slice, pos

	default:
		panic(fmt.Sprintf("byte not supported: %q", char))
	}

	return slice, 0
}

func Decode(data []byte, pos uint64) (interface{}, uint64) {
	d, p := _Decode(data, pos)
	// decompress data after rlp decoding
	if COMPRESS {
		// since d is type interface{}, it may be []interface{} recursively
		d = recursiveDecompress0(d)
	}

	return d, p
}

func _Decode(data []byte, pos uint64) (interface{}, uint64) {
	var slice []interface{}
	char := int(data[pos])
	var ret []byte
	var retN uint64
	switch {
	case char <= 0x7f:
		return data[pos], pos + 1

	case char <= 0xb7:
		b := uint64(data[pos]) - 0x80

		ret = data[pos+1 : pos+1+b]
		retN = pos + 1 + b

	case char <= 0xbf:
		b := uint64(data[pos]) - 0xb7

		b2 := ReadVarInt(data[pos+1 : pos+1+b])

		ret = data[pos+1+b : pos+1+b+b2]
		retN = pos + 1 + b + b2

	case char <= 0xf7:
		b := uint64(data[pos]) - 0xc0
		prevPos := pos
		pos++
		for i := uint64(0); i < b; {
			var obj interface{}

			// Get the next item in the data list and append it
			obj, prevPos = _Decode(data, pos)
			slice = append(slice, obj)

			// Increment i by the amount bytes read in the previous
			// read
			i += (prevPos - pos)
			pos = prevPos
		}
		return slice, pos

	case char <= 0xff:
		// how many bytes for the length
		l := uint64(data[pos]) - 0xf7
		// the length
		b := ReadVarInt(data[pos+1 : pos+1+l])

		pos = pos + l + 1

		prevPos := b
		for i := uint64(0); i < uint64(b); {
			var obj interface{}

			obj, prevPos = _Decode(data, pos)
			slice = append(slice, obj)

			i += (prevPos - pos)
			pos = prevPos
		}
		return slice, pos

	default:
		panic(fmt.Sprintf("byte not supported: %q", char))
	}
	return ret, retN
	//return slice, 0
}

// compresison/decompression of zero bytes
// called by Encode and Decode

// Compress all 0s in a byte array
func compress0(d []byte) []byte {
	matches := [][]int{} // list of pairs of indices delineating 0 patches
	beg := 0
	in := false
	// find all matches
	for i := 0; i < len(d); i++ {
		if !in && d[i] == byte(0) {
			beg = i
			in = true
		}
		if in && d[i] != byte(0) {
			matches = append(matches, []int{beg, i})
			in = false
		}
	}
	// if the array itself ends in a 0
	if in {
		matches = append(matches, []int{beg, len(d)})
	}
	// compressed version
	dd := []byte{}
	lasti := 0
	for _, ii := range matches {
		i := ii[0]
		// append the non-zeros
		dd = append(dd, d[lasti:i]...)
		l := ii[1] - i
		// we only compress if its more than one zero
		if l > 1 {
			if l < 256 {
				dd = append(dd, []byte{0, 0, byte(ii[1] - i)}...)
			} else {
				bb := []byte{0, 0, 0}
				lenn := big.NewInt(int64(l)).Bytes()
				lenN := byte(len(lenn))
				bb = append(bb, lenN)
				bb = append(bb, lenn...)
				dd = append(dd, bb...)
			}
		} else {
			dd = append(dd, byte(0))
		}
		lasti = ii[1]
	}
	// if the array doesn't end in zeros, append the rest
	if lasti != len(d) {
		dd = append(dd, d[lasti:]...)
	}
	return dd
}

// to save on the reflection that builtin append must do
func AppendBytes(slice []byte, elements ...byte) []byte {
	n := len(slice)
	total := len(slice) + len(elements)
	if total > cap(slice) {
		// Reallocate.
		newSize := total * 2
		newSlice := make([]byte, total, newSize)
		copy(newSlice, slice)
		slice = newSlice
	}
	slice = slice[:total]
	copy(slice[n:], elements)
	return slice
}

// Decompress byte array with compressed zeros
// All sequences of 2 or 3 zeros are markers of a compressed sequence.
// 4 or more are invalid and will probably lead to an error.
// unfortunately, it's slow
func decompress0(d []byte) []byte {
	l := len(d)
	dd := []byte{}
	lasti := 0
	for i := 0; i < l; i++ {
		if l-i < 3 {
			// if we are basically at the end
			continue
		} else if bytes.Compare(d[i:i+3], []byte{0, 0, 0}) == 0 {
			// expand arbitrary number of zeros
			dd = AppendBytes(dd, d[lasti:i]...) // append non zeros up to now
			n := int(d[i+3])
			nzb := d[i+4 : i+4+n]
			bigN := BigD(nzb)
			nzeros := int(bigN.Int64())
			dd = AppendBytes(dd, bytes.Repeat([]byte{0}, nzeros)...)
			lasti = i + 4 + n
			i += 3
		} else if bytes.Compare(d[i:i+2], []byte{0, 0}) == 0 {
			// expand fewer than 256 zeros
			dd = AppendBytes(dd, d[lasti:i]...) // append non zeros up to now
			n := int(d[i+2])
			dd = AppendBytes(dd, bytes.Repeat([]byte{0}, n)...)
			lasti = i + 3
			i += 2
		}
	}
	// if the byte array does not end in zeros, append the rest
	if lasti != l {
		dd = AppendBytes(dd, d[lasti:]...)
	}
	return dd[:len(dd)]
}

// Recursively decompress an interface type if it is a list
func recursiveDecompress0(d interface{}) interface{} {
	switch t := d.(type) {
	case byte:
		return t
	case []byte:
		l := make([]byte, len(t))
		copy(l, t)
		r := decompress0(l)
		return r
	case []interface{}:
		ret := make([]interface{}, len(t))
		for i, k := range t {
			ret[i] = recursiveDecompress0(k)
		}
		return ret
	default:
		// should never get here...
		panic(t)
	}
	// or here
	return nil
}
