package indexer

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"

	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func (g *BRC20ModuleIndexer) GetCommitInfoByKey(createIdxKey string) (
	commitData *model.InscriptionBRC20Data, isInvalid bool) {
	var ok bool
	// commit
	commitData, ok = g.InscriptionsValidCommitMap[createIdxKey]
	if !ok {
		commitData, ok = g.InscriptionsInvalidCommitMap[createIdxKey]
		if !ok {
			commitData = nil
		}
		isInvalid = true
	}

	return commitData, isInvalid
}

func (g *BRC20ModuleIndexer) ProcessCommit(dataFrom, dataTo *model.InscriptionBRC20Data, isInvalid bool) error {
	inscriptionId := dataFrom.GetInscriptionId()
	log.Printf("parse move commit. inscription id: %s", inscriptionId)

	// Delete the already sent commit
	delete(g.InscriptionsValidCommitMapById, inscriptionId)

	var body *model.InscriptionBRC20ModuleSwapCommitContent
	if err := json.Unmarshal(dataFrom.ContentBody, &body); err != nil {
		log.Printf("parse module commit json failed. txid: %s",
			hex.EncodeToString(utils.ReverseBytes([]byte(dataTo.TxId))),
		)
		return errors.New("json")
	}

	// Check the inscription reception address, it must be a module address.
	moduleId, ok := utils.GetModuleFromScript([]byte(dataTo.PkScript))
	if !ok || moduleId != body.Module {
		return errors.New("commit, not send to module")
	}

	// check module exist
	moduleInfo, ok := g.ModulesInfoMap[body.Module]
	if !ok {
		return errors.New("commit, module not exist")
	}

	// preset invalid
	moduleInfo.CommitInvalidMap[inscriptionId] = struct{}{}

	// Check the inscription sending address, it must be the sequencer address.
	if moduleInfo.SequencerPkScript != dataFrom.PkScript {
		return errors.New("module sequencer invalid")
	}

	eachFuntionSize, err := GetEachItemLengthOfCommitJsonData(dataFrom.ContentBody)
	if err != nil || len(body.Data) != len(eachFuntionSize) {
		return errors.New("commit, get function size failed")
	}

	log.Printf("ProcessCommitVerify commit[%s] ", inscriptionId)
	var pickUsersPkScript = make(map[string]bool, 0)
	var pickTokensTick = make(map[string]bool, 0)
	var pickPoolsPair = make(map[string]bool, 0)
	g.InitCherryPickFilter(body, pickUsersPkScript, pickTokensTick, pickPoolsPair)
	swapState := g.CherryPick(body.Module, pickUsersPkScript, pickTokensTick, pickPoolsPair)

	// Need to cherrypick, then verify on the copy.
	if idx, _, err := swapState.ProcessCommitVerify(inscriptionId, body, eachFuntionSize, nil); err != nil {
		log.Printf("commit invalid, function[%d] %s, txid: %s", idx, err, hex.EncodeToString([]byte(dataTo.TxId)))
		return err
	}
	// Execute in reality if successful.
	if idx, _, err := g.ProcessCommitVerify(inscriptionId, body, eachFuntionSize, nil); err != nil {
		log.Printf("commit invalid, function[%d] %s, txid: %s", idx, err, hex.EncodeToString([]byte(dataTo.TxId)))
		return err
	}

	// set commit id
	moduleInfo.CommitIdMap[inscriptionId] = struct{}{}
	moduleInfo.CommitIdChainMap[body.Parent] = struct{}{}

	// valid
	delete(moduleInfo.CommitInvalidMap, inscriptionId)

	history := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_SWAP_TYPE_N_COMMIT, dataFrom, dataTo, nil, true)
	moduleInfo.History = append(moduleInfo.History, history)
	return nil
}

func GetCommitParentFromData(data *model.InscriptionBRC20Data) (string, error) {
	var body *model.InscriptionBRC20ModuleSwapCommitContent
	if err := json.Unmarshal(data.ContentBody, &body); err != nil {
		return "", errors.New("json")
	}
	return body.Parent, nil
}

func (g *BRC20ModuleIndexer) ProcessCommitCheck(data *model.InscriptionBRC20Data) (int, error) {
	var body *model.InscriptionBRC20ModuleSwapCommitContent
	if err := json.Unmarshal(data.ContentBody, &body); err != nil {
		return -1, errors.New("json")
	}

	// check module exist
	moduleInfo, ok := g.ModulesInfoMap[body.Module]
	if !ok {
		return -1, errors.New("commit, module not exist")
	}

	eachFuntionSize, err := GetEachItemLengthOfCommitJsonData(data.ContentBody)
	if err != nil || len(body.Data) != len(eachFuntionSize) {
		return -1, errors.New("commit, get function size failed")
	}

	inscriptionId := data.GetInscriptionId()
	log.Printf("ProcessCommitVerify commit[%s] ", inscriptionId)
	idx, _, err := g.ProcessCommitVerify(inscriptionId, body, eachFuntionSize, nil)
	if err != nil {
		return idx, err
	}

	// set commit id
	moduleInfo.CommitIdMap[inscriptionId] = struct{}{}
	moduleInfo.CommitIdChainMap[body.Parent] = struct{}{}

	// Delete the already sent commit
	delete(g.InscriptionsValidCommitMapById, inscriptionId)

	return 0, nil
}
