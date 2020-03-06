/*
	bucket model of a wallet

	keystore
		|——"aid"
		|——keydata-of-each-account(wallet)
			|——xxxx
			|——xxxx
	txstore


	utxostore


	syncstore
		|——"ws"						(bucketWalletStatus)
		    |——<wallet_id, height>
		|——"sync"					(syncBucketName)
			|——<"syncedto", height>
			|——<height, block>
*/

package masswallet
