package indexer

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/unisat-wallet/libbrc20-indexer/conf"
	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func (g *BRC20ModuleIndexer) ProcessCreateModule(data *model.InscriptionBRC20Data) error {
	var body model.InscriptionBRC20ModuleDeploySwapContent
	if err := json.Unmarshal(data.ContentBody, &body); err != nil {
		log.Printf("parse create module json failed. txid: %s",
			hex.EncodeToString(utils.ReverseBytes([]byte(data.TxId))),
		)
		return err
	}

	if conf.MODULE_SWAP_SOURCE_INSCRIPTION_ID != body.Source {
		return errors.New(fmt.Sprintf("source not match: %s", body.Source))
	}

	inscriptionId := data.GetInscriptionId()
	log.Printf("create module: %s", inscriptionId)

	if _, ok := g.ModulesInfoMap[inscriptionId]; ok {
		return errors.New("dup module deploy") // impossible
	}

	// feeRateSwap
	feeRateSwap, ok := body.Init["swap_fee_rate"]
	if !ok {
		feeRateSwap = "0"
	}
	if _, ok := CheckAmountVerify(feeRateSwap, 3); !ok {
		return errors.New("swap fee invalid")
	}

	// gasTick
	gasTick, ok := body.Init["gas_tick"]
	if !ok {
		// gas is not optional
		return errors.New("gas_tick missing")
	}
	if _, ok := g.CheckTickVerify(gasTick, ""); !ok {
		log.Printf("create module gas tick[%s] invalid", gasTick)
		return errors.New("gas_tick invalid")
	}

	// sequencer default
	sequencerPkScript := data.PkScript
	if sequencer, ok := body.Init["sequencer"]; ok {
		if pk, err := utils.GetPkScriptByAddress(sequencer, conf.GlobalNetParams); err != nil {
			return errors.New("sequencer invalid")
		} else {
			sequencerPkScript = string(pk)
		}
	} else {
		return errors.New("sequencer missing")
	}

	// gasTo default
	gasToPkScript := data.PkScript
	if gasTo, ok := body.Init["gas_to"]; ok {
		if pk, err := utils.GetPkScriptByAddress(gasTo, conf.GlobalNetParams); err != nil {
			return errors.New("gasTo invalid")
		} else {
			gasToPkScript = string(pk)
		}
	} else {
		return errors.New("gas_to missing")
	}

	// lpFeeTo default
	lpFeeToPkScript := data.PkScript
	if lpFeeTo, ok := body.Init["fee_to"]; ok {
		if pk, err := utils.GetPkScriptByAddress(lpFeeTo, conf.GlobalNetParams); err != nil {
			return errors.New("lpFeeTo invalid")
		} else {
			lpFeeToPkScript = string(pk)
		}
	} else {
		return errors.New("fee_to missing")
	}

	m := &model.BRC20ModuleSwapInfo{
		ID:                inscriptionId,
		Name:              body.Name,
		DeployerPkScript:  data.PkScript,     // deployer
		SequencerPkScript: sequencerPkScript, // Sequencer
		GasToPkScript:     gasToPkScript,
		LpFeePkScript:     lpFeeToPkScript,

		FeeRateSwap: feeRateSwap,
		GasTick:     gasTick,

		History: make([]*model.BRC20ModuleHistory, 0),

		// runtime for commit
		CommitInvalidMap: make(map[string]struct{}, 0),
		CommitIdChainMap: make(map[string]struct{}, 0),
		CommitIdMap:      make(map[string]struct{}, 0),

		// runtime for holders
		// token holders in module
		// ticker of users in module [address][tick]balanceData
		UsersTokenBalanceDataMap: make(map[string]map[string]*model.BRC20ModuleTokenBalance, 0),

		// token balance of address in module [tick][address]balanceData
		TokenUsersBalanceDataMap: make(map[string]map[string]*model.BRC20ModuleTokenBalance, 0),

		// swap
		// lp token balance of address in module [pool][address]balance
		LPTokenUsersBalanceMap:        make(map[string]map[string]*decimal.Decimal, 0),
		LPTokenUsersBalanceUpdatedMap: make(map[string]struct{}, 0),

		// lp token of users in module [moduleid][address][pool]balance
		UsersLPTokenBalanceMap: make(map[string]map[string]*decimal.Decimal, 0),

		// swap total balance
		// total balance of pool in module [pool]balanceData
		SwapPoolTotalBalanceDataMap: make(map[string]*model.BRC20ModulePoolTotalBalance, 0),

		ConditionalApproveStateBalanceDataMap: make(map[string]*model.BRC20ModuleConditionalApproveStateBalance, 0),
	}

	m.UpdateHeight = data.Height

	// deployInfo := model.NewInscriptionBRC20SwapInfo(data)
	// deployInfo.Module = inscriptionId

	history := model.NewBRC20ModuleHistory(false, constant.BRC20_HISTORY_MODULE_TYPE_N_INSCRIBE_MODULE, data, data, nil, true)
	m.History = append(m.History, history)

	g.ModulesInfoMap[inscriptionId] = m

	return nil
}
