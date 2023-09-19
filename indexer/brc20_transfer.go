package indexer

import (
	"log"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func (g *BRC20Indexer) GetTransferInfoByKey(createIdxKey string) (
	transferInfo *model.InscriptionBRC20TickTransferInfo, isInvalid bool) {
	var ok bool
	// transfer
	transferInfo, ok = g.InscriptionsValidTransferMap[createIdxKey]
	if !ok {
		transferInfo, ok = g.InscriptionsInvalidTransferMap[createIdxKey]
		if !ok {
			transferInfo = nil
		} else {
			delete(g.InscriptionsInvalidTransferMap, createIdxKey)
		}
		isInvalid = true
	} else {
		delete(g.InscriptionsValidTransferMap, createIdxKey)
	}

	return transferInfo, isInvalid
}

func (g *BRC20Indexer) ProcessTransfer(progress int, data *model.InscriptionBRC20Data, transferInfo *model.InscriptionBRC20TickTransferInfo, isInvalid bool) {
	// ticker
	uniqueLowerTicker := strings.ToLower(transferInfo.BRC20Tick)
	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		log.Printf("(%d%%) ProcessBRC20Transfer send transfer, but ticker invalid. txid: %s",
			progress,
			utils.GetReversedStringHex(data.TxId),
		)
		return
	}

	// global history
	history := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_TRANSFER, !isInvalid, true, &transferInfo.InscriptionBRC20TickInfo, nil, data)
	tokenInfo.History = append(tokenInfo.History, history)
	tokenInfo.HistoryTransfer = append(tokenInfo.HistoryTransfer, history)

	// from
	// get user's tokens to update
	fromUserTokens, ok := g.UserTokensBalanceData[string(transferInfo.PkScript)]
	if !ok {
		log.Printf("(%d%%) ProcessBRC20Transfer send from user missing. height: %d, txidx: %d",
			progress,
			data.Height,
			data.TxIdx,
		)
		return
	}
	// get tokenBalance to update
	fromTokenBalance, ok := fromUserTokens[uniqueLowerTicker]
	if !ok {
		log.Printf("(%d%%) ProcessBRC20Transfer send from ticker missing. height: %d, txidx: %d",
			progress,
			data.Height,
			data.TxIdx,
		)
		return
	}

	if isInvalid {
		fromHistory := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_SEND, false, true, &transferInfo.InscriptionBRC20TickInfo, fromTokenBalance, data)
		fromTokenBalance.History = append(fromTokenBalance.History, fromHistory)
		fromTokenBalance.HistorySend = append(fromTokenBalance.HistorySend, fromHistory)
		return
	}

	if _, ok := fromTokenBalance.ValidTransferMap[data.CreateIdxKey]; !ok {
		log.Printf("(%d%%) ProcessBRC20Transfer send from transfer missing(dup transfer?). height: %d, txidx: %d",
			progress,
			data.Height,
			data.TxIdx,
		)
		return
	}

	// to
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
		tokenBalance = &model.BRC20TokenBalance{Ticker: transferInfo.BRC20Tick, PkScript: data.PkScript}
		userTokens[uniqueLowerTicker] = tokenBalance

		// set token's users
		tokenUsers := g.TokenUsersBalanceData[uniqueLowerTicker]
		tokenUsers[string(data.PkScript)] = tokenBalance
	} else {
		tokenBalance = token
	}

	// set from
	fromTokenBalance.OverallBalanceSafe = fromTokenBalance.OverallBalanceSafe.Sub(transferInfo.Amount)
	fromTokenBalance.OverallBalance = fromTokenBalance.OverallBalance.Sub(transferInfo.Amount)
	fromTokenBalance.TransferableBalance = fromTokenBalance.TransferableBalance.Sub(transferInfo.Amount)
	delete(fromTokenBalance.ValidTransferMap, data.CreateIdxKey)

	fromHistory := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_SEND, true, true, &transferInfo.InscriptionBRC20TickInfo, fromTokenBalance, data)
	fromTokenBalance.History = append(fromTokenBalance.History, fromHistory)
	fromTokenBalance.HistorySend = append(fromTokenBalance.HistorySend, fromHistory)

	// set to
	if data.BlockTime > 0 {
		tokenBalance.OverallBalanceSafe = tokenBalance.OverallBalanceSafe.Add(transferInfo.Amount)
	}
	tokenBalance.OverallBalance = tokenBalance.OverallBalance.Add(transferInfo.Amount)

	toHistory := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_RECEIVE, true, true, &transferInfo.InscriptionBRC20TickInfo, tokenBalance, data)
	tokenBalance.History = append(tokenBalance.History, toHistory)
	tokenBalance.HistoryReceive = append(tokenBalance.HistoryReceive, toHistory)
}

