package wirepb

import (
	"bytes"
	"encoding/binary"
)

func (m *Tx) Bytes() []byte {
	var version [4]byte
	binary.LittleEndian.PutUint32(version[:], uint32(m.Version))
	var lockTime [4]byte
	binary.LittleEndian.PutUint32(lockTime[:], m.LockTime)
	txIns := make([]byte, 0)
	for i := 0; i < len(m.TxIn); i++ {
		txIns = bytes.Join([][]byte{txIns, m.TxIn[i].Bytes()}, []byte(""))
	}
	txOuts := make([]byte, 0)
	for i := 0; i < len(m.TxOut); i++ {
		txOuts = bytes.Join([][]byte{txOuts, m.TxOut[i].Bytes()}, []byte(""))
	}
	return bytes.Join([][]byte{version[:], txIns, txOuts, lockTime[:], m.Payload}, []byte(""))
}

func (m *Tx) BytesNoWitness() []byte {
	var version [4]byte
	binary.LittleEndian.PutUint32(version[:], uint32(m.Version))
	var lockTime [4]byte
	binary.LittleEndian.PutUint32(lockTime[:], m.LockTime)
	txInsNoWitness := make([]byte, 0)
	for i := 0; i < len(m.TxIn); i++ {
		txInsNoWitness = bytes.Join([][]byte{txInsNoWitness, m.TxIn[i].BytesNoWitness()}, []byte(""))
	}
	txOuts := make([]byte, 0)
	for i := 0; i < len(m.TxOut); i++ {
		txOuts = bytes.Join([][]byte{txOuts, m.TxOut[i].Bytes()}, []byte(""))
	}
	return bytes.Join([][]byte{version[:], txInsNoWitness, txOuts, lockTime[:], m.Payload}, []byte(""))
}

func (m *TxIn) Bytes() []byte {
	var sequence [4]byte
	binary.LittleEndian.PutUint32(sequence[:], m.Sequence)
	witness := make([]byte, 0)
	for i := 0; i < len(m.Witness); i++ {
		witness = bytes.Join([][]byte{witness, m.Witness[i]}, []byte(""))
	}
	return bytes.Join([][]byte{m.PreviousOutPoint.Bytes(), witness, sequence[:]}, []byte(""))
}

func (m *TxIn) BytesNoWitness() []byte {
	var sequence [4]byte
	binary.LittleEndian.PutUint32(sequence[:], m.Sequence)
	return bytes.Join([][]byte{m.PreviousOutPoint.Bytes(), sequence[:]}, []byte(""))
}

func (m *OutPoint) Bytes() []byte {
	var index [4]byte
	binary.LittleEndian.PutUint32(index[:], m.Index)
	return bytes.Join([][]byte{m.Hash.Bytes(), index[:]}, []byte(""))
}

func (m *TxOut) Bytes() []byte {
	var value [8]byte
	binary.LittleEndian.PutUint64(value[:], uint64(m.Value))
	return bytes.Join([][]byte{value[:], m.PkScript}, []byte(""))
}
