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
      "version": 0,
      "remarks": "for test",   
      "status": 0|1|2,      // 0-ready, 2-removing, 1-syncing
      "status_msg": "ready"|"removing"|<synced_height>
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
  "version": 0,                                                    
  "total_balance": "1234.00001428",                            
  "external_key_count": 10,                                       //external pk(address) total
  "internal_key_count": 0,                                        //internal pk total ,not currently in use
  "remarks": "for test"
}
```

## createwallet
    createwallet [entropy=?] [remarks=?]

Parameter:  

    entropy       The initial entropy length for generating mnemonics must be an integer multiple of 32 in the range of [128,256]. The default is 128.
    remarks       Note information of wallet, without any chain semantics.

Example:  
```bash
> masswallet-cli createwallet entropy=160 remarks="for test"

// Set 6 to 40 valid characters to encrypt the wallet
> Enter password: 
```

Return:  
```json
{
  "wallet_id": "ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds",     
  "mnemonic": "figure vapor flame artwork clarify local right insect fall pulp dwarf steel tip author pulse",            //mnemonics
  "version": 1
}
```

## getwalletmnemonic
    getwalletmnemonic
Returns the mnemonic of currently used wallet.

Parameter:  

    wallet_id

Example:  
```bash
> masswallet-cli getwalletmnemonic ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds
```

Return:  
```json
{
  "mnemonic": "figure vapor flame artwork clarify local right insect fall pulp dwarf steel tip author pulse",
  "version": 0
}
```

## exportwallet
    exportwallet <wallet_id>

Parameter:  

    wallet_id

Example:  
```bash
> masswallet-cli exportwallet ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds
```

Return:  
```json
{
  "keystore": "{\"remarks\":\"for test\",\"crypto\":{\"cipher\":\"Stream cipher\",\"entropyEnc\":\"ac6de145205481dc95ab6a22f8b978edb3f1b2d69f5036a2158339bc7465c0a7311d6d08983ef8e2f5d443dad7dc7580095b024cd809ab6006743624\",\"kdf\":\"scrypt\",\"pubParams\":\"\",\"privParams\":\"b999d54a2369d9be570928c32e0a727e95eb395ad8ba5383608c9cb800a4cad82c52fe054cdcdfae5b3d751810452425704095a32fcb6cea85913575ac39174c000004000000000008000000000000000100000000000000\",\"cryptoKeyPubEnc\":\"\",\"cryptoKeyPrivEnc\":\"\",\"cryptoKeyEntropyEnc\":\"cac4a9b04cca089d396430cb9115627d97e914590183b5a835e7cbf76ae6b85ae4a6cbaafe12e2fa1a43f52c78c5aa62e9d3f5265baf9eb2618e020b1d5ce5555d9cc69f005c6a44\"},\"hdPath\":{\"Purpose\":44,\"Coin\":1,\"Account\":1,\"ExternalChildNum\":0,\"InternalChildNum\":0}}"
}
```

## removewallet
    removewallet <wallet_id>

Parameter:  

    wallet_id

Example:  
```bash
> masswallet-cli removewallet ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds
```

Return:  
```json
{
  "ok": true
}
```

## importwallet
    importwallet <keystore>
Imports a wallet by keystore.

Parameter:  

    keystore        json data

Example1:  
```bash
> masswallet-cli importwallet '{"remarks":"for test","crypto":{"cipher":"Stream cipher","entropyEnc":"ac6de145205481dc95ab6a22f8b978edb3f1b2d69f5036a2158339bc7465c0a7311d6d08983ef8e2f5d443dad7dc7580095b024cd809ab6006743624","kdf":"scrypt","pubParams":"","privParams":"b999d54a2369d9be570928c32e0a727e95eb395ad8ba5383608c9cb800a4cad82c52fe054cdcdfae5b3d751810452425704095a32fcb6cea85913575ac39174c000004000000000008000000000000000100000000000000","cryptoKeyPubEnc":"","cryptoKeyPrivEnc":"","cryptoKeyEntropyEnc":"cac4a9b04cca089d396430cb9115627d97e914590183b5a835e7cbf76ae6b85ae4a6cbaafe12e2fa1a43f52c78c5aa62e9d3f5265baf9eb2618e020b1d5ce5555d9cc69f005c6a44"},"hdPath":{"Purpose":44,"Coin":1,"Account":1,"ExternalChildNum":0,"InternalChildNum":0}}'
```

Example2:  
```bash
> masswallet-cli importwallet "{\"remarks\":\"for test\",\"crypto\":{\"cipher\":\"Stream cipher\",\"entropyEnc\":\"ac6de145205481dc95ab6a22f8b978edb3f1b2d69f5036a2158339bc7465c0a7311d6d08983ef8e2f5d443dad7dc7580095b024cd809ab6006743624\",\"kdf\":\"scrypt\",\"pubParams\":\"\",\"privParams\":\"b999d54a2369d9be570928c32e0a727e95eb395ad8ba5383608c9cb800a4cad82c52fe054cdcdfae5b3d751810452425704095a32fcb6cea85913575ac39174c000004000000000008000000000000000100000000000000\",\"cryptoKeyPubEnc\":\"\",\"cryptoKeyPrivEnc\":\"\",\"cryptoKeyEntropyEnc\":\"cac4a9b04cca089d396430cb9115627d97e914590183b5a835e7cbf76ae6b85ae4a6cbaafe12e2fa1a43f52c78c5aa62e9d3f5265baf9eb2618e020b1d5ce5555d9cc69f005c6a44\"},\"hdPath\":{\"Purpose\":44,\"Coin\":1,\"Account\":1,\"ExternalChildNum\":0,\"InternalChildNum\":0}}"
```

Return:  
```json
{
  "ok": true,
  "wallet_id": "ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds",
  "type": 1,
  "version": 0,
  "remarks": "for test"
}
```

## importmnemonic
    importmnemonic <mnemonic> [initial=?] [remarks=?]
Imports a wallet backup mnemonic.

Parameter:  

    mnemonic        
    initial   optional, number of initial addresses
    remarks   optional

Example:  
```bash
> masswallet-cli importmnemonic "figure vapor flame artwork clarify local right insect fall pulp dwarf steel tip author pulse"
```

Return:  
```json
{
  "ok": true,
  "wallet_id": "ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds",
  "type": 1,
  "version": 0,
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

## decoderawtransaction
    decoderawtransaction <hex>
Decodes hex-encoded transaction.

Parameter:

    <hex>     hex encoded transaction.

Example:
```bash
```

## createrawtransaction
    createrawtransaction <json_data>
Creates a transaction spending given inputs of current wallet.

Parameter:  

    <json_data>:
        - inputs              required
        - amounts             required
        - lock_time           optional
        - change_address      optional, the first sender address will be used by default.
        - subtractfeefrom     optional, if not provided, the sender pays the fee.

Example:  
```bash
> masswallet-cli createrawtransaction '{"inputs":[{"tx_id": "af03d3916639143e343628ba9286c33a70752bf6bc495512dbd093c18e033bc0", "vout": 1}],"amounts":{"ms1qqwmyrmca0zfcpyhjv7tdek2mvsrtr6yzrm8g227r4ryadn42hs0hst2gvut": "0.999"},"change_address":"ms1qq8mg72nwy02g0zpej0247rwtccycy3zrjmv8na5vl3yp6dgttd7ds0pa2df","subtractfeefrom": ["ms1qqwmyrmca0zfcpyhjv7tdek2mvsrtr6yzrm8g227r4ryadn42hs0hst2gvut"]}'
```

Return:  
```json
{
  "hex": "080112310a260a2409bff543ef730be608118e613f104f953fcf19df4f41ba9599c68b21953686418809d29719ffffffffffffffff1a2a0898a0d6b90712220020403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064"
}
```

## autocreaterawtransaction
    autocreaterawtransaction <json_data>
Creates a transaction with randomly selected utxos from the current wallet.

Parameter:   

    <json_data>: 
        - amounts             required  
        - fee                 optional, floating fee with max 8 decimal places
        - lock_time           optional
        - change_address      optional, the first sender address will be used by default.
        - from_address        optional, specific sender, if not provided, the inputs may be selected from any address of current wallet.

Example:  
```bash
> masswallet-cli autocreaterawtransaction '{"amounts":{"ms1qqwmyrmca0zfcpyhjv7tdek2mvsrtr6yzrm8g227r4ryadn42hs0hst2gvut": "1.01"},"change_address":"ms1qq8mg72nwy02g0zpej0247rwtccycy3zrjmv8na5vl3yp6dgttd7ds0pa2df","fee":"0.005"}'
```

Return:  
```json
{
    "hex":"080112310a260a2409bff543ef730be608118e613f104f953fcf19df4f41ba9599c68b21953686418809d29719ffffffffffffffff1a2a0898a0d6b90712220020403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064"    
}
```

## signrawtransaction
    signrawtransaction <hexstring> [mode=?]
Signs a transaction.

Parameter:  

    hexstring       Transactions to be signed
    mode            optional.default "ALL"
                    ALL
                    NONE
                    SINGLE
                    ALL|ANYONECANPAY
                    NONE|ANYONECANPAY
                    SINGLE|ANYONECANPAY

Example:  
```bash
> masswallet-cli signrawtransaction 080112310a260a2409bff543ef730be608118e613f104f953fcf19df4f41ba9599c68b21953686418809d29719ffffffffffffffff1a2a0898a0d6b90712220020403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064

// Enter wallet password
> Enter password:
```

Return:  
```json
{
  "hex": "080112a2010a260a2409bff543ef730be608118e613f104f953fcf19df4f41ba9599c68b21953686418809d2971248473044022010b789a4ac96e6c6f3caafc2b9f548a0e97cf1ad6bfe4a850ad73a3a4818861a0220532098aa7fa238511fdac8ab7aaf2541027e29f4b4d3eccb882d45263c860a0901122551210203a70a76734af100ed151ecf69ec776e4ef03ca0df06457176c1d61bb4c9d52e51ae19ffffffffffffffff1a2a0898a0d6b90712220020403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064",
  "complete": true
}
```

## gettransactionfee
    gettransactionfee <outputs> <inputs> [binding=true]
Estimates transaction fee.

Parameter:  

    outputs         format:{"address1":"vlaue1","address2":"vlaue2",...}
    inputs          format:[{"tx_id":"...","vout":0},...]
    binding         optional.Include staking or not,default false

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
> masswallet-cli getrawtransaction fe7104abbc30b56ec62c092966eccdabaec7034bf87ae1eb653572ad648902e9
```

Return:  
```json
{  "hex": "080112a5010a280a2409338ec05a33cd12c5117564055e95df7bc4197d62963fb196bdac21b8eaf3a6a8aa698010011249483045022100cf67a3663729ddf708500a293274fce5694e68df92d4fadf6c1cdaa2d90105be02203b0162b309017e46e0334999b7270cb8c0f8d5fa415fa7fc946a623397a3717301122551210356830b4780dc5f5463aa91eeaa6508f93698e039fce8cf17db363ba99afd790451ae19ffffffffffffffff12a4010a280a2409017da021af9f30b911f18c999e15c4934a19ee0733fc5aadeacc21cfc1e765b6ee3ab110011248473044022053839e7e9afe0b56b4da9e58e9da7c0371a7cec6eb383d74dab704ec8f4b7d2a02203c635076d031779de22231afd355195b346551ead655aaeee81df8249cc75d6501122551210356830b4780dc5f5463aa91eeaa6508f93698e039fce8cf17db363ba99afd790451ae19ffffffffffffffff1a3f08e0efc1a0251237002076c83de3af1270125e4cf2db9b2b6c80d63d1043d9d0a57875193ad9d55783ef14f5000204050607080102030605060708000203041a2a08d0a7fcf41e122200200c315878dffef12a9f2c9dfde6a68b43c0fd2ffe63f94e7cf3459411d3866907",
  "txId": "fe7104abbc30b56ec62c092966eccdabaec7034bf87ae1eb653572ad648902e9",
  "version": 1,
  "block": {
    "height": "3322",
    "blockHash": "c012ce1b7617fcd4a8711ea0d63e80d4a8691971b9aef9d3e729ece9b6252fad",
    "timestamp": "1624690938"
  },
  "vin": [
    {
      "value": "91.9989",
      "type": 1,
      "redeemDetail": {
        "txId": "c512cd335ac08e33c47bdf955e056475acbd96b13f96627d8069aaa8a6f3eab8",
        "vout": 1,
        "sequence": "18446744073709551615",
        "witness": [
          "483045022100cf67a3663729ddf708500a293274fce5694e68df92d4fadf6c1cdaa2d90105be02203b0162b309017e46e0334999b7270cb8c0f8d5fa415fa7fc946a623397a3717301",
          "51210356830b4780dc5f5463aa91eeaa6508f93698e039fce8cf17db363ba99afd790451ae"
        ],
        "fromAddress": "ms1qqpsc4s7xllmcj48evnh77df5tg0q06tl7v0u5ul8ngk2pr5uxdyrspx4x5g"
      }
    },
    {
      "value": "90.9879",
      "n": 1,
      "type": 1,
      "redeemDetail": {
        "txId": "b9309faf21a07d014a93c4159e998cf1cceaad5afc3307eeb13aeeb665e7c1cf",
        "vout": 1,
        "sequence": "18446744073709551615",
        "witness": [
          "473044022053839e7e9afe0b56b4da9e58e9da7c0371a7cec6eb383d74dab704ec8f4b7d2a02203c635076d031779de22231afd355195b346551ead655aaeee81df8249cc75d6501",
          "51210356830b4780dc5f5463aa91eeaa6508f93698e039fce8cf17db363ba99afd790451ae"
        ],
        "fromAddress": "ms1qqpsc4s7xllmcj48evnh77df5tg0q06tl7v0u5ul8ngk2pr5uxdyrspx4x5g"
      }
    }
  ],
  "vout": [
    {
      "value": "100.003",
      "type": 3,
      "scriptDetail": {
        "asm": "0 76c83de3af1270125e4cf2db9b2b6c80d63d1043d9d0a57875193ad9d55783ef f500020405060708010203060506070800020304",
        "hex": "002076c83de3af1270125e4cf2db9b2b6c80d63d1043d9d0a57875193ad9d55783ef14f500020405060708010203060506070800020304",
        "reqSigs": 1,
        "recipientAddress": "ms1qqwmyrmca0zfcpyhjv7tdek2mvsrtr6yzrm8g227r4ryadn42hs0hst2gvut",
        "bindingTarget": "1PLSZYeBhp6UW1MXaBtCqesJ5oQq4YEoyc:MASS:0"
      }
    },
    {
      "value": "82.9837",
      "n": 1,
      "type": 1,
      "scriptDetail": {
        "asm": "0 0c315878dffef12a9f2c9dfde6a68b43c0fd2ffe63f94e7cf3459411d3866907",
        "hex": "00200c315878dffef12a9f2c9dfde6a68b43c0fd2ffe63f94e7cf3459411d3866907",
        "reqSigs": 1,
        "recipientAddress": "ms1qqpsc4s7xllmcj48evnh77df5tg0q06tl7v0u5ul8ngk2pr5uxdyrspx4x5g"
      }
    }
  ],
  "confirmations": "370",
  "size": 424,
  "fee": "0.0001",
  "status": 1
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

## getblockstakingreward
    getblockstakingreward [height]
Returns staking reward list at target height.

Parameter:  

    height      optional, returns the list at best block by default.

Example:  
```bash
> masswallet-cli getblockstakingreward
```

Return:  
```json
{
  "height": 12999,
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

    liststakingtransactions [all]
Returns staking transactions of current wallet.

Parameter: 

    [all]  - returns all stakings, including withdrawn.

Example:  
```bash
> masswallet-cli liststakingtransactions all
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
Returns binding transactions of current wallet.

    listbindingtransactions [all]

Parameter:  

    [all] - returns all bindings, including withdrawn.

Example:  
```bash
> masswallet-cli listbindingtransactions all
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
        "amount": "0.15625",
        "target_type": "MASS",
        "target_size": 0
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
        "binding_address": "18W8DkbtU6i8advRSkUaSocZqkE2JnDDZEbGH",
        "amount": "0.15625",
        "target_type": "MASS",
        "target_size": 32
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
        "binding_address": "18W8DkbtU6i8advRSkUaSocZqkE2Jni2JNFhT",
        "amount": "0.15625",
        "target_type": "Chia",
        "target_size": 32
      },
      "from_addresses": [
        "ms1qqku7shpmxnwj08evpxng29p8sk79vvm7u9g4h3l7lqqm6m069xmgsd7mz4z"
      ]
    }
  ]
}
```

## checkpoolpkcoinbase
    checkpoolpkcoinbase <chia pool pubkey> <chia pool pubkey> ...

Parameter:  

  \<chia pool pubkey\>

Example:  
```bash
> masswallet-cli checkpoolpkcoinbase 8919b3715c0e8998c5d2f36f1236c7ab0d44b8285644effe2ee0d9f54a6dadf0efc6bbd0917371b2e9462186ac99c948 7719b3715c0e8998c5d2f36f1236c7ab0d44b8285644effe2ee0d9f54a6dadf0efc6bbd0917371b2e9462186ac99c948
```

Return:  
```json
{
  "result": {
    "8919b3715c0e8998c5d2f36f1236c7ab0d44b8285644effe2ee0d9f54a6dadf0efc6bbd0917371b2e9462186ac99c948": {        // has bound coinbase
      "nonce": 6,
      "coinbase": "ms1qq2gyvf5khdpnafyhedcm3syvla5ntzhdz2zj69nf65v5yw35zy2fsc7s6vs"
    },
    "7719b3715c0e8998c5d2f36f1236c7ab0d44b8285644effe2ee0d9f54a6dadf0efc6bbd0917371b2e9462186ac99c948": {        // never bound
      // nonce is 0, 0 means never bound
      // coinbase is ""
    },
    "97d5be5d8612daf12a1658afe2ed2b8e708bb1d4128d0f31d71fa1272eff3ee66a4edec12aaae0e4f0a3d4421e2624c4": {       // bound coinbase is cleared
      "nonce": 7
      // coinbase is ""
    }
  }
}
```

## getnetworkbinding
    getnetworkbinding [height]

Parameter:  

  height      optional, use best height if not specified or zero

Example:  
```bash
> masswallet-cli getnetworkbinding
```

Return:  
```json
{
  "height": "4932",
  "totalBinding": "45059.5772956 MASS",
  "bindingPriceMassBitlength": {
    "32": "0.50860595 MASS",
    "34": "2.0344238 MASS",
    "36": "9.1549071 MASS",
    "38": "38.6540522 MASS",
    "40": "162.753904 MASS"
  },
  "bindingPriceChiaK": {
    "32": "2.0344238 MASS",
    "33": "4.0688476 MASS",
    "34": "8.64630115 MASS",
    "35": "17.80120825 MASS",
    "36": "37.12823435 MASS",
    "37": "76.2908925 MASS",
    "38": "156.6506326 MASS",
    "39": "321.4389604 MASS",
    "40": "659.1533112 MASS"
  }
}
```

## checktargetbinding
    checktargetbinding <target> <target>
Returns total bound MASS on the specified __target address__

Parameter:  

    target     base58 encoded address.

Example:  
```bash
> masswallet-cli checktargetbinding 146hGPwfYRDde6tJ6trbyhkSoPwt69AqyZ 1EgzSkV7vJ7xhC5g38ULLPoMBhHVW38VZN 14LQhx7dGPFyfRS7rYv4uKVdKjoyAJejcVVqw 18gsEwbYu65Qjwz4dUtKpYqfyYawQF8yga
```

Return:  
```json
{
  "result": {
    "146hGPwfYRDde6tJ6trbyhkSoPwt69AqyZ": {
      "targetType": "MASS",
      "amount": "0 MASS"
    },
    "14LQhx7dGPFyfRS7rYv4uKVdKjoyAJejcVVqw": {
      "targetType": "MASS",
      "targetSize": 34,
      "amount": "4.5776367 MASS"
    },
    "18gsEwbYu65Qjwz4dUtKpYqfyYawQF8yga": {
      "targetType": "MASS",
      "amount": "100.002 MASS"
    },
    "1EgzSkV7vJ7xhC5g38ULLPoMBhHVW38VZN": {
      "targetType": "MASS",
      "amount": "0 MASS"
    }
  }
}
```

## batchbinding
    batchbinding -c <file>
    batchbinding <file> <from>
Batch check or send binding transactions from file.

Parameter:  

  file        - Required, file storing targets to be bound. Exported by 'massminercli'.
  from        - Specify the address to pay for bindings. Ignored if flag '-c' is set. 
  
  file sample
  ```json
  {
    "plots": [
      {
        "target": "17rkPoiqpWwdyuFnM2buHrs8kwfXZGEvx3iqp",
        "type": 0,  // MASS
        "size": 34
      },
      {
        "target": "17JDi7zj8PpgDVTdZvZAvmQy2t785EQfgSzRe",
        "type": 1,  // Chia
        "size": 32
      }
    ],
    "total_count": 2,
    "default_count": 1,
    "chia_count": 1
  }
  ```

Example:  
```bash
> masswallet-cli batchbinding binding_list.json ms1qqg8qxsllfkpmt6mpfu9rpk3k86f45fvy2hsu0997ck37mnklvpsuqzf8fme

// Enter wallet password to sign transactions.
> Enter password: 
```

## batchbindpoolpk
    batchbindpoolpk -c <chiaKeystore>
    batchbindpoolpk <chiaKeystore> <from> [coinbase]
Check or bind coinbase for chia pool pubkey.

Parameter:  

  chiaKeystore    - Required, keystore storing chia poolSks/poolPks. Exported by 'massminercli'.  
  from            - Specify the address to pay for the transaction. Ensure it has at least 1.01 MASS. 
  coinbase        - Specify coinbase to be bound to poolpk, clear already bound coinbase if not provided.


Example:  
```bash
> masswallet-cli batchbindpoolpk chia-miner-keystore.json ms1qqpsc4s7xllmcj48evnh77df5tg0q06tl7v0u5ul8ngk2pr5uxdyrspx4x5g  ms1qqg8qxsllfkpmt6mpfu9rpk3k86f45fvy2hsu0997ck37mnklvpsuqzf8fme
```

Return:  
```json
{
  "hex": "080112330a280a2409da45322b721715a0117513a7ee4ea8bfac193850811d324a992e2131eba0232bd9d2ad100119ffffffffffffffff1a2808c0843d122200200c315878dffef12a9f2c9dfde6a68b43c0fd2ffe63f94e7cf3459411d38669071a2a08a0adf98f13122200200c315878dffef12a9f2c9dfde6a68b43c0fd2ffe63f94e7cf3459411d38669072ab60100018919b3715c0e8998c5d2f36f1236c7ab0d44b8285644effe2ee0d9f54a6dadf0efc6bbd0917371b2e9462186ac99c948b3a20ffb39ad711c2fe6c102f028a12f9bd16b6d99b676598529ac3bee094e0a069562ab2c9f5d6fdb56be73a7aafb6403c97488e3621fc1eede30bf65e702658a479e7716268b9097d2dae7886f58ab97603c3c60f91189cca0a4d0241e00620000000741c0687fe9b076bd6c29e1461b46c7d26b44b08abc38f297d8b47db9dbec0c38"
}
```