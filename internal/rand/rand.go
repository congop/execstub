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

package rand

import (
	"math"
	"math/rand"
	"strconv"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NextRandInt63AsHexStr returns the next non-negative pseudo-random 63-bit
// integer as hex string.
func NextRandInt63AsHexStr() string {
	// we are okay with weak randon number generator,
	// since we are just randomizing file names
	return strconv.FormatInt(rand.Int63(), 16) // #nosec G404
}

// NextUint16 returns the next non-negative pseudo-random integer in uint16 range.
func NextUint16() uint16 {
	// we are okay with weak randon number generator,
	// since we are just randomizing file names
	return uint16(rand.Intn(math.MaxUint16)) // #nosec G404
}
