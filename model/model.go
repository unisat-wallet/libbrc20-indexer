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
	TxIdx      uint32 `json:"-"`

	Satoshi  uint64 `json:"-"`
	PkScript string `json:"-"`

	InscriptionNumber int64
	ContentBody       []byte
	CreateIdxKey      string
	Height            uint32 // Height of NFT show in block onCreate
	BlockTime         uint32
}

type InscriptionBRC20InfoResp struct {
	Operation    string `json:"op,omitempty"`
	BRC20Tick    string `json:"tick,omitempty"`
	BRC20Max     string `json:"max,omitempty"`
	BRC20Limit   string `json:"lim,omitempty"`
	BRC20Amount  string `json:"amt,omitempty"`
	BRC20To      string `json:"to,omitempty"`
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
	Ticker  string
	Deploy  *InscriptionBRC20TickInfo

	History []*BRC20History
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
	MintTimes uint64           `json:"-"`
	Decimal   uint8            `json:"-"`

	TxId  string `json:"-"`
	TxIdx uint32 `json:"-"`

	Satoshi  uint64 `json:"-"`
	PkScript string `json:"-"`

	InscriptionNumber int64  `json:"inscriptionNumber"`
	InscriptionId     string `json:"inscriptionId"`
	CreateIdxKey      string `json:"-"`
	Height            uint32 `json:"-"`
	BlockTime         uint32 `json:"-"`

	Confirmations int `json:"confirmations"`

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

		TxId:  data.TxId,
		TxIdx: data.TxIdx,

		Satoshi:  data.Satoshi,
		PkScript: data.PkScript,

		InscriptionNumber: data.InscriptionNumber,
		InscriptionId:     fmt.Sprintf("%si%d", hex.EncodeToString(utils.ReverseBytes([]byte(data.TxId))), data.TxIdx),
		CreateIdxKey:      data.CreateIdxKey,
		Height:            data.Height,
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
	Deploy              *InscriptionBRC20TickInfo

	History []*BRC20History
}
type BRC20History struct {
	Type        string // inscribe-deploy/inscribe-mint/inscribe-transfer/transfer/send/receive
	Valid       bool
	Inscription *InscriptionBRC20TickInfo

	TxId  string
	TxIdx uint32

	PkScriptFrom string
	PkScriptTo   string
	Satoshi      uint64

	Amount              string
	OverallBalance      string
	TransferableBalance string
	AvailableBalance    string

	Height    uint32
	BlockTime uint32
}

func NewBRC20History(historyType string, isValid bool, isTransfer bool,
	info *InscriptionBRC20TickInfo, bal *BRC20TokenBalance, data *InscriptionBRC20Data) *BRC20History {
	history := &BRC20History{
		Type:        historyType,
		Valid:       isValid,
		Inscription: info,
		Amount:      info.Amount.String(),
		Height:      data.Height,
		BlockTime:   data.BlockTime,
	}
	if isTransfer {
		history.TxId = data.TxId
		history.TxIdx = data.TxIdx
		history.PkScriptFrom = info.PkScript
		history.PkScriptTo = data.PkScript
		history.Satoshi = data.Satoshi
	} else {
		history.TxId = info.TxId
		history.TxIdx = info.TxIdx
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