func (g *BRC20Indexer) ProcessInscribeTransfer(progress int, data *model.InscriptionBRC20Data, body *model.InscriptionBRC20Content) {
	// check tick
	uniqueLowerTicker := strings.ToLower(body.BRC20Tick)
	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		return
	}
	tinfo := tokenInfo.Deploy

	// check amount
	amt, precision, err := decimal.NewDecimalFromString(body.BRC20Amount)
	if err != nil {
		log.Printf("(%d%%) ProcessInscribeTransfer, but amount invalid. ticker: %s, amount: '%s'",
			progress,
			tokenInfo.Ticker,
			body.BRC20Amount,
		)

		return
	}
	if precision > int(tinfo.Decimal) {
		return
	}
	if amt.Sign() <= 0 || amt.Cmp(tinfo.Max) > 0 {
		return
	}

	balanceTransfer := decimal.NewDecimalCopy(amt)

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

		// set token's users
		tokenUsers := g.TokenUsersBalanceData[uniqueLowerTicker]
		tokenUsers[string(data.PkScript)] = tokenBalance
	} else {
		tokenBalance = token
	}

	body.BRC20Tick = tokenInfo.Ticker
	transferInfo := model.NewInscriptionBRC20TickTransferInfo(body, data)

	transferInfo.Decimal = tinfo.Decimal
	transferInfo.Amount = balanceTransfer

	history := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_INSCRIBE_TRANSFER, true, false, &transferInfo.InscriptionBRC20TickInfo, tokenBalance, data)
	if tokenBalance.OverallBalance.Sub(tokenBalance.TransferableBalance).Cmp(balanceTransfer) < 0 { // invalid
		history.Valid = false
		// user history
		tokenBalance.History = append(tokenBalance.History, history)
		tokenBalance.HistoryInscribeTransfer = append(tokenBalance.HistoryInscribeTransfer, history)
		// global history
		tokenInfo.History = append(tokenInfo.History, history)
		tokenInfo.HistoryInscribeTransfer = append(tokenInfo.HistoryInscribeTransfer, history)

		tokenBalance.InvalidTransferList = append(tokenBalance.InvalidTransferList, transferInfo)
		g.InscriptionsInvalidTransferMap[data.CreateIdxKey] = transferInfo
	} else {
		tokenBalance.TransferableBalance = tokenBalance.TransferableBalance.Add(balanceTransfer)
		history.TransferableBalance = tokenBalance.TransferableBalance.String()                               // update  balance
		history.AvailableBalance = tokenBalance.OverallBalance.Sub(tokenBalance.TransferableBalance).String() // update  balance

		history.Valid = true
		// user history
		tokenBalance.History = append(tokenBalance.History, history)
		tokenBalance.HistoryInscribeTransfer = append(tokenBalance.HistoryInscribeTransfer, history)
		// global history
		tokenInfo.History = append(tokenInfo.History, history)
		tokenInfo.HistoryInscribeTransfer = append(tokenInfo.HistoryInscribeTransfer, history)

		if tokenBalance.ValidTransferMap == nil {
			tokenBalance.ValidTransferMap = make(map[string]*model.InscriptionBRC20TickTransferInfo, 1)
		}
		tokenBalance.ValidTransferMap[data.CreateIdxKey] = transferInfo
		g.InscriptionsValidTransferMap[data.CreateIdxKey] = transferInfo

		g.InscriptionsValidBRC20DataMap[data.CreateIdxKey] = &transferInfo.InscriptionBRC20TickInfo
	}
}
