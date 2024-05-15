package indexer

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func (g *BRC20ModuleIndexer) GetApproveInfoByKey(createIdxKey string) (
	approveInfo *model.InscriptionBRC20SwapInfo, isInvalid bool) {
	var ok bool
	// approve
	approveInfo, ok = g.InscriptionsValidApproveMap[createIdxKey]
	if !ok {
		approveInfo, ok = g.InscriptionsInvalidApproveMap[createIdxKey]
		if !ok {
			approveInfo = nil
		}
		isInvalid = true
	}

	return approveInfo, isInvalid
}

func (g *BRC20ModuleIndexer) ProcessApprove(data *model.InscriptionBRC20Data, approveInfo *model.InscriptionBRC20SwapInfo, isInvalid bool) error {
	// ticker
	uniqueLowerTicker := strings.ToLower(approveInfo.Tick)
	if _, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]; !ok {
		return errors.New("approve, invalid ticker")
	}

	moduleInfo, ok := g.ModulesInfoMap[approveInfo.Module]
	if !ok {
		log.Printf("ProcessBRC20Approve send approve, but ticker invalid. txid: %s",
			hex.EncodeToString(utils.ReverseBytes([]byte(data.TxId))),
		)
		return errors.New("approve, module invalid")
	}

	// from
	// get user's tokens to update
	fromUserTokens, ok := moduleInfo.UsersTokenBalanceDataMap[string(approveInfo.Data.PkScript)]
	if !ok {
		log.Printf("ProcessBRC20Approve send from user missing. height: %d, txidx: %d",
			data.Height,
			data.TxIdx,
		)
		return errors.New("approve, send from user missing")
	}
	// get tokenBalance to update
	fromTokenBalance, ok := fromUserTokens[uniqueLowerTicker]
	if !ok {
		log.Printf("ProcessBRC20Approve send from ticker missing. height: %d, txidx: %d",
			data.Height,
			data.TxIdx,
		)
		return errors.New("approve, send from ticker missing")
	}

	// Cross-check whether the approve-inscription exists.
	if _, ok := fromTokenBalance.ValidApproveMap[data.CreateIdxKey]; !ok {
		log.Printf("ProcessBRC20Approve send from approve missing(dup approve?). height: %d, txidx: %d",
			data.Height,
			data.TxIdx,
		)
		return errors.New("approve, send from approve missing(dup)")
	}

	// to address
	receiverPkScript := string(data.PkScript)
	if data.Satoshi == 0 {
		receiverPkScript = string(approveInfo.Data.PkScript)
		data.PkScript = receiverPkScript
	}

	// global history
	historyData := &model.BRC20SwapHistoryApproveData{
		Tick:   approveInfo.Tick,
		Amount: approveInfo.Amount.String(),
	}
	history := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_SWAP_TYPE_N_APPROVE, approveInfo.Data, data, historyData, !isInvalid)
	moduleInfo.History = append(moduleInfo.History, history)
	if isInvalid {
		// from invalid history
		fromHistory := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_SWAP_TYPE_N_APPROVE_FROM, approveInfo.Data, data, nil, false)
		fromTokenBalance.History = append(fromTokenBalance.History, fromHistory)
		return nil
	}

	// to
	tokenBalance := moduleInfo.GetUserTokenBalance(approveInfo.Tick, receiverPkScript)

	// set from
	fromTokenBalance.UpdateHeight = g.BestHeight

	fromTokenBalance.ApproveableBalance = fromTokenBalance.ApproveableBalance.Sub(approveInfo.Amount)
	delete(fromTokenBalance.ValidApproveMap, data.CreateIdxKey)

	fromHistory := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_SWAP_TYPE_N_APPROVE_FROM, approveInfo.Data, data, nil, true)
	fromTokenBalance.History = append(fromTokenBalance.History, fromHistory)

	// set to
	tokenBalance.UpdateHeight = g.BestHeight
	if data.BlockTime > 0 {
		tokenBalance.SwapAccountBalanceSafe = tokenBalance.SwapAccountBalanceSafe.Add(approveInfo.Amount)
	}
	tokenBalance.SwapAccountBalance = tokenBalance.SwapAccountBalance.Add(approveInfo.Amount)

	toHistory := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_SWAP_TYPE_N_APPROVE_TO, approveInfo.Data, data, nil, true)
	tokenBalance.History = append(tokenBalance.History, toHistory)

	return nil
}

