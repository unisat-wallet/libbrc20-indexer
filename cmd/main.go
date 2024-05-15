package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/unisat-wallet/libbrc20-indexer/conf"
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
		conf.GlobalNetParams = &chaincfg.TestNet3Params
	}

	if ticks := os.Getenv("TICKS_ENABLED"); ticks != "" {
		conf.TICKS_ENABLED = ticks
	}

	if id := os.Getenv("MODULE_SWAP_SOURCE_INSCRIPTION_ID"); id != "" {
		conf.MODULE_SWAP_SOURCE_INSCRIPTION_ID = id
	}

	if heightStr := os.Getenv("BRC20_ENABLE_SELF_MINT_HEIGHT"); heightStr != "" {
		if h, err := strconv.Atoi(heightStr); err != nil {
			conf.ENABLE_SELF_MINT_HEIGHT = uint32(h)
		}
	}
}

func main() {
	brc20Datas := make(chan interface{}, 10240)
	go func() {
		if err := loader.LoadBRC20InputData(inputfile, brc20Datas); err != nil {
			log.Printf("invalid input, %s", err)
		}
		close(brc20Datas)
	}()

	g := &indexer.BRC20ModuleIndexer{}
	g.Init()
	g.ProcessUpdateLatestBRC20Loop(brc20Datas, nil)

	loader.DumpTickerInfoMap(outputfile,
		g.HistoryData,
		g.InscriptionsTickerInfoMap,
		g.UserTokensBalanceData,
		g.TokenUsersBalanceData,
	)

	loader.DumpModuleInfoMap(outputModulefile,
		g.ModulesInfoMap,
	)
}
