package wirepb

import (
	"bytes"
	"encoding/binary"
)

func (m *ProposalArea) Bytes() []byte {
	punishments := make([]byte, 0)
	for i := 0; i < len(m.Punishments); i++ {
		punishments = bytes.Join([][]byte{punishments, m.Punishments[i].Bytes()}, []byte(""))
	}
	placeholder := make([]byte, 0)
	if len(m.Punishments) == 0 {
		placeholder = m.PlaceHolder.Bytes()
	}
	proposals := make([]byte, 0)
	for i := 0; i < len(m.OtherProposals); i++ {
		proposals = bytes.Join([][]byte{proposals, m.OtherProposals[i].Bytes()}, []byte(""))
	}
	return bytes.Join([][]byte{punishments, placeholder, proposals}, []byte(""))
}

func (m *Punishment) Bytes() []byte {
	var version [4]byte
	binary.LittleEndian.PutUint32(version[:], uint32(m.Version))
	var mType [4]byte
	binary.LittleEndian.PutUint32(mType[:], uint32(m.Type))
	return bytes.Join([][]byte{version[:], mType[:], m.TestimonyA.Bytes(), m.TestimonyB.Bytes()}, []byte(""))
}

func (m *Proposal) Bytes() []byte {
	var version [4]byte
	binary.LittleEndian.PutUint32(version[:], uint32(m.Version))
	var mType [4]byte
	binary.LittleEndian.PutUint32(mType[:], uint32(m.Type))
	return bytes.Join([][]byte{version[:], mType[:], m.Content}, []byte(""))
}
