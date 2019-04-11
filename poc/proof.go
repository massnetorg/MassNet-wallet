package poc

type Proof struct {
	X         []byte
	X_prime   []byte
	BitLength int
}

func NewEmptyProof() *Proof {
	return &Proof{
		X:         make([]byte, 0),
		X_prime:   make([]byte, 0),
		BitLength: 0,
	}
}
