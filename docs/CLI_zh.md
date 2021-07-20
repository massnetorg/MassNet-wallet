# 配置文件
> 文件路径`./conf/config.json`  
> 如果使用https协议，TLS证书需要放在目录`./conf/`下，文件名必须为`cert.crt`和`cert.key`

配置选项
```json
{
    "server": "https://localhost:9688",
    "log_dir": "./logs",
    "log_level": "info"
}
```

# 查看全部可用命令
```bash
> masswallet-cli -h,--help
```
# 基本用法
```bash
> masswallet-cli [command <required args> [optional args]]
```
# 命令帮助
```bash
> masswallet-cli [command] --help
```

# Command

> 注意：
>- 以`<>`标注的参数均为必填项
>- 以`[]`标注的参数均为选填项

## createcert
    createcert <directory>
生成TLS证书。

参数：

    directory       存放路径

示例：
```bash
> masswallet-cli createcert .
```

## getclientstatus
    getclientstatus
查询钱包节点信息。

参数：

    无

示例：
```bash
> masswallet-cli getclientstatus
```

返回结果：
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
查询节点的最新区块信息。

参数：
    无

示例：
```bash
> masswallet-cli getbestblock
```

返回结果：
```json
{
  "height": "8993",         //高度
  "target": "1f56abb05"     //挖矿难度，16进制表示
}
```

## stop
    stop
关闭节点。

参数：
    无

示例：
```bash
> masswallet-cli stop
```

返回结果：
```json
{
  "code": "200",
  "msg": "wait for client quitting process"
}
```

## listwallets
    listwallets
显示节点管理的所有钱包摘要。

参数：
    无

示例：
```bash
> masswallet-cli listwallets
```

返回结果：
```json
{
  "wallets": [
    {
      "wallet_id": "ac102yfx0q2v6v3aug35hw42jn8k6sljeypffn85w3",        //钱包ID，用于钱包相关操作
      "type": 1,                //固定值 1
      "version": 0,             //钱包版本，0或1
      "remarks": "for test",    //备注信息
      "status": 0|1|2,      // 0-ready, 2-removing, 1-syncing
      "status_msg": "ready"|"removing"|<synced_height>
    }
  ]
}
```

## usewallet
    usewallet <wallet_id>
切换当前正在使用的钱包上下文。所有交易相关命令仅在具体钱包上下文才能使用

参数：

    wallet_id       钱包ID

示例：
```bash
> masswallet-cli usewallet ac102yfx0q2v6v3aug35hw42jn8k6sljeypffn85w3
```

返回结果：
```json
{
  "chain_id": "e931abb77f2568f752a29ed28d442558764a5961ed773df7188430a0e0f7cf18",
  "wallet_id": "ac102yfx0q2v6v3aug35hw42jn8k6sljeypffn85w3",
  "type": 1,                                                   //固定值 1
  "version": 0,             //钱包版本，0或1
  "total_balance": "1234.00001428",                            //总余额，单位mass
  "external_key_count": 10,                                    //external地址总数
  "internal_key_count": 0,                                     //internal地址总数，目前未使用
  "remarks": "for test"
}
```

## createwallet
    createwallet [entropy=?] [remarks=?]
创建最新版本钱包。

参数：

    entropy       生成助记词的初始熵长度，必须为[128,256]范围内32的整数倍数字，默认128。
    remarks       钱包备注信息，没有任何链上语义。

示例：
```bash
> masswallet-cli createwallet entropy=160 remarks="仅供测试"
```

返回结果：
```json
{
  "wallet_id": "ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds",     //钱包ID
  "mnemonic": "figure vapor flame artwork clarify local right insect fall pulp dwarf steel tip author pulse",            //助记词
  "version": 1
}
```

## getwalletmnemonic
    getwalletmnemonic <wallet_id>
查询钱包助记词。

参数：

    wallet_id

示例：
```bash
> masswallet-cli getwalletmnemonic ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds
```

