package indexer

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/conf"
	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func (g *BRC20ModuleIndexer) GetConditionalApproveInfoByKey(createIdxKey string) (
	approveInfo *model.InscriptionBRC20SwapConditionalApproveInfo, isInvalid bool) {
	var ok bool
	// approve
	approveInfo, ok = g.InscriptionsValidConditionalApproveMap[createIdxKey]
	if !ok {
		approveInfo, ok = g.InscriptionsInvalidConditionalApproveMap[createIdxKey]
		if !ok {
			approveInfo = nil
		}
		isInvalid = true
	}

	return approveInfo, isInvalid
}

func (g *BRC20ModuleIndexer) ProcessConditionalApprove(data *model.InscriptionBRC20Data, approveInfo *model.InscriptionBRC20SwapConditionalApproveInfo, isInvalid bool) error {
	inscriptionId := approveInfo.Data.GetInscriptionId()
	log.Printf("parse move approve. inscription id: %s", inscriptionId)

	// ticker
	uniqueLowerTicker := strings.ToLower(approveInfo.Tick)
	if _, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]; !ok {
		return errors.New("approve, invalid ticker")
	}

	moduleInfo, ok := g.ModulesInfoMap[approveInfo.Module]
	if !ok {
		log.Printf("ProcessBRC20ConditionalApprove send approve, but module invalid. txid: %s",
			hex.EncodeToString(utils.ReverseBytes([]byte(data.TxId))),
		)
		return errors.New("approve, module invalid")
	}

	// global invalid history
	if isInvalid {
		// global history
		history := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_SWAP_TYPE_N_CONDITIONAL_APPROVE, approveInfo.Data, data, nil, !isInvalid)
		moduleInfo.History = append(moduleInfo.History, history)
		return nil
	}

	var amt *decimal.Decimal
	var events []*model.ConditionalApproveEvent

	// First move
	//   If sent to self or if the fee is deducted, all balances will be directly refunded to the user's swap balance, and the inscription balance will be used up and become void
	//   If sent to another party, mark current receiver as an agent; if the address cannot be unlocked, it will cause the balance to be stuck, money loss needs to be borne by the user.
	// Multiple moves
	//   If the sending address and the receiving address are different, or if the fee is deducted, all balances will be directly refunded to the user's swap balance, and the inscription balance will be used up and become void
	//   If the sending address and the receiving address are the same, carry out transfer scanning
	//
	// In the same transaction, it's possible to deposit repeatedly into different instances of the same type of module, or deposit into different modules
	// Although this causes the issue of repeated deposits, it can maintain complete independence between module instances.

	senderPkScript := string(approveInfo.Data.PkScript)
	receiverPkScript := string(data.PkScript)
	if !approveInfo.HasMoved {
		approveInfo.HasMoved = true
		if data.Satoshi == 0 || senderPkScript == receiverPkScript {
			receiverPkScript = senderPkScript
			amt = approveInfo.Amount
			approveInfo.Balance = nil

			log.Printf("generate new approve event return self with out move, id: %s", inscriptionId)
			log.Printf("generate new approve event amt: %s", amt.String())
			// Returned directly from the start
			event := model.NewConditionalApproveEvent(senderPkScript, receiverPkScript, amt, approveInfo.Balance, data, approveInfo, "", "")
			events = append(events, event)

		} else if senderPkScript != receiverPkScript {
			approveInfo.DelegatorPkScript = receiverPkScript
			return nil
		} // no else
	} else {
		senderPkScript = approveInfo.DelegatorPkScript
		if data.Satoshi == 0 || senderPkScript != receiverPkScript {
			senderPkScript = approveInfo.OwnerPkScript
			receiverPkScript = senderPkScript
			amt = approveInfo.Balance
			approveInfo.Balance = nil

			log.Printf("generate new approve event return self after move, id: %s", inscriptionId)
			log.Printf("generate new approve event amt: %s", amt.String())
			// Subsequent direct return
			event := model.NewConditionalApproveEvent(senderPkScript, receiverPkScript, amt, approveInfo.Balance, data, approveInfo, "", "")
			events = append(events, event)

		} else if senderPkScript == receiverPkScript {
			if g.ThisTxId != data.TxId {
				g.TxStaticTransferStatesForConditionalApprove = nil
				g.ThisTxId = data.TxId
			}
			events = g.GenerateApproveEventsByApprove(approveInfo.OwnerPkScript, approveInfo.Balance,
				data, approveInfo)
		} // no else
	}

	return g.ProcessConditionalApproveEvents(events)
}

