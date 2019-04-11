package wirepb

import (
	"bytes"
	"encoding/binary"
)

func (m *BlockHeader) Bytes() []byte {
	var version, height, timestamp [8]byte
	binary.LittleEndian.PutUint64(version[:], m.Version)
	binary.LittleEndian.PutUint64(height[:], m.Height)
	binary.LittleEndian.PutUint64(timestamp[:], uint64(m.Timestamp))
	banList := make([]byte, 0)
	for i := 0; i < len(m.BanList); i++ {
		banList = bytes.Join([][]byte{banList, m.BanList[i].Bytes()}, []byte(""))
	}

	return bytes.Join([][]byte{m.ChainID.Bytes(), version[:], height[:], timestamp[:],
		m.Previous.Bytes(), m.TransactionRoot.Bytes(), m.ProposalRoot.Bytes(), m.Target.Bytes(),
		m.Challenge.Bytes(), m.PubKey.Bytes(), m.Proof.Bytes(), m.SigQ.Bytes(),
		m.Sig2.Bytes(), banList}, []byte(""))
}

func (m *BlockHeader) BytesPoC() []byte {
	var version, height, timestamp [8]byte
	binary.LittleEndian.PutUint64(version[:], m.Version)
	binary.LittleEndian.PutUint64(height[:], m.Height)
	binary.LittleEndian.PutUint64(timestamp[:], uint64(m.Timestamp))
	banList := make([]byte, 0)
	for i := 0; i < len(m.BanList); i++ {
		banList = bytes.Join([][]byte{banList, m.BanList[i].Bytes()}, []byte(""))
	}

	return bytes.Join([][]byte{m.ChainID.Bytes(), version[:], height[:], timestamp[:],
		m.Previous.Bytes(), m.TransactionRoot.Bytes(), m.ProposalRoot.Bytes(), m.Target.Bytes(),
		m.Challenge.Bytes(), m.PubKey.Bytes(), m.Proof.Bytes(), m.SigQ.Bytes(),
		banList}, []byte(""))
}

func (m *BlockHeader) BytesChainID() []byte {
	var version, height, timestamp [8]byte
	binary.LittleEndian.PutUint64(version[:], m.Version)
	binary.LittleEndian.PutUint64(height[:], m.Height)
	binary.LittleEndian.PutUint64(timestamp[:], uint64(m.Timestamp))
	banList := make([]byte, 0)
	for i := 0; i < len(m.BanList); i++ {
		banList = bytes.Join([][]byte{banList, m.BanList[i].Bytes()}, []byte(""))
	}

	return bytes.Join([][]byte{version[:], height[:], timestamp[:], m.Previous.Bytes(),
		m.TransactionRoot.Bytes(), m.ProposalRoot.Bytes(), m.Target.Bytes(),
		m.Challenge.Bytes(), m.PubKey.Bytes(), m.Proof.Bytes(), m.SigQ.Bytes(),
		m.Sig2.Bytes(), banList}, []byte(""))
}

func (m *Proof) Bytes() []byte {
	var bl [4]byte
	binary.LittleEndian.PutUint32(bl[:], uint32(m.BitLength))
	return bytes.Join([][]byte{m.X, m.XPrime, bl[:]}, []byte(""))
}
