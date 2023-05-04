# UniSat brc-20 Indexer library


This library fully implements the specification protocol of brc-20.

Developers can integrate this library in the code according to their needs.


# Example `cmd/main.go`

When the `cmd/main.go` sample program is running, it will analyze and index the latest data based on the input data, and output a detailed list of all Token information and holder balances in text form.

	unisat@ordinals:~/brc20/brc20-indexer$ go build ./cmd/main.go
	unisat@ordinals:~/brc20/brc20-indexer$ ./main
	2023/05/04 20:35:11 ProcessUpdateLatestBRC20 update. total 2340503
	2023/05/04 20:35:12 ProcessUpdateLatestBRC20 deploy, but max invalid. ticker:🫠, max: 😵
	2023/05/04 20:35:12 ProcessUpdateLatestBRC20 deploy, but max missing. ticker: 500k
	2023/05/04 20:35:13 ProcessUpdateLatestBRC20 deploy, but max missing. ticker: peng
	2023/05/04 20:35:13 ProcessUpdateLatestBRC20 deploy, but max missing. ticker: peng
	2023/05/04 20:35:13 ProcessUpdateLatestBRC20 deploy, but max missing. ticker: wass
	2023/05/04 20:35:14 ProcessUpdateLatestBRC20 inscribe transfer, but amount invalid. ticker:<10K, amount: NaN
	2023/05/04 20:35:14 ProcessUpdateLatestBRC20 inscribe transfer, but amount invalid. ticker:meme, amount: NaN
	2023/05/04 20:35:14 ProcessUpdateLatestBRC20 inscribe transfer, but amount invalid. ticker:meme, amount: NaN
	2023/05/04 20:35:14 ProcessUpdateLatestBRC20 inscribe transfer, but amount invalid. ticker:$OG$, amount: NaN
	2023/05/04 20:35:14 ProcessUpdateLatestBRC20 inscribe transfer, but amount invalid. ticker:$OG$, amount: NaN
	2023/05/04 20:35:14 ProcessUpdateLatestBRC20 inscribe transfer, but amount invalid. ticker:bits, amount: NaN
	2023/05/04 20:35:15 ProcessUpdateLatestBRC20 inscribe transfer, but amount invalid. ticker:$OG$, amount: NaN
	2023/05/04 20:35:17 ProcessUpdateLatestBRC20 inscribe transfer, but amount invalid. ticker:QUHU, amount: 998,899
	2023/05/04 20:35:18 ProcessUpdateLatestBRC20 inscribe transfer, but amount invalid. ticker:SHIB, amount: 161,000,000,000
	2023/05/04 20:35:18 ProcessUpdateLatestBRC20 inscribe transfer, but amount invalid. ticker:SHIB, amount: 161,000,000,000
	2023/05/04 20:35:18 ProcessUpdateLatestBRC20 deploy, but max missing. ticker:  nft
	2023/05/04 20:35:18 ProcessUpdateLatestBRC20 deploy, but max missing. ticker: luki
	2023/05/04 20:35:18 ProcessUpdateLatestBRC20 deploy, but max missing. ticker: sogs
	2023/05/04 20:35:20 ProcessUpdateLatestBRC20 inscribe transfer, but amount invalid. ticker:SHIB, amount: 1,000,000,000
	2023/05/04 20:35:21 ProcessUpdateLatestBRC20 deploy, but max missing. ticker: octo
	2023/05/04 20:35:21 ProcessUpdateLatestBRC20 inscribe transfer, but amount invalid. ticker:SHIB, amount: 100,000,000,000
	2023/05/04 20:35:22 ProcessUpdateLatestBRC20 deploy, but limit invalid. ticker:$neo, limit: 10,000
	2023/05/04 20:35:23 ProcessUpdateLatestBRC20 deploy, but max missing. ticker: pixo
	2023/05/04 20:35:24 ProcessUpdateLatestBRC20 deploy, but max invalid. ticker:錢$, max: 21,000,000
	2023/05/04 20:35:24 ProcessUpdateLatestBRC20 inscribe transfer, but amount invalid. ticker:punk, amount: .1
	2023/05/04 20:35:31 ProcessUpdateLatestBRC20 inscribe transfer, but amount invalid. ticker:ordi, amount: .99999
	2023/05/04 20:35:31 ProcessUpdateLatestBRC20 finish. ticker: 8667, users: 40806, tokens: 8667, validTransfer: 41941, invalidTransfer: 2742