返回结果：
```json
{
  "mnemonic": "figure vapor flame artwork clarify local right insect fall pulp dwarf steel tip author pulse",
  "version": 1
}
```

## exportwallet
    exportwallet <wallet_id>
导出指定钱包。

参数：

    wallet_id

示例：
```bash
> masswallet-cli exportwallet ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds
```

返回结果：
```json
{
  "keystore": "{\"remarks\":\"仅供测试\",\"crypto\":{\"cipher\":\"Stream cipher\",\"entropyEnc\":\"ac6de145205481dc95ab6a22f8b978edb3f1b2d69f5036a2158339bc7465c0a7311d6d08983ef8e2f5d443dad7dc7580095b024cd809ab6006743624\",\"kdf\":\"scrypt\",\"pubParams\":\"\",\"privParams\":\"b999d54a2369d9be570928c32e0a727e95eb395ad8ba5383608c9cb800a4cad82c52fe054cdcdfae5b3d751810452425704095a32fcb6cea85913575ac39174c000004000000000008000000000000000100000000000000\",\"cryptoKeyPubEnc\":\"\",\"cryptoKeyPrivEnc\":\"\",\"cryptoKeyEntropyEnc\":\"cac4a9b04cca089d396430cb9115627d97e914590183b5a835e7cbf76ae6b85ae4a6cbaafe12e2fa1a43f52c78c5aa62e9d3f5265baf9eb2618e020b1d5ce5555d9cc69f005c6a44\"},\"hdPath\":{\"Purpose\":44,\"Coin\":1,\"Account\":1,\"ExternalChildNum\":0,\"InternalChildNum\":0}}"
}
```

## removewallet
    removewallet <wallet_id> 
删除指定钱包。

参数：

    wallet_id

示例：
```bash
> masswallet-cli removewallet ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds
```

返回结果：
```json
{
  "ok": true
}
```

## importwallet
    importwallet <keystore>
导入钱包，通过keystore。

参数：

    keystore        json data

示例1：
```bash
> masswallet-cli importwallet '{"remarks":"仅供测试","crypto":{"cipher":"Stream cipher","entropyEnc":"ac6de145205481dc95ab6a22f8b978edb3f1b2d69f5036a2158339bc7465c0a7311d6d08983ef8e2f5d443dad7dc7580095b024cd809ab6006743624","kdf":"scrypt","pubParams":"","privParams":"b999d54a2369d9be570928c32e0a727e95eb395ad8ba5383608c9cb800a4cad82c52fe054cdcdfae5b3d751810452425704095a32fcb6cea85913575ac39174c000004000000000008000000000000000100000000000000","cryptoKeyPubEnc":"","cryptoKeyPrivEnc":"","cryptoKeyEntropyEnc":"cac4a9b04cca089d396430cb9115627d97e914590183b5a835e7cbf76ae6b85ae4a6cbaafe12e2fa1a43f52c78c5aa62e9d3f5265baf9eb2618e020b1d5ce5555d9cc69f005c6a44"},"hdPath":{"Purpose":44,"Coin":1,"Account":1,"ExternalChildNum":0,"InternalChildNum":0}}'
```

示例2：
```bash
> masswallet-cli importwallet "{\"remarks\":\"仅供测试\",\"crypto\":{\"cipher\":\"Stream cipher\",\"entropyEnc\":\"ac6de145205481dc95ab6a22f8b978edb3f1b2d69f5036a2158339bc7465c0a7311d6d08983ef8e2f5d443dad7dc7580095b024cd809ab6006743624\",\"kdf\":\"scrypt\",\"pubParams\":\"\",\"privParams\":\"b999d54a2369d9be570928c32e0a727e95eb395ad8ba5383608c9cb800a4cad82c52fe054cdcdfae5b3d751810452425704095a32fcb6cea85913575ac39174c000004000000000008000000000000000100000000000000\",\"cryptoKeyPubEnc\":\"\",\"cryptoKeyPrivEnc\":\"\",\"cryptoKeyEntropyEnc\":\"cac4a9b04cca089d396430cb9115627d97e914590183b5a835e7cbf76ae6b85ae4a6cbaafe12e2fa1a43f52c78c5aa62e9d3f5265baf9eb2618e020b1d5ce5555d9cc69f005c6a44\"},\"hdPath\":{\"Purpose\":44,\"Coin\":1,\"Account\":1,\"ExternalChildNum\":0,\"InternalChildNum\":0}}"
```

