package indexer

import (
	"log"
	"strconv"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
)

func (g *BRC20Indexer) ProcessDeploy(progress int, data *model.InscriptionBRC20Data, body *model.InscriptionBRC20Content) {
	// check tick
	uniqueLowerTicker := strings.ToLower(body.BRC20Tick)
	if _, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]; ok { // dup ticker
		return
	}
	if body.BRC20Max == "" { // without max
		log.Printf("(%d%%) ProcessBRC20Deploy, but max missing. ticker: %s",
			progress,
			uniqueLowerTicker,
		)
		return
	}

	tinfo := model.NewInscriptionBRC20TickDeployInfo(body, data)
	tinfo.InscriptionNumberStart = data.InscriptionNumber

	// dec
	if dec, err := strconv.ParseUint(body.BRC20Decimal, 10, 64); err != nil || dec > 18 {
		// dec invalid
		log.Printf("(%d%%) ProcessBRC20Deploy, but dec invalid. ticker: %s, dec: %s",
			progress,
			uniqueLowerTicker,
			body.BRC20Decimal,
		)
		return
	} else {
		tinfo.Decimal = uint8(dec)
	}

	// max
	if max, precision, err := decimal.NewDecimalFromString(body.BRC20Max); err != nil {
		// max invalid
		log.Printf("(%d%%) ProcessBRC20Deploy, but max invalid. ticker: %s, max: '%s'",
			progress,
			uniqueLowerTicker,
			body.BRC20Max,
		)
		return
	} else {
		if max.Sign() <= 0 || max.IsOverflowUint64() || precision > int(tinfo.Decimal) {
			return
		}
		tinfo.Max = max
	}

	// lim
	if lim, precision, err := decimal.NewDecimalFromString(body.BRC20Limit); err != nil {
		// limit invalid
		log.Printf("(%d%%) ProcessBRC20Deploy, but limit invalid. ticker: %s, limit: '%s'",
			progress,
			uniqueLowerTicker,
			body.BRC20Limit,
		)
		return
	} else {
		if lim.Sign() <= 0 || lim.IsOverflowUint64() || precision > int(tinfo.Decimal) {
			return
		}
		tinfo.Limit = lim
	}

	tokenInfo := &model.BRC20TokenInfo{Ticker: body.BRC20Tick, Deploy: tinfo}
	g.InscriptionsTickerInfoMap[uniqueLowerTicker] = tokenInfo

	tokenBalance := &model.BRC20TokenBalance{Ticker: body.BRC20Tick, PkScript: data.PkScript}

	history := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_INSCRIBE_DEPLOY, true, false, &tinfo.InscriptionBRC20TickInfo, nil, data)
	tokenBalance.History = append(tokenBalance.History, history)
	tokenInfo.History = append(tokenInfo.History, history)

	// init user tokens
	var userTokens map[string]*model.BRC20TokenBalance
	if tokens, ok := g.UserTokensBalanceData[string(data.PkScript)]; !ok {
		userTokens = make(map[string]*model.BRC20TokenBalance, 0)
		g.UserTokensBalanceData[string(data.PkScript)] = userTokens
	} else {
		userTokens = tokens
	}
	userTokens[uniqueLowerTicker] = tokenBalance

	// init token users
	tokenUsers := make(map[string]*model.BRC20TokenBalance, 0)
	tokenUsers[string(data.PkScript)] = tokenBalance
	g.TokenUsersBalanceData[uniqueLowerTicker] = tokenUsers

	g.InscriptionsValidBRC20DataMap[data.CreateIdxKey] = &tinfo.InscriptionBRC20TickInfo
}
