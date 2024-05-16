package indexer

import (
	"errors"
	"log"

	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
)

func (g *BRC20ModuleIndexer) ProcessCommitFunctionDeployPool(moduleInfo *model.BRC20ModuleSwapInfo, f *model.SwapFunctionData) error {
	token0, token1 := f.Params[0], f.Params[1]
	poolPair := GetLowerInnerPairNameByToken(token0, token1)
	if _, ok := moduleInfo.SwapPoolTotalBalanceDataMap[poolPair]; ok {
		return errors.New("deploy: twice")
	}

	// lp token balance of address in module [pool][address]balance
	moduleInfo.LPTokenUsersBalanceMap[poolPair] = make(map[string]*decimal.Decimal, 0)

	token0Amt, _ := g.CheckTickVerify(token0, "0")
	token1Amt, _ := g.CheckTickVerify(token1, "0")

	// swap total balance
	// total balance of pool in module [pool]balanceData
	moduleInfo.SwapPoolTotalBalanceDataMap[poolPair] = &model.BRC20ModulePoolTotalBalance{
		Tick:    [2]string{token0, token1},
		History: make([]*model.BRC20ModuleHistory, 0), // fixme:
		// balance
		TickBalance: [2]*decimal.Decimal{token0Amt, token1Amt},
	}
	log.Printf("[%s] pool deploy pool [%s]", moduleInfo.ID, poolPair)
	return nil
}
