package main

import (
	"flag"
	"log"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/indexer"
	"github.com/unisat-wallet/libbrc20-indexer/loader"
)

var (
	inputfile        string
	outputfile       string
	outputModulefile string
	testnet          bool
)

func init() {
	flag.BoolVar(&testnet, "testnet", false, "testnet")
	flag.StringVar(&inputfile, "input", "./data/brc20.input.txt", "the filename of input data, default(./data/brc20.input.txt)")
	flag.StringVar(&outputfile, "output", "./data/brc20.output.txt", "the filename of output data, default(./data/brc20.output.txt)")
	flag.StringVar(&outputModulefile, "output_module", "./data/module.output.txt", "the filename of output data, default(./data/module.output.txt)")

	flag.Parse()

	if testnet {
		constant.GlobalNetParams = &chaincfg.TestNet3Params
	}
}

func main() {
	brc20Datas, err := loader.LoadBRC20InputData(inputfile)
	if err != nil {
		log.Fatalf("invalid input, %s", err)
	}

	g := &indexer.BRC20ModuleIndexer{}
	g.ProcessUpdateLatestBRC20Init(brc20Datas)

	loader.DumpTickerInfoMap(outputfile,
		g.InscriptionsTickerInfoMap,
		g.UserTokensBalanceData,
		g.TokenUsersBalanceData,
	)

	loader.DumpModuleInfoMap(outputModulefile,
		g.ModulesInfoMap,
	)
}
