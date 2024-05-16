package indexer

import (
	"encoding/gob"
	"log"
	"os"

	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
)

type BRC20ModuleIndexerStore struct {
	BestHeight    uint32
	EnableHistory bool

	HistoryCount uint32

	FirstHistoryByHeight map[uint32]uint32
	LastHistoryHeight    uint32

	// brc20 base
	AllHistory     []uint32
	UserAllHistory map[string]*model.BRC20UserHistory

	InscriptionsTickerInfoMap map[string]*model.BRC20TokenInfo
	UserTokensBalanceData     map[string]map[string]*model.BRC20TokenBalance

	InscriptionsValidBRC20DataMap map[string]*model.InscriptionBRC20InfoResp

	// inner valid transfer
	InscriptionsValidTransferMap map[string]*model.InscriptionBRC20TickInfo
	// inner invalid transfer
	InscriptionsInvalidTransferMap map[string]*model.InscriptionBRC20TickInfo

	// module
	// all modules info
	ModulesInfoMap map[string]*model.BRC20ModuleSwapInfoStore

	// module of users [address]moduleid
	UsersModuleWithTokenMap map[string]string

	// module lp of users [address]moduleid
	UsersModuleWithLpTokenMap map[string]string

	// runtime for approve
	InscriptionsValidApproveMap   map[string]*model.InscriptionBRC20SwapInfo // inner valid approve
	InscriptionsInvalidApproveMap map[string]*model.InscriptionBRC20SwapInfo

	// runtime for conditional approve
	InscriptionsValidConditionalApproveMap   map[string]*model.InscriptionBRC20SwapConditionalApproveInfo
	InscriptionsInvalidConditionalApproveMap map[string]*model.InscriptionBRC20SwapConditionalApproveInfo

	// runtime for commit
	InscriptionsValidCommitMap   map[string]*model.InscriptionBRC20Data // inner valid commit by key
	InscriptionsInvalidCommitMap map[string]*model.InscriptionBRC20Data
}

func (g *BRC20ModuleIndexer) Load(fname string) {
	log.Printf("loading brc20 ...")
	gobFile, err := os.Open(fname)
	if err != nil {
		log.Printf("open brc20 file failed: %s", err)
		return
	}

	gob.Register(model.BRC20SwapHistoryApproveData{})
	gob.Register(model.BRC20SwapHistoryCondApproveData{})
	gobDec := gob.NewDecoder(gobFile)

	store := &BRC20ModuleIndexerStore{}
	if err := gobDec.Decode(&store); err != nil {
		log.Printf("load store failed: %s", err)
		return
	}

	g.LoadStore(store)

	log.Printf("load brc20 ok")
}

func (g *BRC20ModuleIndexer) Save(fname string) {
	log.Printf("saving brc20 ...")

	gobFile, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Printf("open brc20 file failed: %s", err)
		return
	}
	defer gobFile.Close()

	gob.Register(model.BRC20SwapHistoryApproveData{})
	gob.Register(model.BRC20SwapHistoryCondApproveData{})

	enc := gob.NewEncoder(gobFile)
	if err := enc.Encode(g.GetStore()); err != nil {
		log.Printf("save store failed: %s", err)
		return
	}

	log.Printf("save brc20 ok")
}

func (g *BRC20ModuleIndexer) LoadHistory(fname string) {
	log.Printf("loading brc20 history...")
	gobFile, err := os.Open(fname)
	if err != nil {
		log.Printf("open brc20 history file failed: %s", err)
		return
	}

	gobDec := gob.NewDecoder(gobFile)

	for {
		var h []byte
		if err := gobDec.Decode(&h); err != nil {
			log.Printf("load history data end: %s", err)
			break
		}
		g.HistoryData = append(g.HistoryData, h)
	}
	log.Printf("load brc20 history ok: %d", len(g.HistoryData))
}

func (g *BRC20ModuleIndexer) SaveHistory(fname string) {
	log.Printf("saving brc20 history...")

	gobFile, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Printf("open brc20 history file failed: %s", err)
		return
	}
	defer gobFile.Close()

	enc := gob.NewEncoder(gobFile)
	for _, h := range g.HistoryData {
		if err := enc.Encode(h); err != nil {
			log.Printf("save history data failed: %s", err)
			return
		}
	}
	log.Printf("save brc20 history ok")
}

