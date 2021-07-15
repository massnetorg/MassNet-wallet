# Endpoint
| URL | Protocol |
| ------ | ------ |
| http://localhost:9688 | HTTP |

# API methods
* [GetBestBlock](#getbestblock)
* [GetBlockByHeight](#getblockbyheight)
* [GetClientStatus](#getclientstatus)
* [Wallets](#wallets)
* [CreateWallet](#createwallet)
* [UseWallet](#usewallet)
* [ImportWallet](#importwallet)
* [ImportMnemonic](#importmnemonic)
* [ExportWallet](#exportwallet)
* [RemoveWallet](#removewallet)
* [GetWalletMnemonic](#getwalletmnemonic)
* [GetWalletBalance](#getwalletbalance)
* [CreateAddress](#createaddress)
* [GetAddresses](#getaddresses)
* [GetAddressBalance](#getaddressbalance)
* [ValidateAddress](#validateaddress)
* [GetUtxo](#getutxo)
* [DecodeRawTransaction](#decoderawtransaction)
* [CreateRawTransaction](#createrawtransaction)
* [AutoCreateTransaction](#autocreatetransaction)
* [SignRawTransaction](#signrawtransaction)
* [GetTransactionFee](#gettransactionfee)
* [SendRawTransaction](#sendrawtransaction)
* [GetRawTransaction](#getrawtransaction)
* [GetTxStatus](#gettxstatus)
* [CreateStakingTransaction](#createstakingtransaction)
* [GetStakingHistory](#getstakinghistory)
* [GetBlockStakingReward](#getblockstakingreward)
* [TxHistory](#txhistory)
* [GetBindingHistory](#getbindinghistory)
* [CreateBindingTransaction](#createbindingtransaction)
* [CreatePoolPkCoinbaseTransaction](#CreatePoolPkCoinbaseTransaction)
* [GetNetworkBinding](#GetNetworkBinding)
* [CheckPoolPkCoinbase](#CheckPoolPkCoinbase)
* [CheckTargetBinding](#CheckTargetBinding)
---

## GetBestBlock
    GET /v1/blocks/best
### Parameters
null
### Returns
Same with response of GetBlockByHeight


## GetBlockByHeight
    POST /v1/blocks/height/{height}
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| height | uint64 |  |  |
### Returns
- `String` - hash
- `String` - chain_id
- `Integer` - version
- `Integer` - height
- `Integer` - confirmations 
- `Integer` - time 
- `String` - previous_hash 
- `String` - next_hash 
- `String` - transaction_root
- `String` - witness_root
- `String` - proposal_root
- `String` - target 
- `String` - quality 
- `String` - challenge 
- `String` - public_key 
- `Object` - proof
    - `String` - x 
    - `String` - x_prime
    - `Integer` - bit_length
- `Object` - block_signature 
    - `String` - r
    - `String` - s
- `Array of String` - ban_list
- `Object` - proposal_area
    - `Array of FaultPubKey` - punishment_area
        - FaultPubKey
            - `Integer` - version
            - `Integer` - proposal_type
            - `String` - public_key 
            - `Array of Header` - testimony
            - Header
                - `String` - hash
                - `String` - chain_id
                - `Integer` - version
                - `Integer` - height
                - `Integer` - time
                - `String` - previous_hash
                - `String` - transaction_root
                - `String` - witness_root
                - `String` - proposal_root
                - `String` - target 
                - `String` - challenge 
                - `String` - public_key 
                - `Object` - proof 
                - `Object` - block_signature 
                - `Array of String` - ban_list 
    - `Array of NormalProposal` - other_area 
        - NormalProposal
            - `Integer` - version
            - `Integer` - proposal_type
            - `String` - data
- `Array of TxRawResult` - raw_tx 
    - TxRawResult
        - `String` - txid
        - `Integer` - version
        - `Integer` - locktime
        - `Array of Vin` - vin
            - Vin
                - `String` - value 
                - `Integer` - n 
                - `Integer` - type  
                - `Object` - redeem_detail 
                    - `String` - tx_id 
                    - `Integer` - vout 
                    - `Integer` - sequence 
                    - `Array of String` - witness 
                    - `String` - from_address
                    - `String` - staking_address, empty when type != 2
                    - `String` - binding_target, empty when type != 3. The format is `{target}:{type}:{size}`
                        - target, binding address
                        - type, "MASS" or "Chia"
                        - size, bitlength of MASS or K size of Chia, `0` means it is an old binding.
        - `Array of Vout` - vout
            - Vout
                - `String` - value 
                - `Integer` - n 
                - `Integer` - type  
                - `Object` - script_detail 
                    - `String` - asm 
                    - `String` - hex 
                    - `Integer` - req_sigs 
                    - `String` - recipient_address
                    - `String` - staking_address, empty when type != 2
                    - `String` - binding_target, empty when type != 3. The format is `{target}:{type}:{size}`
                        - target, binding address
                        - type, "MASS" or "Chia"
                        - size, bitlength of MASS or K size of Chia, `0` means it is an old binding.
        - `String` - payload
        - `Integer` - confirmations 
        - `Integer` - size 
        - `String` - fee 
        - `Integer` - status 
        - `Integer` - type 
- `Integer` - size 
- `String` - time_utc
- `Integer` - tx_count
### Example
```json
{
    "hash": "c80fa760fafa2786569fe08b1806c58d7fcd3b4ce627c1394c4336ad0e5135d2",
    "chain_id": "5433524b370b149007ba1d06225b5d8e53137a041869834cff5860b02bebc5c7",
    "version": "1",
    "height": "1398566",
    "confirmations": "1",
    "time": "1626323043",
    "previous_hash": "81f54be45ff8aebe1a6d7963dc66688e7a6637b9de9c8170aeb4707fcc0712e1",
    "next_hash": "",
    "transaction_root": "502dbdd6d87eeab02381b69f154bb57f2d077d0e0242c933c14eda6f706199d4",
    "witness_root": "502dbdd6d87eeab02381b69f154bb57f2d077d0e0242c933c14eda6f706199d4",
    "proposal_root": "9663440551fdcd6ada50b1fa1b0003d19bc7944955820b54ab569eb9a7ab7999",
    "target": "8c63a876b2a009c",
    "quality": "90679a49084f800",
    "challenge": "8970c9f24a744394f8ba329b949632657845ec5b7e36fa6836f5ab0847611f57",
    "public_key": "023cdd1692353875ccbb48310cf75fe446f14bcf6afaddd34fc0a5469a9b548260",
    "proof": {
        "x": "e2665d0200",
        "x_prime": "2441199300",
        "bit_length": 34
    },
    "block_signature": {
        "r": "ba02ee8d3ea95a44354dce44151773b25083cff4695df94f9041f642ffcce0de",
        "s": "574a50167b99fdfae6ab48b0f995dbbe7ca4faff5b453b5baa949dd5c3b576bb"
    },
    "ban_list": [],
    "proposal_area": {
        "punishment_area": [],
        "other_area": []
    },
    "raw_tx": [
        {
            "txid": "502dbdd6d87eeab02381b69f154bb57f2d077d0e0242c933c14eda6f706199d4",
            "version": 1,
            "lock_time": "0",
            "vin": [],
            "vout": [
                {
                    "value": "0.84002164",
                    "n": 0,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 a954e0df16658df017a43b04e5684be723be4c1438f37984cecacd2094bdfe53",
                        "hex": "0020a954e0df16658df017a43b04e5684be723be4c1438f37984cecacd2094bdfe53",
                        "req_sigs": 1,
                        "recipient_address": "ms1qq492wphckvkxlq9ay8vzw26ztuu3munq58rehnpxwetxjp99alefsnxfv9g",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.82277638",
                    "n": 1,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 dbd539881041e3f8fa2bc5a65c48863beacb3a2f1704dbf585cb8637bda79b42",
                        "hex": "0020dbd539881041e3f8fa2bc5a65c48863beacb3a2f1704dbf585cb8637bda79b42",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqm02nnzqsg83l373tckn9cjyx804vkw30zuzdhav9ewrr00d8ndpqhc8jr0",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.82154566",
                    "n": 2,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 2e86cb356002b17f8e4f9c627efa0d3987d4cd609f2c74a0e56355e89ba7248a",
                        "hex": "00202e86cb356002b17f8e4f9c627efa0d3987d4cd609f2c74a0e56355e89ba7248a",
                        "req_sigs": 1,
                        "recipient_address": "ms1qq96rvkdtqq2chlrj0n338a7sd8xrafntqnuk8fg89vd273xa8yj9qf9efyg",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.81365581",
                    "n": 3,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 416d9bf54b2db5445f8fb2e51af4278c0e90d16c9c6f45df91311bf83d3418fa",
                        "hex": "0020416d9bf54b2db5445f8fb2e51af4278c0e90d16c9c6f45df91311bf83d3418fa",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqg9keha2t9k65ghu0ktj34ap83s8fp5tvn3h5thu3xydls0f5rraqff5q4g",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.79997894",
                    "n": 4,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 05b96775c7a40dd77b65c2da33dd01b46619287b19812fdb9f9eb4e24c96224d",
                        "hex": "002005b96775c7a40dd77b65c2da33dd01b46619287b19812fdb9f9eb4e24c96224d",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqqkukwaw85sxaw7m9ctdr8hgpk3npj2rmrxqjlkuln66wynykyfxs8yxe3j",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.69829291",
                    "n": 5,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 1353a175870d78a185766c86df75346982d488f011dae47590657206f3114f69",
                        "hex": "00201353a175870d78a185766c86df75346982d488f011dae47590657206f3114f69",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqzdf6zav8p4u2rptkdjrd7af5dxpdfz8sz8dwgavsv4eqduc3fa5slpu8y6",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.6665162",
                    "n": 6,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 6dae90c56ddac07b5df818e035793c7114fd9545a986d442cc83466ef0a2b5fc",
                        "hex": "00206dae90c56ddac07b5df818e035793c7114fd9545a986d442cc83466ef0a2b5fc",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqdkhfp3tdmtq8kh0crrsr27fuwy20m9294xrdgskvsdrxau9zkh7qs7v3yd",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.64804178",
                    "n": 7,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 1e2b652e7b3453d4ad5483699195eac1809769656a127661737890d8b742a61d",
                        "hex": "00201e2b652e7b3453d4ad5483699195eac1809769656a127661737890d8b742a61d",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqrc4k2tnmx3faft25sd5er902cxqfw6t9dgf8vctn0zgd3d6z5cwstexsex",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.61995858",
                    "n": 8,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 595896193ead25938460df7a3617493a96cab89529cf1e0107d54f9d91e604a3",
                        "hex": "0020595896193ead25938460df7a3617493a96cab89529cf1e0107d54f9d91e604a3",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqt9vfvxf745je8prqmaarv96f82tv4wy49883uqg8648emy0xqj3s89emk2",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.56065103",
                    "n": 9,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 ecd9869679694103ec60e0084d9362a666b03f8d47c3e89c237b61854b2b7da7",
                        "hex": "0020ecd9869679694103ec60e0084d9362a666b03f8d47c3e89c237b61854b2b7da7",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqanvcd9ned9qs8mrquqyymymz5entq0udglp738pr0dsc2jet0knsszzp3c",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.55839668",
                    "n": 10,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 9cb5cbd134b5562276841489c8a29d5b73c022bd0f59e1556f3881218bc23d65",
                        "hex": "00209cb5cbd134b5562276841489c8a29d5b73c022bd0f59e1556f3881218bc23d65",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqnj6uh5f5k4tzya5yzjyu3g5atdeuqg4apav7z4t08zqjrz7z84js06lgkz",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.50461321",
                    "n": 11,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 283c0cd1c9617f57184257e67634030aeb1f06e4838026c921a7bd2e3d8d1ce2",
                        "hex": "0020283c0cd1c9617f57184257e67634030aeb1f06e4838026c921a7bd2e3d8d1ce2",
                        "req_sigs": 1,
                        "recipient_address": "ms1qq9q7qe5wfv9l4wxzz2ln8vdqrpt437physwqzdjfp577ju0vdrn3qnyc03w",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.46977882",
                    "n": 12,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 9d3c6fc05664d2160705516a14ca3c0656c1ed3a5d4c82e814e2ce4712daaacf",
                        "hex": "00209d3c6fc05664d2160705516a14ca3c0656c1ed3a5d4c82e814e2ce4712daaacf",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqn57xlszkvnfpvpc9294pfj3uqetvrmf6t4xg96q5ut8ywyk64t8sp5cu4m",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.46316098",
                    "n": 13,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 6b81e287c465fe225b18d3ab7a3ffe569a2b567d3239142827f8998c4147f35c",
                        "hex": "00206b81e287c465fe225b18d3ab7a3ffe569a2b567d3239142827f8998c4147f35c",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqdwq79p7yvhlzykcc6w4h50l726dzk4naxgu3g2p8lzvccs287dwqyudwea",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.43534377",
                    "n": 14,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 71fdf38bafc9f8956ea1e8a24e6ee8e6f5a4de17af45cf0a660b582ee95bc862",
                        "hex": "002071fdf38bafc9f8956ea1e8a24e6ee8e6f5a4de17af45cf0a660b582ee95bc862",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqw87l8za0e8uf2m4paz3yumhgum66fhsh4azu7znxpdvza62mep3q89r4hh",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.43356502",
                    "n": 15,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 e3bb7cf9f714e3881774efdbb826458e034935c232dc1b5eee2329e4211afcd2",
                        "hex": "0020e3bb7cf9f714e3881774efdbb826458e034935c232dc1b5eee2329e4211afcd2",
                        "req_sigs": 1,
                        "recipient_address": "ms1qquwahe70hzn3cs9m5aldmsfj93cp5jdwzxtwpkhhwyv57ggg6lnfqmnwhaw",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.41792372",
                    "n": 16,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 c1b5659037d4ca43ced9eb4f922b47f2fabb0eb1367821b9058f3479b91a2b62",
                        "hex": "0020c1b5659037d4ca43ced9eb4f922b47f2fabb0eb1367821b9058f3479b91a2b62",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqcx6ktyph6n9y8nkead8ey2687tatkr43xeuzrwg93u68nwg69d3qg80afr",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.41747878",
                    "n": 17,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 ec74c6742eeb505fccb98d08a178b2253d981a4b199f929155e88a6ad16f43fc",
                        "hex": "0020ec74c6742eeb505fccb98d08a178b2253d981a4b199f929155e88a6ad16f43fc",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqa36vvapwadg9ln9e35y2z79jy57esxjtrx0e9y24az9x45t0g07qcfv2qr",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.32032566",
                    "n": 18,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 b3ff09544456b32249e20bcda98f4ec4d04f43973eb75803360f28745f02d7be",
                        "hex": "0020b3ff09544456b32249e20bcda98f4ec4d04f43973eb75803360f28745f02d7be",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqk0lsj4zy26ejyj0zp0x6nr6wcngy7suh86m4sqekpu58ghcz67lq5m520h",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.31187198",
                    "n": 19,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 6f645d618ea82fae5c132c5120830b29e2688476846948a6f5d8df3104e77628",
                        "hex": "00206f645d618ea82fae5c132c5120830b29e2688476846948a6f5d8df3104e77628",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqdaj96cvw4qh6uhqn93gjpqct983x3prks3553fh4mr0nzp88wc5q0mkse9",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.19607747",
                    "n": 20,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 367670e711bc3732a8744df20043186893db0ffb5a04a82130c3cf9a39ad7a04",
                        "hex": "0020367670e711bc3732a8744df20043186893db0ffb5a04a82130c3cf9a39ad7a04",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqxem8pec3hsmn92r5fheqqsccdzfakrlmtgz2sgfsc08e5wdd0gzqkdqqnc",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.19543884",
                    "n": 21,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 a0e8f42b3c3f2d75269d4099e6499583fde0c2e959788844b5e260fe5015db9e",
                        "hex": "0020a0e8f42b3c3f2d75269d4099e6499583fde0c2e959788844b5e260fe5015db9e",
                        "req_sigs": 1,
                        "recipient_address": "ms1qq5r50g2eu8ukh2f5agzv7vjv4s077pshft9ugs394ufs0u5q4mw0qj9mtms",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.1932686",
                    "n": 22,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 6735726aebb42644a4a672b9f92c4571d01e314424ec149e405f6e7ce8b3c1f2",
                        "hex": "00206735726aebb42644a4a672b9f92c4571d01e314424ec149e405f6e7ce8b3c1f2",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqvu6hy6htksnyff9xw2uljtz9w8gpuv2yynkpf8jqtah8e69nc8eqgfe36a",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.16105773",
                    "n": 23,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 543d442ee507ed9b6385dc964a85f583dd7a65d12012857acc545ac8cb07bea6",
                        "hex": "0020543d442ee507ed9b6385dc964a85f583dd7a65d12012857acc545ac8cb07bea6",
                        "req_sigs": 1,
                        "recipient_address": "ms1qq2s75gth9qlkekcu9mjty4p04s0wh5ew3yqfg27kv23dv3jc8h6nq36av6k",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.13711493",
                    "n": 24,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 eedfcf628770bd84b4abe7744da28ea9caf7bc77b1902bbe9c20fb78a5ae60ea",
                        "hex": "0020eedfcf628770bd84b4abe7744da28ea9caf7bc77b1902bbe9c20fb78a5ae60ea",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqam0u7c58wz7cfd9tua6ymg5w489000rhkxgzh05uyrah3fdwvr4q95zr6d",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.12639269",
                    "n": 25,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 2b2c6f7db5b0e21ed03c18ec2dd00fabb1b3fdb87883f582b713756b928a0f88",
                        "hex": "00202b2c6f7db5b0e21ed03c18ec2dd00fabb1b3fdb87883f582b713756b928a0f88",
                        "req_sigs": 1,
                        "recipient_address": "ms1qq9vkx7ld4kr3pa5purrkzm5q04wcm8ldc0zpltq4hzd6khy52p7yq4rpfdn",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.0966342",
                    "n": 26,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 4eccc77c13bce6de758bdfb97dd839c71d42138df292a7ea8e105ff444c95152",
                        "hex": "00204eccc77c13bce6de758bdfb97dd839c71d42138df292a7ea8e105ff444c95152",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqfmxvwlqnhnnduavtm7uhmkpecuw5yyud72f2065wzp0lg3xf29fqf2r04c",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.09409324",
                    "n": 27,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 598804cada1384f3b41448c5867ce91a8745dabc3736efcdaeea4d108808f954",
                        "hex": "0020598804cada1384f3b41448c5867ce91a8745dabc3736efcdaeea4d108808f954",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqtxyqfjk6zwz08dq5frzcvl8fr2r5tk4uxumwlndwafx3pzqgl92q9z78n3",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.09319501",
                    "n": 28,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 330fbdb6b46f700795bb0b23ccaa99bdb0f978d010aa092dbc88587560475784",
                        "hex": "0020330fbdb6b46f700795bb0b23ccaa99bdb0f978d010aa092dbc88587560475784",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqxv8mmd45dacq09dmpv3ue25ehkc0j7xszz4qjtdu3pv82cz827zq0lwdmu",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "0.08282957",
                    "n": 29,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 ecad9e706c80102eaa09448c1b41d2a8addb11e51d1ec1ab89363bd69eb8c1f0",
                        "hex": "0020ecad9e706c80102eaa09448c1b41d2a8addb11e51d1ec1ab89363bd69eb8c1f0",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqajkeuurvsqgza2sfgjxpkswj4zkaky09r50vr2ufxcaad84cc8cqsng8pe",
                        "staking_address": "",
                        "binding_target": ""
                    }
                },
                {
                    "value": "3.00000017",
                    "n": 30,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 25846db545ade6ebb08effa808bfa0babfc1027ee1d447f0b1c10a487364f869",
                        "hex": "002025846db545ade6ebb08effa808bfa0babfc1027ee1d447f0b1c10a487364f869",
                        "req_sigs": 1,
                        "recipient_address": "ms1qqykzxmd294hnwhvywl75q30aqh2luzqn7u82y0u93cy9ysumylp5sy3rm2k",
                        "staking_address": "",
                        "binding_target": ""
                    }
                }
            ],
            "payload": "26571500000000001e000000",
            "confirmations": "1",
            "size": 1370,
            "fee": "0",
            "status": 4,
            "type": 4
        }
    ],
    "size": 2712,
    "time_utc": "2021-07-15T04:24:03Z",
    "tx_count": 1,
    "binding_root": "0000000000000000000000000000000000000000000000000000000000000000"
}
```

## GetBlockByHeight
    POST /v1/blocks/height/{height}
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| height | uint64 |  |  |
### Returns
- `String` - hash
- `String` - chain_id
- `Integer` - version
- `Integer` - height
- `Integer` - confirmations 
- `Integer` - time 
- `String` - previous_hash 
- `String` - next_hash 
- `String` - transaction_root
- `String` - witness_root
- `String` - proposal_root
- `String` - target 
- `String` - quality 
- `String` - challenge 
- `String` - public_key 
- `Object` - proof
    - `String` - x 
    - `String` - x_prime
    - `Integer` - bit_length
- `Object` - block_signature 
    - `String` - r
    - `String` - s
- `Array of String` - ban_list
- `Object` - proposal_area
    - `Array of FaultPubKey` - punishment_area
        - FaultPubKey
            - `Integer` - version
            - `Integer` - proposal_type
            - `String` - public_key 
            - `Array of Header` - testimony
            - Header
                - `String` - hash
                - `String` - chain_id
                - `Integer` - version
                - `Integer` - height
                - `Integer` - time
                - `String` - previous_hash
                - `String` - transaction_root
                - `String` - witness_root
                - `String` - proposal_root
                - `String` - target 
                - `String` - challenge 
                - `String` - public_key 
                - `Object` - proof 
                - `Object` - block_signature 
                - `Array of String` - ban_list 
    - `Array of NormalProposal` - other_area 
        - NormalProposal
            - `Integer` - version
            - `Integer` - proposal_type
            - `String` - data
- `Array of TxRawResult` - raw_tx 
    - TxRawResult
        - `String` - txid
        - `Integer` - version
        - `Integer` - locktime
        - `Array of Vin` - vin
            - Vin
                - `String` - value 
                - `Integer` - n 
                - `Integer` - type  
                - `Object` - redeem_detail 
                    - `String` - tx_id 
                    - `Integer` - vout 
                    - `Integer` - sequence 
                    - `Array of String` - witness 
                    - `Array of String` - addresses // addresses[0]
                                                    //          holder_address of input utxo
                                                    // addresses[1]    -  not exists when type=1
                                                    //          type=2: staking_address
                                                    //          type=3: binding_address
        - `Array of Vout` - vout
            - Vout
                - `String` - value 
                - `Integer` - n 
                - `Integer` - type  
                - `Object` - script_detail 
                    - `String` - asm 
                    - `String` - hex 
                    - `Integer` - req_sigs 
                    - `Array of String` - addresses // addresses[0]
                                                    //          holder_address of input utxo
                                                    // addresses[1]    -  not exists when type=1
                                                    //          type=2: staking_address
                                                    //          type=3: binding_address
        - `String` - payload
        - `Integer` - confirmations 
        - `Integer` - size 
        - `String` - fee 
        - `Integer` - status 
        - `Integer` - type 
- `Integer` - size 
- `String` - time_utc
- `Integer` - tx_count
### Example
```json
{
    "hash": "a2e014a87e7388261dd46ad3cb8ba5560d054881cf3acb22b0d3ba832cebb989",
    "chain_id": "5433524b370b149007ba1d06225b5d8e53137a041869834cff5860b02bebc5c7",
    "version": "1",
    "height": "10000",
    "confirmations": "740805",
    "time": "1567702158",
    "previous_hash": "391f0842ad2fa8b18c47a84c9f688cd79685c61358ce3643f5b6fc979655494c",
    "next_hash": "c31b9ad4bb038d259219ae2dc8ccbba947984e30dc12f7ae2c364ea2b89cb233",
    "transaction_root": "2644e75e1e07dca71eff356f51fff406c637791aaac58910edb42c80b50d4390",
    "witness_root": "2644e75e1e07dca71eff356f51fff406c637791aaac58910edb42c80b50d4390",
    "proposal_root": "9663440551fdcd6ada50b1fa1b0003d19bc7944955820b54ab569eb9a7ab7999",
    "target": "9ae99aa3e8b7",
    "quality": "14438790b1cdd",
    "challenge": "65b5c71764830dfe47a873765cb827da51aae1c38bcfceeb1ad49e1431688d76",
    "public_key": "02c35b466747bc743d6cb9b4186fc32792959ab0835001173af1c3d0370dcdb02e",
    "proof": {
        "x": "0e78dbc5",
        "x_prime": "b7585fbc",
        "bit_length": 32
    },
    "block_signature": {
        "r": "7455a07660426c8c427747a9f56a6dfdf694771b7206fa44507d2bbd736b9768",
        "s": "7995979d051afeec267fba2e878ca0e320890952a65e8257f0c18dcc294da506"
    },
    "ban_list": [],
    "proposal_area": {
        "punishment_area": [],
        "other_area": []
    },
    "raw_tx": [
        {
            "txid": "2644e75e1e07dca71eff356f51fff406c637791aaac58910edb42c80b50d4390",
            "version": 1,
            "lock_time": "0",
            "vin": [],
            "vout": [
                {
                    "value": "50.20401505",
                    "n": 0,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 2924a0e1557fd282588aa1fb493ae37b9eb45d50b7e8bb6902f3763753e871b8",
                        "hex": "00202924a0e1557fd282588aa1fb493ae37b9eb45d50b7e8bb6902f3763753e871b8",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qq9yj2pc240lfgyky258a5jwhr0w0tgh2skl5tk6gz7dmrw5lgwxuqf4dltz"
                        ]
                    }
                },
                {
                    "value": "49.79348849",
                    "n": 1,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 e5125049d6929dc5f42b3a1c58c0a7a5ca1d12c2db6b734d6eee129705290f3a",
                        "hex": "0020e5125049d6929dc5f42b3a1c58c0a7a5ca1d12c2db6b734d6eee129705290f3a",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqu5f9qjwkj2wutapt8gw93s985h9p6ykzmd4hxntwacffwpffpuaq499cf5"
                        ]
                    }
                },
                {
                    "value": "49.19004188",
                    "n": 2,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 8a6ac1d090a01262f1bdc25d075541214b1e067a295fb96d3bbe9382e8a09e7c",
                        "hex": "00208a6ac1d090a01262f1bdc25d075541214b1e067a295fb96d3bbe9382e8a09e7c",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qq3f4vr5ys5qfx9udacfwsw42py993upn6990mjmfmh6fc969qne7q007qgg"
                        ]
                    }
                },
                {
                    "value": "48.7008844",
                    "n": 3,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 2e86cb356002b17f8e4f9c627efa0d3987d4cd609f2c74a0e56355e89ba7248a",
                        "hex": "00202e86cb356002b17f8e4f9c627efa0d3987d4cd609f2c74a0e56355e89ba7248a",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qq96rvkdtqq2chlrj0n338a7sd8xrafntqnuk8fg89vd273xa8yj9qf9efyg"
                        ]
                    }
                },
                {
                    "value": "45.71473799",
                    "n": 4,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 2e90cfed85f9bc5cd0f2ca15ea421bf9328b4a99bafefc2dec1ad54873caf9c0",
                        "hex": "00202e90cfed85f9bc5cd0f2ca15ea421bf9328b4a99bafefc2dec1ad54873caf9c0",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qq96gvlmv9lx79e58jeg275ssmlyegkj5ehtl0ct0vrt25su72l8qqw9q548"
                        ]
                    }
                },
                {
                    "value": "41.10751455",
                    "n": 5,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 db6b1546b7e8bf1819894d767c119acbe78466b7dd80e02fc572439804ab594c",
                        "hex": "0020db6b1546b7e8bf1819894d767c119acbe78466b7dd80e02fc572439804ab594c",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqmd43234hazl3sxvff4m8cyv6e0ncge4hmkqwqt79wfpesp9tt9xqanrnr2"
                        ]
                    }
                },
                {
                    "value": "37.14671044",
                    "n": 6,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 ec74c6742eeb505fccb98d08a178b2253d981a4b199f929155e88a6ad16f43fc",
                        "hex": "0020ec74c6742eeb505fccb98d08a178b2253d981a4b199f929155e88a6ad16f43fc",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqa36vvapwadg9ln9e35y2z79jy57esxjtrx0e9y24az9x45t0g07qcfv2qr"
                        ]
                    }
                },
                {
                    "value": "36.57252184",
                    "n": 7,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 0776d270ba9842944c32cade52a2d8991870c056cecb7c7a206a837048193fc4",
                        "hex": "00200776d270ba9842944c32cade52a2d8991870c056cecb7c7a206a837048193fc4",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqqamdyu96nppfgnpjet099gkcnyv8pszkem9hc73qd2phqjqe8lzq8y9tn2"
                        ]
                    }
                },
                {
                    "value": "36.38965923",
                    "n": 8,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 66367d146b1165ebd6bae3c52d408e44dff27e05d38ebf1f1b2655f595427f36",
                        "hex": "002066367d146b1165ebd6bae3c52d408e44dff27e05d38ebf1f1b2655f595427f36",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqvcm869rtz9j7h446u0zj6sywgn0lyls96w8t78cmye2lt92z0umqx5a4mq"
                        ]
                    }
                },
                {
                    "value": "35.57774925",
                    "n": 9,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 5c69278b61b94c2ba3bfc6f84cd2d18279fd63f94438f4d9c6f8be9c2c09b101",
                        "hex": "00205c69278b61b94c2ba3bfc6f84cd2d18279fd63f94438f4d9c6f8be9c2c09b101",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqt35j0zmph9xzhgalcmuye5k3sful6clegsu0fkwxlzlfctqfkyqslxkr4y"
                        ]
                    }
                },
                {
                    "value": "34.54000394",
                    "n": 10,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 ac03c642537f7ced93edaf9dad2a0348257315ad2cbfd59f03a360c0d70f1dd2",
                        "hex": "0020ac03c642537f7ced93edaf9dad2a0348257315ad2cbfd59f03a360c0d70f1dd2",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qq4spuvsjn0a7wmyld47w662srfqjhx9dd9jlat8cr5dsvp4c0rhfqr90q0h"
                        ]
                    }
                },
                {
                    "value": "29.30830469",
                    "n": 11,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 716b35f9e36fb8b9a9fbf3cfe0f1fb952f61277f00888e3039537a2d6807219b",
                        "hex": "0020716b35f9e36fb8b9a9fbf3cfe0f1fb952f61277f00888e3039537a2d6807219b",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqw94nt70rd7utn20m7087pu0mj5hkzfmlqzyguvpe2daz66q8yxdsljzngn"
                        ]
                    }
                },
                {
                    "value": "28.87421812",
                    "n": 12,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 f9a8490dd2c9832f5d3c8535da81617ae00d60727cb5a2d688c29a17481c17ff",
                        "hex": "0020f9a8490dd2c9832f5d3c8535da81617ae00d60727cb5a2d688c29a17481c17ff",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqlx5yjrwjexpj7hfus56a4qtp0tsq6crj0j66945gc2dpwjquzllsvatggv"
                        ]
                    }
                },
                {
                    "value": "28.80086095",
                    "n": 13,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 ecd9869679694103ec60e0084d9362a666b03f8d47c3e89c237b61854b2b7da7",
                        "hex": "0020ecd9869679694103ec60e0084d9362a666b03f8d47c3e89c237b61854b2b7da7",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqanvcd9ned9qs8mrquqyymymz5entq0udglp738pr0dsc2jet0knsszzp3c"
                        ]
                    }
                },
                {
                    "value": "28.06621042",
                    "n": 14,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 1d4d5cca10a52431dad17a705c27b26a1818756e111493c10a26a7062efe7b13",
                        "hex": "00201d4d5cca10a52431dad17a705c27b26a1818756e111493c10a26a7062efe7b13",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqr4x4ejss55jrrkk30fc9cfajdgvpsatwzy2f8sg2y6nsvth70vfsmywrzc"
                        ]
                    }
                },
                {
                    "value": "27.42939138",
                    "n": 15,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 598804cada1384f3b41448c5867ce91a8745dabc3736efcdaeea4d108808f954",
                        "hex": "0020598804cada1384f3b41448c5867ce91a8745dabc3736efcdaeea4d108808f954",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqtxyqfjk6zwz08dq5frzcvl8fr2r5tk4uxumwlndwafx3pzqgl92q9z78n3"
                        ]
                    }
                },
                {
                    "value": "27.42847707",
                    "n": 16,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 a41b202370bbfee42959880602021b96226648d1d92d1b3e2d784e0ea40bc526",
                        "hex": "0020a41b202370bbfee42959880602021b96226648d1d92d1b3e2d784e0ea40bc526",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qq5sdjqgmsh0lwg22e3qrqyqsmjc3xvjx3myk3k03d0p8qafqtc5nq3hxzsq"
                        ]
                    }
                },
                {
                    "value": "26.38707451",
                    "n": 17,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 cef84c79e45e6374766f2cd0e64a24146641db983cfd8454436263a113280781",
                        "hex": "0020cef84c79e45e6374766f2cd0e64a24146641db983cfd8454436263a113280781",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqemuyc70yte3hgan09ngwvj3yz3nyrkuc8n7cg4zrvf36zyegq7qs9npxv9"
                        ]
                    }
                },
                {
                    "value": "23.30201771",
                    "n": 18,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 884c5bc44925ce47b1c6782b140ef7b03ec2a62d5d0c5f8704ac1be38b9bb3d6",
                        "hex": "0020884c5bc44925ce47b1c6782b140ef7b03ec2a62d5d0c5f8704ac1be38b9bb3d6",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qq3px9h3zfyh8y0vwx0q43grhhkqlv9f3dt5x9lpcy4sd78zumk0tq6c0cz7"
                        ]
                    }
                },
                {
                    "value": "20.2611771",
                    "n": 19,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 a2f9f2583b6ce426925a798fc523dceb1650555e7edc68e19b1edf68e5ca1fb7",
                        "hex": "0020a2f9f2583b6ce426925a798fc523dceb1650555e7edc68e19b1edf68e5ca1fb7",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qq5tulykpmdnjzdyj60x8u2g7uavt9q4270mwx3cvmrm0k3ew2r7mscl3252"
                        ]
                    }
                },
                {
                    "value": "19.18960221",
                    "n": 20,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 0ff26d43dbb689d88f2d1ff69c4e10fab553c2cdd544417566849ff9fd22edf0",
                        "hex": "00200ff26d43dbb689d88f2d1ff69c4e10fab553c2cdd544417566849ff9fd22edf0",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqplex6s7mk6ya3redrlmfcnssl2648skd64zyzatxsj0lnlfzahcqcs225x"
                        ]
                    }
                },
                {
                    "value": "17.4999517",
                    "n": 21,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 8da3b8464cb706d554538307050185e1bf72e6744cbbaef0e5369dce63941d23",
                        "hex": "00208da3b8464cb706d554538307050185e1bf72e6744cbbaef0e5369dce63941d23",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qq3k3ms3jvkurd24znsvrs2qv9uxlh9en5fja6au89x6wuucu5r53s24q7yt"
                        ]
                    }
                },
                {
                    "value": "15.43360422",
                    "n": 22,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 58798602c98693eac3d038f135c39f1a704b13ab78ea1ecd494a986de7a3b296",
                        "hex": "002058798602c98693eac3d038f135c39f1a704b13ab78ea1ecd494a986de7a3b296",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqtpucvqkfs6f74s7s8rcntsulrfcykyat0r4pan2ff2vxmeark2tq0te9r9"
                        ]
                    }
                },
                {
                    "value": "14.26328352",
                    "n": 23,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 d893b43ea69e22a63c41312339be312c743cf8f8afd493034923f83b3e741683",
                        "hex": "0020d893b43ea69e22a63c41312339be312c743cf8f8afd493034923f83b3e741683",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqmzfmg04xnc32v0zpxy3nn033936re78c4l2fxq6fy0urk0n5z6psd2nc0w"
                        ]
                    }
                },
                {
                    "value": "12.80038264",
                    "n": 24,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 101911496c4d8d76da791b232b0e2a5cef51cb1f57cc38ea91237213653921c3",
                        "hex": "0020101911496c4d8d76da791b232b0e2a5cef51cb1f57cc38ea91237213653921c3",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqzqv3zjtvfkxhdknerv3jkr32tnh4rjcl2lxr3653ydepxefey8psmy9cz5"
                        ]
                    }
                },
                {
                    "value": "10.97175655",
                    "n": 25,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 4fb1654024bd62151e51e953f3b76f5401be17906a970b8236ca063190c90e70",
                        "hex": "00204fb1654024bd62151e51e953f3b76f5401be17906a970b8236ca063190c90e70",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqf7ck2spyh43p28j3a9fl8dm02sqmu9usd2tshq3kegrrryxfpecq9386vl"
                        ]
                    }
                },
                {
                    "value": "9.59114385",
                    "n": 26,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 5f4200521ea94fd86f9b9abdb2d771711703ea95c29e14b5702a137ead6ddd96",
                        "hex": "00205f4200521ea94fd86f9b9abdb2d771711703ea95c29e14b5702a137ead6ddd96",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqtapqq5s7498asmumn27m94m3wyts8654c20pfdts9gfhattdmktqp2j7jt"
                        ]
                    }
                },
                {
                    "value": "9.18701748",
                    "n": 27,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 db98a0a191a92a6b1d1fc3196dcaf1b56a8d57262c0fcd7ff3dfdab41c789a7f",
                        "hex": "0020db98a0a191a92a6b1d1fc3196dcaf1b56a8d57262c0fcd7ff3dfdab41c789a7f",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqmwv2pgv34y4xk8glcvvkmjh3k44g64ex9s8u6llnmldtg8rcnfls20k8mt"
                        ]
                    }
                },
                {
                    "value": "9.14313046",
                    "n": 28,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 82adcc674fc9cadfbfe7b9f37e97654437b5951f6df14417500dfe0474ee7aa6",
                        "hex": "002082adcc674fc9cadfbfe7b9f37e97654437b5951f6df14417500dfe0474ee7aa6",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqs2kuce60e89dl0l8h8eha9m9gsmmt9gldhc5g96sphlqga8w02nq86lq99"
                        ]
                    }
                },
                {
                    "value": "9.1250682",
                    "n": 29,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 330fbdb6b46f700795bb0b23ccaa99bdb0f978d010aa092dbc88587560475784",
                        "hex": "0020330fbdb6b46f700795bb0b23ccaa99bdb0f978d010aa092dbc88587560475784",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qqxv8mmd45dacq09dmpv3ue25ehkc0j7xszz4qjtdu3pv82cz827zq0lwdmu"
                        ]
                    }
                },
                {
                    "value": "192.00000016",
                    "n": 30,
                    "type": 1,
                    "script_detail": {
                        "asm": "0 f36de9ba772fa487f30e0f3d5b9b09c3ed778f6dfc475a7d66fc481ecec055f0",
                        "hex": "0020f36de9ba772fa487f30e0f3d5b9b09c3ed778f6dfc475a7d66fc481ecec055f0",
                        "req_sigs": 1,
                        "addresses": [
                            "ms1qq7dk7nwnh97jg0ucwpu74hxcfc0kh0rmdl3r45ltxl3ypankq2hcqsntrlq"
                        ]
                    }
                }
            ],
            "payload": "10270000000000001e000000",
            "confirmations": "1",
            "size": 1370,
            "fee": "0",
            "status": 1,
            "type": 4
        }
    ],
    "size": 2707,
    "time_utc": "2019-09-05T16:49:18Z",
    "tx_count": 1
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
- `Object` peer_count
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
        - `Integer` - type      // default 1
        - `Integer` - version   // 0 or 1
        - `String` - remarks
        - `Integer` - status    // 0-ready, 1-syncing, 2-removing
        - `String` - status_msg 
          - "ready" - when status=0
          - "removing" - when status=2
          - {synced_height} - when status=1
### Example
```json
{
    "wallets": [
        {
            "wallet_id": "ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz",
            "type": 1,
            "remarks": "init",
            "status": 0,
            "status_msg": "ready"
        },
        {
            "wallet_id": "ac10nge8pha03mdp32ndhtxr7lmscc4s0lkg9eee2j",
            "type": 1,
            "remarks": "init-2",
            "status": 1,
            "status_msg": "109830"
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
- `Integer` - version 
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
    "mnemonic": "tribe belt hand odor beauty pelican switch pluck toe pigeon zero future acoustic enemy panda twice endless motion",
    "version": 1
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
- `Integer` - version  // version of this wallet, 0 or 1
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
- `Integer` - version
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
    "version": 0,
    "remarks": "init"
}
```

## ImportMnemonic
    POST /v1/wallets/import/mnemonic
### Parameter
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| mnemonic | string |  | required |
| passphrase | string |  | required |
| remarks | string |  |  |
| external_index | int | initial external address num |  |
| internal_index | int | initial internal address num |  |

### Returns
- `Boolean` - ok 
- `String` - wallet_id 
- `Integer` - type 
- `Integer` - version
- `String` - remarks 
### Example
```json
// Request
{
	"mnemonic":"tribe belt hand odor beauty pelican switch pluck toe pigeon zero future acoustic enemy panda twice endless motion",
	"passphrase":"123456",
    "remarks":"e.g."
}

// Response
{
    "ok": true,
    "wallet_id": "ac10nge8pha03mdp32ndhtxr7lmscc4s0lkg9eee2j",
    "type": 1,
    "version": 0,
    "remarks": "e.g."
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

## DecodeRawTransaction
    POST /v1/transactions/decode
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| hex | string | hex-encoded transaction |  |

### Returns
- `String` - tx_id 
- `Integer` - version 
- `Integer` - lock_time 
- `Integer` - size 
- `Array of Vin` - vin, inputs of transaction
    - Vin
        - `String` - tx_id 
        - `Integer` - vout 
        - `Integer` - sequence 
        - `Array of String`, witness
- `Array of Vout` - vout, outputs of transaction
    - Vout
        - `String` - value 
        - `Integer` - n 
        - `Integer` - type
            - `1` - transfer
            - `2` - staking
            - `3` - binding
        - `String` - script_asm 
        - `String` - script_hex 
        - `String` - recipient_address
        - `String` - staking_address, empty when type != 2
        - `String` - binding_target, empty when type != 3. The format is `{target}:{type}:{size}`
            - target, binding address
            - type, "MASS" or "Chia"
            - size, bitlength of MASS or K size of Chia, `0` means it is an old binding.
- `String` - payload_hex 
- `String` - payload_decode, the decoded payload or '**Unknown**'.
### Example
```json
// Request
{
	"hex":"080112a2010a260a240961f6851b9720350011d2d0930f84a0f0e419911f24fd9305dc242190b0cbb5f4eb720912484730440220213a3196a6043a8dd8565dfe04df3465b33fd8bc057f1d547ba579542f4450f602207b33e25bd4192a85d24eb2a7a843e13425789f0a4e16fadfeaaea05ab651a07501122551210356830b4780dc5f5463aa91eeaa6508f93698e039fce8cf17db363ba99afd790451ae19ffffffffffffffff1a410880b1d0ba071239002076c83de3af1270125e4cf2db9b2b6c80d63d1043d9d0a57875193ad9d55783ef16f50003040506070801020304050607080002030400201a2a08f080d28840122200200c315878dffef12a9f2c9dfde6a68b43c0fd2ffe63f94e7cf3459411d3866907"
}

// Response
{
    "tx_id": "a8758742862e69028f9fe7b30aa2a17b77d794102ce76733216828006b215cee",
    "version": 1,
    "lock_time": "0",
    "size": 272,
    "vin": [
        {
            "tx_id": "003520971b85f661e4f0a0840f93d0d224dc0593fd241f910972ebf4b5cbb090",
            "vout": 0,
            "sequence": "18446744073709551615",
            "witness": [
                "4730440220213a3196a6043a8dd8565dfe04df3465b33fd8bc057f1d547ba579542f4450f602207b33e25bd4192a85d24eb2a7a843e13425789f0a4e16fadfeaaea05ab651a07501",
                "51210356830b4780dc5f5463aa91eeaa6508f93698e039fce8cf17db363ba99afd790451ae"
            ]
        }
    ],
    "vout": [
        {
            "value": "20.02",
            "n": 0,
            "type": 3,
            "script_asm": "0 76c83de3af1270125e4cf2db9b2b6c80d63d1043d9d0a57875193ad9d55783ef f5000304050607080102030405060708000203040020",
            "script_hex": "002076c83de3af1270125e4cf2db9b2b6c80d63d1043d9d0a57875193ad9d55783ef16f5000304050607080102030405060708000203040020",
            "recipient_address": "ms1qqwmyrmca0zfcpyhjv7tdek2mvsrtr6yzrm8g227r4ryadn42hs0hst2gvut",
            "staking_address": "",
            "binding_target": "18W8DkbtU6i8advRSkUaSocZqkE2JnDDZEbGH:MASS:32"
        },
        {
            "value": "171.9799",
            "n": 1,
            "type": 1,
            "script_asm": "0 0c315878dffef12a9f2c9dfde6a68b43c0fd2ffe63f94e7cf3459411d3866907",
            "script_hex": "00200c315878dffef12a9f2c9dfde6a68b43c0fd2ffe63f94e7cf3459411d3866907",
            "recipient_address": "ms1qqpsc4s7xllmcj48evnh77df5tg0q06tl7v0u5ul8ngk2pr5uxdyrspx4x5g",
            "staking_address": "",
            "binding_target": ""
        }
    ],
    "payload_hex": "",
    "payload_decode": ""
}
```

## CreateRawTransaction
    POST /v1/transactions/create
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| inputs | Array&lt;TransactionInput&gt; |  |  |
| amounts | Map&lt;string,string&gt; |  | key is paid address, value is in unit MASS  |
| change_address | string |  | optional, if not specified, the first sender will be used.  |
| subtractfeefrom | Array&lt;string&gt; |  | optional, equally deduct fee from amount of selected address.If not specified, sender pays the fee.  |
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
    },
    "subtractfeefrom": ["ms1qqc7773md3ux8wkha6td2q9vcxfae39xvuzgj063q4l2mwymp2h0aqunux9z"]
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
| change_address | string |  | optional, if not specified, the first sender will be used.  |
| from_address | string | who will pay for this transaction | optional. |
| lock_time | int |  | optional.|
| fee | string |  | optional. |
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
            - `String` - from_address
            - `String` - staking_address, empty when type != 2
            - `String` - binding_target, empty when type != 3. The format is `{target}:{type}:{size}`
                - target, binding address
                - type, "MASS" or "Chia"
                - size, bitlength of MASS or K size of Chia, `0` means it is an old binding.
- `Array of Vout`, outputs of transaction
    - Vout
        - `String` - value 
        - `Integer` - n 
        - `Integer` - type 
        - `Object`, script_detail
            - `String` - asm 
            - `String` - hex 
            - `Integer` - req_sigs 
            - `String` - recipient_address
            - `String` - staking_address, empty when type != 2
            - `String` - binding_target, empty when type != 3. The format is `{target}:{type}:{size}`
                - target, binding address
                - type, "MASS" or "Chia"
                - size, bitlength of MASS or K size of Chia, `0` means it is an old binding.
- `String` - payload 
- `Integer` - confirmations
- `Integer` - size 
- `String` - fee 
- `Integer` - status 
- `Boolean` - coinbase, if it is a coinbase transaction
### Example
```json
{
    "hex": "080112a2010a260a240961f6851b9720350011d2d0930f84a0f0e419911f24fd9305dc242190b0cbb5f4eb7209124847304402207977ec8fe14e83993a7712af3c7330ba35dfe4d17977a7c005d82dbacea48c9d0220646ad239ccf5967698a834e08a54f80d950ed31140c4773347e8dc58c425bcee01122551210356830b4780dc5f5463aa91eeaa6508f93698e039fce8cf17db363ba99afd790451ae19ffffffffffffffff1a3f08a0d5b5a0251237002076c83de3af1270125e4cf2db9b2b6c80d63d1043d9d0a57875193ad9d55783ef14f5000204050607080102030405060708000203041a2a08d0dceca222122200200c315878dffef12a9f2c9dfde6a68b43c0fd2ffe63f94e7cf3459411d3866907",
    "tx_id": "c512cd335ac08e33c47bdf955e056475acbd96b13f96627d8069aaa8a6f3eab8",
    "version": 1,
    "lock_time": "0",
    "block": {
        "height": "3311",
        "block_hash": "5acdf5ca4bf6042d3fbb33aa489073c0558f1407083af39819c6eb0c81327efb",
        "timestamp": "1624690416"
    },
    "vin": [
        {
            "value": "192",
            "n": 0,
            "type": 1,
            "redeem_detail": {
                "tx_id": "003520971b85f661e4f0a0840f93d0d224dc0593fd241f910972ebf4b5cbb090",
                "vout": 0,
                "sequence": "18446744073709551615",
                "witness": [
                    "47304402207977ec8fe14e83993a7712af3c7330ba35dfe4d17977a7c005d82dbacea48c9d0220646ad239ccf5967698a834e08a54f80d950ed31140c4773347e8dc58c425bcee01",
                    "51210356830b4780dc5f5463aa91eeaa6508f93698e039fce8cf17db363ba99afd790451ae"
                ],
                "from_address": "ms1qqpsc4s7xllmcj48evnh77df5tg0q06tl7v0u5ul8ngk2pr5uxdyrspx4x5g",
                "staking_address": "",
                "binding_target": ""
            }
        }
    ],
    "vout": [
        {
            "value": "100.001",
            "n": 0,
            "type": 3,
            "script_detail": {
                "asm": "0 76c83de3af1270125e4cf2db9b2b6c80d63d1043d9d0a57875193ad9d55783ef f500020405060708010203040506070800020304",
                "hex": "002076c83de3af1270125e4cf2db9b2b6c80d63d1043d9d0a57875193ad9d55783ef14f500020405060708010203040506070800020304",
                "req_sigs": 1,
                "recipient_address": "ms1qqwmyrmca0zfcpyhjv7tdek2mvsrtr6yzrm8g227r4ryadn42hs0hst2gvut",
                "staking_address": "",
                "binding_target": "1PLSZYeBhp6UW1MXa2DpGjdvtyXBKjdaGt:MASS:0"
            }
        },
        {
            "value": "91.9989",
            "n": 1,
            "type": 1,
            "script_detail": {
                "asm": "0 0c315878dffef12a9f2c9dfde6a68b43c0fd2ffe63f94e7cf3459411d3866907",
                "hex": "00200c315878dffef12a9f2c9dfde6a68b43c0fd2ffe63f94e7cf3459411d3866907",
                "req_sigs": 1,
                "recipient_address": "ms1qqpsc4s7xllmcj48evnh77df5tg0q06tl7v0u5ul8ngk2pr5uxdyrspx4x5g",
                "staking_address": "",
                "binding_target": ""
            }
        }
    ],
    "payload": "",
    "confirmations": "381",
    "size": 270,
    "fee": "0.0001",
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
    ## excluding withdrawn
    GET /v1/transactions/staking/history

    ## including withdrawn
    GET /v1/transactions/staking/history/all
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

## GetBlockStakingReward
    GET /v1/blocks/{height}/stakingreward
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| height | int |  | optional. default 0 for the latest block. |
### Returns
- `Integer` - height
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

## GetBindingHistory
    ## excluding withdrawn
    GET /v1/transactions/binding/history

    ## including withdrawn
    GET /v1/transactions/binding/history/all
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
            - `String` - amount, in MASS
            - `String` - holder_address
            - `String` - binding_target, bound MASS or Chia target
            - `String` - target_type, 'MASS' or 'Chia'
            - `Integer` - target_size, bitlength for MASS or K for Chia, 0 for old binding
        - `Array of String`, from_addresses

### Example
```json
{
    "histories": [
        {
            "tx_id": "fe7104abbc30b56ec62c092966eccdabaec7034bf87ae1eb653572ad648902e9",
            "status": 1,
            "block_height": "3322",
            "utxo": {
                "tx_id": "fe7104abbc30b56ec62c092966eccdabaec7034bf87ae1eb653572ad648902e9",
                "vout": 0,
                "holder_address": "ms1qqwmyrmca0zfcpyhjv7tdek2mvsrtr6yzrm8g227r4ryadn42hs0hst2gvut",
                "amount": "100.003",
                "binding_target": "1PLSZYeBhp6UW1MXaBtCqesJ5oQq4YEoyc",
                "target_type": "MASS",
                "target_size": 0
            },
            "from_addresses": [
                "ms1qqpsc4s7xllmcj48evnh77df5tg0q06tl7v0u5ul8ngk2pr5uxdyrspx4x5g"
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

## CreatePoolPkCoinbaseTransaction
    POST /v1/transactions/poolpkcoinbase

Set coinbase for chia pool pubkey, it tells miners on chia to replace default coinbase with this one.  
Block is invalid if the coinbase is not the one associated with the pool pubkey in proof, **consensus required**.  
This type of transaction will be charged at least 1 MASS fee, **consensus required**.

### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| from_address | string | who will pay for this transaction | required |
| payload | string | A hex-encoded payload which set coinbase for chia pool pubkey. Use cmd tools to generate this payload. | required |


### Returns
- `String` - hex 
### Example
```json
// Request
{
	"from_address":"ms1qqpsc4s7xllmcj48evnh77df5tg0q06tl7v0u5ul8ngk2pr5uxdyrspx4x5g",
	"payload": "00018919b3715c0e8998c5d2f36f1236c7ab0d44b8285644effe2ee0d9f54a6dadf0efc6bbd0917371b2e9462186ac99c94886188f7aeadf4c58662869557a9dd3ccba6f04ccad4841d7cf1dfae08024f844356b9b41b183a27e1794a6a52fbc2cca08fa6d1ae6c63bff3e01637a4cc74a0f14e0e09a63d4d9728c8506ee376baf5d6af961344691272cf4da079b439470365208c48ad76867d492f96e3718119fed26b15da2509da2cd3aa3284746822289"
}

// Response
{
    "hex": "080112310a260a2409948a5b9b45293100110c57c9c0f02c1d4119aab544ea91b587c22114d9a8bab7ae9f5519ffffffffffffffff1a2808c0843d122200200c315878dffef12a9f2c9dfde6a68b43c0fd2ffe63f94e7cf3459411d38669071a2a08c0b98e9347122200200c315878dffef12a9f2c9dfde6a68b43c0fd2ffe63f94e7cf3459411d38669072ab20100018919b3715c0e8998c5d2f36f1236c7ab0d44b8285644effe2ee0d9f54a6dadf0efc6bbd0917371b2e9462186ac99c94886188f7aeadf4c58662869557a9dd3ccba6f04ccad4841d7cf1dfae08024f844356b9b41b183a27e1794a6a52fbc2cca08fa6d1ae6c63bff3e01637a4cc74a0f14e0e09a63d4d9728c8506ee376baf5d6af961344691272cf4da079b439470365208c48ad76867d492f96e3718119fed26b15da2509da2cd3aa3284746822289"
}
```

## CheckPoolPkCoinbase
    POST /v1/bindings/poolpubkeys

Query coinbases bound to specified chia pool pubkeys.

### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| pool_pubkeys | Array\<string\> | hex-encoded | required |


### Returns
- `Map<pubkey, Info>` 
- `Info`
    - `Integer` - nonce, to prevent a replay attack, 0 means never be bound.
    - `String` - coinbase
### Example
```json
// Request
{
	"pool_pubkeys": ["8919b3715c0e8998c5d2f36f1236c7ab0d44b8285644effe2ee0d9f54a6dadf0efc6bbd0917371b2e9462186ac99c948", "7719b3715c0e8998c5d2f36f1236c7ab0d44b8285644effe2ee0d9f54a6dadf0efc6bbd0917371b2e9462186ac99c948","97d5be5d8612daf12a1658afe2ed2b8e708bb1d4128d0f31d71fa1272eff3ee66a4edec12aaae0e4f0a3d4421e2624c4"]
}

// Response
{
    "result": {
        "8919b3715c0e8998c5d2f36f1236c7ab0d44b8285644effe2ee0d9f54a6dadf0efc6bbd0917371b2e9462186ac99c948": {
            "nonce": 6,
            "coinbase": "ms1qq2gyvfzkhdpnafyhedcm3syvla5ntzhdz2zw69nf65v5yw35zy2ysc7s6vt"
        },
        "7719b3715c0e8998c5d2f36f1236c7ab0d44b8285644effe2ee0d9f54a6dadf0efc6bbd0917371b2e9462186ac99c948": {
            "nonce": 0, // 0 means never be bound
            "coinbase": ""
        },
        "97d5be5d8612daf12a1658afe2ed2b8e708bb1d4128d0f31d71fa1272eff3ee66a4edec12aaae0e4f0a3d4421e2624c4": {
            "nonce": 9,
            "coinbase": ""
        }
    }
}
```

## GetNetworkBinding
    GET /v1/bindings/networkbinding  
    GET /v1/bindings/networkbinding/{height}

Query total network binding and new binding price.

### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| height | Integer |  | optional |


### Example
```json
{
    "height": "3691",
    "total_binding": "0 MASS",
    "binding_price_mass_bitlength": {
        "32": "0.91552734 MASS",
        "34": "4.5776367 MASS",
        "36": "18.3105468 MASS",
        "38": "73.2421872 MASS",
        "40": "292.9687488 MASS"
    },
    "binding_price_chia_k": {
        "32": "3.66210936 MASS",
        "33": "7.32421872 MASS",
        "34": "15.56396478 MASS",
        "35": "32.0434569 MASS",
        "36": "66.83349582 MASS",
        "37": "137.329101 MASS",
        "38": "281.98242072 MASS",
        "39": "578.61327888 MASS",
        "40": "1186.52343264 MASS"
    }
}
```

## CheckTargetBinding
    POST /v1/bindings/targets
### Parameters
| param | type | meaning | notes |
| ------ | ------ | ------ | ------ |
| targets | []string | target addresses to query | not wallet address |

### Returns
- `Map`, amounts
    - `key` - string, target
    - `value` - object, binding info
        - `target_type` - string, 'MASS' or 'Chia' or 'Unknown'.
        - `target_size` - integer, bitlength of MASS or K size of Chia, **0** for old binding.
        - `amount` - string, total MASS bound to this target.
### Example
```json
// Request
{
	"targets":[
        "146hGPwfYRDde6tJ6trbyhkSoPwt69AqyZ",
        "1EgzSkV7vJ7xhC5g38ULLPoMBhHVW38VZN", 
        "14LQhx7dGPFyfRS7rYv4uKVdKjoyAJejcVVqw",
        "18gsEwbYu65Qjwz4dUtKpYqfyYawQF8yga"
    ]
}

// Response
{
    "result": {
        "146hGPwfYRDde6tJ6trbyhkSoPwt69AqyZ": {
            "target_type": "MASS",
            "target_size": 0,
            "amount": "0 MASS"
        },
        "14LQhx7dGPFyfRS7rYv4uKVdKjoyAJejcVVqw": {
            "target_type": "MASS",
            "target_size": 34,
            "amount": "4.5776367 MASS"
        },
        "18gsEwbYu65Qjwz4dUtKpYqfyYawQF8yga": {
            "target_type": "MASS",
            "target_size": 0,
            "amount": "100.002 MASS"
        },
        "1EgzSkV7vJ7xhC5g38ULLPoMBhHVW38VZN": {
            "target_type": "MASS",
            "target_size": 0,
            "amount": "0 MASS"
        }
    }
}
```