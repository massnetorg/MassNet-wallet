package api

const (
	// transaction err
	ErrAPINoTxInfo           = 1101
	ErrAPIRawTx              = 1102
	ErrAPIUserTxFee          = 1103
	ErrAPIGetStakingTxDetail = 1105
	ErrAPISignRawTx          = 1106
	ErrAPIUnspendable        = 1107
	ErrAPIDoubleSpend        = 1108
	ErrAPIOverfullInputs     = 1109
	ErrAPIBigTransactionFee  = 1110

	// block err
	ErrAPINewestHash          = 1201
	ErrAPIBlockHeaderNotFound = 1202
	ErrAPIBlockNotFound       = 1203

	// wallet err
	ErrAPINoAddressInWallet         = 1301
	ErrAPIGetAddresses              = 1302
	ErrAPINoWalletInUse             = 1303
	ErrAPIInsufficientWalletBalance = 1304
	ErrAPIOutputNotExist            = 1305
	ErrAPIWalletUnready             = 1306
	ErrAPIGapLimit                  = 1307
	ErrAPIUnusedAddressLimit        = 1308
	ErrAPIChangePassUnsupported     = 1309
	ErrAPIWalletUnlocked            = 1310
	ErrAPIInvalidKeystoreVersion    = 1311

	// txScript
	ErrAPIRejectTx          = 1401
	ErrAPITxAlreadyExists   = 1402
	ErrAPINonStandardTxSize = 1403

	// Invalid Parameter
	ErrAPIInvalidParameter       = 1501
	ErrAPIInvalidLockTime        = 1502
	ErrAPIInvalidAmount          = 1503
	ErrAPIInvalidAddress         = 1504
	ErrAPIInvalidFlag            = 1505
	ErrAPIInvalidTxHex           = 1506
	ErrAPIInvalidPassphrase      = 1507
	ErrAPIInvalidOldPassphrase   = 1508
	ErrAPIInvalidNewPassphrase   = 1509
	ErrAPIInvalidSeed            = 1510
	ErrAPIInvalidWalletId        = 1511
	ErrAPIInvalidKeystoreJson    = 1512
	ErrAPIInvalidVersion         = 1513
	ErrAPIDuplicateSeed          = 1514
	ErrAPIPrivPassSameAsPubPass  = 1515
	ErrAPIInvalidMnemonic        = 1516
	ErrAPIInvalidBitSize         = 1517
	ErrAPIInvalidTxId            = 1518
	ErrAPIInvalidTxHistoryCount  = 1519
	ErrAPIMismatchedKeystoreJson = 1520
	ErrAPIUnknownSubfeefrom      = 1521
	ErrAPIDustChange             = 1522
	ErrAPIDustAmount             = 1523
	ErrAPINotEnoughInputs        = 1524

	// other err
	ErrAPIUnknownErr      = 1701
	ErrAPIQueryDataFailed = 1702
	ErrAPIAbnormalData    = 1703
	ErrAPIUnacceptable    = 1704
)

var ErrCode = map[uint32]string{
	ErrAPINoTxInfo:                  "No information available about transaction",
	ErrAPIGetAddresses:              "Failed to get addresses",
	ErrAPINoAddressInWallet:         "There is no address in the wallet",
	ErrAPIInvalidParameter:          "Invalid parameter",
	ErrAPIInvalidLockTime:           "Invalid locktime",
	ErrAPIInvalidAmount:             "Invalid amount",
	ErrAPIInvalidAddress:            "Invalid address",
	ErrAPIInvalidFlag:               "Invalid flag of sighash parameter",
	ErrAPIRejectTx:                  "Reject receive transaction",
	ErrAPINewestHash:                "Failed to get newest hash",
	ErrAPIRawTx:                     "Failed to create raw transaction",
	ErrAPIBlockHeaderNotFound:       "Failed to find block header",
	ErrAPIBlockNotFound:             "Failed to find block",
	ErrAPIUnknownErr:                "Unknown error",
	ErrAPIUserTxFee:                 "Invalid userTxFee",
	ErrAPIGetStakingTxDetail:        "Failed to query staking tx detail",
	ErrAPIInvalidPassphrase:         "Invalid passphrase",
	ErrAPIInvalidOldPassphrase:      "Invalid old passphrase",
	ErrAPIInvalidNewPassphrase:      "Invalid new passphrase",
	ErrAPIInvalidSeed:               "Invalid seed",
	ErrAPIInvalidWalletId:           "Invalid walletId",
	ErrAPIInvalidKeystoreJson:       "Invalid keystore json",
	ErrAPINoWalletInUse:             "No wallet in use",
	ErrAPIInvalidVersion:            "Invalid version",
	ErrAPIInsufficientWalletBalance: "Insufficient wallet balance",
	ErrAPIInvalidTxHex:              "Invalid txHex",
	ErrAPIOutputNotExist:            "Output not exist",
	ErrAPIDuplicateSeed:             "Duplicated wallet",
	ErrAPIPrivPassSameAsPubPass:     "New private passphrase same as public passphrase",
	ErrAPIWalletUnready:             "Wallet is unready, need to wait util wallet imported",
	ErrAPITxAlreadyExists:           "Transaction already exists",
	ErrAPINonStandardTxSize:         "Transaction size is larger than max allowed size",
	ErrAPIInvalidMnemonic:           "Invalid mnemonic",
	ErrAPIInvalidBitSize:            "Invalid bit size",
	ErrAPIGapLimit:                  "Too many unused addresses",
	ErrAPIInvalidTxId:               "Invalid transaction id",
	ErrAPIInvalidTxHistoryCount:     "Invalid count for transaction history",
	ErrAPIMismatchedKeystoreJson:    "Keystore json does not match the client or network",
	ErrAPIInvalidKeystoreVersion:    "Invalid keystore version",
	ErrAPIUnknownSubfeefrom:         "Unknown subtractfeefrom",
	ErrAPIDustChange:                "Change is dust",
	ErrAPIDustAmount:                "Amount is dust",
	ErrAPINotEnoughInputs:           "Not enough inputs",

	ErrAPISignRawTx:             "Failed to sign raw transaction",
	ErrAPIQueryDataFailed:       "Query for data failed",
	ErrAPIAbnormalData:          "Abnormal data",
	ErrAPIUnusedAddressLimit:    "Too many unused address",
	ErrAPIUnspendable:           "Unspendable output",
	ErrAPIDoubleSpend:           "Output already spent",
	ErrAPIOverfullInputs:        "Overfull inputs",
	ErrAPIChangePassUnsupported: "Unsupported to change passphrase of current wallet",
	ErrAPIWalletUnlocked:        "Wallet unlocked, try again later",
	ErrAPIBigTransactionFee:     "Big transaction fee",
	ErrAPIUnacceptable:          "Request is unacceptable",
}