func (g *BRC20ModuleIndexer) GetStore() (store *BRC20ModuleIndexerStore) {
	store = &BRC20ModuleIndexerStore{
		BestHeight:    g.BestHeight,
		EnableHistory: g.EnableHistory,

		HistoryCount: g.HistoryCount,

		FirstHistoryByHeight: g.FirstHistoryByHeight,
		LastHistoryHeight:    g.LastHistoryHeight,

		// brc20 base
		AllHistory:     g.AllHistory,
		UserAllHistory: g.UserAllHistory,

		InscriptionsTickerInfoMap: g.InscriptionsTickerInfoMap,
		UserTokensBalanceData:     g.UserTokensBalanceData,

		InscriptionsValidBRC20DataMap: g.InscriptionsValidBRC20DataMap,

		// inner valid transfer
		InscriptionsValidTransferMap: g.InscriptionsValidTransferMap,
		// inner invalid transfer
		InscriptionsInvalidTransferMap: g.InscriptionsInvalidTransferMap,

		// module
		// all modules info
		// module of users [address]moduleid
		UsersModuleWithTokenMap: g.UsersModuleWithTokenMap,

		// module lp of users [address]moduleid
		UsersModuleWithLpTokenMap: g.UsersModuleWithLpTokenMap,

		// runtime for approve
		InscriptionsValidApproveMap:   g.InscriptionsValidApproveMap,
		InscriptionsInvalidApproveMap: g.InscriptionsInvalidApproveMap,

		// runtime for conditional approve
		InscriptionsValidConditionalApproveMap:   g.InscriptionsValidConditionalApproveMap,
		InscriptionsInvalidConditionalApproveMap: g.InscriptionsInvalidConditionalApproveMap,

		// runtime for commit
		InscriptionsValidCommitMap:   g.InscriptionsValidCommitMap,
		InscriptionsInvalidCommitMap: g.InscriptionsInvalidCommitMap,
	}

	store.ModulesInfoMap = make(map[string]*model.BRC20ModuleSwapInfoStore)
	for module, info := range g.ModulesInfoMap {
		infoStore := &model.BRC20ModuleSwapInfoStore{
			ID:                info.ID,
			Name:              info.Name,
			DeployerPkScript:  info.DeployerPkScript,
			SequencerPkScript: info.SequencerPkScript,
			GasToPkScript:     info.GasToPkScript,
			LpFeePkScript:     info.LpFeePkScript,

			FeeRateSwap: info.FeeRateSwap,
			GasTick:     info.GasTick,

			History: info.History, // fixme

			// runtime for commit
			CommitInvalidMap: info.CommitInvalidMap,
			CommitIdMap:      info.CommitIdMap,
			CommitIdChainMap: info.CommitIdChainMap,

			// token holders in module
			// ticker of users in module [address][tick]balanceData
			UsersTokenBalanceDataMap: info.UsersTokenBalanceDataMap,

			// swap
			// lp token balance of address in module [pool][address]balance
			LPTokenUsersBalanceMap: info.LPTokenUsersBalanceMap,

			// swap total balance
			// total balance of pool in module [pool]balanceData
			SwapPoolTotalBalanceDataMap: info.SwapPoolTotalBalanceDataMap,

			// module deposit/withdraw state [tick]balanceData
			ConditionalApproveStateBalanceDataMap: info.ConditionalApproveStateBalanceDataMap,
		}

		store.ModulesInfoMap[module] = infoStore
	}

	return store

}

