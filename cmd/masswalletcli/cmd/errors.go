package cmd

import "errors"

const (
	LogMsgIncorrectArgsNumber = "incorrect number of arguments"
)

var (
	ErrInvalidArgument = errors.New("invalid argument")
)
