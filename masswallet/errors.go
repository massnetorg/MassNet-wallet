package masswallet

import "errors"

var (
	ErrShaHashFromStr        = errors.New("Failed to decode hash from string")
	ErrInvalidAmount         = errors.New("Invalid amount")
	ErrFailedDecodeAddress   = errors.New("Failed to decode address")
	ErrInvalidAddress        = errors.New("Invalid address")
	ErrInvalidStakingAddress = errors.New("Invalid staking address")
	ErrNet                   = errors.New("Mismatched network")
	ErrCreatePkScript        = errors.New("Failed to create pkScript")
	ErrInvalidLockTime       = errors.New("Invalid locktime")
	ErrInvalidParameter      = errors.New("Invalid parameter")
	ErrEncode                = errors.New("Failed to encode data")
	ErrInsufficientFunds     = errors.New("Insufficient funds")
	ErrOverfullUtxo          = errors.New("Overfull utxo")
	ErrInvalidFlag           = errors.New("Invalid sighash parameter")
	ErrInvalidIndex          = errors.New("Invalid OutPoint index")
	ErrDoubleSpend           = errors.New("Output already spent")
	ErrUnknownSubfeefrom     = errors.New("Unknown subtractfeefrom")
	ErrDustChange            = errors.New("Change is dust")
	ErrDustAmount            = errors.New("Amount is dust")
	ErrNotEnoughInputs       = errors.New("Not enough inputs")

	ErrSignWitnessTx = errors.New("Failed to sign witness tx")

	ErrNoWalletInUse     = errors.New("no wallet in use")
	ErrIllegalReorgBlock = errors.New("illegal reorg block")
	ErrNilDB             = errors.New("db is nil")
	ErrChangeInUseWallet = errors.New("failed to change in-use wallet")
	ErrInvalidVersion    = errors.New("unknown version")
	ErrNoAddressInWallet = errors.New("no address in wallet")
	ErrUTXONotExists     = errors.New("utxo not exists")

	ErrImportingContinuable = errors.New("importing continuable")
	ErrWalletUnready        = errors.New("wallet is unready")
	ErrTooManyTask          = errors.New("too many task")
	ErrTaskAbort            = errors.New("task abort")
)