func (g *BRC20ModuleIndexer) LoadStore(store *BRC20ModuleIndexerStore) {
	g.BestHeight = store.BestHeight
	g.EnableHistory = store.EnableHistory

	g.HistoryCount = store.HistoryCount

	g.FirstHistoryByHeight = store.FirstHistoryByHeight
	g.LastHistoryHeight = store.LastHistoryHeight

	// brc20 base
	g.AllHistory = store.AllHistory
	g.UserAllHistory = store.UserAllHistory

	g.InscriptionsTickerInfoMap = store.InscriptionsTickerInfoMap
	g.UserTokensBalanceData = store.UserTokensBalanceData

	// balance
	for u, userTokens := range g.UserTokensBalanceData {
		for uniqueLowerTicker, balance := range userTokens {
			tokenUsers, ok := g.TokenUsersBalanceData[uniqueLowerTicker]
			if !ok {
				tokenUsers = make(map[string]*model.BRC20TokenBalance, 0)
				g.TokenUsersBalanceData[uniqueLowerTicker] = tokenUsers
			}
			if balance.OverallBalance().Sign() > 0 {
				tokenUsers[u] = balance
			}
		}
	}

	g.InscriptionsValidBRC20DataMap = store.InscriptionsValidBRC20DataMap

	// inner valid transfer
	g.InscriptionsValidTransferMap = store.InscriptionsValidTransferMap
	// inner invalid transfer
	g.InscriptionsInvalidTransferMap = store.InscriptionsInvalidTransferMap

	// module
	// all modules info
	// module of users [address]moduleid
	g.UsersModuleWithTokenMap = store.UsersModuleWithTokenMap

	// module lp of users [address]moduleid
	g.UsersModuleWithLpTokenMap = store.UsersModuleWithLpTokenMap

	// runtime for approve
	g.InscriptionsValidApproveMap = store.InscriptionsValidApproveMap
	g.InscriptionsInvalidApproveMap = store.InscriptionsInvalidApproveMap

	// runtime for conditional approve
	g.InscriptionsValidConditionalApproveMap = store.InscriptionsValidConditionalApproveMap
	g.InscriptionsInvalidConditionalApproveMap = store.InscriptionsInvalidConditionalApproveMap

	// runtime for commit
	g.InscriptionsValidCommitMap = store.InscriptionsValidCommitMap
	g.InscriptionsInvalidCommitMap = store.InscriptionsInvalidCommitMap

	// InscriptionsValidCommitMapById
	for _, v := range g.InscriptionsValidCommitMap {
		g.InscriptionsValidCommitMapById[v.GetInscriptionId()] = v
	}

	for module, infoStore := range store.ModulesInfoMap {
		info := &model.BRC20ModuleSwapInfo{
			ID:                infoStore.ID,
			Name:              infoStore.Name,
			DeployerPkScript:  infoStore.DeployerPkScript,
			SequencerPkScript: infoStore.SequencerPkScript,
			GasToPkScript:     infoStore.GasToPkScript,
			LpFeePkScript:     infoStore.LpFeePkScript,

			FeeRateSwap: infoStore.FeeRateSwap,
			GasTick:     infoStore.GasTick,

			History: infoStore.History,

			// runtime for commit
			CommitInvalidMap: infoStore.CommitInvalidMap,
			CommitIdMap:      infoStore.CommitIdMap,
			CommitIdChainMap: infoStore.CommitIdChainMap,

			// token holders in module
			// ticker of users in module [address][tick]balanceData
			UsersTokenBalanceDataMap: infoStore.UsersTokenBalanceDataMap,
			TokenUsersBalanceDataMap: make(map[string]map[string]*model.BRC20ModuleTokenBalance, 0),

			// swap
			// lp token balance of address in module [pool][address]balance
			LPTokenUsersBalanceMap: infoStore.LPTokenUsersBalanceMap,
			UsersLPTokenBalanceMap: make(map[string]map[string]*decimal.Decimal, 0),

			// swap total balance
			// total balance of pool in module [pool]balanceData
			SwapPoolTotalBalanceDataMap: infoStore.SwapPoolTotalBalanceDataMap,

			// module deposit/withdraw state [tick]balanceData
			ConditionalApproveStateBalanceDataMap: infoStore.ConditionalApproveStateBalanceDataMap,
		}

		// tick/user: balance
		for address, dataMap := range info.UsersTokenBalanceDataMap {
			for uniqueLowerTicker, tokenBalance := range dataMap {
				tokenUsers, ok := info.TokenUsersBalanceDataMap[uniqueLowerTicker]
				if !ok {
					tokenUsers = make(map[string]*model.BRC20ModuleTokenBalance, 0)
					info.TokenUsersBalanceDataMap[uniqueLowerTicker] = tokenUsers
				}
				tokenUsers[address] = tokenBalance
			}
		}

		// pair/user: lpbalance
		for pair, dataMap := range info.LPTokenUsersBalanceMap {
			for address, lpBalance := range dataMap {
				userTokens, ok := info.UsersLPTokenBalanceMap[address]
				if !ok {
					userTokens = make(map[string]*decimal.Decimal, 0)
					info.UsersLPTokenBalanceMap[address] = userTokens
				}
				userTokens[pair] = lpBalance
			}
		}

		g.ModulesInfoMap[module] = info
	}
}
