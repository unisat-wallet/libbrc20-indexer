package main

import (
	"flag"
	"log"

	brc20 "github.com/unisat-wallet/libbrc20-indexer"
	"github.com/unisat-wallet/libbrc20-indexer/loader"
)

var (
	inputfile  string
	outputfile string
)

func init() {
	flag.StringVar(&inputfile, "input", "./data/brc20.input.txt", "the filename of input data, default(./data/brc20.input.txt)")
	flag.StringVar(&outputfile, "output", "./data/brc20.output.txt", "the filename of output result, default(./data/brc20.output.txt)")

	flag.Parse()
}

func main() {
	brc20Datas, err := loader.LoadBRC20InputData(inputfile)
	if err != nil {
		log.Fatalf("invalid input, %s", err)
	}

	inscriptionsTickerInfoMap, userTokensBalanceData, tokenUsersBalanceData, inscriptionsValidTransferDataMap := brc20.ProcessUpdateLatestBRC20(brc20Datas)

	loader.DumpTickerInfoMap(outputfile, inscriptionsTickerInfoMap, userTokensBalanceData, tokenUsersBalanceData, inscriptionsValidTransferDataMap)
}