func (g *BRC20ModuleIndexer) ProcessInscribeApprove(data *model.InscriptionBRC20Data) error {
	var body model.InscriptionBRC20ModuleSwapApproveContent
	if err := json.Unmarshal(data.ContentBody, &body); err != nil {
		log.Printf("parse approve json failed. txid: %s",
			hex.EncodeToString(utils.ReverseBytes([]byte(data.TxId))),
		)
		return err
	}

	// lower case moduleid only
	if body.Module != strings.ToLower(body.Module) {
		return errors.New("module id invalid")
	}

	moduleInfo, ok := g.ModulesInfoMap[body.Module]
	if !ok { // invalid module
		return errors.New("module invalid")
	}

	if len(body.Tick) != 4 {
		return errors.New("tick invalid")
	}

	uniqueLowerTicker := strings.ToLower(body.Tick)
	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		return errors.New("tick not exist")
	}
	tinfo := tokenInfo.Deploy

	amt, err := decimal.NewDecimalFromString(body.Amount, int(tinfo.Decimal))
	if err != nil {
		return errors.New(fmt.Sprintf("approve amount invalid: %s", body.Amount))
	}
	if amt.Sign() <= 0 || amt.Cmp(tinfo.Max) > 0 {
		return errors.New("amount out of range")
	}

	balanceApprove := decimal.NewDecimalCopy(amt)

	// Unify ticker case
	body.Tick = tokenInfo.Ticker
	// Set up approve data for subsequent use.
	approveInfo := &model.InscriptionBRC20SwapInfo{
		Data: data,
	}
	approveInfo.Module = body.Module
	approveInfo.Tick = tokenInfo.Ticker
	approveInfo.Amount = balanceApprove

	// global history
	historyData := &model.BRC20SwapHistoryApproveData{
		Tick:   approveInfo.Tick,
		Amount: approveInfo.Amount.String(),
	}
	history := model.NewBRC20ModuleHistory(false, constant.BRC20_HISTORY_SWAP_TYPE_N_INSCRIBE_APPROVE, data, data, historyData, true)
	moduleInfo.History = append(moduleInfo.History, history)

	// Check if the module balance is sufficient to approve
	moduleTokenBalance := moduleInfo.GetUserTokenBalance(approveInfo.Tick, data.PkScript)
	// available > amt
	if moduleTokenBalance.AvailableBalance.Cmp(balanceApprove) < 0 { // invalid
		history.Valid = false
		g.InscriptionsInvalidApproveMap[data.CreateIdxKey] = approveInfo
	} else {
		history.Valid = true
		// The available balance here needs to be directly deducted and transferred to ApproveableBalance.
		moduleTokenBalance.AvailableBalanceSafe = moduleTokenBalance.AvailableBalanceSafe.Sub(balanceApprove)
		moduleTokenBalance.AvailableBalance = moduleTokenBalance.AvailableBalance.Sub(balanceApprove)
		moduleTokenBalance.ApproveableBalance = moduleTokenBalance.ApproveableBalance.Add(balanceApprove)

		// Update personal approve lookup table ValidApproveMap
		if moduleTokenBalance.ValidApproveMap == nil {
			moduleTokenBalance.ValidApproveMap = make(map[string]*model.InscriptionBRC20Data, 1)
		}
		moduleTokenBalance.ValidApproveMap[data.CreateIdxKey] = data

		moduleTokenBalance.UpdateHeight = g.BestHeight
		// Update global approve lookup table
		g.InscriptionsValidApproveMap[data.CreateIdxKey] = approveInfo

		// g.InscriptionsValidBRC20DataMap[data.CreateIdxKey] = approveInfo.Data  // fixme
	}

	return nil
}
