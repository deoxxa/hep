// Copyright 2017 The go-hep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rootio

import (
	"bytes"
	B "encoding/binary"
	"reflect"
)

func (k *Key) DecodeVector(in *bytes.Buffer, dst interface{}) (int, error) {
	// Discard three int16s (like 40 00 00 0e 00 09)
	x := in.Next(6)
	_ = x // sometimes we want to look at this.

	var n int32
	err := B.Read(in, B.BigEndian, &n)
	if err != nil {
		return -1, err
	}

	err = B.Read(in, B.BigEndian, reflect.ValueOf(dst).Slice(0, int(n)).Interface())
	if err != nil {
		return -1, err
	}
	return int(n), nil
}