func (g *BRC20ModuleIndexer) ProcessConditionalApproveEvents(events []*model.ConditionalApproveEvent) error {
	// produce conditional approve events
	for _, event := range events {
		inscriptionId := event.FromData.GetInscriptionId()

		// from address
		addressFrom, err := utils.GetAddressFromScript([]byte(event.From), conf.GlobalNetParams)
		if err != nil {
			addressFrom = hex.EncodeToString([]byte(event.From))
		}
		// to address
		addressTo, err := utils.GetAddressFromScript([]byte(event.To), conf.GlobalNetParams)
		if err != nil {
			addressTo = hex.EncodeToString([]byte(event.From))
		}
		log.Printf("process approve event. inscription id: %s, from: %s, to: %s, amt: %s, balance: %s",
			inscriptionId,
			addressFrom,
			addressTo,
			event.Amount.String(),
			event.Balance.String())

		// ticker
		uniqueLowerTicker := strings.ToLower(event.Tick)
		if _, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]; !ok {
			return errors.New("approve event, invalid ticker")
		}

		moduleInfo, ok := g.ModulesInfoMap[event.Module]
		if !ok {
			return errors.New("approve event, module invalid")
		}

		// global history
		data := &model.BRC20SwapHistoryCondApproveData{
			Tick:                  event.Tick,
			Amount:                event.Amount.String(),
			Balance:               event.Balance.String(),
			TransferInscriptionId: event.TransferInscriptionId,
			TransferMax:           event.TransferMax,
		}
		history := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_SWAP_TYPE_N_CONDITIONAL_APPROVE, &event.FromData, &event.ToData, data, true)
		moduleInfo.History = append(moduleInfo.History, history)

		// from
		// get user's tokens to update
		fromUserTokens, ok := moduleInfo.UsersTokenBalanceDataMap[string(event.From)]
		if !ok {
			log.Printf("ProcessBRC20ConditionalApprove send from user missing. height: %d, txidx: %d",
				event.ToData.Height,
				event.ToData.TxIdx,
			)
			return errors.New("approve, send from user missing")
		}
		// get tokenBalance to update
		fromTokenBalance, ok := fromUserTokens[uniqueLowerTicker]
		if !ok {
			log.Printf("ProcessBRC20ConditionalApprove send from ticker missing. height: %d, txidx: %d",
				event.ToData.Height,
				event.ToData.TxIdx,
			)
			return errors.New("approve, send from ticker missing")
		}

		// Cross-check whether the approve inscription exists.
		if _, ok := fromTokenBalance.ValidConditionalApproveMap[event.ToData.CreateIdxKey]; !ok {
			log.Printf("ProcessBRC20ConditionalApprove send from approve missing(dup approve?). height: %d, txidx: %d",
				event.ToData.Height,
				event.ToData.TxIdx,
			)
			return errors.New("approve, send from approve missing(dup)")
		}

		// to
		tokenBalance := moduleInfo.GetUserTokenBalance(event.Tick, event.To)

		// set from
		fromTokenBalance.CondApproveableBalance = fromTokenBalance.CondApproveableBalance.Sub(event.Amount)
		// delete(fromTokenBalance.ValidConditionalApproveMap, data.CreateIdxKey)

		fromTokenBalance.UpdateHeight = g.BestHeight

		// fixme: history.Data
		fromHistory := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_SWAP_TYPE_N_APPROVE_FROM, &event.FromData, &event.ToData, nil, true)
		fromTokenBalance.History = append(fromTokenBalance.History, fromHistory)

		// set to
		if event.ToData.BlockTime > 0 {
			tokenBalance.SwapAccountBalanceSafe = tokenBalance.SwapAccountBalanceSafe.Add(event.Amount)
		}
		tokenBalance.SwapAccountBalance = tokenBalance.SwapAccountBalance.Add(event.Amount)

		tokenBalance.UpdateHeight = g.BestHeight

		// fixme: history.Data
		toHistory := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_SWAP_TYPE_N_APPROVE_TO, &event.FromData, &event.ToData, nil, true)
		tokenBalance.History = append(tokenBalance.History, toHistory)

		// record state
		stateBalance := moduleInfo.GetTickConditionalApproveStateBalance(event.Tick)
		if event.From == event.To {
			stateBalance.BalanceCancelApprove = stateBalance.BalanceCancelApprove.Add(event.Amount)
		} else {
			stateBalance.BalanceApprove = stateBalance.BalanceApprove.Add(event.Amount)
		}
	}

	for _, event := range events {
		event.ApproveInfo.UpdateHeight = g.BestHeight
		event.ApproveInfo.Balance = event.Balance
	}
	return nil
}

