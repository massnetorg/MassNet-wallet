# v1.0.3
## Configuration Changes
* Add an item `advanced.max_tx_fee` to avoid creating transactions with too big fee. 

## API Changes
* Add a `change_address` field to `CreateRawTransaction` and `AutoCreateTransactionRequest`. By default, the change will be paid back to the first sender address.
* Add a `subtractfeefrom` field to `CreateRawTransaction`, indicating that equally deduct fee from amount of selected address.
* Add a new API `DecodeRawTransaction`.

## CLI Changes
* Simplify usage of `createrawtransaction` and `autocreaterawtransaction`.
* Add a new command `decoderawtransaction`.