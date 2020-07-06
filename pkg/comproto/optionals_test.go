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
	"fmt"
	"math"
	"testing"
	"time"
)

func TestValueUint8(t *testing.T) {
	type args []OptionalUint8
	tests := []struct {
		name      string
		args      args
		wantValue []uint8
		wantErr   bool
	}{
		{
			name: "Should return uint8 value",
			args: []OptionalUint8{
				int(22), int8(8), int16(61), int32(23), int64(44),
				uint(12), uint8(18), uint16(161), uint32(33), uint64(255),
			},
			wantValue: []uint8{22, 8, 61, 23, 44, 12, 18, 161, 33, 255},
		},

		{
			name: "Should fail return uint8 value",
			args: []OptionalUint8{
				nil, int(-1), int8(-2), int16(-22), int32(-33), int64(-66),
				int(300), int16(416), int32(532), int64(666),
				uint(700), uint16(256), uint32(512), uint64(1024),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		for i := len(tt.args) - 1; i >= 0; i-- {
			arg := tt.args[i]
			ttName := fmt.Sprintf("%s___%T", tt.name, arg)
			t.Run(ttName, func(t *testing.T) {
				gotValue, err := ValueUint8(arg)
				if (err != nil) != tt.wantErr {
					t.Errorf("Value(%T=%v) error = %v, wantErr %v", arg, arg, err, tt.wantErr)
					return
				}
				want := uint8(255)
				if err == nil {
					want = tt.wantValue[i]
				}
				if gotValue != want {
					t.Errorf("Value(%T=%v) = %v, want %v", arg, arg, gotValue, want)
				}
			})

		}
	}
}

func TestValueDuration(t *testing.T) {

	type args []OptionalDuration
	tests := []struct {
		name      string
		args      args
		wantValue []time.Duration
		wantErr   bool
	}{
		{
			name: "Should return uint8 value",
			args: []OptionalDuration{
				time.Duration(4444),
				int(0), int8(8), int16(61), int32(23), int64(44),
				uint(12), uint8(18), uint16(161), uint32(33), uint64(math.MaxInt64),
				int(-1), int8(-2), int16(-22), int32(-33), math.MinInt64,
			},
			wantValue: []time.Duration{
				4444,
				0, 8, 61, 23, 44,
				12, 18, 161, 33, math.MaxInt64,
				-1, -2, -22, -33, math.MinInt64,
			},
		},

		{
			name: "Should fail return uint8 value",
			args: []OptionalDuration{
				nil,
				uint64(math.MaxUint64),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		for i := len(tt.args) - 1; i >= 0; i-- {
			arg := tt.args[i]
			ttName := fmt.Sprintf("%s___%T", tt.name, arg)
			t.Run(ttName, func(t *testing.T) {
				gotValue, err := ValueDuration(arg)
				if (err != nil) != tt.wantErr {
					t.Errorf("Value(%T=%#v) error = %v, wantErr %v", arg, arg, err, tt.wantErr)
					return
				}
				want := duration10s()
				if err == nil {
					want = tt.wantValue[i]
				}
				if gotValue != want {
					t.Errorf("ValueDuration(%T=%#v) = %v, want %v", arg, arg, gotValue, want)
				}
			})

		}
	}
}
