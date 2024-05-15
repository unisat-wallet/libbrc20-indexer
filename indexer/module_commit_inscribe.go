package indexer

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

// ProcessInscribeCommit inscribed a commit, but it has not taken effect yet.
func (g *BRC20ModuleIndexer) ProcessInscribeCommit(data *model.InscriptionBRC20Data) (err error) {
	inscriptionId := data.GetInscriptionId()
	log.Printf("parse new inscribe commit. inscription id: %s", inscriptionId)

	var body *model.InscriptionBRC20ModuleSwapCommitContent
	if err := json.Unmarshal(data.ContentBody, &body); err != nil {
		log.Printf("parse commit json failed. txid: %s",
			hex.EncodeToString(utils.ReverseBytes([]byte(data.TxId))),
		)
		return errors.New("json invalid")
	}

	// lower case module id only
	if body.Module != strings.ToLower(body.Module) {
		return errors.New("module id invalid")
	}

	// check module exist
	moduleInfo, ok := g.ModulesInfoMap[body.Module]
	if !ok {
		return errors.New("module invalid")
	}

	// preset invalid
	moduleInfo.CommitInvalidMap[inscriptionId] = struct{}{}

	// check sequencer match
	if moduleInfo.SequencerPkScript != data.PkScript {
		return errors.New("module sequencer invalid")
	}

	idx, err := g.ProcessInscribeCommitPreVerify(body)
	if err != nil {
		log.Printf("commit invalid inscribe. function[%d], %s, txid: %s", idx, err, hex.EncodeToString([]byte(data.TxId)))
		return err
	}
	g.InscriptionsValidCommitMap[data.CreateIdxKey] = data
	g.InscriptionsValidCommitMapById[inscriptionId] = data

	// valid
	delete(moduleInfo.CommitInvalidMap, inscriptionId)

	return nil
}
