# v1.1.0

## How to Upgrade
See [cmd/database-upgrade/1.1.0/USAGE.md](https://github.com/mass-community/MassNet-wallet/tree/1.1/cmd/database-upgrade/1.1.0/USAGE.md).

## Behavior Changes
* Now removing wallet is asynchronous.
* Save blocks to disk instead of database.

## API Changes
* Change `Wallets` response field `ready` to `status`, and `synced_height` to `status_msg`.
* Change `GetStakingHistory` and `GetBindingHistory`. By default, they both return only transactions not withrawn.
* Replace `GetLatestRewardList` with `GetBlockStakingReward`.

## Perfomance Improvements
* Speed up wallet startup by reducing unnecessary data reading.
* Reduced hardware usage.
