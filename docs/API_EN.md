# Endpoint
| URL | Protocol |
| ------ | ------ |
| http://localhost:9688 | HTTP |

# API methods
* [GetBestBlock](#getbestblock)
* [GetClientStatus](#getclientstatus)
* [Wallets](#wallets)
* [CreateWallet](#createwallet)
* [UseWallet](#usewallet)
* [ImportWallet](#importwallet)
* [ImportWalletWithMnemonic](#importwalletwithmnemonic)
* [ExportWallet](#exportwallet)
* [RemoveWallet](#removewallet)
* [GetWalletMnemonic](#getwalletmnemonic)
* [GetWalletBalance](#getwalletbalance)
* [CreateAddress](#createaddress)
* [GetAddresses](#getaddresses)
* [GetAddressBalance](#getaddressbalance)
* [ValidateAddress](#validateaddress)
* [GetUtxo](#getutxo)
* [CreateRawTransaction](#createrawtransaction)
* [AutoCreateTransaction](#autocreatetransaction)
* [SignRawTransaction](#signrawtransaction)
* [GetTransactionFee](#gettransactionfee)
* [SendRawTransaction](#sendrawtransaction)
* [GetRawTransaction](#getrawtransaction)
* [GetTxStatus](#gettxstatus)
* [CreateStakingTransaction](#createstakingtransaction)
* [GetStakingHistory](#getstakinghistory)
* [GetLatestRewardList](#getlatestrewardlist)
* [TxHistory](#txhistory)
* [GetAddressBinding](#getaddressbinding)
* [GetBindingHistory](#getbindinghistory)
* [CreateBindingTransaction](#createbindingtransaction)
---

## GetBestBlock
    GET /v1/blocks/best
### Parameters
null
### Returns
- `Integer` - height
- `String` - target, mining difficulty
### Example
```json
{
    "height": "176989",
    "target": "b173f7b71"
}
```

## GetClientStatus
    GET /v1/client/status
### Parameters
null
### Returns
- `Boolean` - peer_listening
- `Boolean` - syncing 
- `String` - chain_id 
- `Integer` - local_best_height 
- `Integer` - known_best_height 
- `Integer` - wallet_sync_height
- `Object`, peer_count
    - `Integer` - total    
    - `Integer` - outbound 
    - `Integer` - inbound           
- `Object`, peers
    - peerList
        - `Array of peerInfo` - outbound
        - `Array of peerInfo` - inbound 
        - `Array of peerInfo` - other   
        - peerInfo
            - `String` - id       
            - `String` - address   
            - `String` - direction 
### Example
```json
{
    "peer_listening": true,
    "syncing": false,
    "chain_id": "e931abb77f2568f752a29ed28d442558764a5961ed773df7188430a0e0f7cf18",
    "local_best_height": "176992",
    "known_best_height": "176992",
    "wallet_sync_height": "176992",
    "peer_count": {
        "total": 2,
        "outbound": 2,
        "inbound": 0
    },
    "peers": {
        "outbound": [
            {
                "id": "0A6AFB3678A1612296AA5FD4338AF9304EA8831455DDC014D3F554357BBBC2EE",
                "address": "39.99.32.37:43453",
                "direction": "outbound"
            },
            {
                "id": "B3664A9AC4AF1DBB457BB82F2F856F25DDE1F9F226D51BCA94A7F71123839100",
                "address": "39.104.206.48:43453",
                "direction": "outbound"
            }
        ],
        "inbound": [],
        "other": []
    }
}
```

# Wallets
    GET /v1/wallets
### Parameters
null
### Returns
 - `Array of WalletSummary`, wallets
    - WalletSummary
        - `String` - wallet_id
        - `Integer` - type             // default 1
        - `String` - remarks
        - `Boolean` - ready              // false-importing, true-importing completed
        - `Integer` - synced_height   // indicates processed blocks height when *ready* is false
### Example
```json
{
    "wallets": [
        {
            "wallet_id": "ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz",
            "type": 1,
            "remarks": "init",
            "ready": true,
            "synced_height": "0"
        },
        {
            "wallet_id": "ac10nge8pha03mdp32ndhtxr7lmscc4s0lkg9eee2j",
            "type": 1,
            "remarks": "init-2",
            "ready": true,
            "synced_height": "0"
        }
    ]
}
```

## CreateWallet
    POST /v1/wallets/create
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| passphrase | string |  |  |
| remarks | string |  |  optional |
| bit_size | int |  |  optional. length of entropy, should be a multiple of 32 between 128 and 256; if not set, it will be the default value(128) |

### Returns
- `String` - wallet_id 
- `String` - mnemonic 
### Example
```json
// Reqeust
{
	"passphrase":"123456",
	"remarks":"init-2",
	"bit_size":192
}

// Response
{
    "wallet_id": "ac10nge8pha03mdp32ndhtxr7lmscc4s0lkg9eee2j",
    "mnemonic": "tribe belt hand odor beauty pelican switch pluck toe pigeon zero future acoustic enemy panda twice endless motion"
}
```

## UseWallet
    POST /v1/wallets/use
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| wallet_id | string | wallet id |  |
### Return
- `String` - chain_id 
- `String` - wallet_id 
- `Integer` - type 
- `String` - total_balance      // include the amount can't be spent yet
- `Integer` - external_key_count 
- `Integer` - internal_key_count 
- `String` - remarks 
### Example
```json
// Request
{
	"wallet_id":"ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz"
}

// Response
{
    "chain_id": "e931abb77f2568f752a29ed28d442558764a5961ed773df7188430a0e0f7cf18",
    "wallet_id": "ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz",
    "type": 1,
    "total_balance": "48994.88593426",
    "external_key_count": 5,
    "internal_key_count": 0,
    "remarks": "init"
}
```

## ImportWallet
    POST /v1/wallets/import
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| keystore | string |  |  |
| passphrase | string |  |  |
### Returns
- `Boolean` - ok 
- `String` - wallet_id 
- `Integer` - type 
- `String` - remarks 
### Example
```json
// Request
{
	"keystore": "{\"remarks\":\"init\",\"crypto\":{\"cipher\":\"Stream cipher\",\"entropyEnc\":\"408998f673619fafe25a820588f12c0b9fed25a0ec2fad33128abc62644cd9d80c5e9f2f1f23df1862058ff7622bb097185c45f6b59697ec\",\"kdf\":\"scrypt\",\"privParams\":\"9d5d2f6de075ed1f8c46d590a823c67bcbdb25159ba3caf50426c27b575821a95daa891a93be42c900f40c1c6f1ae72c19cf3ffbefe45bb3b67643988a517cb2000004000000000008000000000000000100000000000000\",\"cryptoKeyEntropyEnc\":\"8b5d8cf78697d88c7a9e3143862c8db45b7a9729e5976df99ef586c7ebfd3b35a3ab2d82b606eaa9ca1f7c7b0bf21a585e87aec423e48c1e4d0d45745b5a7d4ae5c1c688c2cd9ca1\"},\"hdPath\":{\"Purpose\":44,\"Coin\":297,\"Account\":1,\"ExternalChildNum\":5,\"InternalChildNum\":0}}",
	"passphrase": "111111"
}

// Response
{
    "ok": true,
    "wallet_id": "ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz",
    "type": 1,
    "remarks": "init"
}
```

## ImportWalletWithMnemonic
    POST /v1/wallets/import/mnemonic
### Parameter
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| mnemonic | string |  |  |
| passphrase | string |  |  |
| remarks | string |  |  |
| external_index | int | init external address num |  |
| internal_index | int | init internal address num |  |

### Returns
- `Boolean` - ok 
- `String` - wallet_id 
- `Integer` - type 
- `String` - remarks 
### Example
```json
// Request
{
	"mnemonic":"tribe belt hand odor beauty pelican switch pluck toe pigeon zero future acoustic enemy panda twice endless motion",
	"passphrase":"123456",
	"remarks":"init-2"
}

// Response
{
    "ok": true,
    "wallet_id": "ac10nge8pha03mdp32ndhtxr7lmscc4s0lkg9eee2j",
    "type": 1,
    "remarks": "init-2"
}
```

## ExportWallet
    POST /v1/wallets/export
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| wallet_id | string |  |  |
| passphrase | string |  |  |

### Returns
- `String` - keystore 
### Example
```json
// Request
{
	"wallet_id":"ac10nge8pha03mdp32ndhtxr7lmscc4s0lkg9eee2j",
	"passphrase":"123456"
}

// Response
{
    "keystore": "{\"remarks\":\"init-2\",\"crypto\":{\"cipher\":\"Stream cipher\",\"entropyEnc\":\"8e5d6c3fba1bd23a75fd545287f41828a0f7d1c75c8e3166cbc266d0ffb95997764ecc631b995c3b4696aaf7c58c6e887fc0b89ebf4ccfd0f3f82d4c33913650\",\"kdf\":\"scrypt\",\"privParams\":\"551147d50b72305cf0769f3f524e67a9ebda3fb256aaedb53c43dc5b24e99c2bb2c39425fa4fe08afafda88eb2a096e3395c499bae8aafe4bc6436ee70c0a150000004000000000008000000000000000100000000000000\",\"cryptoKeyEntropyEnc\":\"61089855ec95a5f0214506aefcd2f633ef330a774698d1e8a465dc86f68146c13dd95eb562012a8601aed6f8c3803d4283bd8b8ecd2613629a272c5911a5449aa002254c147ff3c2\"},\"hdPath\":{\"Purpose\":44,\"Coin\":297,\"Account\":1,\"ExternalChildNum\":0,\"InternalChildNum\":0}}"
}
```

## RemoveWallet
    POST /v1/wallets/remove
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| wallet_id | string |  |  |
| passphrase | string |  |  |
### Returns
- `Boolean` - ok 
### Example
```json
// Request
{
	"wallet_id":"ac10nge8pha03mdp32ndhtxr7lmscc4s0lkg9eee2j",
	"passphrase":"123456"
}

// Response
{
    "ok": true
}
```

## GetWalletMnemonic
    POST /v1/wallets/mnemonic
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| wallet_id | string |  |  |
| passphrase | string |  |  |
### Returns
- `String` - mnemonic 
### Example
```json
// Request
{
	"wallet_id": "ac10nge8pha03mdp32ndhtxr7lmscc4s0lkg9eee2j",
	"passphrase":"123456"
}

// Response
{
    "mnemonic": "tribe belt hand odor beauty pelican switch pluck toe pigeon zero future acoustic enemy panda twice endless motion"
}
```

## GetWalletBalance
    POST /v1/wallets/current/balance
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| required_confirmations | int | only filter utxos that have been confirmed no less than `required_confirmations` |  |
| detail | bool | whether to return details |  |
### Returns
- `String` - wallet_id 
- `String` - total 
- `Object`, detail 
    - `String` - spendable string
    - `String` - withdrawable_staking
    - `String` - withdrawable_binding
### Example
```json
// Request
{
	"required_confirmations":1,
	"detail":true
}

// Response
{
    "wallet_id": "ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz",
    "total": "48994.88593426",
    "detail": {
        "spendable": "48994.77943826",
        "withdrawable_staking": "0",
        "withdrawable_binding": "0.106496"
    }
}
```

## CreateAddress
    POST /v1/addresses/create
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| version | int | which type of address to create | 0-standard address, 1-staking address |
### Returns
- `String` - address 
### Example
```json
// Request
{
	"version":0
}

// Response
{
    "address": "ms1qqc7773md3ux8wkha6td2q9vcxfae39xvuzgj063q4l2mwymp2h0aqunux9z"
}
```

## GetAddresses
    GET /v1/addresses/{version}
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| version | int | which type of address to query | 0-standard address, 1-staking address |
### Returns-
-  []AddressDetail details
    -  AddressDetail
         - `String` - address 
         - `Integer` - version   
         - `Boolean` - used               // whether there is a transaction related to this address on the main chain
         - `String` - std_address      // withdrawal address corresponding to staking address, omitted when *version* is 0
### Example
```json
{
    "details": [
        {
            "address": "ms1qqc7773md3ux8wkha6td2q9vcxfae39xvuzgj063q4l2mwymp2h0aqunux9z",
            "version": 0,
            "used": false,
            "std_address": ""
        }
    ]
}
```

## GetAddressBalance
    POST /v1/addresses/balance
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| required_confirmations | int |  |  |
| addresses | array<string> | which addresses to query | optional, if not provided, balances of all addresses will be returned |
### Returns
- `Array of AddressAndBalance`, balances
    - AddressAndBalance
        - `String` - address 
        - `String` - total
        - `String` - spendable
        - `String` - withdrawable_staking
        - `String` - withdrawable_binding
### Example
```json
// Request
{
	"required_confirmations":1,
	"addresses":[ "ms1qqehh47s0hvzrqqjl77ayj78yytstjkrsltcna343p8yg7ndskvveql4z3vl"]
}

// Response
{
    "balances": [
        {
            "address": "ms1qqehh47s0hvzrqqjl77ayj78yytstjkrsltcna343p8yg7ndskvveql4z3vl",
            "total": "0.053248",
            "spendable": "0",
            "withdrawable_staking": "0",
            "withdrawable_binding": "0.053248"
        }
    ]
}
```

## ValidateAddress
    GET /v1/addresses/{address}/validate
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| address | string |  |  |
### Returns
- `Boolean` - is_valid 
- `Boolean` - is_mine,whether this address belongs to current wallet
- `String` - address 
- `Integer` - version 
### Example
```json
{
    "is_valid": true,
    "is_mine": true,
    "address": "ms1qqc7773md3ux8wkha6td2q9vcxfae39xvuzgj063q4l2mwymp2h0aqunux9z",
    "version": 0
}
```

## GetUtxo
    POST /v1/addresses/utxos
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| address | string |  | return utxo of all addresses if this parameter is null |
### Returns
- `Array of AddressUTXO`, address_utxos
    - AddressUTXO
        - `String` - address 
        - `Array of UTXO`, utxos
            - `String` - tx_id 
            - `Integer` - vout 
            - `String` - amount, in MASS
            - `Integer` - block_height 
            - `Integer` - maturity 
            - `Integer` - confirmations, number of blocks that this utxo has been confirmed since packing
            - `Boolean` - spent_by_unmined
### Example
```json
// Request
{
	"addresses":["ms1qqehh47s0hvzrqqjl77ayj78yytstjkrsltcna343p8yg7ndskvveql4z3vl"]
}

// Response
{
    "address_utxos": [
        {
            "address": "ms1qqehh47s0hvzrqqjl77ayj78yytstjkrsltcna343p8yg7ndskvveql4z3vl",
            "utxos": [
                {
                    "tx_id": "9e4c191a29a4eb018d7904ca1cd0d6f1568356426f0a4a1c5f388c91b768d80e",
                    "vout": 0,
                    "amount": "0.026624",
                    "block_height": "117649",
                    "maturity": 0,
                    "confirmations": 59412,
                    "spent_by_unmined": false
                },
                {
                    "tx_id": "9e4c191a29a4eb018d7904ca1cd0d6f1568356426f0a4a1c5f388c91b768d80e",
                    "vout": 1,
                    "amount": "0.026624",
                    "block_height": "117649",
                    "maturity": 0,
                    "confirmations": 59412,
                    "spent_by_unmined": false
                }
            ]
        }
    ]
}
```

## CreateRawTransaction
    POST /v1/transactions/create
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| inputs | []TransactionInput |  |  |
| amounts | map<string,string> |  | key is paid address, value is in unit MASS  |
| lock_time | int |  | optional.  |
 - TransactionInput
    - `String` - tx_id 
    - `Integer` - vout 
### Returns
- `String` - hex 
### Example
```json
// Request
{
	"inputs":[
		{
			"tx_id": "0054de1e7262dd1238df8283fc2cc940a038502bfb6b03ee7a761b82816f63d2",
            "vout": 9
		},{
			"tx_id": "00d52e0ff62c35c4c7b66e163261fd00278c5815b55375ff75552b4e4ee82db1",
            "vout": 9
		}
	],
	"amounts":{
		"ms1qqc7773md3ux8wkha6td2q9vcxfae39xvuzgj063q4l2mwymp2h0aqunux9z":"200.00000001"
	}
}

// Response
{
    "hex": "080112330a280a240912dd62721ede54001140c92cfc8382df3819ee036bfb2b5038a021d2636f81821b767a100919ffffffffffffffff12330a280a2409c4352cf60f2ed5001100fd6132166eb6c719ff7553b515588c2721b12de84e4e2b5575100919ffffffffffffffff1a2a088190dfc04a12220020c7bde8edb1e18eeb5fba5b5402b3064f7312999c1224fd4415fab6e26c2abbfa"
}
```

## AutoCreateTransaction
    POST /v1/transactions/create/auto
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| amounts | map<string,string> |  | key is paid address, value is in unit MASS |
| lock_time | int |  | optional.|
| fee | string |  | optional. |
| from_address | string | who will pay for this transaction | optional. |
### Returns
- `String` - hex 
### Example
```json
// Request
{
    "amounts":{
        "ms1qqc7773md3ux8wkha6td2q9vcxfae39xvuzgj063q4l2mwymp2h0aqunux9z":"100.00000001"
    },
    "fee": "0.0001"
}

// Response
{
    "hex": "080112330a280a2409429e0e2dd1493404117ca032badd059e1f19e2651e53570b235c21fbe2d10d152dc48e100819ffffffffffffffff1a2a0881c8afa02512220020c7bde8edb1e18eeb5fba5b5402b3064f7312999c1224fd4415fab6e26c2abbfa1a2a08f2b9ddbe0112220020409ce12fde171d5824fc07100951dac06daff7fd8560c36a5dc29f690ee471a2"
}
```

## SignRawTransaction
    POST /v1/transactions/sign
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| raw_tx | string |  |  |
| flags | string |  | optional. default "`ALL`"(else-`NONE`、`SINGLE`、`ALL|ANYONECANPAY`、`NONE|ANYONECANPAY`、`SINGLE|ANYONECANPAY`) |
| passphrase | string |  |  |
### Returns
- `String` - hex 
- `Boolean` - complete 
### Example
```json
// Request
{
	"raw_tx":"080112330a280a240912dd62721ede54001140c92cfc8382df3819ee036bfb2b5038a021d2636f81821b767a100919ffffffffffffffff12330a280a2409c4352cf60f2ed5001100fd6132166eb6c719ff7553b515588c2721b12de84e4e2b5575100919ffffffffffffffff1a2a088190dfc04a12220020c7bde8edb1e18eeb5fba5b5402b3064f7312999c1224fd4415fab6e26c2abbfa",
	"passphrase": "111111"
}

// Response
{
    "hex": "080112a4010a280a240912dd62721ede54001140c92cfc8382df3819ee036bfb2b5038a021d2636f81821b767a10091248473044022003161aa740d89984ef995103735bc6f6a0e0db76bb4eb224914bb797cf9df9ab02202765b0dd7ecb4bf5835e1a1bdce6686b26b3f6e37977668aaefdfa9a29e0a5f4011225512103d0cd7443a5e8dcc030793bea363fe328c84d2daf75f0f2db17d36c07642777b151ae19ffffffffffffffff12a4010a280a2409c4352cf60f2ed5001100fd6132166eb6c719ff7553b515588c2721b12de84e4e2b55751009124847304402205f3a8d2ea86971a7cebba0a07aeb93372732bcefc4e566e8d29009a8cc5598720220759fd2b87292cc9633f16e151d6d34e28dfbfde6b35fa329177b985f80388c14011225512103d0cd7443a5e8dcc030793bea363fe328c84d2daf75f0f2db17d36c07642777b151ae19ffffffffffffffff1a2a088190dfc04a12220020c7bde8edb1e18eeb5fba5b5402b3064f7312999c1224fd4415fab6e26c2abbfa",
    "complete": true
}
```

## GetTransactionFee
    POST /v1/transactions/fee
- ### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| amounts | map<string,string> |  |  |
| lock_time | int |  |  |
| inputs | []TransactionInput |  |  |
| has_binding | bool | whether transaction contains binding output |  |
- TransactionInput
     - `String` - tx_id 
     - `Integer` - vout 
### Returns
- `String` - fee 
### Example
```json
// Request
{
	"inputs":[],
	"amounts":{
		"ms1qqc7773md3ux8wkha6td2q9vcxfae39xvuzgj063q4l2mwymp2h0aqunux9z":"100.00000001"
	}
}

// Response
{
    "fee": "0.0001"
}
```

## SendRawTransaction
    POST /v1/transactions/send
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| hex | string |  |  |
### Returns
- `String` - tx_id 
### Example
```json
// Request
{
	"hex":"080112a4010a280a240912dd62721ede54001140c92cfc8382df3819ee036bfb2b5038a021d2636f81821b767a10091248473044022003161aa740d89984ef995103735bc6f6a0e0db76bb4eb224914bb797cf9df9ab02202765b0dd7ecb4bf5835e1a1bdce6686b26b3f6e37977668aaefdfa9a29e0a5f4011225512103d0cd7443a5e8dcc030793bea363fe328c84d2daf75f0f2db17d36c07642777b151ae19ffffffffffffffff12a4010a280a2409c4352cf60f2ed5001100fd6132166eb6c719ff7553b515588c2721b12de84e4e2b55751009124847304402205f3a8d2ea86971a7cebba0a07aeb93372732bcefc4e566e8d29009a8cc5598720220759fd2b87292cc9633f16e151d6d34e28dfbfde6b35fa329177b985f80388c14011225512103d0cd7443a5e8dcc030793bea363fe328c84d2daf75f0f2db17d36c07642777b151ae19ffffffffffffffff1a2a088190dfc04a12220020c7bde8edb1e18eeb5fba5b5402b3064f7312999c1224fd4415fab6e26c2abbfa"
}

// Response
{
    "tx_id": "b7f7cab1dcb748987aa5694a6c021828cbf18f07154991467417dbe4f98e9707"
}
```

## GetRawTransaction
    GET /v1/transactions/{tx_id}/details
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| tx_id | string |  |  |
### Returns
- `String` - hex 
- `String` - tx_id 
- `Integer` - version 
- `Integer` - lock_time 
- `Array of BlockInfoForTx`, info of the block that packed this transaction
    - BlockInfoForTx
        - `Integer` - height 
        - `String` - block_hash 
        - `Integer` - timestamp 
- `Array of Vin`, inputs of transaction
    - Vin
        - `String` - value, of spent utxo 
        - `Integer` - n 
        - `Integer` - type, of spent utxo, 1-normal, 2-staking, 3-binding
        - `Object`, redeem_detail  
            - `String` - tx_id 
            - `Integer` - vout 
            - `Integer` - sequence 
            - `Array of String`, witness
            - `Array of String`, addresses
                - `0` - holder address of this utxo
                - `1` - staking address(type=2),
                        binding address(type=3),
                        nothing(other)
- `Array of Vout`, outputs of transaction
    - Vout
        - `String` - value 
        - `Integer` - n 
        - `Integer` - type 
        - `Object`, script_detail
            - `String` - asm 
            - `String` - hex 
            - `Integer` - req_sigs 
            - `Array of String`, addresses
                - `0` - holder address of this utxo
                - `1` - staking address(type=2),
                        binding address(type=3),
                        nothing(other)
- `String` - payload 
- `Integer` - confirmations
- `Integer` - size 
- `String` - fee 
- `Integer` - status 
- `Boolean` - coinbase, if it is a coinbase transaction
### Example
```json
// tx_id b7f7cab1dcb748987aa5694a6c021828cbf18f07154991467417dbe4f98e9707
{
    "hex": "080112a4010a280a240912dd62721ede54001140c92cfc8382df3819ee036bfb2b5038a021d2636f81821b767a10091248473044022003161aa740d89984ef995103735bc6f6a0e0db76bb4eb224914bb797cf9df9ab02202765b0dd7ecb4bf5835e1a1bdce6686b26b3f6e37977668aaefdfa9a29e0a5f4011225512103d0cd7443a5e8dcc030793bea363fe328c84d2daf75f0f2db17d36c07642777b151ae19ffffffffffffffff12a4010a280a2409c4352cf60f2ed5001100fd6132166eb6c719ff7553b515588c2721b12de84e4e2b55751009124847304402205f3a8d2ea86971a7cebba0a07aeb93372732bcefc4e566e8d29009a8cc5598720220759fd2b87292cc9633f16e151d6d34e28dfbfde6b35fa329177b985f80388c14011225512103d0cd7443a5e8dcc030793bea363fe328c84d2daf75f0f2db17d36c07642777b151ae19ffffffffffffffff1a2a088190dfc04a12220020c7bde8edb1e18eeb5fba5b5402b3064f7312999c1224fd4415fab6e26c2abbfa",
    "tx_id": "b7f7cab1dcb748987aa5694a6c021828cbf18f07154991467417dbe4f98e9707",
    "version": 1,
    "lock_time": "0",
    "block": {
        "height": "177083",
        "block_hash": "78bd7128f00f5186e18b5b2f692b9cb49fc0abcebb9e1bb86f06b818b0c6432a",
        "timestamp": "1574849967"
    },
    "vin": [
        {
            "value": "104.00000007",
            "n": 0,
            "type": 1,
            "redeem_detail": {
                "tx_id": "0054de1e7262dd1238df8283fc2cc940a038502bfb6b03ee7a761b82816f63d2",
                "vout": 9,
                "sequence": "18446744073709551615",
                "witness": [
                    "473044022003161aa740d89984ef995103735bc6f6a0e0db76bb4eb224914bb797cf9df9ab02202765b0dd7ecb4bf5835e1a1bdce6686b26b3f6e37977668aaefdfa9a29e0a5f401",
                    "512103d0cd7443a5e8dcc030793bea363fe328c84d2daf75f0f2db17d36c07642777b151ae"
                ],
                "addresses": [
                    "ms1qqgzwwzt77zuw4sf8uqugqj5w6cpk6lalas4svx6jac20kjrhywx3qnshys8"
                ]
            }
        },
        {
            "value": "104.00000007",
            "n": 1,
            "type": 1,
            "redeem_detail": {
                "tx_id": "00d52e0ff62c35c4c7b66e163261fd00278c5815b55375ff75552b4e4ee82db1",
                "vout": 9,
                "sequence": "18446744073709551615",
                "witness": [
                    "47304402205f3a8d2ea86971a7cebba0a07aeb93372732bcefc4e566e8d29009a8cc5598720220759fd2b87292cc9633f16e151d6d34e28dfbfde6b35fa329177b985f80388c1401",
                    "512103d0cd7443a5e8dcc030793bea363fe328c84d2daf75f0f2db17d36c07642777b151ae"
                ],
                "addresses": [
                    "ms1qqgzwwzt77zuw4sf8uqugqj5w6cpk6lalas4svx6jac20kjrhywx3qnshys8"
                ]
            }
        }
    ],
    "vout": [
        {
            "value": "200.00000001",
            "n": 0,
            "type": 1,
            "script_detail": {
                "asm": "0 c7bde8edb1e18eeb5fba5b5402b3064f7312999c1224fd4415fab6e26c2abbfa",
                "hex": "0020c7bde8edb1e18eeb5fba5b5402b3064f7312999c1224fd4415fab6e26c2abbfa",
                "req_sigs": 1,
                "addresses": [
                    "ms1qqc7773md3ux8wkha6td2q9vcxfae39xvuzgj063q4l2mwymp2h0aqunux9z"
                ]
            }
        }
    ],
    "payload": "",
    "confirmations": "6",
    "size": 360,
    "fee": "8.00000013",
    "status": 1,
    "coinbase": false
}
```

## GetTxStatus
    GET /v1/transactions/{tx_id}/status
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| tx_id | string |  |  |
### Returns
- `Integer` - code
- `String`, status, corresponding description of code
    - 1-confirmed
    - 2-missing
    - 3-pending
    - 4-confirming
### Example
```json
// tx_id b7f7cab1dcb748987aa5694a6c021828cbf18f07154991467417dbe4f98e9707
{
    "code": 1,
    "status": "confirmed"
}
```

##  CreateStakingTransaction
    POST /v1/transactions/staking
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| from_address | string |  | optional. Who will pay for this transaction. |
| staking_address | string |  |  |
| amount | string |  | number in `MASS` |
| frozen_period | int |  | number of blocks from been packed |
| fee | string | | number in `MASS` | 
### Returns
- `String` - hex 
### Example
```json
// Request
{
	"from_address":"ms1qqgzwwzt77zuw4sf8uqugqj5w6cpk6lalas4svx6jac20kjrhywx3qnshys8",
	"staking_address":"ms1qp3fjnfxx3v2pja3gkatyrc3nzvfw52p08w4xnnuap47ey4wfg7xtq5yrwrx",
	"amount": "3000",
	"frozen_period": 65000
}

// Response
//      tx_id 383e5e934e20fedc7ca077a9bb789c4831ae4d6af9cae4e164c1b9741976e38c
{
    "hex": "080112330a280a2409c08a24daa23a187111503af2b73a2d029119d01ca48ea3d3fb7321c180911b3772077d100919ffffffffffffffff12330a280a24093fe5847b633f46ef11cde41ad1eee2784d1905aa9a9f7bb3982c2189b769f14a35fb23100919ffffffffffffffff12330a280a240971355420be4c13ee115998502a92bef97a197ce4837bf57a302621747ccd2ece4a320b100919ffffffffffffffff12330a280a2409105025166f2843ed11febcda2f1065c04d1921ef48e5fee03f8421c111504a19032dcb100919ffffffffffffffff12330a280a240965072cc121ed16ed110b3f9e857daa0c70198d49ab97a084b8c021dbea53458b172e3b100919ffffffffffffffff12330a280a24097327b2954f2df5eb11f72c7feeb32f5bc2199f3d25bbf82bdfc3218d7ba70603c997ec100919ffffffffffffffff12330a280a24090f40e74479d0f2eb11ced218bc533d9980197cbd8fce9e3da710215ab51a0bb9218133100919ffffffffffffffff12330a280a240935bf7f6d1317b7eb11441b4eed208cdfeb19de1a93300afbceed21675ebd6e91ee0c7b100919ffffffffffffffff12330a280a2409a0b69836a94de5ea113b5024cc2271e30b19c8b37f4c8d8be8ce210351e03f5a048a64100919ffffffffffffffff12330a280a2409767ca308914b1ce91158a913e85e056f8e19f302f76899d6087921e7a90b47b9afc8c5100919ffffffffffffffff12330a280a2409c23bc87849c69ae811fd21d9b8e7c9c67c19515d31fc2678794a219ba96458c3064f41100919ffffffffffffffff12330a280a2409cce9f1d564a402e711bdd8de8127b629e019c19838269ee5542d2151d54a1702871ea7100919ffffffffffffffff12330a280a24097c50f9a33482dfe511d4e0124e3fdd191e191114e4c3fb06834421e7e2406655209d37100919ffffffffffffffff12330a280a2409bdc49e8b92f681e41128470994a3b959bd19926c12f44aa9029d217e48cf58bce4dca5100919ffffffffffffffff12330a280a24097392d6d41c1409e4116fc6b0874235bfb119fb5852099bb5bfe321c01221976db9589a100919ffffffffffffffff12330a280a24096c15eb9e3127b2e311de99d2709f35618f196583af7f2871313f21cb29ff658496da85100919ffffffffffffffff12330a280a24093a7186eee43081e211deda1239f5559941190f95145271a8bed721e5aa25190f189019100919ffffffffffffffff12330a280a24099070f585f75b70e211e0902c050e8664d319bf7e774ed53eede6211bb822951dcc07dc100919ffffffffffffffff12330a280a2409a99a8a09d58c0ce211eedfd0bd9ae566b41959dfe6fb9be455a12190b297246495ee92100919ffffffffffffffff12330a280a240951b4591be5907ee1117da970c7fb0399e61946a991481878a78921413c957ba764a250100919ffffffffffffffff12330a280a240975c4115be8312ce011d6345cfbc7e63be419669bd5764a8a3d32217d3d1140f4a31724100919ffffffffffffffff12330a280a24095da32008253a25e0115ec997d4f4288bb019b137eb578370d242214fed6fcba1211a6e100919ffffffffffffffff12330a280a2409e905c1c538e2fadc11067decbd0f333f1b1993b8b7b23d4850c92122103f721844283c100919ffffffffffffffff12330a280a2409eed8930b826e32dc117efadce801e0c92b1932f8596c227502dd21b0421ce5d4aa967e100919ffffffffffffffff12330a280a2409ab75d27d5f390bdb115fd209ac5728042319dead7676e771957321c52c101caf217ae7100919ffffffffffffffff12330a280a2409421fb3eb817555d9111b6eb2db91de07b019df4035c6175aba942133f71dcf7ac58dc2100919ffffffffffffffff12330a280a2409c9db2fd3f8c29dd811094ac2f24f6ea18119d293abcabb980d62214f1c8bf34111f4fe100919ffffffffffffffff12330a280a240987fa33c5411436d711edc1e2a4d4fbd74f19e5b37993be78037621f25be9ee360cee92100919ffffffffffffffff12330a280a24090e27b3e6f6af4e7211a2891ec3772186f21900c14b3fcc03efed21705cf79f8d67ef21100819ffffffffffffffff1a340880f092cbdd08122b00208a653498d162832ec516eac83c4662625d4505e7754d39f3a1afb24ab928f19608e8fd0000000000001a2a08efb9f5fa0512220020409ce12fde171d5824fc07100951dac06daff7fd8560c36a5dc29f690ee471a2"
}
```

## GetStakingHistory
    GET /v1/transactions/staking/history
### Parameters
null
### Returns
- `Array of Tx`, txs
    - Tx
        - `String` - tx_id 
        - `Integer` - status 
        - `Integer` - block_height      // height of the block that packed this transaction, 0 if still packing
        - `Object`, utxo 
            - `String` - tx_id 
            - `Integer` - vout  
            - `String` - address          // staking address
            - `String` - amount           // staking value in MASS
            - `Integer` - frozen_period
- `Object`, weights
### Example
```json
{
    "txs": [
        {
            "tx_id": "383e5e934e20fedc7ca077a9bb789c4831ae4d6af9cae4e164c1b9741976e38c",
            "status": 1,
            "block_height": "177102",
            "utxo": {
                "tx_id": "383e5e934e20fedc7ca077a9bb789c4831ae4d6af9cae4e164c1b9741976e38c",
                "vout": 0,
                "address": "ms1qp3fjnfxx3v2pja3gkatyrc3nzvfw52p08w4xnnuap47ey4wfg7xtq5yrwrx",
                "amount": "3000",
                "frozen_period": 65000
            }
        }
    ],
    "weights": {}
}
```

## GetLatestRewardList
    GET /v1/transactions/staking/latestreward
### Parameters
null
### Returns
- `Array of RewardDetail`, details
    - RewardDetail
        - `Integer` - rank 
        - `String` - amount, in MASS
        - `Integer` - weight 
        - `String` - address 
        - `String` - profit, in MASS
### Example
```json
{
    "details": [
        {
            "rank": 0,
            "amount": "3000",
            "weight": 19493100000000000,
            "address": "ms1qp3fjnfxx3v2pja3gkatyrc3nzvfw52p08w4xnnuap47ey4wfg7xtq5yrwrx",
            "profit": "61.79441473"
        },
        {
            "rank": 1,
            "amount": "2049",
            "weight": 2309427900000000,
            "address": "ms1qpv24szcpxphktpea9pd6caer9j4nf66jepp3qldtp6ahdu7ldxsnsp28g3u",
            "profit": "42.20558526"
        }
    ]
}
```

## TxHistory
    POST /v1/transactions/history
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| count | int | return the `count` most recent transactions | optional. 500 by default |
| address | string | which addresses to query | optional. If not provided, all addresses of current wallet will be used. |
### Returns
- `Array of TxHistoryDetails`, histories
    - TxHistoryDetails
        - `String` - tx_id
        - `Integer` - block_height
        - `Array of Input`, inputs
            - `String` - tx_id
            - `Integer` - index 
        - `Array of Output`, outputs
            - `String` - address 
            - `String` - amount, in MASS
        - `Array of String` - from_addresses, address collection of inputs
### Example
```json
{
    "histories": [
        {
            "tx_id": "b7f7cab1dcb748987aa5694a6c021828cbf18f07154991467417dbe4f98e9707",
            "block_height": "177083",
            "inputs": [
                {
                    "tx_id": "0054de1e7262dd1238df8283fc2cc940a038502bfb6b03ee7a761b82816f63d2",
                    "index": "9"
                },
                {
                    "tx_id": "00d52e0ff62c35c4c7b66e163261fd00278c5815b55375ff75552b4e4ee82db1",
                    "index": "9"
                }
            ],
            "outputs": [
                {
                    "address": "ms1qqc7773md3ux8wkha6td2q9vcxfae39xvuzgj063q4l2mwymp2h0aqunux9z",
                    "amount": "200.00000001"
                }
            ],
            "from_addresses": [
                "ms1qqgzwwzt77zuw4sf8uqugqj5w6cpk6lalas4svx6jac20kjrhywx3qnshys8"
            ]
        }
    ]
}
```

## GetAddressBinding
    POST /v1/addresses/binding
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| address | []string | witch addresses to query | poc miner address, not wallet address |
### Returns
- `Map`, amounts
    - `String` - key, miner address
    - `String` - value, total binding value in MASS
### Example
```json
// Request
{
	"addresses":["146hGPwfYRDde6tJ6trbyhkSoPwt69AqyZ","1EgzSkV7vJ7xhC5g38ULLPoMBhHVW38VZN"]
}

// Response
{
    "amounts": {
        "146hGPwfYRDde6tJ6trbyhkSoPwt69AqyZ": "0.026624",
        "1EgzSkV7vJ7xhC5g38ULLPoMBhHVW38VZN": "0.026624"
    }
}
```

## GetBindingHistory
    GET /v1/transactions/binding/history
### Parameters
null
### Returns
- `Array of History`, histories
    - History
        - `String` - tx_id 
        - `Integer` - status, 0-pending, 1-confirmed, 2-withdrawing, 3-withdrawn
        - `Integer` - block_height
        - `Object`, utxo
            - `String` - tx_id 
            - `Integer` - vout 
            - `String` - holder_address
            - `String` - binding_address, poc miner address
            - `String` - amount, in MASS
        - `Array of String`, from_addresses

### Example
```json
{
    "histories": [
        {
            "tx_id": "9e4c191a29a4eb018d7904ca1cd0d6f1568356426f0a4a1c5f388c91b768d80e",
            "status": 1,
            "block_height": "117649",
            "utxo": {
                "tx_id": "9e4c191a29a4eb018d7904ca1cd0d6f1568356426f0a4a1c5f388c91b768d80e",
                "vout": 0,
                "holder_address": "ms1qqehh47s0hvzrqqjl77ayj78yytstjkrsltcna343p8yg7ndskvveql4z3vl",
                "binding_address": "146hGPwfYRDde6tJ6trbyhkSoPwt69AqyZ",
                "amount": "0.026624"
            },
            "from_addresses": [
                "ms1qq20yfsypqjuz305j2nhhu8khsj07mxfq2sa8ua685l2leayk02hrsk9kjvx"
            ]
        },
        {
            "tx_id": "9e4c191a29a4eb018d7904ca1cd0d6f1568356426f0a4a1c5f388c91b768d80e",
            "status": 1,
            "block_height": "117649",
            "utxo": {
                "tx_id": "9e4c191a29a4eb018d7904ca1cd0d6f1568356426f0a4a1c5f388c91b768d80e",
                "vout": 1,
                "holder_address": "ms1qqehh47s0hvzrqqjl77ayj78yytstjkrsltcna343p8yg7ndskvveql4z3vl",
                "binding_address": "1EgzSkV7vJ7xhC5g38ULLPoMBhHVW38VZN",
                "amount": "0.026624"
            },
            "from_addresses": [
                "ms1qq20yfsypqjuz305j2nhhu8khsj07mxfq2sa8ua685l2leayk02hrsk9kjvx"
            ]
        },
        {
            "tx_id": "436a2d092493590d96b4782067326c9f04fe5b4e3602203cea920c100dffb66b",
            "status": 1,
            "block_height": "117292",
            "utxo": {
                "tx_id": "436a2d092493590d96b4782067326c9f04fe5b4e3602203cea920c100dffb66b",
                "vout": 0,
                "holder_address": "ms1qq93pq8kphrtax7m5t52km4x84rrvplty4ttpjz27y3ve6rmhhuqys7cr2s4",
                "binding_address": "1HWyjsiqaMuHjdMJpSxDwQ9NG1Rrca4Jjx",
                "amount": "0.026624"
            },
            "from_addresses": [
                "ms1qq20yfsypqjuz305j2nhhu8khsj07mxfq2sa8ua685l2leayk02hrsk9kjvx"
            ]
        },
        {
            "tx_id": "436a2d092493590d96b4782067326c9f04fe5b4e3602203cea920c100dffb66b",
            "status": 1,
            "block_height": "117292",
            "utxo": {
                "tx_id": "436a2d092493590d96b4782067326c9f04fe5b4e3602203cea920c100dffb66b",
                "vout": 1,
                "holder_address": "ms1qq93pq8kphrtax7m5t52km4x84rrvplty4ttpjz27y3ve6rmhhuqys7cr2s4",
                "binding_address": "1MaTXJGHeXxPmDtusRtESWkcdu9RhTiu65",
                "amount": "0.026624"
            },
            "from_addresses": [
                "ms1qq20yfsypqjuz305j2nhhu8khsj07mxfq2sa8ua685l2leayk02hrsk9kjvx"
            ]
        }
    ]
}
```

## CreateBindingTransaction
    POST /v1/transactions/binding
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| from_address | string | who will pay for this transaction | optional, if not provided, payer is indefinite |
| fee | string |  | optional, in MASS |
| outputs | Output |  |  |
- Output
    - `String` - holder_address
    - `String` - binding_address, poc miner address
    - `String` - amount, in MASS

### Returns
- `String` - hex 
### Example
```json
// Request
{
	"outputs":[{
		"holder_address":"ms1qq2hr9cfgrrjekah9uy2nwsgpv5dtmckzh3vls6zgcv572k5hm5u8sd9nzjv",
		"binding_address":"146hGPwfYRDde6tJ6trbyhkSoPwt69AqyZ",
		"amount":"2.5"
	}],
	"from_address":"ms1qqgzwwzt77zuw4sf8uqugqj5w6cpk6lalas4svx6jac20kjrhywx3qnshys8",
	"fee": "0.001"
}

// Response
{
    "hex": "080112330a280a2409dcfe204e935e3e3811489c78bba977a07c19e1e4caf96a4dae31218ce3761974b9c164100119ffffffffffffffff1a3e0880e59a771237002055c65c25031cb36edcbc22a6e8202ca357bc58578b3f0d0918653cab52fba70f1421fc11adc05e340fe9e5c6548f0846b74a9c28751a2a08cfc7d4830512220020409ce12fde171d5824fc07100951dac06daff7fd8560c36a5dc29f690ee471a2"
}
```