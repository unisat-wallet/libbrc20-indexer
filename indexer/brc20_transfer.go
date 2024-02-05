package indexer

import (
	"errors"
	"log"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func (g *BRC20ModuleIndexer) GetTransferInfoByKey(createIdxKey string) (
	transferInfo *model.InscriptionBRC20TickInfo, isInvalid bool) {
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

func (g *BRC20ModuleIndexer) ProcessTransfer(data *model.InscriptionBRC20Data, transferInfo *model.InscriptionBRC20TickInfo, isInvalid bool) error {
	// ticker
	uniqueLowerTicker := strings.ToLower(transferInfo.Tick)
	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		log.Printf("ProcessBRC20Transfer send transfer, but ticker invalid. txid: %s",
			utils.HashString([]byte(data.TxId)),
		)
		return errors.New("transfer, invalid ticker")
	}

	// to
	senderPkScript := string(transferInfo.PkScript)
	receiverPkScript := string(data.PkScript)
	if data.Satoshi == 0 {
		receiverPkScript = senderPkScript
		data.PkScript = senderPkScript
	}

	// global history
	history := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_TRANSFER, !isInvalid, true, transferInfo, nil, data)
	tokenInfo.History = append(tokenInfo.History, history)
	tokenInfo.HistoryTransfer = append(tokenInfo.HistoryTransfer, history)

	// from
	// get user's tokens to update
	fromUserTokens, ok := g.UserTokensBalanceData[senderPkScript]
	if !ok {
		log.Printf("ProcessBRC20Transfer send from user missing. height: %d, txidx: %d",
			data.Height,
			data.TxIdx,
		)
		return errors.New("transfer, invalid from data")
	}
	// get tokenBalance to update
	fromTokenBalance, ok := fromUserTokens[uniqueLowerTicker]
	if !ok {
		log.Printf("ProcessBRC20Transfer send from ticker missing. height: %d, txidx: %d",
			data.Height,
			data.TxIdx,
		)
		return errors.New("transfer, invalid from balance")
	}

	if isInvalid {
		fromHistory := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_SEND, false, true, transferInfo, fromTokenBalance, data)
		fromTokenBalance.History = append(fromTokenBalance.History, fromHistory)
		fromTokenBalance.HistorySend = append(fromTokenBalance.HistorySend, fromHistory)
		return nil
	}

	if _, ok := fromTokenBalance.ValidTransferMap[data.CreateIdxKey]; !ok {
		log.Printf("ProcessBRC20Transfer send from transfer missing(dup transfer?). height: %d, txidx: %d",
			data.Height,
			data.TxIdx,
		)
		return errors.New("transfer, invalid transfer")
	}

	// to
	// get user's tokens to update
	var userTokens map[string]*model.BRC20TokenBalance
	if tokens, ok := g.UserTokensBalanceData[receiverPkScript]; !ok {
		userTokens = make(map[string]*model.BRC20TokenBalance, 0)
		g.UserTokensBalanceData[receiverPkScript] = userTokens
	} else {
		userTokens = tokens
	}
	// get tokenBalance to update
	var tokenBalance *model.BRC20TokenBalance
	if token, ok := userTokens[uniqueLowerTicker]; !ok {
		tokenBalance = &model.BRC20TokenBalance{Ticker: transferInfo.Tick, PkScript: receiverPkScript}
		userTokens[uniqueLowerTicker] = tokenBalance

		// set token's users
		tokenUsers := g.TokenUsersBalanceData[uniqueLowerTicker]
		tokenUsers[receiverPkScript] = tokenBalance
	} else {
		tokenBalance = token
	}

	// set from
	fromTokenBalance.TransferableBalance = fromTokenBalance.TransferableBalance.Sub(transferInfo.Amount)
	delete(fromTokenBalance.ValidTransferMap, data.CreateIdxKey)

	fromHistory := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_SEND, true, true, transferInfo, fromTokenBalance, data)
	fromTokenBalance.History = append(fromTokenBalance.History, fromHistory)
	fromTokenBalance.HistorySend = append(fromTokenBalance.HistorySend, fromHistory)

	// set to
	if data.BlockTime > 0 {
		tokenBalance.AvailableBalanceSafe = tokenBalance.AvailableBalanceSafe.Add(transferInfo.Amount)
	}
	tokenBalance.AvailableBalance = tokenBalance.AvailableBalance.Add(transferInfo.Amount)

	toHistory := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_RECEIVE, true, true, transferInfo, tokenBalance, data)
	tokenBalance.History = append(tokenBalance.History, toHistory)
	tokenBalance.HistoryReceive = append(tokenBalance.HistoryReceive, toHistory)

	////////////////////////////////////////////////////////////////
	// module conditional approve (black withdraw)
	if g.ThisTxId != data.TxId {
		g.TxStaticTransferStatesForConditionalApprove = nil
		g.ThisTxId = data.TxId
	}

	inscriptionId := transferInfo.Meta.GetInscriptionId()
	events := g.GenerateApproveEventsByTransfer(inscriptionId, transferInfo.Tick, senderPkScript, receiverPkScript, transferInfo.Amount)
	if err := g.ProcessConditionalApproveEvents(events); err != nil {
		return err
	}

	////////////////////////////////////////////////////////////////
	// module deposit
	moduleId, ok := utils.GetModuleFromScript([]byte(history.PkScriptTo))
	if !ok {
		// errors.New("module transfer, not module")
		return nil
	}
	moduleInfo, ok := g.ModulesInfoMap[moduleId]
	if !ok { // invalid module
		return nil
		// return errors.New(fmt.Sprintf("module transfer, module(%s) not exist", moduleId))
	}

	// global history
	mHistory := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_TYPE_N_TRANSFER, transferInfo.Meta, data, nil, true)
	moduleInfo.History = append(moduleInfo.History, mHistory)

	// get user's tokens to update

	moduleTokenBalance := moduleInfo.GetUserTokenBalance(transferInfo.Tick, history.PkScriptFrom)
	// 设置module充值
	if data.BlockTime > 0 { // 多少个确认ok
		moduleTokenBalance.SwapAccountBalanceSafe = moduleTokenBalance.SwapAccountBalanceSafe.Add(transferInfo.Amount)
	}
	moduleTokenBalance.SwapAccountBalance = moduleTokenBalance.SwapAccountBalance.Add(transferInfo.Amount)

	// record state
	stateBalance := moduleInfo.GetTickConditionalApproveStateBalance(transferInfo.Tick)
	stateBalance.BalanceDeposite = stateBalance.BalanceDeposite.Add(transferInfo.Amount)

	return nil
}

