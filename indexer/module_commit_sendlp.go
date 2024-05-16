package indexer

import (
	"errors"
	"fmt"
	"log"

	"github.com/unisat-wallet/libbrc20-indexer/conf"
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func (g *BRC20ModuleIndexer) ProcessCommitFunctionSendLp(moduleInfo *model.BRC20ModuleSwapInfo, f *model.SwapFunctionData) error {
	addressTo := f.Params[0]
	pkScriptTo, _ := utils.GetPkScriptByAddress(addressTo, conf.GlobalNetParams)

	token0, token1 := f.Params[1], f.Params[2]
	poolPair := GetLowerInnerPairNameByToken(token0, token1)
	if _, ok := moduleInfo.SwapPoolTotalBalanceDataMap[poolPair]; !ok {
		return errors.New("sendlp: pool invalid")
	}
	usersLpBalanceInPool, ok := moduleInfo.LPTokenUsersBalanceMap[poolPair]
	if !ok {
		return errors.New("sendlp: lps balance map missing pair")
	}

	// Check whether the lp user's balance storage is consistent (consider storing only one copy)
	lpsBalanceFrom, ok := moduleInfo.UsersLPTokenBalanceMap[f.PkScript]
	if !ok {
		return errors.New("sendlp: users balance map missing user")
	}
	lpBalanceFrom := lpsBalanceFrom[poolPair]

	userbalanceFrom := usersLpBalanceInPool[f.PkScript]
	if userbalanceFrom.Cmp(lpBalanceFrom) != 0 {
		return errors.New("sendlp: user's tokenLp balance miss match")
	}

	tokenAmtStr := f.Params[3]
	tokenLpAmt, _ := CheckAmountVerify(tokenAmtStr, 18)
	// Check if the user's lp balance is sufficient.
	if userbalanceFrom.Cmp(tokenLpAmt) < 0 {
		return errors.New(fmt.Sprintf("sendlp: user's tokenLp balance insufficient, %s < %s", userbalanceFrom, tokenLpAmt))
	}
	if lpBalanceFrom.Cmp(tokenLpAmt) < 0 {
		return errors.New(fmt.Sprintf("sendlp: user's tokenLp balance insufficient, %s < %s", lpBalanceFrom, tokenLpAmt))
	}

	// update from lp balance
	usersLpBalanceInPool[f.PkScript] = userbalanceFrom.Sub(tokenLpAmt)
	lpsBalanceFrom[poolPair] = lpBalanceFrom.Sub(tokenLpAmt)

	// update to lp balance
	lpBalanceTo := usersLpBalanceInPool[string(pkScriptTo)]
	lpBalanceTo = lpBalanceTo.Add(tokenLpAmt)
	usersLpBalanceInPool[string(pkScriptTo)] = lpBalanceTo

	// set update flag
	moduleInfo.LPTokenUsersBalanceUpdatedMap[poolPair+f.PkScript] = struct{}{}
	moduleInfo.LPTokenUsersBalanceUpdatedMap[poolPair+string(pkScriptTo)] = struct{}{}

	// touser-lp-balance
	lpsBalanceTo, ok := moduleInfo.UsersLPTokenBalanceMap[string(pkScriptTo)]
	if !ok {
		lpsBalanceTo = make(map[string]*decimal.Decimal, 0)
		moduleInfo.UsersLPTokenBalanceMap[string(pkScriptTo)] = lpsBalanceTo
	}
	lpsBalanceTo[poolPair] = lpBalanceTo

	log.Printf("pool sendlp [%s] lp: %s -> %s", poolPair, lpBalanceFrom, lpBalanceTo)

	return nil
}
