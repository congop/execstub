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
	"math"
	"reflect"
	"time"

	"github.com/pkg/errors"
)

// OptionalUint8 is an uint8 or nil to fake optional type for unit8.
// It may legaly holds the following types as long as their value
// is within the rang of uint8: int,int8/16/32/64,uint,uint8/16/32/64
type OptionalUint8 interface{}

// OptionalDuration is an time.Duration or nil to fake optional type for time.Duration.
// It may legly holds the following type as long as the valze is
// within the rang of time.Duration: int,int8/16/32/64,uint,uint8/16/32/64
type OptionalDuration interface{}

// ValueUint8 returns the uint8 value if present or an error.
func ValueUint8(o OptionalUint8) (value uint8, err error) {
	if nil == o {
		return math.MaxUint8, errors.New("value absent")
	}

	switch ot := o.(type) {
	case uint8:
		return o.(uint8), nil
	case int, int8, int16, int32, int64:
		rvalue := reflect.ValueOf(o)
		valInt64 := rvalue.Int()
		if valInt64 < 0 || valInt64 > math.MaxUint8 {
			return math.MaxUint8, errors.Errorf("value (%d) not >=0 and <=255", valInt64)
		}
		return uint8(valInt64), nil
	case uint, uint16, uint32, uint64:
		rvalue := reflect.ValueOf(o)
		valUint64 := rvalue.Uint()
		if valUint64 > math.MaxUint8 {
			return math.MaxUint8, errors.Errorf("value not less than 255: %d", valUint64)
		}
		return uint8(valUint64), nil
	default:
		err := errors.Errorf(
			"value must be an integer(int,int8/16/32/64,uint,uint8/16/32/64) "+
				"within uint8 range but was type %T=%v",
			ot, o)
		return math.MaxUint8, err
	}
}

// ValueDuration returns the duration value if present or an error.
func ValueDuration(o OptionalDuration) (valueX time.Duration, err error) {
	switch ot := o.(type) {
	case time.Duration:
		return o.(time.Duration), nil
	case int, int8, int16, int32, int64:
		rvalue := reflect.ValueOf(o)
		valInt64 := rvalue.Int()
		return time.Duration(valInt64), nil
	case uint, uint8, uint16, uint32, uint64:
		rvalue := reflect.ValueOf(o)
		valUint64 := rvalue.Uint()
		if valUint64 > math.MaxInt64 {
			err = errors.Errorf(
				"value (=%x) not be greater maxint64(=%x)",
				valUint64, math.MaxInt64)
			return duration10s(), err
		}
		return time.Duration(int64(valUint64)), nil
	default:
		err := errors.Errorf(
			"timeout must be of type time.Duration or int,int8/16/32/64 or "+
				",uint,uint8/16/32/64 having a value within in64 range "+
				"but was type %T=%v",
			ot, o)
		return duration10s(), err
	}
}

func duration10s() time.Duration {
	return time.Duration(10 * time.Second)
}