返回结果：
```json
{
  "ok": true,
  "wallet_id": "ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds",
  "type": 1,
  "version":0,
  "remarks": "仅供测试"
}
```

## importmnemonic
    importmnemonic <version> <mnemonic> [initial=?] [remarks=?]
导入钱包助记词，若version与mnemonic不匹配则不能正确恢复钱包。

参数：

    version         助记词的版本，v0或v1
    mnemonic        标注助记词短语
    initial         选填。初始地址数目
    remarks         选填。

示例：
```bash
> masswallet-cli importmnemonic v0 "figure vapor flame artwork clarify local right insect fall pulp dwarf steel tip author pulse"
```

返回结果：
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
查询当前使用的钱包的余额。

参数：

    minconf        选填。仅统计至少经过min_conf次区块确认的utxo，默认值1
    detail         选填。是否统计可花费（提现）的各个部分总额，默认false

示例：
```bash
> masswallet-cli getwalletbalance minfconf=100 detail=true
```

返回结果：
```json
{
  "wallet_id": "ac10uz28q8yjevkvvfva84txu2dztsahu7mqxlvxds",
  "total": "100.00001428",            //总余额
  "detail": {
    "spendable": "100.00001428",      //总余额中可花费的非抵押（或锁定）部分
    "withdrawable_staking": "0",      //总余额中锁定到期可赎回部分
    "withdrawable_binding": "0"       //总余额中抵押的可赎回部分
  }
}
```

## listaddresses
    listaddresses <version>
显示当前钱包指定类型地址。

参数：

    version     0  - 创建普通地址
                1  - 创建锁定地址

示例：
```bash
> masswallet-cli listaddresses 1
```

返回结果：
```json
{
  "details": [
    {
      "address": "ms1qp0czrc8errz8gdmpjgxd59kwvydf3g3ch72d6qm2kqwzlgm232pksqw0eky",
      "version": 1,             //0-普通地址，1-锁定交易地址
      "used": true|false,       //该地址在链上是否发生交易
      "std_address": "ms1qq0czrc8errz8gdmpjgxd59kwvydf3g3ch72d6qm2kqwzlgm232pksl9lut6"          //当version=1时，对应的收益地址，否则为空
    }
  ]
}
```

## createaddress
    createaddress <version>
创建新地址。（当前钱包）

参数：

    version     0  - 创建普通地址
                1  - 创建锁定地址

示例：
```bash
> masswallet-cli createaddress 0
```

返回结果：
```json
{
    "address": "ms1qqgrq0g20u8tpq2vv0596vm3uxh0ptn72449wvpr86gaqk0gx78scqmp7jyl"  
}
```

## validateaddress
    validateaddress <address>
检验地址是否格式正确且属于当前钱包。

参数；

    address     任意地址

示例：
```bash
> masswallet-cli validateaddress ms1qqgrq0g20u8tpq2vv0596vm3uxh0ptn72449wvpr86gaqk0gx78scqmp7jyl
```

返回结果：
```json
{
  "is_valid": true|false,        //格式是否正确  
  "is_mine": true|false,         //是否属于当前钱包，当is_valid=false无意义 
  "address": "",                 //原样返回被验证的地址     
  "version": 0|1                 //0-普通地址，1-锁定地址，当is_valid=false无意义 
}
```

## getaddressbalance
    getaddressbalance <min_conf> [<address> <address> ...]
查询当前钱包指定地址上的余额。

参数：

    min_conf        必填。仅统计至少经过min_conf次区块确认的utxo
    address         选填。待查询的地址

