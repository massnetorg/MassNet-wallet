// Modified for MassNet
// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"
)

const CommandSize = 12

const MaxMessagePayload = (1024 * 1024 * 32) // 32MB

const (
	CmdBlock  = "block"
	CmdTx     = "tx"
	CmdReject = "reject"
)

type MessageEncoding uint32

const (
	BaseEncoding MessageEncoding = 1 << iota
	WitnessEncoding
)

type Message interface {
	MassDecode(io.Reader, CodecMode) error
	MassEncode(io.Writer, CodecMode) error
	Command() string
	MaxPayloadLength(uint32) uint32
}
