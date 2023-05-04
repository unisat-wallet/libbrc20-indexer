# UniSat Indexer (brc-20) library


This library fully implements the specification protocol of brc-20.

Developers can integrate this library in the code according to their needs.


# Example `cmd/main.go`

When the `cmd/main.go` sample program is running, it will analyze and index the latest data based on the input data, and output a detailed list of all Token information and holder balances in text form.

	unisat@ordinals:~/brc20/brc20-indexer$ go build ./cmd/main.go
	unisat@ordinals:~/brc20/brc20-indexer$ ./main
	2023/05/04 22:17:13 ProcessUpdateLatestBRC20 update. total 10569
	2023/05/04 22:17:13 ProcessUpdateLatestBRC20 finish. ticker: 59, users: 2282, tokens: 59, validTransfer: 3, invalidTransfer: 2

The example input data is until block height 783025 (2023-03-09 19:19:43 UTC) , the results:

	...
	ordi history: 9356, valid: 9355, minted: 8961661, holders: 2181
	ordi bc1q9zzrdj7w0c4f8j6tkwcr8z3c5a6qav9tfgp70s history: 10, transfer: 0, balance: 10000, tokens: 1
	ordi bc1qghsuvxj70q0ckh0d8j2m3g9ndjz4lpyehtq4ln history: 1, transfer: 0, balance: 1000, tokens: 4
	ordi bc1qf8t5z9j3fhrwnltha7u9lqvvtw4uux02k0600v history: 95, transfer: 0, balance: 95000, tokens: 1
	ordi bc1qtncrdztdrygchjgelsyr9prrdr224x3vt9p2hd history: 4, transfer: 0, balance: 4000, tokens: 1
	ordi bc1qvgm4ae49cktgpm33p0pr3s3ln7j203xu9z59t7 history: 3, transfer: 0, balance: 3000, tokens: 1
	ordi bc1q3j2djzt4j5mr0v9wduhq7p0xzs3fzufzggfma8 history: 101, transfer: 0, balance: 101000, tokens: 1
	ordi bc1qe594y2hzlhqgksefxslluza2mm2nz02hpe7vp5 history: 14, transfer: 0, balance: 14000, tokens: 1
	ordi bc1qatp97a6pnxutnncrrth39lff5zcreuuscjawf6 history: 3, transfer: 0, balance: 3000, tokens: 1
	ordi bc1qa4nt24mpz4evk44xqnjcz9xl66f5enew82ru8y history: 26, transfer: 0, balance: 26000, tokens: 1
	ordi bc1ql5vdt2pkcu4hc9uu460l58m6fy78euwgsamf97 history: 11, transfer: 0, balance: 11000, tokens: 2
	ordi bc1pqqvxt3h05z0e0prq87kup4kzvnrmhscs6gual9zshg7xrhant2lquvr4uz history: 1, transfer: 0, balance: 1000, tokens: 1
	ordi bc1pqq48cd7drp9v4z4gwvtdjhune7gdt384gy39794vzj2d48eqyjqqfm4y9g history: 1, transfer: 0, balance: 1000, tokens: 1
	ordi bc1pqqkcju49grmppll9m4s63x4drzyzt65sxtjrjkwmr8d57gzkaxwqwphkz5 history: 1, transfer: 0, balance: 1000, tokens: 1
	...
