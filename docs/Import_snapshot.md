# How to Use
Snapshot file can be used for both fullnode wallet and miner.

## Import For Miner
Download right one for your OS and extract.
* [massminer-darwin-amd64.tgz](https://github.com/massnetorg/MassNet-miner/releases/download/v2.0.0/massminer-darwin-amd64.tgz)  
* [massminer-linux-amd64.tgz](https://github.com/massnetorg/MassNet-miner/releases/download/v2.0.0/massminer-linux-amd64.tgz)
* [massminer-windows-amd64.zip](https://github.com/massnetorg/MassNet-miner/releases/download/v2.0.0/massminer-windows-amd64.zip)

Commands,
```
# linux/darwin
./massminercli importchain xxxx.gz <path/to/put/chaindata>

# windows
massminercli.exe importchain xxxx.gz <path/to/put/chaindata> 
```
* Replace `xxxx.gz` with the snapshot file path you downloaded.
* Replace `<path/to/put/chaindata>` with the directory path you want to store chain data.

## Import For Wallet
Download right one for your OS and extract.
* [masswallet-darwin-amd64.tgz](https://github.com/massnetorg/MassNet-wallet/releases/download/v2.0.0/masswallet-darwin-amd64.tgz)  
* [masswallet-linux-amd64.tgz](https://github.com/massnetorg/MassNet-wallet/releases/download/v2.0.0/masswallet-linux-amd64.tgz)
* [masswallet-windows-amd64.zip](https://github.com/massnetorg/MassNet-wallet/releases/download/v2.0.0/masswallet-windows-amd64.zip)

Copy `conf/walletcli-config.json` to the same folder with `masswalletcli` or `masswalletcli.exe`.   
Commands,
```
# linux/darwin
./masswalletcli importchain xxxx.gz <path/to/put/chaindata>

# windows
masswalletcli.exe importchain xxxx.gz <path/to/put/chaindata> 
```
* Replace `xxxx.gz` with the snapshot file path you downloaded.
* Replace `<path/to/put/chaindata>` with the directory path you want to store chain data.