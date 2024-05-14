package event

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/conf"
	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func InitTickDataFromFile(fname string) (brc20Datas []*model.InscriptionBRC20Data, err error) {
	// Open our jsonFile
	jsonFile, err := os.Open(fname)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	var ticksExternal []*model.InscriptionBRC20DeployContent
	err = json.Unmarshal([]byte(byteValue), &ticksExternal)
	if err != nil {
		return nil, err
	}

	for idx, info := range ticksExternal {
		var data model.InscriptionBRC20Data

		data.TxId = fmt.Sprintf("10%030x", idx)
		data.Idx = 0
		data.Vout = 0
		data.Offset = 0
		data.Satoshi = 330

		data.InscriptionId = fmt.Sprintf("10%030xi0", idx)
		data.InscriptionNumber = 100 + int64(idx)

		data.Height = 1
		data.TxIdx = uint32(idx)
		data.BlockTime = 100
		var key model.NFTCreateIdxKey = model.NFTCreateIdxKey{
			Height:     data.Height,
			IdxInBlock: uint64(idx), // fake idx
		}
		data.CreateIdxKey = key.String()
		data.IsTransfer = false

		data.ContentBody, _ = json.Marshal(info)
		data.Sequence = 0

		brc20Datas = append(brc20Datas, &data)
	}
	return brc20Datas, nil
}

func GenerateBRC20InputDataFromEvents(fname string) (brc20Datas []*model.InscriptionBRC20Data, err error) {
	// Open our jsonFile
	jsonFile, err := os.Open(fname)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	var events []*model.BRC20ModuleHistoryInfoEvent
	err = json.Unmarshal([]byte(byteValue), &events)
	if err != nil {
		return nil, err
	}

	for idx, e := range events {
		var data model.InscriptionBRC20Data

		txid, _ := hex.DecodeString(e.TxIdHex)
		data.TxId = string(txid)

		data.Idx = e.Idx
		data.Vout = e.Vout
		data.Offset = e.Offset
		data.Satoshi = e.Satoshi

		data.InscriptionId = e.InscriptionId // preset cache
		data.InscriptionNumber = e.InscriptionNumber

		data.Height = e.Height
		data.TxIdx = e.TxIdx
		data.BlockTime = e.BlockTime

		var key model.NFTCreateIdxKey = model.NFTCreateIdxKey{
			Height:     data.Height,
			IdxInBlock: uint64(idx), // fake idx
		}
		data.CreateIdxKey = key.String()

		var pkScriptFrom, pkScriptTo string
		if pk, err := utils.GetPkScriptByAddress(e.AddressFrom, conf.GlobalNetParams); err != nil {
			log.Printf("GenerateBRC20InputDataFromEvents [%d] pk invalid: %s", idx, err)
		} else {
			pkScriptFrom = string(pk)
		}
		if pk, err := utils.GetPkScriptByAddress(e.AddressTo, conf.GlobalNetParams); err != nil {
			pk, _ := hex.DecodeString(e.AddressTo)
			pkScriptTo = string(pk)
		} else {
			pkScriptTo = string(pk)
		}

		data.PkScript = pkScriptTo

		var inscribe model.InscriptionBRC20Data
		inscribe = data
		inscribe.IsTransfer = false

		inscribe.ContentBody = []byte(e.ContentBody)
		inscribe.Sequence = 0

		if e.Type == "transfer" {
			// mint
			var mint model.InscriptionBRC20Data
			mint = data
			mint.ContentBody = []byte(strings.Replace(e.ContentBody, "transfer", "mint", 1))
			mint.Sequence = 0
			mint.PkScript = pkScriptFrom
			brc20Datas = append(brc20Datas, &mint)

			// transfer
			inscribe.PkScript = pkScriptFrom
			brc20Datas = append(brc20Datas, &inscribe)

			// transfer send
			data.IsTransfer = true
			data.Sequence = 1
			data.PkScript = pkScriptTo
			brc20Datas = append(brc20Datas, &data)

		} else if e.Type == "commit" {
			inscribe.PkScript = pkScriptFrom

			brc20Datas = append(brc20Datas, &inscribe)

			// commit send
			data.IsTransfer = true
			data.Sequence = 1
			brc20Datas = append(brc20Datas, &data)

		} else if e.Type == "inscribe-module" {
			inscribe.PkScript = pkScriptTo

			brc20Datas = append(brc20Datas, &inscribe)

		} else if e.Type == "inscribe-conditional-approve" {
			inscribe.PkScript = pkScriptTo
			brc20Datas = append(brc20Datas, &inscribe)

			// send to delegator
			data.IsTransfer = true
			data.Sequence = 1
			data.PkScript = constant.ZERO_ADDRESS_PKSCRIPT
			brc20Datas = append(brc20Datas, &data)

		} else if e.Type == "conditional-approve" {
			content := strings.Replace(e.ContentBody, "brc20-swap", "brc-20", 1)

			// fixme: always use mint may fail
			// mint
			var mint model.InscriptionBRC20Data
			mint = data

			// e.Data.Amount fixme: replace e.Data.Amount with

			m1 := regexp.MustCompile(`"amt" *: *"[0-9\.]+"`)
			content = m1.ReplaceAllString(content, fmt.Sprintf(`"amt":"%s"`, e.Data.Amount))

			mintContent := strings.Replace(content, "conditional-approve", "mint", 1)
			mint.ContentBody = []byte(mintContent)
			mint.IsTransfer = false
			mint.Sequence = 0
			mint.PkScript = pkScriptTo
			brc20Datas = append(brc20Datas, &mint)

			// transfer
			var transfer model.InscriptionBRC20Data
			transfer = data
			transferContent := strings.Replace(content, "conditional-approve", "transfer", 1)
			transfer.ContentBody = []byte(transferContent)
			transfer.IsTransfer = false
			transfer.Sequence = 0
			transfer.PkScript = pkScriptTo
			brc20Datas = append(brc20Datas, &transfer)

			// transfer send
			var send model.InscriptionBRC20Data
			send = data
			send.IsTransfer = true
			send.Sequence = 1
			send.PkScript = pkScriptFrom
			brc20Datas = append(brc20Datas, &send)

			// send to delegator
			data.IsTransfer = true
			data.Sequence = 2
			data.PkScript = constant.ZERO_ADDRESS_PKSCRIPT
			brc20Datas = append(brc20Datas, &data)

		} else {
			log.Printf("GenerateBRC20InputDataFromEvents [%d] op invalid: %s", idx, e.Type)
			// fixme: inscribe-conditional-approve / conditional-approve
		}

	}

	return brc20Datas, nil
}
