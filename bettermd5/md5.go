// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run gen.go -full -output md5block.go

// Package bettermd5 implements the MD5 hash algorithm as defined in RFC 1321.
package bettermd5

import (
	"bytes"
	"encoding/gob"
)

// The size of an MD5 checksum in bytes.
const Size = 16

// The blocksize of MD5 in bytes.
const BlockSize = 64

const (
	chunk = 64
	init0 = 0x67452301
	init1 = 0xEFCDAB89
	init2 = 0x98BADCFE
	init3 = 0x10325476
)

type betterDigestState struct {
	S   [4]uint32
	X   [chunk]byte
	Nx  int
	Len uint64
}

// BetterDigest represents the partial evaluation of a checksum.
type BetterDigest struct {
	s   [4]uint32
	x   [chunk]byte
	nx  int
	len uint64
}

func (d *BetterDigest) Reset() {
	d.s[0] = init0
	d.s[1] = init1
	d.s[2] = init2
	d.s[3] = init3
	d.nx = 0
	d.len = 0
}

// New returns a new hash.Hash computing the MD5 checksum.
func New() *BetterDigest {
	d := new(BetterDigest)
	d.Reset()
	return d
}

// New returns a new hash.Hash computing the MD5 checksum from existing state
func NewFromState(state []byte) *BetterDigest {
	d := new(BetterDigest)
	d.Reset()
	d.SetState(state)
	return d
}

func (d *BetterDigest) GetState() []byte {
	var state bytes.Buffer

	enc := gob.NewEncoder(&state)

	enc.Encode(betterDigestState{
		S:   d.s,
		X:   d.x,
		Nx:  d.nx,
		Len: d.len,
	})

	return state.Bytes()
}

func (d *BetterDigest) SetState(state []byte) error {
	dec := gob.NewDecoder(bytes.NewBuffer(state))

	var s betterDigestState

	err := dec.Decode(&s)

	if err != nil {
		return err
	}

	d.s = s.S
	d.x = s.X
	d.nx = s.Nx
	d.len = s.Len

	return nil
}

func (d *BetterDigest) Size() int { return Size }

func (d *BetterDigest) BlockSize() int { return BlockSize }

func (d *BetterDigest) Write(p []byte) (nn int, err error) {
	nn = len(p)
	d.len += uint64(nn)
	if d.nx > 0 {
		n := len(p)
		if n > chunk-d.nx {
			n = chunk - d.nx
		}
		for i := 0; i < n; i++ {
			d.x[d.nx+i] = p[i]
		}
		d.nx += n
		if d.nx == chunk {
			block(d, d.x[0:chunk])
			d.nx = 0
		}
		p = p[n:]
	}
	if len(p) >= chunk {
		n := len(p) &^ (chunk - 1)
		block(d, p[:n])
		p = p[n:]
	}
	if len(p) > 0 {
		d.nx = copy(d.x[:], p)
	}
	return
}

func (d0 *BetterDigest) Sum(in []byte) []byte {
	// Make a copy of d0 so that caller can keep writing and summing.
	d := *d0
	hash := d.checkSum()
	return append(in, hash[:]...)
}

func (d *BetterDigest) checkSum() [Size]byte {
	// Padding.  Add a 1 bit and 0 bits until 56 bytes mod 64.
	len := d.len
	var tmp [64]byte
	tmp[0] = 0x80
	if len%64 < 56 {
		d.Write(tmp[0 : 56-len%64])
	} else {
		d.Write(tmp[0 : 64+56-len%64])
	}

	// Length in bits.
	len <<= 3
	for i := uint(0); i < 8; i++ {
		tmp[i] = byte(len >> (8 * i))
	}
	d.Write(tmp[0:8])

	if d.nx != 0 {
		panic("d.nx != 0")
	}

	var digest [Size]byte
	for i, s := range d.s {
		digest[i*4] = byte(s)
		digest[i*4+1] = byte(s >> 8)
		digest[i*4+2] = byte(s >> 16)
		digest[i*4+3] = byte(s >> 24)
	}

	return digest
}

// Sum returns the MD5 checksum of the data.
func Sum(data []byte) [Size]byte {
	var d BetterDigest
	d.Reset()
	d.Write(data)
	return d.checkSum()
}
