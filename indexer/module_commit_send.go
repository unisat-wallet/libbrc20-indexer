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

func (g *BRC20ModuleIndexer) ProcessCommitFunctionSend(moduleInfo *model.BRC20ModuleSwapInfo, f *model.SwapFunctionData) error {
	addressTo := f.Params[0]
	pkScriptTo, _ := utils.GetPkScriptByAddress(addressTo, conf.GlobalNetParams)

	tokenOrPair := f.Params[1]
	tokenAmtStr := f.Params[2]
	if len(tokenOrPair) == 4 {
		tokenAmt, _ := g.CheckTickVerify(tokenOrPair, tokenAmtStr)
		tokenBalanceFrom := moduleInfo.GetUserTokenBalance(tokenOrPair, f.PkScript)

		// fixme: Must use the confirmed amount
		if tokenBalanceFrom.SwapAccountBalance.Cmp(tokenAmt) < 0 {
			log.Printf("token[%s] user[%s], balance %s", tokenOrPair, f.Address, tokenBalanceFrom)
			return errors.New("send: token balance insufficient")
		}

		tokenBalanceTo := moduleInfo.GetUserTokenBalance(tokenOrPair, string(pkScriptTo))

		// User Real-time Balance Update
		tokenBalanceFrom.SwapAccountBalance = tokenBalanceFrom.SwapAccountBalance.Sub(tokenAmt)
		tokenBalanceTo.SwapAccountBalance = tokenBalanceTo.SwapAccountBalance.Add(tokenAmt)

		tokenBalanceFrom.UpdateHeight = g.BestHeight
		tokenBalanceTo.UpdateHeight = g.BestHeight

		log.Printf("pool send [%s] swappable: %s -> %s", tokenOrPair, tokenBalanceFrom.SwapAccountBalance, tokenBalanceTo.SwapAccountBalance)
	} else {
		token0, token1, _ := utils.DecodeTokensFromSwapPair(tokenOrPair)
		poolPair := GetLowerInnerPairNameByToken(token0, token1)
		if _, ok := moduleInfo.SwapPoolTotalBalanceDataMap[poolPair]; !ok {
			return errors.New("send: pool invalid")
		}
		usersLpBalanceInPool, ok := moduleInfo.LPTokenUsersBalanceMap[poolPair]
		if !ok {
			return errors.New("send: lps balance map missing pair")
		}

		// Check whether the lp user's balance storage is consistent (consider storing only one copy)
		lpsBalanceFrom, ok := moduleInfo.UsersLPTokenBalanceMap[f.PkScript]
		if !ok {
			return errors.New("send: users balance map missing user")
		}
		lpBalanceFrom := lpsBalanceFrom[poolPair]

		userbalanceFrom := usersLpBalanceInPool[f.PkScript]
		if userbalanceFrom.Cmp(lpBalanceFrom) != 0 {
			return errors.New("send: user's tokenLp balance miss match")
		}

		tokenLpAmt, _ := CheckAmountVerify(tokenAmtStr, 18)
		// Check if the user's lp balance is sufficient.
		if userbalanceFrom.Cmp(tokenLpAmt) < 0 {
			return errors.New(fmt.Sprintf("send: user's tokenLp balance insufficient, %s < %s", userbalanceFrom, tokenLpAmt))
		}
		if lpBalanceFrom.Cmp(tokenLpAmt) < 0 {
			return errors.New(fmt.Sprintf("send: user's tokenLp balance insufficient, %s < %s", lpBalanceFrom, tokenLpAmt))
		}

		// update from lp balance
		usersLpBalanceInPool[f.PkScript] = userbalanceFrom.Sub(tokenLpAmt)
		lpsBalanceFrom[poolPair] = lpBalanceFrom.Sub(tokenLpAmt)

		// update to lp balance
		lpBalanceTo := usersLpBalanceInPool[string(pkScriptTo)]
		lpBalanceTo = lpBalanceTo.Add(tokenLpAmt)
		usersLpBalanceInPool[string(pkScriptTo)] = lpBalanceTo
		// touser-lp-balance
		lpsBalanceTo, ok := moduleInfo.UsersLPTokenBalanceMap[string(pkScriptTo)]
		if !ok {
			lpsBalanceTo = make(map[string]*decimal.Decimal, 0)
			moduleInfo.UsersLPTokenBalanceMap[string(pkScriptTo)] = lpsBalanceTo
		}
		lpsBalanceTo[poolPair] = lpBalanceTo

		log.Printf("pool send [%s] lp: %s -> %s", tokenOrPair, lpBalanceFrom, lpBalanceTo)
	}

	return nil
}
