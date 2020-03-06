# configuration file
> path`./conf/config.json`  
> If HTTPS protocol is used, the TLS certificate needs to be placed in the directory`./conf/`,named `cert.crt`and `cert.key`.

config option
```json
{
    "server": "https://localhost:9688", // or http://localhost:9688
    "log_dir": "./logs",
    "log_level": "info"
}
```

# check all cmd
```bash
> masswallet-cli -h,--help
```
# basic usage
```bash
> masswallet-cli [command <required args> [optional args]]
```
# help
```bash
> masswallet-cli [command] --help
```

# Command

> note:  
>- All parameters marked with '< >' are required
>- Parameters marked with '[]' are optional

## createcert
    createcert <directory>
Creates a new TLS certificate.

Example:  
```bash
> masswallet-cli createcert .
```

## getclientstatus
    getclientstatus
Returns current node status.

Parameter:  

    null

Example:  
```bash
> masswallet-cli getclientstatus
```

Return:  
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
                "address": "[host]:[port]",
                "direction": "outbound"
            },
            {
                "id": "B3664A9AC4AF1DBB457BB82F2F856F25DDE1F9F226D51BCA94A7F71123839100",
                "address": "[host]:[port]",
                "direction": "outbound"
            }
        ],
        "inbound": [],
        "other": []
    }
}
```

## getbestblock
    getbestblock
Query the latest block information of the node.

Parameter:  

    null

Example:  
```bash
> masswallet-cli getbestblock
```

Return:  
```json
{
  "height": "8993",         
  "target": "1f56abb05"     //mining difficulty (hex)
}
```

## listwallets
    listwallets
Returns all imported wallets.

Parameter:  

    null

Example:  
```bash
> masswallet-cli listwallets
```

Return:  
```json
{
  "wallets": [
    {
      "wallet_id": "ac102yfx0q2v6v3aug35hw42jn8k6sljeypffn85w3",       
      "type": 1,                //fixed value
      "remarks": "for test",   
      "ready": true|false,      // false-importing, true-import completed
      "synced_height": "0"      // Indicates processed blocks height when ready=false
    }
  ]
}
```

## usewallet
    usewallet <wallet_id>
Toggles the wallet context currently in use. All transaction related commands can only be used in specific wallet context.

Parameter:  

    wallet_id       

Example:  
```bash
> masswallet-cli usewallet ac102yfx0q2v6v3aug35hw42jn8k6sljeypffn85w3
```

Return:  
```json
{
  "chain_id": "...",
  "wallet_id": "ac102yfx0q2v6v3aug35hw42jn8k6sljeypffn85w3",
  "type": 1,                                                    
  "total_balance": "1234.00001428",                            
  "external_key_count": 10,                                       //external pk(address) total
  "internal_key_count": 0,                                        //internal pk total ,not currently in use
  "remarks": "for test"
}
```

## createwallet
    createwallet <passphrase> [entropy=?] [remarks=?]

Parameter:  

    passphrase    6 to 40 valid characters to encrypt the wallet.
    entropy       The initial entropy length for generating mnemonics must be an integer multiple of 32 in the range of [128,256]. The default is 128.
    remarks       Note information of wallet, without any chain semantics.

Example:  
```bash
> masswallet-cli createwallet 123456 entropy=160 remarks="for test"
```

Return:  
```json
{
  "wallet_id": "ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds",     
  "mnemonic": "figure vapor flame artwork clarify local right insect fall pulp dwarf steel tip author pulse"            //mnemonics
}
```

## getwalletmnemonic
    getwalletmnemonic
Returns the mnemonic of currently used wallet.

Parameter:  

    wallet_id
    passphrase      

Example:  
```bash
> masswallet-cli getwalletmnemonic ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds 123456
```

Return:  
```json
{
  "mnemonic": "figure vapor flame artwork clarify local right insect fall pulp dwarf steel tip author pulse"
}
```

## exportwallet
    exportwallet <wallet_id> <passphrase>

Parameter:  

    wallet_id
    passphrase      

Example:  
```bash
> masswallet-cli exportwallet ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds 123456
```

Return:  
```json
{
  "keystore": "{\"remarks\":\"for test\",\"crypto\":{\"cipher\":\"Stream cipher\",\"entropyEnc\":\"ac6de145205481dc95ab6a22f8b978edb3f1b2d69f5036a2158339bc7465c0a7311d6d08983ef8e2f5d443dad7dc7580095b024cd809ab6006743624\",\"kdf\":\"scrypt\",\"pubParams\":\"\",\"privParams\":\"b999d54a2369d9be570928c32e0a727e95eb395ad8ba5383608c9cb800a4cad82c52fe054cdcdfae5b3d751810452425704095a32fcb6cea85913575ac39174c000004000000000008000000000000000100000000000000\",\"cryptoKeyPubEnc\":\"\",\"cryptoKeyPrivEnc\":\"\",\"cryptoKeyEntropyEnc\":\"cac4a9b04cca089d396430cb9115627d97e914590183b5a835e7cbf76ae6b85ae4a6cbaafe12e2fa1a43f52c78c5aa62e9d3f5265baf9eb2618e020b1d5ce5555d9cc69f005c6a44\"},\"hdPath\":{\"Purpose\":44,\"Coin\":1,\"Account\":1,\"ExternalChildNum\":0,\"InternalChildNum\":0}}"
}
```

## removewallet
    removewallet <wallet_id> <passphrase>

Parameter:  

    wallet_id
    passphrase    

Example:  
```bash
> masswallet-cli removewallet ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds 123456
```

Return:  
```json
{
  "ok": true
}
```

## importwallet
    importwallet <keystore> <passphrase>
Imports a wallet by keystore.

Parameter:  

    keystore        json data
    passphrase      

Example1:  
```bash
> masswallet-cli importwallet '{"remarks":"for test","crypto":{"cipher":"Stream cipher","entropyEnc":"ac6de145205481dc95ab6a22f8b978edb3f1b2d69f5036a2158339bc7465c0a7311d6d08983ef8e2f5d443dad7dc7580095b024cd809ab6006743624","kdf":"scrypt","pubParams":"","privParams":"b999d54a2369d9be570928c32e0a727e95eb395ad8ba5383608c9cb800a4cad82c52fe054cdcdfae5b3d751810452425704095a32fcb6cea85913575ac39174c000004000000000008000000000000000100000000000000","cryptoKeyPubEnc":"","cryptoKeyPrivEnc":"","cryptoKeyEntropyEnc":"cac4a9b04cca089d396430cb9115627d97e914590183b5a835e7cbf76ae6b85ae4a6cbaafe12e2fa1a43f52c78c5aa62e9d3f5265baf9eb2618e020b1d5ce5555d9cc69f005c6a44"},"hdPath":{"Purpose":44,"Coin":1,"Account":1,"ExternalChildNum":0,"InternalChildNum":0}}' 123456
```

Example2:  
```bash
> masswallet-cli importwallet "{\"remarks\":\"for test\",\"crypto\":{\"cipher\":\"Stream cipher\",\"entropyEnc\":\"ac6de145205481dc95ab6a22f8b978edb3f1b2d69f5036a2158339bc7465c0a7311d6d08983ef8e2f5d443dad7dc7580095b024cd809ab6006743624\",\"kdf\":\"scrypt\",\"pubParams\":\"\",\"privParams\":\"b999d54a2369d9be570928c32e0a727e95eb395ad8ba5383608c9cb800a4cad82c52fe054cdcdfae5b3d751810452425704095a32fcb6cea85913575ac39174c000004000000000008000000000000000100000000000000\",\"cryptoKeyPubEnc\":\"\",\"cryptoKeyPrivEnc\":\"\",\"cryptoKeyEntropyEnc\":\"cac4a9b04cca089d396430cb9115627d97e914590183b5a835e7cbf76ae6b85ae4a6cbaafe12e2fa1a43f52c78c5aa62e9d3f5265baf9eb2618e020b1d5ce5555d9cc69f005c6a44\"},\"hdPath\":{\"Purpose\":44,\"Coin\":1,\"Account\":1,\"ExternalChildNum\":0,\"InternalChildNum\":0}}" 123456
```

Return:  
```json
{
  "ok": true,
  "wallet_id": "ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds",
  "type": 1,
  "remarks": "for test"
}
```

## importwalletbymnemonic
    importwalletbymnemonic <mnemonic> <passphrase> [externalindex=?] [remarks=?]
Imports a wallet by mnemonic.

Parameter:  

    mnemonic        
    passphrase      
    externalindex   optional, init pk num
    remarks         optional

Example:  
```bash
> masswallet-cli importwalletbymnemonic "figure vapor flame artwork clarify local right insect fall pulp dwarf steel tip author pulse" 123456
```

Return:  
```json
{
  "ok": true,
  "wallet_id": "ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds",
  "type": 1,
  "remarks": ""
}
```

## getwalletbalance
    getwalletbalance [minconf=?] [detail=?]


Parameter:  

    minconf        Optional, only utxos that have been confirmed by at least <minconf> blocks would be count, default 1.
    detail         Optional, whether to count the total amount of that spendable, default false.

Example:  
```bash
> masswallet-cli getwalletbalance minfconf=100 detail=true
```

Return:  
```json
{
  "wallet_id": "ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds",
  "total": "100.00001428",         
  "detail": {
    "spendable": "100.00001428",     
    "withdrawable_staking": "0",      
    "withdrawable_binding": "0"      
  }
}
```

## listaddresses
    listaddresses <version>
Returns all addresses of currently used wallet.

Parameter:  

    version     0  - create normal address
                1  - create staking address

Example:  
```bash
> masswallet-cli listaddresses 1
```

Return:  
```json
{
  "details": [
    {
      "address": "ms1qp0czrc8errz8gdmpjgxd59kwvydf3g3ch72d6qm2kqwzlgm232pksqw0eky",
      "version": 1,             //0-normal address,1-staking address
      "used": true|false,       
      "std_address": "ms1qq0czrc8errz8gdmpjgxd59kwvydf3g3ch72d6qm2kqwzlgm232pksl9lut6"          //indicates revenue address when version=1
    }
  ]
}
```

## createaddress
    createaddress <version>
Creates a new address of currently used wallet.

Parameter:  

    version     0  - create a normal address
                1  - create a staking address

Example:  
```bash
> masswallet-cli createaddress 0
```

Return:  
```json
{
    "address": "ms1qqgrq0g20u8tpq2vv0596vm3uxh0ptn72449wvpr86gaqk0gx78scqmp7jyl"  
}
```

## validateaddress
    validateaddress <address>


parameter；

    address    

Example:  
```bash
> masswallet-cli validateaddress ms1qqgrq0g20u8tpq2vv0596vm3uxh0ptn72449wvpr86gaqk0gx78scqmp7jyl
```

Return:  
```json
{
  "is_valid": true|false,       
  "is_mine": true|false,         //belongs to the current Wallet
  "address": "",                   
  "version": 0|1                 //0-normal address,1-staking address
}
```

## getaddressbalance
    getaddressbalance <min_conf> [<address> <address> ...]


Parameter:  

    min_conf        fixed value.Only count utxo confirmed by Min conf block at least
    address         optional.If null, return the balance of all addresses

Example:  
```bash
> masswallet-cli getaddressbalance 1 ms1qqf8870v59cdaanj3cxgfq97d3xpz94g9gqqsvz0wnj7lmlp9ehr2sxdj0um ms1qqgq750hhj0lcem3pmv6gymvvmfmrzmvhyxqwyp4fey2cmec372pjq3uw7yg
```

Return:  
```json
{
  "balances": [
    {
      "address": "ms1qqf8870v59cdaanj3cxgfq97d3xpz94g9gqqsvz0wnj7lmlp9ehr2sxdj0um",
      "total": "20",                // in MASS
      "spendable": "20",            // mature part of total
      "withdrawable_staking": "0",  // expired and unspent staking utxo    
      "withdrawable_binding": "0"       
    },
    {
      "address": "ms1qqgq750hhj0lcem3pmv6gymvvmfmrzmvhyxqwyp4fey2cmec372pjq3uw7yg",
      "total": "0",
      "spendable": "0",
      "withdrawable_staking": "0",
      "withdrawable_binding": "0"
    }
  ]
}
```

## listutxo
    listutxo <address> <address> ...
Querys utxos of the current wallet address.

Parameter:  

    address     optional.If null, return the balance of all addresses

Example:  
```bash
> masswallet-cli listutxo ms1qqf8870v59cdaanj3cxgfq97d3xpz94g9gqqsvz0wnj7lmlp9ehr2sxdj0um
```

Return:  
```json
{
    "address_utxos":[{
        "address": "ms1qqf8870v59cdaanj3cxgfq97d3xpz94g9gqqsvz0wnj7lmlp9ehr2sxdj0um",
        "utxos":[{
          "tx_id": "08e60b73ef43f5bfcf3f954f103f618e8bc69995ba414fdf97d2098841863695",
          "vout": 0,
          "amount": "20",
          "block_height": "1279",
          "maturity": 0,
          "confirmations": 13935,
          "spent_by_unmined": false  
        },...]
    }]
}
```

## createrawtransaction
    createrawtransaction <inputs> <outputs> [locktime=?]
Creates a normal transaction.

Parameter:  

    inputs      
                [{"tx_id":"...","vout":0},...]
    outputs     
                {"address1":"vlaue1","address2":"vlaue2",...}, value unit:  MASS
    locktime    optional.

Example:  
```bash
> masswallet-cli createrawtransaction '[{"tx_id":"08e60b73ef43f5bfcf3f954f103f618e8bc69995ba414fdf97d2098841863695","vout":0}]' '{"ms1qqgq750hhj0lcem3pmv6gymvvmfmrzmvhyxqwyp4fey2cmec372pjq3uw7yg":"19.99999"}'
```

Return:  
```json
{
  "hex": "080112310a260a2409bff543ef730be608118e613f104f953fcf19df4f41ba9599c68b21953686418809d29719ffffffffffffffff1a2a0898a0d6b90712220020403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064"
}
```

## autocreaterawtransaction
    autocreaterawtransaction <outputs> [locktime=?] [fee=?] [from=?]
Creates a transaction with randomly selected utxos from the current wallet.

Parameter:    

    outputs     fixed value,format:  {"address":"value",...}
    fee         optional
    from        optional.Specify the source address of utxo, otherwise select randomly from all addresses of wallet

Example:  
```bash
> masswallet-cli autocreaterawtransaction '{"ms1qqgq750hhj0lcem3pmv6gymvvmfmrzmvhyxqwyp4fey2cmec372pjq3uw7yg": "19.99999", ...}' from=ms1qqf8870v59cdaanj3cxgfq97d3xpz94g9gqqsvz0wnj7lmlp9ehr2sxdj0um

