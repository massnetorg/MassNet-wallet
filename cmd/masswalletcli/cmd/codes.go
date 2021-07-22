package cmd

const (
	ExitBindPlotInsufficientBalance = 10 + iota
	ExitBindPlotNoUnbound
	ExitBindPlotPartialDone
)

const (
	ExitBindPoolPkInsufficientBalance = 10 + iota
	ExitBindPoolPkNone
	ExitBindPoolPkInvalidMnemonic
)
