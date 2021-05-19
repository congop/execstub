// Copyright 2020 The Execstub Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package comproto

import (
	"bytes"
	"reflect"
	"testing"
)

func TestExecOutcomeDecoderFunc(t *testing.T) {
	type args struct {
		outcome ExecOutcome
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Should be able to encode decode",
			args: args{
				outcome: ExecOutcome{
					InternalErrTxt: "iTxt",
					ExitCode:       210,
					Key:            "k1",
					Stderr:         "stderr1",
					Stdout:         "stdout2",
				},
			},
		},
		{
			name: "Should be able to encode decode null execoutome",
			args: args{
				outcome: ExecOutcome{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			encoderFunc := ExecOutcomeEncoderFunc(&buf)
			erre := encoderFunc(&tt.args.outcome)
			if erre != nil {
				t.Errorf("could not encode [%#v]: %v", tt.args.outcome, erre)
				return
			}
			bufStr := buf.String()
			buf.Reset()
			buf.WriteString(bufStr)
			got, errDec := ExecOutcomeDecoderFunc(&buf)
			if (errDec != nil) != tt.wantErr {
				t.Errorf("Could not decode= %v, wantErr %v", errDec, tt.wantErr)
				return
			}
			want := tt.args.outcome
			if !reflect.DeepEqual(*got, want) {
				t.Errorf("ExecOutcomeDecoderFunc() \ngot = %v, \nwant= %v \nbuf=%s",
					*got, want, bufStr)
			}
		})
	}
}

func TestStubbingExecEncodeDecode(t *testing.T) {
	type args struct {
		req StubRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Should be able to encode decode",
			args: args{
				req: StubRequest{
					Args: []string{"arg1", "argb", "arg3"},
					Key:  "k_1",
				},
			},
		},
		{
			name: "Should not be able to encode decode null stubrequest",
			args: args{
				req: StubRequest{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			encoderFunc := StubRequestEncoderFunc(&buf)
			erre := encoderFunc(&tt.args.req)
			if (erre != nil) != tt.wantErr {
				t.Errorf(
					"unexpected error status while encoding stubbing request [%#v]: wantError=%t error=%v",
					tt.args.req, tt.wantErr, erre)
				return
			}
			bufStr := buf.String()
			buf.Reset()
			buf.WriteString(bufStr)
			got, errDec := StubRequestDecoderFunc(&buf)
			if (errDec != nil) != tt.wantErr {
				t.Errorf("unexpected error status while decoding stubbing request: error=%v, wantErr %v", errDec, tt.wantErr)
				return
			}
			if errDec == nil {
				wantReq := tt.args.req
				if !reflect.DeepEqual(*got, wantReq) {
					t.Errorf("ExecOutcomeDecoderFunc() \ngot = %#v, \nwant= %#v buf=%s",
						*got, wantReq, bufStr)
				}
			}
		})
	}
}
