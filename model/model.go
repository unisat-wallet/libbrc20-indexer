package model

import (
	"encoding/hex"
	"fmt"

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

type InscriptionBRC20InfoResp struct {
	Operation    string `json:"op,omitempty"`
	BRC20Tick    string `json:"tick,omitempty"`
	BRC20Max     string `json:"max,omitempty"`
	BRC20Limit   string `json:"lim,omitempty"`
	BRC20Amount  string `json:"amt,omitempty"`
	BRC20Decimal string `json:"decimal,omitempty"`
	BRC20Minted  string `json:"minted,omitempty"`
}

type InscriptionBRC20Content struct {
	Proto        string `json:"p,omitempty"`
	Operation    string `json:"op,omitempty"`
	BRC20Tick    string `json:"tick,omitempty"`
	BRC20Max     string `json:"max,omitempty"`
	BRC20Limit   string `json:"lim,omitempty"`
	BRC20Amount  string `json:"amt,omitempty"`
	BRC20Decimal string `json:"dec,omitempty"`
}

type BRC20TokenInfo struct {
	Ticker string
	Deploy *InscriptionBRC20TickInfo

	History                 []*BRC20History
	HistoryMint             []*BRC20History
	HistoryInscribeTransfer []*BRC20History
	HistoryTransfer         []*BRC20History
}

type InscriptionBRC20TickInfoResp struct {
	Height            uint32                   `json:"-"`
	Data              InscriptionBRC20InfoResp `json:"data"`
	InscriptionNumber int64                    `json:"inscriptionNumber"`
	InscriptionId     string                   `json:"inscriptionId"`
	Confirmations     int                      `json:"confirmations"`
}

type InscriptionBRC20TickInfo struct {
	Data  InscriptionBRC20InfoResp `json:"data"`
	Max   *decimal.Decimal         `json:"-"`
	Limit *decimal.Decimal         `json:"-"`

	TotalMinted        *decimal.Decimal `json:"-"`
	ConfirmedMinted    *decimal.Decimal `json:"-"`
	ConfirmedMinted1h  *decimal.Decimal `json:"-"`
	ConfirmedMinted24h *decimal.Decimal `json:"-"`

	Amount    *decimal.Decimal `json:"-"`
	MintTimes uint32           `json:"-"`
	Decimal   uint8            `json:"-"`

	TxId string `json:"-"`
	Idx  uint32 `json:"-"`
	Vout uint32 `json:"-"`

	Satoshi  uint64 `json:"-"`
	PkScript string `json:"-"`

	InscriptionNumber int64  `json:"inscriptionNumber"`
	CreateIdxKey      string `json:"-"`
	Height            uint32 `json:"-"`
	TxIdx             uint32 `json:"-"`
	BlockTime         uint32 `json:"-"`

	CompleteHeight    uint32 `json:"-"`
	CompleteBlockTime uint32 `json:"-"`

	InscriptionNumberStart int64 `json:"-"`
	InscriptionNumberEnd   int64 `json:"-"`
}

func NewInscriptionBRC20TickInfo(body *InscriptionBRC20Content, data *InscriptionBRC20Data) *InscriptionBRC20TickInfo {
	info := &InscriptionBRC20TickInfo{
		Data: InscriptionBRC20InfoResp{
			Operation: body.Operation,
			BRC20Tick: body.BRC20Tick,
		},
		Decimal: 18,

		TxId: data.TxId,
		Idx:  data.Idx,
		Vout: data.Vout,

		Satoshi:  data.Satoshi,
		PkScript: data.PkScript,

		InscriptionNumber: data.InscriptionNumber,
		CreateIdxKey:      data.CreateIdxKey,
		Height:            data.Height,
		TxIdx:             data.TxIdx,
		BlockTime:         data.BlockTime,
	}
	return info
}

type BRC20TokenBalance struct {
	Ticker              string
	PkScript            string
	OverallBalanceSafe  *decimal.Decimal
	OverallBalance      *decimal.Decimal
	TransferableBalance *decimal.Decimal
	InvalidTransferList []*InscriptionBRC20TickInfo
	ValidTransferMap    map[string]*InscriptionBRC20TickInfo

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
			Data:              info.Data,
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
