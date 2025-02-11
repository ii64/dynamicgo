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

package thrift

import (
	"bytes"
	"context"
	"encoding/hex"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"testing"
	"time"

	"github.com/cloudwego/dynamicgo/meta"
	thrift_iter "github.com/thrift-iterator/go"
	"github.com/thrift-iterator/go/general"
	"github.com/thrift-iterator/go/protocol"
)

var (
	debugAsyncGC = os.Getenv("SONIC_NO_ASYNC_GC") == ""
)

func TestMain(m *testing.M) {
	go func() {
		if !debugAsyncGC {
			return
		}
		println("Begin GC looping...")
		for {
			runtime.GC()
			debug.FreeOSMemory()
		}
	}()
	time.Sleep(time.Millisecond)
	m.Run()
}

func getExampleDesc() *TypeDescriptor {
	svc, err := NewDescritorFromPath(context.Background(), "../testdata/idl/example.thrift")
	if err != nil {
		panic(err)
	}
	return svc.Functions()["ExampleMethod"].Request().Struct().FieldByKey("req").Type()
}

func getExampleData() []byte {
	out, err := ioutil.ReadFile("../testdata/data/example.bin")
	if err != nil {
		panic(err)
	}
	return out
}

func TestBinaryUnwrapReply(t *testing.T) {

	type args struct {
		proto meta.Encoding
		msg   general.Message
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "simple",
			args: args{
				proto: meta.EncodingThriftBinary,
				msg: general.Message{
					MessageHeader: protocol.MessageHeader{
						MessageName: "GetFavoriteMethod",
						MessageType: protocol.MessageTypeReply,
					},
					Arguments: general.Struct{
						1: general.Struct{
							1: int32(7749),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// pair of buffer
			inputBytes, err := thrift_iter.Marshal(tt.args.msg)
			if err != nil {
				t.Fatalf("failed to marshal input %s", err)
			}
			wantBytes, err := thrift_iter.Marshal(tt.args.msg.Arguments[1].(general.Struct))
			if err != nil {
				t.Fatalf("failed to marshal want bytes %s", err)
			}
			//
			_, _, _, _, got, err := UnwrapBinaryMessage(inputBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnwrapReply() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, wantBytes) {
				t.Errorf("UnwrapReply() = %s, want %s", hex.EncodeToString(got), hex.EncodeToString(wantBytes))
			}
			header, footer, err := GetBinaryMessageHeaderAndFooter(tt.args.msg.MessageHeader.MessageName, TMessageType(tt.args.msg.MessageHeader.MessageType), 1, 0)
			if err != nil {
				t.Fatalf("failed to wrap message %s", err)
			}
			header = append(header, got...)
			header = append(header, footer...)
			if !bytes.Equal(header, inputBytes) {
				t.Errorf("WrapBinaryMessage() = %s, want %s", hex.EncodeToString(header), hex.EncodeToString(inputBytes))
			}

		})
	}
}