func (g *BRC20ModuleIndexer) ProcessInscribeConditionalApprove(data *model.InscriptionBRC20Data) error {
	if data.Height >= conf.ENABLE_SWAP_WITHDRAW_HEIGHT {
		return errors.New("invalid operation")
	}

	var body model.InscriptionBRC20ModuleSwapApproveContent
	if err := json.Unmarshal(data.ContentBody, &body); err != nil {
		log.Printf("parse approve json failed. txid: %s",
			hex.EncodeToString(utils.ReverseBytes([]byte(data.TxId))),
		)
		return err
	}

	// lower case only
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

	// check amount
	amt, err := decimal.NewDecimalFromString(body.Amount, int(tinfo.Decimal))
	if err != nil {
		return errors.New(fmt.Sprintf("cond approve amount invalid: %s", body.Amount))
	}
	if amt.Sign() <= 0 || amt.Cmp(tinfo.Max) > 0 {
		return errors.New("amount out of range")
	}

	balanceCondApprove := decimal.NewDecimalCopy(amt)

	body.Tick = tokenInfo.Ticker
	condApproveInfo := &model.InscriptionBRC20SwapConditionalApproveInfo{
		Data: data,
	}
	condApproveInfo.UpdateHeight = g.BestHeight

	condApproveInfo.Module = body.Module
	condApproveInfo.Tick = tokenInfo.Ticker
	condApproveInfo.Amount = balanceCondApprove
	condApproveInfo.Balance = decimal.NewDecimalCopy(balanceCondApprove)
	condApproveInfo.OwnerPkScript = data.PkScript

	// global history
	historyData := &model.BRC20SwapHistoryCondApproveData{
		Tick:    condApproveInfo.Tick,
		Amount:  condApproveInfo.Amount.String(),
		Balance: condApproveInfo.Balance.String(),
	}
	history := model.NewBRC20ModuleHistory(false, constant.BRC20_HISTORY_SWAP_TYPE_N_INSCRIBE_CONDITIONAL_APPROVE, data, data, historyData, true)
	moduleInfo.History = append(moduleInfo.History, history)

	moduleTokenBalance := moduleInfo.GetUserTokenBalance(condApproveInfo.Tick, data.PkScript)
	if moduleTokenBalance.AvailableBalance.Cmp(balanceCondApprove) < 0 { // invalid
		history.Valid = false
		g.InscriptionsInvalidConditionalApproveMap[data.CreateIdxKey] = condApproveInfo
	} else {
		history.Valid = true
		// The available balance here will be directly deducted and transferred to ApproveableBalance.
		moduleTokenBalance.AvailableBalanceSafe = moduleTokenBalance.AvailableBalanceSafe.Sub(balanceCondApprove)
		moduleTokenBalance.AvailableBalance = moduleTokenBalance.AvailableBalance.Sub(balanceCondApprove)

		moduleTokenBalance.CondApproveableBalance = moduleTokenBalance.CondApproveableBalance.Add(balanceCondApprove)

		// Update personal approve lookup table ValidApproveMap
		if moduleTokenBalance.ValidConditionalApproveMap == nil {
			moduleTokenBalance.ValidConditionalApproveMap = make(map[string]*model.InscriptionBRC20Data, 1)
		}
		moduleTokenBalance.ValidConditionalApproveMap[data.CreateIdxKey] = data

		moduleTokenBalance.UpdateHeight = g.BestHeight

		// Update global approve lookup table
		g.InscriptionsValidConditionalApproveMap[data.CreateIdxKey] = condApproveInfo
		// g.InscriptionsValidBRC20DataMap[data.CreateIdxKey] = condApproveInfo.Data  // fixme

		// record state
		stateBalance := moduleInfo.GetTickConditionalApproveStateBalance(condApproveInfo.Tick)
		stateBalance.BalanceNewApprove = stateBalance.BalanceNewApprove.Add(balanceCondApprove)
	}

	return nil
}