示例：
```bash
> masswallet-cli getaddressbalance 1 ms1qqf8870v59cdaanj3cxgfq97d3xpz94g9gqqsvz0wnj7lmlp9ehr2sxdj0um ms1qqgq750hhj0lcem3pmv6gymvvmfmrzmvhyxqwyp4fey2cmec372pjq3uw7yg
```

返回结果：
```json
{
  "balances": [
    {
      "address": "ms1qqf8870v59cdaanj3cxgfq97d3xpz94g9gqqsvz0wnj7lmlp9ehr2sxdj0um",
      "total": "20",                    //总余额
      "spendable": "20",                //总余额中可花费的非抵押（或锁定）部分
      "withdrawable_staking": "0",      //总余额中锁定到期可赎回部分
      "withdrawable_binding": "0"       //总余额中抵押的可赎回部分
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
查询当前钱包地址地址的utxo。

参数：

    address     地址，0（查询当前钱包所有地址）或多个

示例：
```bash
> masswallet-cli listutxo ms1qqf8870v59cdaanj3cxgfq97d3xpz94g9gqqsvz0wnj7lmlp9ehr2sxdj0um
```

返回结果：
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
创建交易。

参数：

    inputs      交易输入
                [{"tx_id":"...","vout":0},...]
    outputs     交易输出
                {"address1":"vlaue1","address2":"vlaue2",...}，value单位：mass
    locktime    选填。交易锁定时间/高度，默认0

示例：
```bash
> masswallet-cli createrawtransaction '[{"tx_id":"08e60b73ef43f5bfcf3f954f103f618e8bc69995ba414fdf97d2098841863695","vout":0}]' '{"ms1qqgq750hhj0lcem3pmv6gymvvmfmrzmvhyxqwyp4fey2cmec372pjq3uw7yg":"19.99999"}'
```

返回结果：
```json
{
  "hex": "080112310a260a2409bff543ef730be608118e613f104f953fcf19df4f41ba9599c68b21953686418809d29719ffffffffffffffff1a2a0898a0d6b90712220020403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064"
}
```

## autocreaterawtransaction
    autocreaterawtransaction <outputs> [locktime=?] [fee=?] [from=?]
从当前钱包随机选择utxo构建交易。（当前钱包）

参数：  

    outputs     必填。交易输出，格式：{"address":"value",...}
                    address  -  普通mass地址
                    value    -  金额（单位：mass)， 最多8位小数
    fee         选填。指定交易费（单位：mass），最多8位小数
    from        选填。指定utxo的来源地址，否则从钱包的所有地址随机选择

示例：
```bash
> masswallet-cli autocreaterawtransaction '{"ms1qqgq750hhj0lcem3pmv6gymvvmfmrzmvhyxqwyp4fey2cmec372pjq3uw7yg": "19.99999", ...}' from=ms1qqf8870v59cdaanj3cxgfq97d3xpz94g9gqqsvz0wnj7lmlp9ehr2sxdj0um

