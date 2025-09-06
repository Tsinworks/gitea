// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package util

import (
	"bytes"

	"github.com/vmihailenco/msgpack/v5"
)

// PackData uses msgpack to encode the given data in sequence
func PackData(data ...any) ([]byte, error) {
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	for _, datum := range data {
		if err := enc.Encode(datum); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// UnpackData uses msgpack to decode the given data in sequence
func UnpackData(buf []byte, data ...any) error {
	r := bytes.NewReader(buf)
	dec := msgpack.NewDecoder(r)
	for _, datum := range data {
		if err := dec.Decode(datum); err != nil {
			return err
		}
	}
	return nil
}
