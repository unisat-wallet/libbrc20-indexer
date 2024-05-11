package indexer

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
)

func (g *BRC20ModuleIndexer) ProcessMint(data *model.InscriptionBRC20Data) error {
	body := new(model.InscriptionBRC20MintTransferContent)
	if err := body.Unmarshal(data.ContentBody); err != nil {
		return nil
	}

	// check tick
	if len(body.BRC20Tick) != 4 {
		return nil
		// return errors.New("mint, tick length not 4")
	}
	uniqueLowerTicker := strings.ToLower(body.BRC20Tick)
	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		return nil
		// return errors.New(fmt.Sprintf("mint %s, but tick not exist", body.BRC20Tick))
	}
	tinfo := tokenInfo.Deploy

	// check mint amount
	amt, err := decimal.NewDecimalFromString(body.BRC20Amount, int(tinfo.Decimal))
	if err != nil {
		return errors.New(fmt.Sprintf("mint %s, but invalid amount(%s)", body.BRC20Tick, body.BRC20Amount))
	}
	if amt.Sign() <= 0 || amt.Cmp(tinfo.Limit) > 0 {
		return errors.New(fmt.Sprintf("mint %s, invalid amount(%s), limit(%s)", body.BRC20Tick, body.BRC20Amount, tinfo.Limit))
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
	} else {
		tokenBalance = token
	}
	// init token's users
	tokenUsers := g.TokenUsersBalanceData[uniqueLowerTicker]
	tokenUsers[string(data.PkScript)] = tokenBalance

	body.BRC20Tick = tokenInfo.Ticker
	mintInfo := model.NewInscriptionBRC20TickInfo(body.BRC20Tick, body.Operation, data)
	mintInfo.Data.BRC20Amount = body.BRC20Amount
	mintInfo.Data.BRC20Minted = amt.String()
	mintInfo.Decimal = tinfo.Decimal
	mintInfo.Amount = amt
	if tinfo.TotalMinted.Cmp(tinfo.Max) >= 0 {
		// invalid history
		history := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_INSCRIBE_MINT, false, false, mintInfo, tokenBalance, data)
		tokenBalance.History = append(tokenBalance.History, history)
		tokenBalance.HistoryMint = append(tokenBalance.HistoryMint, history)
		tokenInfo.History = append(tokenInfo.History, history)
		tokenInfo.HistoryMint = append(tokenInfo.HistoryMint, history)
		return errors.New(fmt.Sprintf("mint %s, but mint out", body.BRC20Tick))
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
	tinfo.Data.BRC20Minted = tinfo.TotalMinted.String()
	// valid mint inscriptionNumber range
	tinfo.InscriptionNumberEnd = data.InscriptionNumber

	// update mint info
	mintInfo.Data.BRC20Minted = balanceMinted.String()
	mintInfo.Amount = balanceMinted

	// update tokenBalance
	if data.BlockTime > 0 {
		tokenBalance.AvailableBalanceSafe = tokenBalance.AvailableBalanceSafe.Add(balanceMinted)
	}
	tokenBalance.AvailableBalance = tokenBalance.AvailableBalance.Add(balanceMinted)

	// history
	history := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_INSCRIBE_MINT, true, false, mintInfo, tokenBalance, data)
	// tick history
	tokenBalance.History = append(tokenBalance.History, history)
	tokenBalance.HistoryMint = append(tokenBalance.HistoryMint, history)
	tokenInfo.History = append(tokenInfo.History, history)
	tokenInfo.HistoryMint = append(tokenInfo.HistoryMint, history)

	// g.InscriptionsValidBRC20DataMap[data.CreateIdxKey] = mintInfo.Data
	return nil
}
