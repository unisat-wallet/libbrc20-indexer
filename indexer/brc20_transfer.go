package indexer

import (
	"errors"
	"log"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/conf"
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
			// don't remove. use for api valid data
			// delete(g.InscriptionsInvalidTransferMap, createIdxKey)
		}
		isInvalid = true
	} else {
		// don't remove. use for api valid data
		// delete(g.InscriptionsValidTransferMap, createIdxKey)
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
	if g.EnableHistory {
		historyObj := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_TRANSFER, !isInvalid, true, transferInfo, nil, data)
		history := g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)

		tokenInfo.History = append(tokenInfo.History, history)
		tokenInfo.HistoryTransfer = append(tokenInfo.HistoryTransfer, history)
		if !isInvalid {
			// all history
			g.AllHistory = append(g.AllHistory, history)
		}
	}

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
		if g.EnableHistory {
			historyObj := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_SEND, false, true, transferInfo, fromTokenBalance, data)
			fromHistory := g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)

			fromTokenBalance.History = append(fromTokenBalance.History, fromHistory)
			fromTokenBalance.HistorySend = append(fromTokenBalance.HistorySend, fromHistory)

			userHistory := g.GetBRC20HistoryByUser(senderPkScript)
			userHistory.History = append(userHistory.History, fromHistory)
		}
		return nil
	}

	if _, ok := fromTokenBalance.ValidTransferMap[data.CreateIdxKey]; !ok {
		log.Printf("ProcessBRC20Transfer send from transfer missing(dup transfer?). height: %d, txidx: %d",
			data.Height,
			data.TxIdx,
		)
		return errors.New("transfer, invalid transfer")
	}

	// set from
	fromTokenBalance.UpdateHeight = data.Height

	fromTokenBalance.TransferableBalance = fromTokenBalance.TransferableBalance.Sub(transferInfo.Amount)
	delete(fromTokenBalance.ValidTransferMap, data.CreateIdxKey)

	if g.EnableHistory {
		historyObj := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_SEND, true, true, transferInfo, fromTokenBalance, data)
		fromHistory := g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)

		fromTokenBalance.History = append(fromTokenBalance.History, fromHistory)
		fromTokenBalance.HistorySend = append(fromTokenBalance.HistorySend, fromHistory)

		userHistoryFrom := g.GetBRC20HistoryByUser(senderPkScript)
		userHistoryFrom.History = append(userHistoryFrom.History, fromHistory)
	}

	// to
	// get user's tokens to update
	tokenBalance := g.GetUserTokenBalance(transferInfo.Tick, receiverPkScript)
	// set to
	tokenBalance.UpdateHeight = data.Height

	if data.BlockTime > 0 {
		tokenBalance.AvailableBalanceSafe = tokenBalance.AvailableBalanceSafe.Add(transferInfo.Amount)
	}
	tokenBalance.AvailableBalance = tokenBalance.AvailableBalance.Add(transferInfo.Amount)

	// burn
	if len(receiverPkScript) == 1 && []byte(receiverPkScript)[0] == 0x6a {
		tokenInfo.Deploy.Burned = tokenInfo.Deploy.Burned.Add(transferInfo.Amount)
	}

	if g.EnableHistory {
		historyObj := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_RECEIVE, true, true, transferInfo, tokenBalance, data)
		toHistory := g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)

		tokenBalance.History = append(tokenBalance.History, toHistory)
		tokenBalance.HistoryReceive = append(tokenBalance.HistoryReceive, toHistory)

		userHistoryTo := g.GetBRC20HistoryByUser(receiverPkScript)
		userHistoryTo.History = append(userHistoryTo.History, toHistory)
	}

	////////////////////////////////////////////////////////////////
	// skip module deposit if self mint for now
	if tokenInfo.Deploy.SelfMint {
		return nil
	}

	////////////////////////////////////////////////////////////////
	// module conditional approve (black withdraw)
	if g.ThisTxId != data.TxId {
		g.TxStaticTransferStatesForConditionalApprove = nil
		g.ThisTxId = data.TxId
	}

	if data.Height < conf.ENABLE_SWAP_WITHDRAW_HEIGHT {
		inscriptionId := transferInfo.Meta.GetInscriptionId()
		events := g.GenerateApproveEventsByTransfer(inscriptionId, transferInfo.Tick, senderPkScript, receiverPkScript, transferInfo.Amount)
		if err := g.ProcessConditionalApproveEvents(events); err != nil {
			return err
		}
	}
	////////////////////////////////////////////////////////////////
	// module deposit
	moduleId, ok := utils.GetModuleFromScript([]byte(receiverPkScript))
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

	moduleTokenBalance := moduleInfo.GetUserTokenBalance(transferInfo.Tick, senderPkScript)
	moduleTokenBalance.UpdateHeight = data.Height
	// set module deposit
	if data.BlockTime > 0 { // how many confirmes ok
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
	uniqueLowerTicker, err := utils.GetValidUniqueLowerTickerTicker(body.BRC20Tick)
	if err != nil {
		return nil
		// return errors.New("transfer, tick length not 4 or 5")
	}

	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		return nil
		// return errors.New(fmt.Sprintf("transfer %s, but tick not exist", body.BRC20Tick))
	}
	tinfo := tokenInfo.Deploy

	// check amount
	amt, err := decimal.NewDecimalFromString(body.BRC20Amount, int(tinfo.Decimal))
	if err != nil {
		return nil
		// return errors.New("transfer, but invalid amount")
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
	} else {
		tokenBalance = token
	}
	// set token's users
	tokenUsers, ok := g.TokenUsersBalanceData[uniqueLowerTicker]
	if !ok {
		log.Panicf("g.TokenUsersBalanceData[%s] not exist, tick: %s", uniqueLowerTicker, uniqueLowerTicker)
	}
	tokenUsers[string(data.PkScript)] = tokenBalance

	body.BRC20Tick = tokenInfo.Ticker

	transferInfo := model.NewInscriptionBRC20TickInfo(body.BRC20Tick, body.Operation, data)
	transferInfo.Data.BRC20Amount = body.BRC20Amount
	transferInfo.Data.BRC20Limit = tinfo.Data.BRC20Limit
	transferInfo.Data.BRC20Decimal = tinfo.Data.BRC20Decimal

	transferInfo.Tick = tokenInfo.Ticker
	transferInfo.Amount = balanceTransfer
	transferInfo.Meta = data

	if g.EnableHistory {
		history := g.HistoryCount
		historyObj := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_INSCRIBE_TRANSFER, true, false, transferInfo, tokenBalance, data)
		// If use the safe version of the available balance, it will cause the unconfirmed balance to not be able to be used to create a valid transfer inscription.
		if tokenBalance.AvailableBalance.Cmp(balanceTransfer) < 0 {
			historyObj.Valid = false
			// user history
			tokenBalance.History = append(tokenBalance.History, history)
			tokenBalance.HistoryInscribeTransfer = append(tokenBalance.HistoryInscribeTransfer, history)
			// global history
			tokenInfo.History = append(tokenInfo.History, history)
			tokenInfo.HistoryInscribeTransfer = append(tokenInfo.HistoryInscribeTransfer, history)

			userHistory := g.GetBRC20HistoryByUser(string(data.PkScript))
			userHistory.History = append(userHistory.History, history)
		} else {
			historyObj.Valid = true
			// user tick history
			tokenBalance.History = append(tokenBalance.History, history)
			tokenBalance.HistoryInscribeTransfer = append(tokenBalance.HistoryInscribeTransfer, history)
			// user history
			userHistory := g.GetBRC20HistoryByUser(string(data.PkScript))
			userHistory.History = append(userHistory.History, history)
			// global history
			tokenInfo.History = append(tokenInfo.History, history)
			tokenInfo.HistoryInscribeTransfer = append(tokenInfo.HistoryInscribeTransfer, history)
			// all history
			g.AllHistory = append(g.AllHistory, history)
		}

		g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)
	}

	// If use the safe version of the available balance, it will cause the unconfirmed balance to not be able to be used to create a valid transfer inscription.
	if tokenBalance.AvailableBalance.Cmp(balanceTransfer) < 0 {
		g.InscriptionsInvalidTransferMap[data.CreateIdxKey] = transferInfo
	} else {
		// Update available balance

		// fixme: The available safe balance may not decrease, the current transfer usage of available balance source is not accurately distinguished.
		tokenBalance.AvailableBalanceSafe = tokenBalance.AvailableBalanceSafe.Sub(balanceTransfer)

		tokenBalance.AvailableBalance = tokenBalance.AvailableBalance.Sub(balanceTransfer)
		tokenBalance.TransferableBalance = tokenBalance.TransferableBalance.Add(balanceTransfer)

		if tokenBalance.ValidTransferMap == nil {
			tokenBalance.ValidTransferMap = make(map[string]*model.InscriptionBRC20TickInfo, 1)
		}
		tokenBalance.ValidTransferMap[data.CreateIdxKey] = transferInfo
		g.InscriptionsValidTransferMap[data.CreateIdxKey] = transferInfo
		g.InscriptionsValidBRC20DataMap[data.CreateIdxKey] = transferInfo.Data
	}

	return nil
}