func (g *BRC20ModuleIndexer) ProcessInscribeTransfer(data *model.InscriptionBRC20Data) error {
	body := new(model.InscriptionBRC20MintTransferContent)
	if err := body.Unmarshal(data.ContentBody); err != nil {
		return nil
	}

	// check tick
	if len(body.BRC20Tick) != 4 {
		return nil
		// return errors.New("transfer, tick length not 4")
	}
	uniqueLowerTicker := strings.ToLower(body.BRC20Tick)
	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		return nil
		// return errors.New(fmt.Sprintf("transfer %s, but tick not exist", body.BRC20Tick))
	}
	tinfo := tokenInfo.Deploy

	// check amount
	amt, err := decimal.NewDecimalFromString(body.BRC20Amount, int(tinfo.Decimal))
	if err != nil {
		return errors.New("transfer, but invalid amount")
	}
	if amt.Sign() <= 0 || amt.Cmp(tinfo.Max) > 0 {
		return nil
		// return errors.New("transfer, invalid amount(range)")
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

	transferInfo := model.NewInscriptionBRC20TickInfo(body.BRC20Tick, body.Operation, data)
	transferInfo.Data.BRC20Amount = body.BRC20Amount
	transferInfo.Data.BRC20Limit = tinfo.Data.BRC20Limit
	transferInfo.Data.BRC20Decimal = tinfo.Data.BRC20Decimal

	transferInfo.Tick = tokenInfo.Ticker
	transferInfo.Amount = balanceTransfer
	transferInfo.Meta = data

	history := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_INSCRIBE_TRANSFER, true, false, transferInfo, tokenBalance, data)
	// If use the safe version of the available balance, it will cause the unconfirmed balance to not be able to be used to create a valid transfer inscription.
	if tokenBalance.AvailableBalance.Cmp(balanceTransfer) < 0 {
		history.Valid = false
		// user history
		tokenBalance.History = append(tokenBalance.History, history)
		tokenBalance.HistoryInscribeTransfer = append(tokenBalance.HistoryInscribeTransfer, history)
		// global history
		tokenInfo.History = append(tokenInfo.History, history)
		tokenInfo.HistoryInscribeTransfer = append(tokenInfo.HistoryInscribeTransfer, history)

		g.InscriptionsInvalidTransferMap[data.CreateIdxKey] = transferInfo
	} else {
		// Update available balance

		// fixme: The available safe balance may not decrease, the current transfer usage of available balance source is not accurately distinguished.
		tokenBalance.AvailableBalanceSafe = tokenBalance.AvailableBalanceSafe.Sub(balanceTransfer)

		tokenBalance.AvailableBalance = tokenBalance.AvailableBalance.Sub(balanceTransfer)
		tokenBalance.TransferableBalance = tokenBalance.TransferableBalance.Add(balanceTransfer)

		history.AvailableBalance = tokenBalance.AvailableBalance.String()       // update  balance
		history.TransferableBalance = tokenBalance.TransferableBalance.String() // update  balance

		history.Valid = true
		// user tick history
		tokenBalance.History = append(tokenBalance.History, history)
		tokenBalance.HistoryInscribeTransfer = append(tokenBalance.HistoryInscribeTransfer, history)
		// global history
		tokenInfo.History = append(tokenInfo.History, history)
		tokenInfo.HistoryInscribeTransfer = append(tokenInfo.HistoryInscribeTransfer, history)

		if tokenBalance.ValidTransferMap == nil {
			tokenBalance.ValidTransferMap = make(map[string]*model.InscriptionBRC20TickInfo, 1)
		}
		tokenBalance.ValidTransferMap[data.CreateIdxKey] = transferInfo
		g.InscriptionsValidTransferMap[data.CreateIdxKey] = transferInfo
		g.InscriptionsValidBRC20DataMap[data.CreateIdxKey] = transferInfo.Data
	}

	return nil
}
