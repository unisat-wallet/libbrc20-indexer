package model

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

type InscriptionBRC20Data struct {
	IsTransfer bool
	TxId       string `json:"-"`
	Idx        uint32 `json:"-"`
	Vout       uint32 `json:"-"`

	Satoshi  uint64 `json:"-"`
	PkScript string `json:"-"`

	InscriptionNumber int64
	ContentBody       []byte
	CreateIdxKey      string
	Height            uint32 // Height of NFT show in block onCreate
	TxIdx             uint32
	BlockTime         uint32
}

type InscriptionBRC20Content struct {
	Proto        string `json:"p,omitempty"`
	Operation    string `json:"op,omitempty"`
	BRC20Tick    string `json:"tick,omitempty"`
	BRC20Max     string `json:"max,omitempty"`
	BRC20Amount  string `json:"amt,omitempty"`
	BRC20Limit   string `json:"lim,omitempty"` // option
	BRC20Decimal string `json:"dec,omitempty"` // option
}

func (body *InscriptionBRC20Content) Unmarshal(contentBody []byte) (err error) {
	var bodyMap map[string]interface{} = make(map[string]interface{}, 8)
	if err := json.Unmarshal(contentBody, &bodyMap); err != nil {
		return err
	}
	if v, ok := bodyMap["p"].(string); ok {
		body.Proto = v
	}
	if v, ok := bodyMap["op"].(string); ok {
		body.Operation = v
	}
	if v, ok := bodyMap["tick"].(string); ok {
		body.BRC20Tick = v
	}
	if v, ok := bodyMap["max"].(string); ok {
		body.BRC20Max = v
	}
	if v, ok := bodyMap["amt"].(string); ok {
		body.BRC20Amount = v
	}

	if _, ok := bodyMap["lim"]; !ok {
		body.BRC20Limit = body.BRC20Max
	} else {
		if v, ok := bodyMap["lim"].(string); ok {
			body.BRC20Limit = v
		}
	}

	if _, ok := bodyMap["dec"]; !ok {
		body.BRC20Decimal = constant.DEFAULT_DECIMAL_18
	} else {
		if v, ok := bodyMap["dec"].(string); ok {
			body.BRC20Decimal = v
		}
	}

	return nil
}

type BRC20TokenInfo struct {
	Ticker string
	Deploy *InscriptionBRC20TickDeployInfo

	History                 []*BRC20History
	HistoryMint             []*BRC20History
	HistoryInscribeTransfer []*BRC20History
	HistoryTransfer         []*BRC20History
}

type InscriptionBRC20TickInfoResp struct {
	Height            uint32 `json:"-"`
	InscriptionNumber int64  `json:"inscriptionNumber"`
	InscriptionId     string `json:"inscriptionId"`
	Confirmations     int    `json:"confirmations"`
}

type BRC20TokenBalance struct {
	Ticker              string
	PkScript            string
	OverallBalanceSafe  *decimal.Decimal
	OverallBalance      *decimal.Decimal
	TransferableBalance *decimal.Decimal
	InvalidTransferList []*InscriptionBRC20TickTransferInfo
	ValidTransferMap    map[string]*InscriptionBRC20TickTransferInfo

	History                 []*BRC20History
	HistoryMint             []*BRC20History
	HistoryInscribeTransfer []*BRC20History
	HistorySend             []*BRC20History
	HistoryReceive          []*BRC20History
}
type BRC20History struct {
	Type        uint8 // inscribe-deploy/inscribe-mint/inscribe-transfer/transfer/send/receive
	Valid       bool
	Inscription InscriptionBRC20TickInfoResp

	TxId string
	Idx  uint32
	Vout uint32

	PkScriptFrom string
	PkScriptTo   string
	Satoshi      uint64

	Amount              string
	OverallBalance      string
	TransferableBalance string
	AvailableBalance    string

	Height    uint32
	TxIdx     uint32
	BlockTime uint32
}

func NewBRC20History(historyType uint8, isValid bool, isTransfer bool,
	info *InscriptionBRC20TickInfo, bal *BRC20TokenBalance, data *InscriptionBRC20Data) *BRC20History {
	history := &BRC20History{
		Type:  historyType,
		Valid: isValid,
		Inscription: InscriptionBRC20TickInfoResp{
			Height:            data.Height,
			InscriptionNumber: info.InscriptionNumber,
			InscriptionId:     fmt.Sprintf("%si%d", hex.EncodeToString(utils.ReverseBytes([]byte(data.TxId))), data.Idx),
		},
		Amount:    info.Amount.String(),
		Height:    data.Height,
		TxIdx:     data.TxIdx,
		BlockTime: data.BlockTime,
	}
	if isTransfer {
		history.TxId = data.TxId
		history.Vout = data.Vout
		history.Idx = data.Idx
		history.PkScriptFrom = info.PkScript
		history.PkScriptTo = data.PkScript
		history.Satoshi = data.Satoshi
	} else {
		history.TxId = info.TxId
		history.Vout = info.Vout
		history.Idx = info.Idx
		history.PkScriptTo = info.PkScript
		history.Satoshi = info.Satoshi
	}

	if bal != nil {
		history.OverallBalance = bal.OverallBalance.String()
		history.TransferableBalance = bal.TransferableBalance.String()
		history.AvailableBalance = bal.OverallBalance.Sub(bal.TransferableBalance).String()
	}
	return history
}
