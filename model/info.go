package model

import (
	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
)

type InscriptionBRC20TickInfo struct {
	BRC20Tick string           `json:"-"`
	Operation uint8            `json:"-"`
	Decimal   uint8            `json:"-"`
	Amount    *decimal.Decimal `json:"-"`

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
}

// for deploy
type InscriptionBRC20TickDeployInfo struct {
	InscriptionBRC20TickInfo

	Max   *decimal.Decimal `json:"-"`
	Limit *decimal.Decimal `json:"-"`

	TotalMinted        *decimal.Decimal `json:"-"`
	ConfirmedMinted    *decimal.Decimal `json:"-"`
	ConfirmedMinted1h  *decimal.Decimal `json:"-"`
	ConfirmedMinted24h *decimal.Decimal `json:"-"`

	MintTimes uint32 `json:"-"`

	CompleteHeight    uint32 `json:"-"`
	CompleteBlockTime uint32 `json:"-"`

	InscriptionNumberStart int64 `json:"-"`
	InscriptionNumberEnd   int64 `json:"-"`
}

func NewInscriptionBRC20TickDeployInfo(body *InscriptionBRC20Content, data *InscriptionBRC20Data) *InscriptionBRC20TickDeployInfo {
	info := &InscriptionBRC20TickDeployInfo{
		InscriptionBRC20TickInfo: InscriptionBRC20TickInfo{
			BRC20Tick: body.BRC20Tick,
			Operation: constant.BRC20_OP_N_DEPLOY,
			Decimal:   18,

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
		},
	}
	return info
}

// for mint
type InscriptionBRC20TickMintInfo struct {
	InscriptionBRC20TickInfo
}

func NewInscriptionBRC20TickMintInfo(body *InscriptionBRC20Content, data *InscriptionBRC20Data) *InscriptionBRC20TickMintInfo {
	info := &InscriptionBRC20TickMintInfo{
		InscriptionBRC20TickInfo: InscriptionBRC20TickInfo{
			BRC20Tick: body.BRC20Tick,
			Operation: constant.BRC20_OP_N_MINT,
			Decimal:   18,

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
		},
	}
	return info
}

// for transfer
type InscriptionBRC20TickTransferInfo struct {
	InscriptionBRC20TickInfo
}

func NewInscriptionBRC20TickTransferInfo(body *InscriptionBRC20Content, data *InscriptionBRC20Data) *InscriptionBRC20TickTransferInfo {
	info := &InscriptionBRC20TickTransferInfo{
		InscriptionBRC20TickInfo: InscriptionBRC20TickInfo{
			BRC20Tick: body.BRC20Tick,
			Operation: constant.BRC20_OP_N_TRANSFER,
			Decimal:   18,

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
		},
	}
	return info
}
