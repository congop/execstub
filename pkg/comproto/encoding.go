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
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/pkg/errors"
)

// ExecOutcomeEncoderFunc return an encoder which can write exec outcome representation into the writer.
func ExecOutcomeEncoderFunc(writer io.Writer) func(outcome *ExecOutcome) error {
	// return func(outcome *ExecOutcome) error {
	// 	enc := gob.NewEncoder(writer)
	// 	return enc.Encode(outcome)
	// }
	return execOutcomeEncoderFuncBase64CSV(writer)
}

// ExecOutcomeDecoderFunc decodes an exec-outcome from the given reader.
func ExecOutcomeDecoderFunc(reader io.Reader) (*ExecOutcome, error) {
	// dec := gob.NewDecoder(reader)
	// outcome := ExecOutcome{}
	// err := dec.Decode(&outcome)
	// return &outcome, err
	return execOutcomeDecoderFuncBase64CVS(reader)
}

func execOutcomeEncoderFuncBase64CSV(writer io.Writer) func(outcome *ExecOutcome) error {
	return func(outcome *ExecOutcome) error {
		writerCSV := csv.NewWriter(writer)
		rec := make([]string, 0, 5)
		rec = append(rec, strToBase64(strconv.Itoa(int(outcome.ExitCode))))
		rec = append(rec, strToBase64(outcome.InternalErrTxt))
		rec = append(rec, strToBase64(outcome.Key))
		rec = append(rec, strToBase64(outcome.Stderr))
		rec = append(rec, strToBase64(outcome.Stdout))
		err := writerCSV.Write(rec)
		if err != nil {
			return err
		}
		writerCSV.Flush()
		return nil
	}
}

func execOutcomeDecoderFuncBase64CVS(reader io.Reader) (*ExecOutcome, error) {
	csvReader := csv.NewReader(reader)
	rec, err := csvReader.Read()
	if err != nil {
		return nil, errors.Wrap(err, "while execOutcomeDecoderFuncBase64CVS")
	}
	if len(rec) != 5 {
		return nil, fmt.Errorf("expect 5 record items but got %d, rec=%v", len(rec), rec)
	}
	outcome := ExecOutcome{}
	outcome.ExitCode, err = base64ToStrToUint8(rec[0])
	if err != nil {
		err = errors.Wrapf(
			err,
			"while ExitCode <- base64ToStrToUint8('%s') records=%v",
			rec[0], rec)
		return nil, err
	}
	outcome.InternalErrTxt, err = base64ToStr((rec[1]))
	if err != nil {
		err = errors.Wrapf(
			err,
			"while InternaleErrTxt <- base64ToStr('%s') records=%v",
			rec[1], rec)
		return nil, err
	}
	outcome.Key, err = base64ToStr((rec[2]))
	if err != nil {
		err = errors.Wrapf(
			err,
			"while Key <- base64ToStr('%s') records=%v",
			rec[2], rec)
		return nil, err
	}
	outcome.Stderr, err = base64ToStr((rec[3]))
	if err != nil {
		err = errors.Wrapf(
			err,
			"while Stderr <- base64ToStr('%s') records=%v",
			rec[3], rec)
		return nil, err
	}
	outcome.Stdout, err = base64ToStr((rec[4]))
	if err != nil {
		err = errors.Wrapf(
			err,
			"while Stdout <- base64ToStr('%s') records=%v",
			rec[4], rec)
		return nil, err
	}
	return &outcome, err
}

func strToBase64(str string) (strBase64 string) {
	if str == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(str))
}

func base64ToStr(base64Str string) (str string, err error) {
	if base64Str == "" {
		return "", nil
	}
	bytes, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func base64ToStrToUint8(base64Str string) (valUin8 uint8, err error) {
	str, err := base64ToStr(base64Str)
	if err != nil {
		return math.MaxUint8, errors.Wrapf(err, "while base64ToStr('%s')", base64Str)
	}
	val64, err := strconv.ParseUint(str, 10, 60)
	if err != nil {
		return math.MaxUint8, errors.Wrapf(err, "while ParseUint('%s'), base64Str=%s", str, base64Str)
	}
	if val64 > math.MaxUint8 {
		return math.MaxUint8, errors.Wrapf(err, "while base64ToStrToUint8 overflow value %d > %d", val64, math.MaxUint8)
	}
	return uint8(val64), nil
}

// StubRequestEncoderFunc return an encoder which can write stubbing request representation into the writer.
func StubRequestEncoderFunc(writer io.Writer) func(outcome *StubRequest) error {
	// return func(req *StubRequest) error {
	// 	enc := gob.NewEncoder(writer)
	// 	return enc.Encode(req)
	// }
	return stubRequestEncoderFuncBase64CVS(writer)
}

// StubRequestDecoderFunc decodes an stubbing request from the given reader.
func StubRequestDecoderFunc(reader io.Reader) (*StubRequest, error) {
	// dec := gob.NewDecoder(reader)
	// req := StubRequest{}
	// err := dec.Decode(&req)
	// return &req, err
	return stubRequestDecoderFuncBase64CVS(reader)
}

func stubRequestEncoderFuncBase64CVS(writer io.Writer) func(req *StubRequest) error {
	return func(req *StubRequest) error {
		writerCSV := csv.NewWriter(writer)
		rec := make([]string, 0, 2+len(req.Args))
		rec = append(rec, strToBase64(req.Key))
		rec = append(rec, strToBase64(req.CmdName))
		for _, arg := range req.Args {
			rec = append(rec, strToBase64(arg))
		}
		err := writerCSV.Write(rec)
		if err != nil {
			return err
		}
		writerCSV.Flush()
		return nil
	}
}

func stubRequestDecoderFuncBase64CVS(reader io.Reader) (*StubRequest, error) {
	csvReader := csv.NewReader(reader)
	rec, err := csvReader.Read()
	if err != nil {
		return nil, errors.Wrapf(err, "while stubRequestDecoderFuncBase64CVS rec=%v, err=%v", rec, err)
	}
	req := StubRequest{}
	req.Key, err = base64ToStr(rec[0])
	if err != nil {
		err = errors.Wrapf(
			err,
			"while Key <- base64ToStrTo('%s') records=%v",
			rec[0], rec)
		return nil, err
	}
	req.CmdName, err = base64ToStr(rec[1])
	if err != nil {
		err = errors.Wrapf(
			err,
			"while CmdName <- base64ToStrTo('%s') records=%v",
			rec[1], rec)
		return nil, err
	}
	if recLen := len(rec); recLen > 2 {
		// because we will be using asignment to transfer rec to args
		// we need an array with the appopriate length
		args := make([]string, recLen-2)
		for i := recLen - 1; i >= 2; i-- {
			recValI := rec[i]
			args[i-2], err = base64ToStr(recValI)
			if err != nil {
				err = errors.Wrapf(
					err,
					"while args[%d] <- base64ToStrTo(rec[%d]='%s') records=%v",
					i-2, i, recValI, rec)
				return nil, err
			}
		}
		req.Args = args
	}

	return &req, nil
}
