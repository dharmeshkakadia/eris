// +build !windows !cgo

package monkutil

import (

//"github.com/obscuren/serpent-go"
)

// strings and hex only
func PackTxDataArgs(args ...string) []byte {
	//fmt.Println("pack data:", args)
	ret := *new([]byte)
	for _, s := range args {
		if s[:2] == "0x" {
			t := s[2:]
			if len(t)%2 == 1 {
				t = "0" + t
			}
			x := Hex2Bytes(t)
			//fmt.Println(x)
			l := len(x)
			ret = append(ret, LeftPadBytes(x, 32*((l+31)/32))...)
		} else {
			x := []byte(s)
			l := len(x)
			ret = append(ret, RightPadBytes(x, 32*((l+31)/32))...)
		}
	}
	return ret
}

// strings and hex only
func PackTxDataArgs2(args ...string) []byte {
	//fmt.Println("pack data:", args)
	ret := *new([]byte)
	for _, s := range args {
		if len(s) > 1 && s[:2] == "0x" {
			t := s[2:]
			if len(t)%2 == 1 {
				t = "0" + t
			}
			x := Hex2Bytes(t)
			//fmt.Println(x)
			l := len(x)
			ret = append(ret, LeftPadBytes(x, 32*((l+31)/32))...)
		} else {
			x := []byte(s)
			l := len(x)
			ret = append(ret, LeftPadBytes(x, 32*((l+31)/32))...)
		}
	}
	return ret
}

func PackTxDataBytes(args ...[]byte) []byte {
	ret := *new([]byte)
	for _, s := range args {
		l := len(s)
		if l == 0 {
			ret = append(ret, LeftPadBytes([]byte{0}, 32)...)
		}
		ret = append(ret, LeftPadBytes(s, 32*((l+31)/32))...)
	}
	return ret
}
