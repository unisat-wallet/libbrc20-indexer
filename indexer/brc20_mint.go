package indexer

import (
	"strings"
	"time"

	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
)

func (g *BRC20Indexer) ProcessMint(progress int, data *model.InscriptionBRC20Data, body *model.InscriptionBRC20Content) {
	// check tick
	uniqueLowerTicker := strings.ToLower(body.BRC20Tick)
	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		return
	}
	tinfo := tokenInfo.Deploy

	// check mint amount
	amt, precision, err := decimal.NewDecimalFromString(body.BRC20Amount)
	if err != nil {
		return
	}
	if precision > int(tinfo.Decimal) {
		return
	}
	if amt.Sign() <= 0 || amt.Cmp(tinfo.Limit) > 0 {
		return
	}

	// get user's tokens to update
	var userTokens map[string]*model.BRC20TokenBalance
	if tokens, ok := g.UserTokensBalanceData[string(data.PkScript)]; !ok {
		userTokens = make(map[string]*model.BRC20TokenBalance, 0)
		g.UserTokensBalanceData[string(data.PkScript)] = userTokens
	} else {
		userTokens = tokens
	}
	// get tokenBalance to update
	var tokenBalance *model.BRC20TokenBalance
	if token, ok := userTokens[uniqueLowerTicker]; !ok {
		tokenBalance = &model.BRC20TokenBalance{Ticker: tokenInfo.Ticker, PkScript: data.PkScript}
		userTokens[uniqueLowerTicker] = tokenBalance

		// init token's users
		tokenUsers := g.TokenUsersBalanceData[uniqueLowerTicker]
		tokenUsers[string(data.PkScript)] = tokenBalance
	} else {
		tokenBalance = token
	}

	body.BRC20Tick = tokenInfo.Ticker
	mintInfo := model.NewInscriptionBRC20TickMintInfo(body, data)
	mintInfo.Decimal = tinfo.Decimal
	mintInfo.Amount = amt
	if tinfo.TotalMinted.Cmp(tinfo.Max) >= 0 {
		// invalid history
		history := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_INSCRIBE_MINT, false, false, &mintInfo.InscriptionBRC20TickInfo, tokenBalance, data)
		tokenBalance.History = append(tokenBalance.History, history)
		tokenBalance.HistoryMint = append(tokenBalance.HistoryMint, history)
		tokenInfo.History = append(tokenInfo.History, history)
		tokenInfo.HistoryMint = append(tokenInfo.HistoryMint, history)
		return
	}

	// update tinfo
	// minted
	balanceMinted := decimal.NewDecimalCopy(amt)
	if tinfo.TotalMinted.Add(amt).Cmp(tinfo.Max) > 0 {
		balanceMinted = tinfo.Max.Sub(tinfo.TotalMinted)
	}
	tinfo.TotalMinted = tinfo.TotalMinted.Add(balanceMinted)
	if tinfo.TotalMinted.Cmp(tinfo.Max) >= 0 {
		tinfo.CompleteHeight = data.Height
		tinfo.CompleteBlockTime = data.BlockTime
	}
	// confirmed minted
	now := time.Now()
	if data.BlockTime > 0 {
		tinfo.ConfirmedMinted = tinfo.ConfirmedMinted.Add(balanceMinted)
		if data.BlockTime < uint32(now.Unix())-3600 {
			tinfo.ConfirmedMinted1h = tinfo.ConfirmedMinted1h.Add(balanceMinted)
		}
		if data.BlockTime < uint32(now.Unix())-86400 {
			tinfo.ConfirmedMinted24h = tinfo.ConfirmedMinted24h.Add(balanceMinted)
		}
	}
	// count
	tinfo.MintTimes++
	// valid mint inscriptionNumber range
	tinfo.InscriptionNumberEnd = data.InscriptionNumber

	// update mint info
	mintInfo.Amount = balanceMinted

	// update tokenBalance
	if data.BlockTime > 0 {
		tokenBalance.OverallBalanceSafe = tokenBalance.OverallBalanceSafe.Add(balanceMinted)
	}
	tokenBalance.OverallBalance = tokenBalance.OverallBalance.Add(balanceMinted)

	// history
	history := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_INSCRIBE_MINT, true, false, &mintInfo.InscriptionBRC20TickInfo, tokenBalance, data)
	tokenBalance.History = append(tokenBalance.History, history)
	tokenBalance.HistoryMint = append(tokenBalance.HistoryMint, history)
	tokenInfo.History = append(tokenInfo.History, history)
	tokenInfo.HistoryMint = append(tokenInfo.HistoryMint, history)

	g.InscriptionsValidBRC20DataMap[data.CreateIdxKey] = &mintInfo.InscriptionBRC20TickInfo
}