// win
> masswallet-cli autocreaterawtransaction "{\"ms1qqgq750hhj0lcem3pmv6gymvvmfmrzmvhyxqwyp4fey2cmec372pjq3uw7yg\": \"19.99999\", ...}" from=ms1qqf8870v59cdaanj3cxgfq97d3xpz94g9gqqsvz0wnj7lmlp9ehr2sxdj0um
```

返回结果：
```json
{
    "hex":"080112310a260a2409bff543ef730be608118e613f104f953fcf19df4f41ba9599c68b21953686418809d29719ffffffffffffffff1a2a0898a0d6b90712220020403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064"     //未签名、序列化、经hex编码的交易
}
```

## signrawtransaction
    signrawtransaction <hexstring> [mode=?]
交易签名。

参数：

    hexstring       待签名交易
    mode            选填。默认"ALL"
                    ALL
                    NONE
                    SINGLE
                    ALL|ANYONECANPAY
                    NONE|ANYONECANPAY
                    SINGLE|ANYONECANPAY

示例：
```bash
> masswallet-cli signrawtransaction 080112310a260a2409bff543ef730be608118e613f104f953fcf19df4f41ba9599c68b21953686418809d29719ffffffffffffffff1a2a0898a0d6b90712220020403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064
```

返回结果：
```json
{
  "hex": "080112a2010a260a2409bff543ef730be608118e613f104f953fcf19df4f41ba9599c68b21953686418809d2971248473044022010b789a4ac96e6c6f3caafc2b9f548a0e97cf1ad6bfe4a850ad73a3a4818861a0220532098aa7fa238511fdac8ab7aaf2541027e29f4b4d3eccb882d45263c860a0901122551210203a70a76734af100ed151ecf69ec776e4ef03ca0df06457176c1d61bb4c9d52e51ae19ffffffffffffffff1a2a0898a0d6b90712220020403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064",
  "complete": true
}
```

## gettransactionfee
    gettransactionfee <outputs> <inputs> [binding=true]
预估交易费用。

参数：

    outputs         交易输出
                    {"address1":"vlaue1","address2":"vlaue2",...}，value单位：mass
    inputs          交易输出，可以为空
                    [{"tx_id":"...","vout":0},...]
    binding        选填。是否包含抵押，默认false

示例：
```bash
> masswallet-cli gettransactionfee '{"ms1qqgq750hhj0lcem3pmv6gymvvmfmrzmvhyxqwyp4fey2cmec372pjq3uw7yg":"19.99999"}' '[{"tx_id":"08e60b73ef43f5bfcf3f954f103f618e8bc69995ba414fdf97d2098841863695","vout":0}]'
```

返回结果：
```json
{
  "fee": "0.00000229"       //单位：mass
}
```

## sendrawtransaction
    sendrawtransaction <hexstring>
发送交易。

参数：

    hexstring       签名交易

示例：
```bash
> masswallet-cli sendrawtransaction "080112a2010a260a2409bff543ef730be608118e613f104f953fcf19df4f41ba9599c68b21953686418809d2971248473044022010b789a4ac96e6c6f3caafc2b9f548a0e97cf1ad6bfe4a850ad73a3a4818861a0220532098aa7fa238511fdac8ab7aaf2541027e29f4b4d3eccb882d45263c860a0901122551210203a70a76734af100ed151ecf69ec776e4ef03ca0df06457176c1d61bb4c9d52e51ae19ffffffffffffffff1a2a0898a0d6b90712220020403d47def27ff19dc43b66904db19b4ec62db2e4301c40d53922b1bce23e5064"
```

返回结果：
```json
{
  "txId": "2c8d77bb380786822acce767139eb8441027637d0c3dd248cc0ddf070bb52dc8"
}
```

## gettransactionstatus
    gettransactionstatus <txid>
查询交易状态。

参数：

    txid        交易id

示例：
```bash
> masswallet-cli gettransactionstatus 2c8d77bb380786822acce767139eb8441027637d0c3dd248cc0ddf070bb52dc8
```

返回结果：
```json
{
  "code": 1,
  "status": "confirmed"     //交易状态：[missing, packing, confirming, confirmed]
}
```

## getrawtransaction
    getrawtransaction <txid>
查询交易详情。

参数：

    txid        交易id

示例：
```bash
> masswallet-cli getrawtransaction 06a47d7d1c1e3702c944e9bd300b97b9d40080407b61b933db64205f13b6c101
```

返回结果：
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
查询当前钱包最近的N条交易历史。

参数：

    count       选填。查询的最大条数
    address     选填。指定具体地址查询，若没有则返回钱包最近的交易

示例：
```bash
> masswallet-cli listtransactions count=10 address=ms1qqgrq0g20u8tpq2vv0596vm3uxh0ptn72449wvpr86gaqk0gx78scqmp7jyl
```

返回结果：
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
查询指定区块上的锁定奖励。

参数：
    
    height           选填，默认查询最新区块上的锁定奖励

示例：
```bash
> masswallet-cli getblockstakingreward
```

返回结果：
```json
{
  "height": 12999,
  "details": [
    {
      "rank": 0,
      "amount": "100", //单位MASS
      "weight": 1760000000000,
      "address": "ms1qp0czrc8errz8gdmpjgxd59kwvydf3g3ch72d6qm2kqwzlgm232pksqw0eky",
      "profit": "20"
    }
  ]
}
```

## createstakingtransaction
    createstakingtransaction <staking_address> <frozen_period> <value> [fee=?] [from=?]
创建锁定交易（当前钱包）。

参数：

        staking_address     锁定地址
        frozen_period       锁定高度（相对高度），经过该高度之后才能将锁定的mass提出
        value               锁定的金额，单位：mass，最多8位小数
        fee                 选填。指定交易费，单位：mass，最多8位小数
        from                选填。指定支付金额的来源地址

示例：
```bash
> masswallet-cli createstakingtransaction ms1qp0czrc8errz8gdmpjgxd59kwvydf3g3ch72d6qm2kqwzlgm232pksqw0eky 100 100 from=ms1qqku7shpmxnwj08evpxng29p8sk79vvm7u9g4h3l7lqqm6m069xmgsd7mz4z
```

返回结果：
```json
{
  "hex": "080112330a280a240961a30352ce076ae11116685dc3be033d121957f6e6d405beaaaa21767604eedb312ba5100119ffffffffffffffff1a330880c8afa025122b00207e043c1f23188e86ec32419b42d9cc2353144717f29ba06d560385f46d51506d086400000000000000"
}
```

## liststakingtransactions
    liststakingtransactions [all]
查询当前钱包的锁定交易记录。

参数： 

    [all]   选填，查询所有的锁定交易，包括已经提现的。默认不包括

示例：
```bash
> masswallet-cli liststakingtransactions all
```

返回结果：
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
创建抵押交易（当前钱包）。

参数：

    outputs     输出
    fee                 选填。指定交易费，单位：mass，最多8位小数
    from                选填。指定支付金额的来源地址

示例：
```bash
> masswallet-cli createbindingtransaction "[{\"holder_address\": \"ms1qqku7shpmxnwj08evpxng29p8sk79vvm7u9g4h3l7lqqm6m069xmgsd7mz4z\", \"binding_address\": \"18gsEwbYu65Qjwz4dUtKpYqfyYawQF8yga\", \"amount\": \"0.15625\"}]" from=ms1qqku7shpmxnwj08evpxng29p8sk79vvm7u9g4h3l7lqqm6m069xmgsd7mz4z
```

返回结果：
```json
{
  "hex": "080112330a280a24093e9a6e9cf58f7fa01109809b9c6611011219a0ea5be4d297e5ba2194cfad2d9e53a3bb100119ffffffffffffffff1a3e08a8d6b90712370020b73d0b87669ba4f3e58134d0a284f0b78ac66fdc2a2b78ffdf0037adbf4536d11454530aa28d86011c30101584907e26d870bf9c421a2908eef28e3412220020b73d0b87669ba4f3e58134d0a284f0b78ac66fdc2a2b78ffdf0037adbf4536d1"
}
```

## listbindingtransactions
    listbindingtransactions [all]
查询当前钱包的抵押交易记录。

参数：

    [all]   选填，查询所有的绑定交易，包括已经提现的。默认不包括

示例：
```bash
> masswallet-cli listbindingtransactions all
```

返回结果：
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

## getaddresstotalbinding
    getaddresstotalbinding <poc_address>...
查询指定poc地址（非钱包地址）上的抵押总金额，与钱包上下文无关。

参数：

    poc_address     poc地址，非钱包地址

示例：
```bash
> masswallet-cli getaddresstotalbinding 18gsEwbYu65Qjwz4dUtKpYqfyYawQF8yga
```

返回结果：
```json
{
  "amounts": {
    "18gsEwbYu65Qjwz4dUtKpYqfyYawQF8yga": "0.15625" //单位MASS
  }
}
```