package masswallet

import (
	"massnet.org/mass-wallet/logging"
)

const (
	MaxWaitingTaskNum = 3

	WalletTaskImport = iota
	WalletTaskRemove
)

type WalletTask struct {
	taskType int
	walletId string
}

type WalletTaskChan struct {
	C chan WalletTask
}

func NewWalletTaskChan(size int) *WalletTaskChan {
	if size < MaxWaitingTaskNum+1 {
		size = MaxWaitingTaskNum + 1
	}
	return &WalletTaskChan{
		C: make(chan WalletTask, size),
	}
}

func (c *WalletTaskChan) IsBusy() bool {
	return len(c.C) >= MaxWaitingTaskNum
}

func (c *WalletTaskChan) PushImport(walletId string) {
	select {
	case c.C <- WalletTask{
		taskType: WalletTaskImport,
		walletId: walletId,
	}:
	default:
		logging.CPrint(logging.ERROR, "PushImport failed", logging.LogFormat{"walletId": walletId})
	}
}

func (c *WalletTaskChan) PushRemove(walletId string) {
	select {
	case c.C <- WalletTask{
		taskType: WalletTaskRemove,
		walletId: walletId,
	}:
	default:
		logging.CPrint(logging.ERROR, "PushRemove failed", logging.LogFormat{"walletId": walletId})
	}
}