// win
> masswallet-cli autocreaterawtransaction "{\"ms1qqgq750hhj0lcem3pmv6gymvvmfmrzmvhyxqwyp4fey2cmec372pjq3uw7yg\": \"19.99999\", ...}" from=ms1qqf8870v59cdaanj3cxgfq97d3xpz94g9gqqsvz0wnj7lmlp9ehr2sxdj0um
```

Return:  
```json
{
    "hex":"080112310a260a2409bff543ef730be608118e613f104f953fcf19df4f41ba9599c68b21953686418809d29719ffffffffffffffff1a2a0898a0d6b90712220020403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064"    
}
```

## signrawtransaction
    signrawtransaction <hexstring> <passphrase> [mode=?]
Signs a transaction.

Parameter:  

    hexstring       Transactions to be signed
    passphrase      
    mode            optional.default "ALL"
                    ALL
                    NONE
                    SINGLE
                    ALL|ANYONECANPAY
                    NONE|ANYONECANPAY
                    SINGLE|ANYONECANPAY

Example:  
```bash
> masswallet-cli signrawtransaction 080112310a260a2409bff543ef730be608118e613f104f953fcf19df4f41ba9599c68b21953686418809d29719ffffffffffffffff1a2a0898a0d6b90712220020403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064 123456
```

Return:  
```json
{
  "hex": "080112a2010a260a2409bff543ef730be608118e613f104f953fcf19df4f41ba9599c68b21953686418809d2971248473044022010b789a4ac96e6c6f3caafc2b9f548a0e97cf1ad6bfe4a850ad73a3a4818861a0220532098aa7fa238511fdac8ab7aaf2541027e29f4b4d3eccb882d45263c860a0901122551210203a70a76734af100ed151ecf69ec776e4ef03ca0df06457176c1d61bb4c9d52e51ae19ffffffffffffffff1a2a0898a0d6b90712220020403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064",
  "complete": true
}
```

## gettransactionfee
    gettransactionfee <outputs> <inputs> [binding=true] [locktime=?]
Estimates transaction fee.

Parameter:  

    outputs         format:{"address1":"vlaue1","address2":"vlaue2",...}
    inputs          format:[{"tx_id":"...","vout":0},...]
    binding         optional.Include staking or not,default false
    locktime        optional

Example:  
```bash
> masswallet-cli gettransactionfee '{"ms1qqgq750hhj0lcem3pmv6gymvvmfmrzmvhyxqwyp4fey2cmec372pjq3uw7yg":"19.99999"}' '[{"tx_id":"08e60b73ef43f5bfcf3f954f103f618e8bc69995ba414fdf97d2098841863695","vout":0}]'
```

Return:  
```json
{
  "fee": "0.00000229"      
```

## sendrawtransaction
    sendrawtransaction <hexstring>
Sends a signed transactions.

Parameter:  

    hexstring       signatured transaction 

Example:  
```bash
> masswallet-cli sendrawtransaction "080112a2010a260a2409bff543ef730be608118e613f104f953fcf19df4f41ba9599c68b21953686418809d2971248473044022010b789a4ac96e6c6f3caafc2b9f548a0e97cf1ad6bfe4a850ad73a3a4818861a0220532098aa7fa238511fdac8ab7aaf2541027e29f4b4d3eccb882d45263c860a0901122551210203a70a76734af100ed151ecf69ec776e4ef03ca0df06457176c1d61bb4c9d52e51ae19ffffffffffffffff1a2a0898a0d6b90712220020403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064"
```

Return:  
```json
{
  "tx_id": "2c8d77bb380786822acce767139eb8441027637d0c3dd248cc0ddf070bb52dc8"
}
```

## gettransactionstatus
    gettransactionstatus <txid>


Parameter:  

    txid        

Example:  
```bash
> masswallet-cli gettransactionstatus 2c8d77bb380786822acce767139eb8441027637d0c3dd248cc0ddf070bb52dc8
```

Return:  
```json
{
  "code": 1,
  "status": "confirmed"     //state:  [missing, packing, confirming, confirmed]
}
```

## getrawtransaction
    getrawtransaction <txid>


Parameter:  

    txid        

Example:  
```bash
> masswallet-cli getrawtransaction 06a47d7d1c1e3702c944e9bd300b97b9d40080407b61b933db64205f13b6c101
```

Return:  
```json
{
  "hex": "080112a2010a260a2409bff543ef730be608118e613f104f953fcf19df4f41ba9599c68b21953686418809d2971248473044022010b789a4ac96e6c6f3caafc2b9f548a0e97cf1ad6bfe4a850ad73a3a4818861a0220532098aa7fa238511fdac8ab7aaf2541027e29f4b4d3eccb882d45263c860a0901122551210203a70a76734af100ed151ecf69ec776e4ef03ca0df06457176c1d61bb4c9d52e51ae19ffffffffffffffff1a2a0898a0d6b90712220020403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064",
  "tx_id": "2c8d77bb380786822acce767139eb8441027637d0c3dd248cc0ddf070bb52dc8",
  "version": 1,
  "lock_time": "0",
  "block": {
    "height": "15695",
    "block_hash": "ce4c5d1ef257c7b83f0c5117f56ddc663e244fbca0909b11c48d88d97eccba85",
    "timestamp": "1566924154"
  },
  "vin": [
    {
      "value": "20",
      "n": 0,
      "type": 1,
      "redeem_detail": {
        "tx_id": "08e60b73ef43f5bfcf3f954f103f618e8bc69995ba414fdf97d2098841863695",
        "vout": 0,
        "sequence": "18446744073709551615",
        "witness": [
          "473044022010b789a4ac96e6c6f3caafc2b9f548a0e97cf1ad6bfe4a850ad73a3a4818861a0220532098aa7fa238511fdac8ab7aaf2541027e29f4b4d3eccb882d45263c860a0901",
          "51210203a70a76734af100ed151ecf69ec776e4ef03ca0df06457176c1d61bb4c9d52e51ae"
        ],
        "addresses": [
          "ms1qqf8870v59cdaanj3cxgfq97d3xpz94g9gqqsvz0wnj7lmlp9ehr2sxdj0um"
        ]
      }
    }
  ],
  "vout": [
    {
      "value": "19.99999",
      "n": 0,
      "type": 1,
      "script_detail": {
        "asm": "0 403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064",
        "hex": "0020403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064",
        "req_sigs": 1,
        "addresses": [
          "ms1qqgq750hhj0lcem3pmv6gymvvmfmrzmvhyxqwyp4fey2cmec372pjq3uw7yg"
        ]
      }
    }
  ],
  "payload": "",
  "confirmations": "82",
  "size": 207,
  "fee": "0.00001",
  "status": 1,
  "coinbase": false
}
```

## listtransactions
    listtransactions [count=?] [address=?]
Returns the __count__ most recent transactions of current wallet.

Parameter:  

    count       optional.Maximum number of queries
    address     optional.Specify specific address to query, if null, return the latest transaction of wallet

Example:  
```bash
> masswallet-cli listtransactions count=10 address=ms1qqgrq0g20u8tpq2vv0596vm3uxh0ptn72449wvpr86gaqk0gx78scqmp7jyl
```

Return:  
```json
{
  "histories": [
    {
      "tx_id": "2c8d77bb380786822acce767139eb8441027637d0c3dd248cc0ddf070bb52dc8",
      "block_height": "15695",
      "inputs": [
        {
          "tx_id": "08e60b73ef43f5bfcf3f954f103f618e8bc69995ba414fdf97d2098841863695",
          "index": "0"
        }
      ],
      "outputs": [
        {
          "address": "ms1qqgq750hhj0lcem3pmv6gymvvmfmrzmvhyxqwyp4fey2cmec372pjq3uw7yg",
          "amount": "19.99999"
        }
      ],
      "fromAddress": [
        "ms1qqf8870v59cdaanj3cxgfq97d3xpz94g9gqqsvz0wnj7lmlp9ehr2sxdj0um"
      ]
    },
    {
      "tx_id": "509fbdaae3094ad09cb66c2690d6a360f152c2c9f761334611c55688ed1e3969",
      "block_height": "9186",
      "inputs": [
        {
          "tx_id": "fc43279431abf0b21c3aa598e2a456d99d2bb0276b6b91ba5439567a391a0041",
          "index": "2"
        },
        {
          "tx_id": "4e7eaeb45c53a3a889548a9bce6cbad51ec912b75f62ff3e9dd0640676cc2c76",
          "index": "4"
        },
        {
          "tx_id": "7bb4ab913f0b861d7ec304826b8df775cc4cb7e10285e5d507ae71d1991853a8",
          "index": "1"
        }
      ],
      "outputs": [
        {
          "address": "ms1qqvc0l9yjq735ekmnfgccj3v045kxzfurlsw6c980tt88sh8f7dajsy5nh2t",
          "amount": "30"
        },
        {
          "address": "ms1qqxzj6gm4rh3w609a3752lfz8fw0p4tk7haa90juylzrxsjeclzlxqw5kjku",
          "amount": "0.99996001"
        }
      ],
      "fromAddress": [
        "ms1qqxzj6gm4rh3w609a3752lfz8fw0p4tk7haa90juylzrxsjeclzlxqw5kjku",
        "ms1qqf8870v59cdaanj3cxgfq97d3xpz94g9gqqsvz0wnj7lmlp9ehr2sxdj0um",
        "ms1qqku7shpmxnwj08evpxng29p8sk79vvm7u9g4h3l7lqqm6m069xmgsd7mz4z"
      ]
    },
    {
      "tx_id": "08e60b73ef43f5bfcf3f954f103f618e8bc69995ba414fdf97d2098841863695",
      "block_height": "1279",
      "inputs": [
        {
          "tx_id": "fa674e92900c85b198470b5314636e819a0e07ef6d4561b14779f713b62e1eb6",
          "index": "0"
        }
      ],
      "outputs": [
        {
          "address": "ms1qqf8870v59cdaanj3cxgfq97d3xpz94g9gqqsvz0wnj7lmlp9ehr2sxdj0um",
          "amount": "20"
        },
        {
          "address": "ms1qqjvj8m939wttulaf3yeu9znt4r43ce59v0642rvp44xcrszq5gccsg0vpj3",
          "amount": "1.1"
        },
        {
          "address": "ms1qqq2kmgr489z6llkz2c35aszem9umheuw2prayh7ycue2gt3xglnzs07ggjc",
          "amount": "18.89999"
        }
      ],
      "fromAddress": [
        "ms1qqq2kmgr489z6llkz2c35aszem9umheuw2prayh7ycue2gt3xglnzs07ggjc"
      ]
    },
    {
      "tx_id": "4e7eaeb45c53a3a889548a9bce6cbad51ec912b75f62ff3e9dd0640676cc2c76",
      "block_height": "1278",
      "inputs": [
        {
          "tx_id": "c45ebe383900017939cbf4ff2ce336bc2a963e4c7716cd68f404cbcdcf47abb5",
          "index": "0"
        },
        {
          "tx_id": "dada2adc1ec814609b5a81014ddb91cb4106f2ec7859ee644b1b27a2aa5aab58",
          "index": "0"
        }
      ],
      "outputs": [
        {
          "address": "ms1qqjvj8m939wttulaf3yeu9znt4r43ce59v0642rvp44xcrszq5gccsg0vpj3",
          "amount": "1"
        },
        {
          "address": "ms1qqkfmu3m5uxsvzuf47whs0ddylmlzd96x8qvt3snraczq2kwxqawhqe4r4uu",
          "amount": "40.1"
        },
        {
          "address": "ms1qqvfw97vna33laeef9xggwavfkp9ekz4z0gvgjymk6qam7483phy8qq2kx6x",
          "amount": "30.00000001"
        },
        {
          "address": "ms1qqd24an0hvnkfvvlhfx5he6exd5da0yfrm0qwp93runqwp054z4hgqvvs7et",
          "amount": "30.9"
        },
        {
          "address": "ms1qqf8870v59cdaanj3cxgfq97d3xpz94g9gqqsvz0wnj7lmlp9ehr2sxdj0um",
          "amount": "10"
        },
        {
          "address": "ms1qqq2kmgr489z6llkz2c35aszem9umheuw2prayh7ycue2gt3xglnzs07ggjc",
          "amount": "27.99998999"
        }
      ],
      "fromAddress": [
        "ms1qqq2kmgr489z6llkz2c35aszem9umheuw2prayh7ycue2gt3xglnzs07ggjc"
      ]
    }
  ]
}
```

## listlateststakingreward
    listlateststakingreward
Returns staking rewards in the latest block.

Parameter:  
    null

Example:  
```bash
> masswallet-cli listlateststakingreward
```

Return:  
```json
{
  "details": [
    {
      "rank": 0,
      "amount": "100",  // in MASS
      "weight": 1760000000000,
      "address": "ms1qp0czrc8errz8gdmpjgxd59kwvydf3g3ch72d6qm2kqwzlgm232pksqw0eky",
      "profit": "20"    // in MASS
    }
  ]
}
```

## createstakingtransaction
    createstakingtransaction <staking_address> <frozen_period> <value> [fee=?] [from=?]
Creates a transactions with randomly selected utxos from current wallet.

Parameter:  

        staking_address     
        frozen_period       
        value               
        fee                 optional.specify transaction fee
        from                optional.specify the source address of the payment amount

Example:  
```bash
> masswallet-cli createstakingtransaction ms1qp0czrc8errz8gdmpjgxd59kwvydf3g3ch72d6qm2kqwzlgm232pksqw0eky 100 100 from=ms1qqku7shpmxnwj08evpxng29p8sk79vvm7u9g4h3l7lqqm6m069xmgsd7mz4z
```

Return:  
```json
{
  "hex": "080112330a280a240961a30352ce076ae11116685dc3be033d121957f6e6d405beaaaa21767604eedb312ba5100119ffffffffffffffff1a330880c8afa025122b00207e043c1f23188e86ec32419b42d9cc2353144717f29ba06d560385f46d51506d086400000000000000"
}
```

## liststakingtransactions
    liststakingtransactions
Returns all staking transactions of current wallet.

Parameter: 

    null

Example:  
```bash
> masswallet-cli liststakingtransactions
```

Return:  
```json
{
  "txs": [
    {
      "tx_id": "b5785a1aecfcfd54bb2bab50686e48bc107e9e9744d213554123c7c02809d116",
      "status": 1,
      "block_height": "15856",
      "utxo": {
        "tx_id": "b5785a1aecfcfd54bb2bab50686e48bc107e9e9744d213554123c7c02809d116",
        "vout": 0,
        "address": "ms1qp0czrc8errz8gdmpjgxd59kwvydf3g3ch72d6qm2kqwzlgm232pksqw0eky",
        "amount": "100",
        "frozen_period": 200
      }
    },
    {
      "tx_id": "9781f1db9d8f6f8e40c83d27e53c858bd3ba6da2b270c55a3c4a258bf8668334",
      "status": 2,
      "block_height": "162",
      "utxo": {
        "tx_id": "9781f1db9d8f6f8e40c83d27e53c858bd3ba6da2b270c55a3c4a258bf8668334",
        "vout": 0,
        "address": "ms1qp0czrc8errz8gdmpjgxd59kwvydf3g3ch72d6qm2kqwzlgm232pksqw0eky",
        "amount": "1000",
        "frozen_period": 200
      }
    },
    {
      "tx_id": "d12d72e278e5899f3b18d6a08bc878f02f1b6d81620638cc5f8f31951e19cdde",
      "status": 2,
      "block_height": "154",
      "utxo": {
        "tx_id": "d12d72e278e5899f3b18d6a08bc878f02f1b6d81620638cc5f8f31951e19cdde",
        "vout": 0,
        "address": "ms1qpxzj6gm4rh3w609a3752lfz8fw0p4tk7haa90juylzrxsjeclzlxq3lxhtz",
        "amount": "1000",
        "frozen_period": 200
      }
    }
  ],
  "weights": {
    "ms1qp0czrc8errz8gdmpjgxd59kwvydf3g3ch72d6qm2kqwzlgm232pksqw0eky": 1170000000000
  }
}
```

## createbindingtransaction
    createbindingtransaction <outputs> [fee=?] [from=?]
Creates a binding transaction with randomly selected utxos from current wallet.

Parameter:  

    outputs     
    fee                 optional.specify transaction fee
    from                optional.specify the source address of the payment amount

Example:  
```bash
> masswallet-cli createbindingtransaction "[{\"holder_address\": \"ms1qqku7shpmxnwj08evpxng29p8sk79vvm7u9g4h3l7lqqm6m069xmgsd7mz4z\", \"binding_address\": \"18gsEwbYu65Qjwz4dUtKpYqfyYawQF8yga\", \"amount\": \"0.15625\"}]" from=ms1qqku7shpmxnwj08evpxng29p8sk79vvm7u9g4h3l7lqqm6m069xmgsd7mz4z
```

Return:  
```json
{
  "hex": "080112330a280a24093e9a6e9cf58f7fa01109809b9c6611011219a0ea5be4d297e5ba2194cfad2d9e53a3bb100119ffffffffffffffff1a3e08a8d6b90712370020b73d0b87669ba4f3e58134d0a284f0b78ac66fdc2a2b78ffdf0037adbf4536d11454530aa28d86011c30101584907e26d870bf9c421a2908eef28e3412220020b73d0b87669ba4f3e58134d0a284f0b78ac66fdc2a2b78ffdf0037adbf4536d1"
}
```

## listbindingtransactions
    listbindingtransactions
Returns all binding transactions of current wallet.

Parameter:  

    null

Example:  
```bash
> masswallet-cli listbindingtransactions
```

Return:  
```json
{
  "histories": [
    {
      "tx_id": "8fca0d35aa8219b5c9b6bb402ddbcb7fdb1cc12fb73f739f8f2d781d461e0623",
      "status": 1,
      "block_height": "58",
      "utxo": {
        "tx_id": "8fca0d35aa8219b5c9b6bb402ddbcb7fdb1cc12fb73f739f8f2d781d461e0623",
        "vout": 0,
        "holder_address": "ms1qqku7shpmxnwj08evpxng29p8sk79vvm7u9g4h3l7lqqm6m069xmgsd7mz4z",
        "binding_address": "18gsEwbYu65Qjwz4dUtKpYqfyYawQF8yga",
        "amount": "0.15625"
      },
      "from_addresses": [
        "ms1qqku7shpmxnwj08evpxng29p8sk79vvm7u9g4h3l7lqqm6m069xmgsd7mz4z"
      ]
    },
    {
      "tx_id": "8fca0d35aa8219b5c9b6bb402ddbcb7fdb1cc12fb73f739f8f2d781d461e0623",
      "status": 1,
      "block_height": "58",
      "utxo": {
        "tx_id": "8fca0d35aa8219b5c9b6bb402ddbcb7fdb1cc12fb73f739f8f2d781d461e0623",
        "vout": 1,
        "holder_address": "ms1qqku7shpmxnwj08evpxng29p8sk79vvm7u9g4h3l7lqqm6m069xmgsd7mz4z",
        "binding_address": "1EJY8EKpP91T9WbGD1BJihDDdL8x1fSAfc",
        "amount": "0.15625"
      },
      "from_addresses": [
        "ms1qqku7shpmxnwj08evpxng29p8sk79vvm7u9g4h3l7lqqm6m069xmgsd7mz4z"
      ]
    },
    {
      "tx_id": "8fca0d35aa8219b5c9b6bb402ddbcb7fdb1cc12fb73f739f8f2d781d461e0623",
      "status": 1,
      "block_height": "58",
      "utxo": {
        "tx_id": "8fca0d35aa8219b5c9b6bb402ddbcb7fdb1cc12fb73f739f8f2d781d461e0623",
        "vout": 2,
        "holder_address": "ms1qqku7shpmxnwj08evpxng29p8sk79vvm7u9g4h3l7lqqm6m069xmgsd7mz4z",
        "binding_address": "14AkJD4kHBWk8PY7AU8rECAHYrQzg835Qi",
        "amount": "0.15625"
      },
      "from_addresses": [
        "ms1qqku7shpmxnwj08evpxng29p8sk79vvm7u9g4h3l7lqqm6m069xmgsd7mz4z"
      ]
    }
  ]
}
```

## getaddresstotalbinding
    getaddresstotalbinding <poc_address>...
Returns total binding MASS on the specified __poc_address__, which has no relevance to the wallet context.

Parameter:  

    poc_address     poc address

Example:  
```bash
> masswallet-cli getaddresstotalbinding 18gsEwbYu65Qjwz4dUtKpYqfyYawQF8yga
```

Return:  
```json
{
  "amounts": {
    "18gsEwbYu65Qjwz4dUtKpYqfyYawQF8yga": "0.15625" // in MASS
  }
}
```