package wirepb

import "bytes"

func (m *Block) Bytes() []byte {
	txs := make([]byte, 0)
	for i := 0; i < len(m.Transactions); i++ {
		txs = bytes.Join([][]byte{txs, m.Transactions[i].Bytes()}, []byte(""))
	}
	return bytes.Join([][]byte{m.Header.Bytes(), m.Proposals.Bytes(), txs}, []byte(""))
}
