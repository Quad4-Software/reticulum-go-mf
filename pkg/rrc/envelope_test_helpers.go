// SPDX-License-Identifier: 0BSD
package rrc

import "github.com/fxamacker/cbor/v2"

func mustMarshalWithExtra(e Envelope, key uint64, val any) ([]byte, error) {
	m := map[uint64]any{
		KeyVersion:   e.Version,
		KeyType:      e.Type,
		KeyMsgID:     e.MsgID,
		KeyTimestamp: e.Timestamp,
		KeySender:    e.Sender,
		key:          val,
	}
	return cbor.Marshal(m)
}

func mustMarshalWithVersion(e *Envelope, ver uint64) ([]byte, error) {
	m := map[uint64]any{
		KeyVersion:   ver,
		KeyType:      e.Type,
		KeyMsgID:     e.MsgID,
		KeyTimestamp: e.Timestamp,
		KeySender:    e.Sender,
	}
	return cbor.Marshal(m)
}
