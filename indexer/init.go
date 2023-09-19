package indexer

import (
	"bytes"
	"log"

	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/model"
)

type BRC20Indexer struct {
	InscriptionsTickerInfoMap     map[string]*model.BRC20TokenInfo
	UserTokensBalanceData         map[string]map[string]*model.BRC20TokenBalance
	TokenUsersBalanceData         map[string]map[string]*model.BRC20TokenBalance
	InscriptionsValidBRC20DataMap map[string]*model.InscriptionBRC20TickInfo

	// inner valid transfer
	InscriptionsValidTransferMap map[string]*model.InscriptionBRC20TickTransferInfo
	// inner invalid transfer
	InscriptionsInvalidTransferMap map[string]*model.InscriptionBRC20TickTransferInfo
}

func (g *BRC20Indexer) initBRC20() {
	// all ticker info
	g.InscriptionsTickerInfoMap = make(map[string]*model.BRC20TokenInfo, 0)

	// ticker of users
	g.UserTokensBalanceData = make(map[string]map[string]*model.BRC20TokenBalance, 0)

	// ticker holders
	g.TokenUsersBalanceData = make(map[string]map[string]*model.BRC20TokenBalance, 0)

	// valid brc20 inscriptions
	g.InscriptionsValidBRC20DataMap = make(map[string]*model.InscriptionBRC20TickInfo, 0)

	// inner valid transfer
	g.InscriptionsValidTransferMap = make(map[string]*model.InscriptionBRC20TickTransferInfo, 0)
	// inner invalid transfer
	g.InscriptionsInvalidTransferMap = make(map[string]*model.InscriptionBRC20TickTransferInfo, 0)
}

func isJson(contentBody []byte) bool {
	if len(contentBody) < 40 {
		return false
	}

	content := bytes.TrimSpace(contentBody)
	if !bytes.HasPrefix(content, []byte("{")) {
		return false
	}
	if !bytes.HasSuffix(content, []byte("}")) {
		return false
	}

	return true
}

// ProcessUpdateLatestBRC20
func (g *BRC20Indexer) ProcessUpdateLatestBRC20(brc20Datas []*model.InscriptionBRC20Data) {
	totalDataCount := len(brc20Datas)

	log.Printf("ProcessUpdateLatestBRC20Swap update. total %d", len(brc20Datas))

	g.initBRC20()

	for idx, data := range brc20Datas {
		progress := idx * 100 / totalDataCount

		// is sending transfer
		if data.IsTransfer {
			// transfer
			if transferInfo, isInvalid := g.GetTransferInfoByKey(data.CreateIdxKey); transferInfo != nil {
				g.ProcessTransfer(idx, data, transferInfo, isInvalid)
				continue
			}

			continue
		}

		if ok := isJson(data.ContentBody); !ok {
			continue
		}

		body := new(model.InscriptionBRC20Content)
		if err := body.Unmarshal(data.ContentBody); err != nil {
			continue
		}
		data.ContentBody = nil

		// is inscribe deploy/mint/transfer
		if body.Proto != constant.BRC20_P || len(body.BRC20Tick) != 4 {
			continue
		}

		if body.Proto == constant.BRC20_P && body.Operation == constant.BRC20_OP_DEPLOY { // op deploy
			g.ProcessDeploy(progress, data, body)
		} else if body.Proto == constant.BRC20_P && body.Operation == constant.BRC20_OP_MINT { // op mint
			g.ProcessMint(progress, data, body)
		} else if body.Proto == constant.BRC20_P && body.Operation == constant.BRC20_OP_TRANSFER { // op transfer
			g.ProcessInscribeTransfer(progress, data, body)
		} else {
			continue
		}
	}

	for _, holdersBalanceMap := range g.TokenUsersBalanceData {
		for key, balance := range holdersBalanceMap {
			if balance.OverallBalance.Sign() <= 0 {
				delete(holdersBalanceMap, key)
			}
		}
	}

	log.Printf("ProcessUpdateLatestBRC20Swap finish. ticker: %d, users: %d, tokens: %d, validInscription: %d, validTransfer: %d, invalidTransfer: %d",
		len(g.InscriptionsTickerInfoMap),
		len(g.UserTokensBalanceData),
		len(g.TokenUsersBalanceData),

		len(g.InscriptionsValidBRC20DataMap),

		len(g.InscriptionsValidTransferMap),
		len(g.InscriptionsInvalidTransferMap),
	)
}
