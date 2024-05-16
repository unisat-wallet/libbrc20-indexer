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

func (g *BRC20ModuleIndexer) ProcessCommitFunctionRemoveLiquidity(moduleInfo *model.BRC20ModuleSwapInfo, f *model.SwapFunctionData) (err error) {
	token0, token1 := f.Params[0], f.Params[1]
	if g.BestHeight < conf.ENABLE_SWAP_WITHDRAW_HEIGHT {
		token0, token1, err = utils.DecodeTokensFromSwapPair(f.Params[0])
		if err != nil {
			return errors.New("func: removeLiq poolPair invalid")
		}
	}
	poolPair := GetLowerInnerPairNameByToken(token0, token1)
	pool, ok := moduleInfo.SwapPoolTotalBalanceDataMap[poolPair]
	if !ok {
		return errors.New("removeLiq: pool invalid")
	}
	usersLpBalanceInPool, ok := moduleInfo.LPTokenUsersBalanceMap[poolPair]
	if !ok {
		return errors.New("removeLiq: lps balance map missing pair")
	}
	lpsBalance, ok := moduleInfo.UsersLPTokenBalanceMap[f.PkScript]
	if !ok {
		return errors.New("removeLiq: users balance map missing user")
	}

	// log.Printf("[%s] pool before removeliq [%s] %s: %s, %s: %s, lp: %s", moduleInfo.ID, poolPair, pool.Tick[0], pool.TickBalance[0], pool.Tick[1], pool.TickBalance[1], pool.LpBalance)
	log.Printf("pool removeliq params: %v", f.Params)

	offset := 0
	if g.BestHeight >= conf.ENABLE_SWAP_WITHDRAW_HEIGHT {
		offset = 1
	}

	tokenLpAmtStr := f.Params[1+offset]
	token0AmtStr := f.Params[2+offset]
	token1AmtStr := f.Params[3+offset]

	token0Amt, _ := g.CheckTickVerify(token0, token0AmtStr)
	token1Amt, _ := g.CheckTickVerify(token1, token1AmtStr)
	tokenLpAmt, _ := decimal.NewDecimalFromString(tokenLpAmtStr, 18)

	// LP Balance Slippage Check
	slippageAmtStr := f.Params[4+offset]
	slippageAmt, _ := decimal.NewDecimalFromString(slippageAmtStr, 3)

	var token0Idx, token1Idx int
	if token0 == pool.Tick[0] {
		token0Idx = 0
		token1Idx = 1
	} else {
		token0Idx = 1
		token1Idx = 0
	}

	// Increase LP, as a method of collecting service fees.
	feeRateSwapAmt, _ := CheckAmountVerify(moduleInfo.FeeRateSwap, 3)
	if feeRateSwapAmt.Sign() > 0 {
		// lp = (poolLp * (rootK - rootKLast)) / (rootK * 5 + rootKLast)
		rootK := pool.TickBalance[token0Idx].Mul(pool.TickBalance[token1Idx]).Sqrt()

		lpFee := pool.LpBalance.Mul(rootK.Sub(pool.LastRootK)).Div(
			rootK.Mul(decimal.NewDecimal(5, 0)).Add(pool.LastRootK))
		if lpFee.Sign() > 0 {
			// pool lp update
			pool.LpBalance = pool.LpBalance.Add(lpFee)

			// lpFee update
			lpFeelpbalance := usersLpBalanceInPool[moduleInfo.LpFeePkScript]
			lpFeelpbalance = lpFeelpbalance.Add(lpFee)
			usersLpBalanceInPool[moduleInfo.LpFeePkScript] = lpFeelpbalance
			// set update flag
			moduleInfo.LPTokenUsersBalanceUpdatedMap[poolPair+moduleInfo.LpFeePkScript] = struct{}{}
			// lpFee-lp-balance
			lpFeelpsBalance, ok := moduleInfo.UsersLPTokenBalanceMap[moduleInfo.LpFeePkScript]
			if !ok {
				lpFeelpsBalance = make(map[string]*decimal.Decimal, 0)
				moduleInfo.UsersLPTokenBalanceMap[moduleInfo.LpFeePkScript] = lpFeelpsBalance
			}
			lpFeelpsBalance[poolPair] = lpFeelpbalance
		}
	}

	// Slippage Check
	amt0 := pool.TickBalance[token0Idx].Mul(tokenLpAmt).Div(pool.LpBalance)
	if amt0.Cmp(token0Amt.Sub(token0Amt.Mul(slippageAmt))) < 0 {
		log.Printf("user[%s], token0: %s, expect: %s", f.Address, amt0, token0Amt)
		return errors.New("removeLiq: over slippage")
	}
	amt1 := pool.TickBalance[token1Idx].Mul(tokenLpAmt).Div(pool.LpBalance)
	if amt1.Cmp(token1Amt.Sub(token1Amt.Mul(slippageAmt))) < 0 {
		log.Printf("user[%s], token1: %s, expect: %s", f.Address, amt1, token1Amt)
		return errors.New("removeLiq: over slippage")
	}

	// Changes in pool balance
	if pool.LpBalance.Cmp(tokenLpAmt) < 0 {
		return errors.New(fmt.Sprintf("removeLiq: tokenLp balance insufficient, %s < %s", pool.LpBalance, tokenLpAmt))
	}
	if pool.TickBalance[token0Idx].Cmp(amt0) < 0 {
		return errors.New(fmt.Sprintf("removeLiq: pool %s balance insufficient", pool.Tick[token1Idx]))
	}
	if pool.TickBalance[token1Idx].Cmp(amt1) < 0 {
		return errors.New(fmt.Sprintf("removeLiq: pool %s balance insufficient", pool.Tick[token1Idx]))
	}

	// Check whether the user's LP balance is consistent (consider storing only one copy)
	userbalance := usersLpBalanceInPool[f.PkScript]
	lpBalance := lpsBalance[poolPair]
	if userbalance.Cmp(lpBalance) != 0 {
		return errors.New("removeLiq: user's tokenLp balance miss match")
	}
	// Check whether the balance of user LP is sufficient.
	if userbalance.Cmp(tokenLpAmt) < 0 {
		return errors.New(fmt.Sprintf("removeLiq: user's tokenLp balance insufficient, %s < %s", userbalance, tokenLpAmt))
	}
	if lpBalance.Cmp(tokenLpAmt) < 0 {
		return errors.New(fmt.Sprintf("removeLiq: user's tokenLp balance insufficient, %s < %s", lpBalance, tokenLpAmt))
	}

	// update lp balance
	usersLpBalanceInPool[f.PkScript] = userbalance.Sub(tokenLpAmt)
	lpsBalance[poolPair] = lpBalance.Sub(tokenLpAmt)

	token0Balance := moduleInfo.GetUserTokenBalance(token0, f.PkScript)
	token1Balance := moduleInfo.GetUserTokenBalance(token1, f.PkScript)

	// Obtains user token balance
	token0Balance.SwapAccountBalance = token0Balance.SwapAccountBalance.Add(amt0)
	token1Balance.SwapAccountBalance = token1Balance.SwapAccountBalance.Add(amt1)

	// update at height
	token0Balance.UpdateHeight = g.BestHeight
	token1Balance.UpdateHeight = g.BestHeight
	pool.UpdateHeight = g.BestHeight

	pool.LpBalance = pool.LpBalance.Sub(tokenLpAmt) // fixme

	// Deduct token balance in the pool
	pool.TickBalance[token0Idx] = pool.TickBalance[token0Idx].Sub(amt0)
	pool.TickBalance[token1Idx] = pool.TickBalance[token1Idx].Sub(amt1)

	// update lastRootK
	pool.LastRootK = pool.TickBalance[token0Idx].Mul(pool.TickBalance[token1Idx]).Sqrt()

	// log.Printf("[%s] pool after removeliq [%s] %s: %s, %s: %s, lp: %s", moduleInfo.ID, poolPair, pool.Tick[0], pool.TickBalance[0], pool.Tick[1], pool.TickBalance[1], pool.LpBalance)
	return nil
}
