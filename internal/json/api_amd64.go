//go:build amd64 && go1.16
// +build amd64,go1.16

/**
 * Copyright 2023 CloudWeGo Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package json

import (
	"runtime"
	"unsafe"

	"github.com/bytedance/sonic/encoder"
	"github.com/chenzhuoyu/base64x"
	"github.com/cloudwego/dynamicgo/internal/native"
	"github.com/cloudwego/dynamicgo/internal/native/types"
	"github.com/cloudwego/dynamicgo/internal/rt"
	uq "github.com/cloudwego/dynamicgo/internal/unquote"
)

var typeByte = rt.UnpackEface(byte(0)).Type

func quote(buf *[]byte, val string) {
	*buf = append(*buf, '"')
	if len(val) == 0 {
		*buf = append(*buf, '"')
	}
	NoQuote(buf, val)
	
	*buf = append(*buf, '"')
}

//go:nocheckptr
// NoQuote only escape inner '\' and '"' of one string, but it does add quotes besides string.
func NoQuote(buf *[]byte, val string) {
	sp := rt.IndexChar(val, 0)
	nb := len(val)
	b := (*rt.GoSlice)(unsafe.Pointer(buf))

	// input buffer
	for nb > 0 {
		// output buffer
		dp := unsafe.Pointer(uintptr(b.Ptr) + uintptr(b.Len))
		dn := b.Cap - b.Len
		// call native.Quote, dn is byte count it outputs
		ret := native.Quote(sp, nb, dp, &dn, 0)
		// update *buf length
		b.Len += dn

		// no need more output
		if ret >= 0 {
			break
		}

		// double buf size
		*b = rt.Growslice(typeByte, *b, b.Cap*2)
		// ret is the complement of consumed input
		ret = ^ret
		// update input buffer
		nb -= ret
		sp = unsafe.Pointer(uintptr(sp) + uintptr(ret))
	}

	runtime.KeepAlive(buf)
	runtime.KeepAlive(sp)
}

func unquote(src string) (string, types.ParsingError) {
	return uq.String(src)
}

func decodeBase64(src string) ([]byte, error) {
	return base64x.StdEncoding.DecodeString(src)
}

func encodeBase64(src []byte) string {
	return base64x.StdEncoding.EncodeToString(src)
}

func (self *Parser) decodeValue() (val types.JsonState) {
	sv := (*rt.GoString)(unsafe.Pointer(&self.s))
	self.p = native.Value(sv.Ptr, sv.Len, self.p, &val, 0)
	return
}

func (self *Parser) skip() (int, types.ParsingError) {
	fsm := types.NewStateMachine()
	start := native.SkipOne(&self.s, &self.p, fsm)
	types.FreeStateMachine(fsm)

	if start < 0 {
		return self.p, types.ParsingError(-start)
	}
	return start, 0
}

func (self *Node) encodeInterface(buf *[]byte) error {
	//WARN: NOT compatible with json.Encoder
	return encoder.EncodeInto(buf, self.packAny(), 0)
}

func i64toa(buf *[]byte, val int64) int {
	rt.GuardSlice(buf, types.MaxInt64StringLen)
	s := len(*buf)
	ret := native.I64toa((*byte)(rt.IndexPtr((*rt.GoSlice)(unsafe.Pointer(buf)).Ptr, byteType.Size, s)), val)
	if ret < 0 {
		*buf = append((*buf)[s:], '0')
		return 1
	}
	*buf = (*buf)[:s+ret]
	return ret
}

func f64toa(buf *[]byte, val float64) int {
	rt.GuardSlice(buf, types.MaxFloat64StringLen)
	s := len(*buf)
	ret := native.F64toa((*byte)(rt.IndexPtr((*rt.GoSlice)(unsafe.Pointer(buf)).Ptr, byteType.Size, s)), val)
	if ret < 0 {
		*buf = append((*buf)[s:], '0')
		return 1
	}
	*buf = (*buf)[:s+ret]
	return ret
}