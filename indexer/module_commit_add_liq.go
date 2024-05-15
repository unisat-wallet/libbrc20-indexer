package indexer

import (
	"errors"
	"log"

	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func (g *BRC20ModuleIndexer) ProcessCommitFunctionAddLiquidity(moduleInfo *model.BRC20ModuleSwapInfo, f *model.SwapFunctionData) error {
	token0, token1, err := utils.DecodeTokensFromSwapPair(f.Params[0])
	if err != nil {
		return errors.New("func: addLiq poolPair invalid")
	}
	poolPair := GetLowerInnerPairNameByToken(token0, token1)

	pool, ok := moduleInfo.SwapPoolTotalBalanceDataMap[poolPair]
	if !ok {
		return errors.New("addLiq: pool invalid")
	}

	usersLpBalanceInPool, ok := moduleInfo.LPTokenUsersBalanceMap[poolPair]
	if !ok {
		return errors.New("addLiq: users invalid")
	}

	// log.Printf("[%s] pool before addliq [%s] %s: %s, %s: %s, lp: %s", moduleInfo.ID, poolPair, pool.Tick[0], pool.TickBalance[0], pool.Tick[1], pool.TickBalance[1], pool.LpBalance)
	log.Printf("pool addliq params: %v", f.Params)

	token0AmtStr := f.Params[1]
	token1AmtStr := f.Params[2]
	tokenLpAmtStr := f.Params[3]

	token0Amt, _ := g.CheckTickVerify(token0, token0AmtStr)
	token1Amt, _ := g.CheckTickVerify(token1, token1AmtStr)
	tokenLpAmt, _ := decimal.NewDecimalFromString(tokenLpAmtStr, 18)

	// LP Balance Slippage Check
	slippageAmtStr := f.Params[4]
	slippageAmt, _ := decimal.NewDecimalFromString(slippageAmtStr, 3)

	var token0Idx, token1Idx int
	if token0 == pool.Tick[0] {
		token0Idx = 0
		token1Idx = 1
	} else {
		token0Idx = 1
		token1Idx = 0
	}

	var first bool = false
	var lpForPool, lpForUser *decimal.Decimal
	if pool.TickBalance[0].Sign() == 0 && pool.TickBalance[1].Sign() == 0 {
		first = true
		lpForPool = token0Amt.Mul(token1Amt).Sqrt()
		if lpForPool.Cmp(decimal.NewDecimal(1000, 18)) < 0 {
			return errors.New("addLiq: lp less than 1000")
		}
		lpForUser = lpForPool.Sub(decimal.NewDecimal(1000, 18))

	} else {
		// Issuing additional LP, as a way of collecting service fees.
		feeRateSwapAmt, ok := CheckAmountVerify(moduleInfo.FeeRateSwap, 3)
		if !ok {
			log.Printf("pool addliq FeeRateSwap invalid: %s", moduleInfo.FeeRateSwap)
			return errors.New("addLiq: feerate swap invalid")
		}
		if feeRateSwapAmt.Sign() > 0 {
			// lp = (poolLp * (rootK - rootKLast)) / (rootK * 5 + rootKLast)
			rootK := pool.TickBalance[token0Idx].Mul(pool.TickBalance[token1Idx]).Sqrt()

			lpFee := pool.LpBalance.Mul(rootK.Sub(pool.LastRootK)).Div(
				rootK.Mul(decimal.NewDecimal(5, 0)).Add(pool.LastRootK))

			log.Printf("pool addliq issue lp: %s", lpFee.String())
			if lpFee.Sign() > 0 {
				// pool lp update
				pool.LpBalance = pool.LpBalance.Add(lpFee)

				// lpFee lp balance update
				lpFeelpbalance := usersLpBalanceInPool[moduleInfo.LpFeePkScript]
				lpFeelpbalance = lpFeelpbalance.Add(lpFee)
				usersLpBalanceInPool[moduleInfo.LpFeePkScript] = lpFeelpbalance
				// lpFee-lp-balance
				lpFeelpsBalance, ok := moduleInfo.UsersLPTokenBalanceMap[moduleInfo.LpFeePkScript]
				if !ok {
					lpFeelpsBalance = make(map[string]*decimal.Decimal, 0)
					moduleInfo.UsersLPTokenBalanceMap[moduleInfo.LpFeePkScript] = lpFeelpsBalance
				}
				lpFeelpsBalance[poolPair] = lpFeelpbalance
			}
		}

		// Calculate the amount of liquidity tokens acquired
		token1AdjustAmt := pool.TickBalance[token1Idx].Mul(token0Amt).Div(pool.TickBalance[token0Idx])
		if token1Amt.Cmp(token1AdjustAmt) >= 0 {
			token1Amt = token1AdjustAmt
		} else {
			token0AdjustAmt := pool.TickBalance[token0Idx].Mul(token1Amt).Div(pool.TickBalance[token1Idx])
			token0Amt = token0AdjustAmt
		}

		lp0 := pool.LpBalance.Mul(token0Amt).Div(pool.TickBalance[token0Idx])
		lp1 := pool.LpBalance.Mul(token1Amt).Div(pool.TickBalance[token1Idx])
		if lp0.Cmp(lp1) > 0 {
			lpForPool = lp1
		} else {
			lpForPool = lp0
		}
		lpForUser = lpForPool
	}

	if lpForUser.Cmp(tokenLpAmt.Mul(decimal.NewDecimal(1000, 3).Sub(slippageAmt)).Div(decimal.NewDecimal(1000, 3))) < 0 {
		log.Printf("user[%s], lp: %s < expect: %s. * %s", f.Address, lpForUser, tokenLpAmt, tokenLpAmt.Sub(tokenLpAmt.Mul(slippageAmt)))
		return errors.New("addLiq: over slippage")
	}

	// User Balance Check
	token0Balance := moduleInfo.GetUserTokenBalance(token0, f.PkScript)
	token1Balance := moduleInfo.GetUserTokenBalance(token1, f.PkScript)
	// fixme: Must use the confirmed amount
	if token0Balance.SwapAccountBalance.Cmp(token0Amt) < 0 {
		log.Printf("token0[%s] user[%s], balance %s", token0, f.Address, token0Balance)
		return errors.New("addLiq: token0 balance insufficient")
	}
	// fixme: Must use the confirmed amount
	if token1Balance.SwapAccountBalance.Cmp(token1Amt) < 0 {
		log.Printf("token1[%s] user[%s], balance %s", token1, f.Address, token1Balance)
		return errors.New("addLiq: token1 balance insufficient")
	}

	// User Real-time Balance Update
	token0Balance.SwapAccountBalance = token0Balance.SwapAccountBalance.Sub(token0Amt)
	token1Balance.SwapAccountBalance = token1Balance.SwapAccountBalance.Sub(token1Amt)
	// fixme: User safety balance update

	// lp balance update
	// lp-user-balance
	lpbalance := usersLpBalanceInPool[f.PkScript]
	lpbalance = lpbalance.Add(lpForUser)
	usersLpBalanceInPool[f.PkScript] = lpbalance
	// user-lp-balance
	lpsBalance, ok := moduleInfo.UsersLPTokenBalanceMap[f.PkScript]
	if !ok {
		lpsBalance = make(map[string]*decimal.Decimal, 0)
		moduleInfo.UsersLPTokenBalanceMap[f.PkScript] = lpsBalance
	}
	lpsBalance[poolPair] = lpbalance

	// zero address lp balance update
	if first {
		zerolpbalance := usersLpBalanceInPool[constant.ZERO_ADDRESS_PKSCRIPT]
		zerolpbalance = zerolpbalance.Add(decimal.NewDecimal(1000, 18))
		usersLpBalanceInPool[constant.ZERO_ADDRESS_PKSCRIPT] = zerolpbalance
		// zerouser-lp-balance
		zerolpsBalance, ok := moduleInfo.UsersLPTokenBalanceMap[constant.ZERO_ADDRESS_PKSCRIPT]
		if !ok {
			zerolpsBalance = make(map[string]*decimal.Decimal, 0)
			moduleInfo.UsersLPTokenBalanceMap[constant.ZERO_ADDRESS_PKSCRIPT] = zerolpsBalance
		}
		zerolpsBalance[poolPair] = zerolpbalance
	}

	// Changes in pool balance
	pool.TickBalance[token0Idx] = pool.TickBalance[token0Idx].Add(token0Amt)
	pool.TickBalance[token1Idx] = pool.TickBalance[token1Idx].Add(token1Amt)
	pool.LpBalance = pool.LpBalance.Add(lpForPool)

	// update lastRootK
	pool.LastRootK = pool.TickBalance[token0Idx].Mul(pool.TickBalance[token1Idx]).Sqrt()

	// log.Printf("[%s] pool after addliq [%s] %s: %s, %s: %s, lp: %s", moduleInfo.ID, poolPair, pool.Tick[0], pool.TickBalance[0], pool.Tick[1], pool.TickBalance[1], pool.LpBalance)
	return nil
}
