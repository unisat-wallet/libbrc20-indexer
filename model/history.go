package model

type BRC20HistoryBase struct {
	Type  uint8 // inscribe-deploy/inscribe-mint/inscribe-transfer/transfer/send/receive
	Valid bool

	TxId   string
	Idx    uint32
	Vout   uint32
	Offset uint64

	PkScriptFrom string
	PkScriptTo   string
	Satoshi      uint64
	Fee          int64

	Height    uint32
	TxIdx     uint32
	BlockTime uint32
}

// history
type BRC20History struct {
	BRC20HistoryBase

	Inscription *InscriptionBRC20TickInfoResp

	// param
	Amount string

	// state
	OverallBalance      string
	TransferableBalance string
	AvailableBalance    string
}

func NewBRC20History(historyType uint8, isValid bool, isTransfer bool,
	from *InscriptionBRC20TickInfo, bal *BRC20TokenBalance, to *InscriptionBRC20Data) *BRC20History {
	history := &BRC20History{
		BRC20HistoryBase: BRC20HistoryBase{
			Type:      historyType,
			Valid:     isValid,
			Height:    to.Height,
			TxIdx:     to.TxIdx,
			BlockTime: to.BlockTime,
		},
		Inscription: &InscriptionBRC20TickInfoResp{
			Height:            from.Height,
			Data:              from.Data,
			InscriptionNumber: from.InscriptionNumber,
			InscriptionId:     from.GetInscriptionId(),
			Satoshi:           from.Satoshi,
		},
		Amount: from.Amount.String(),
	}
	if isTransfer {
		history.TxId = to.TxId
		history.Vout = to.Vout
		history.Offset = to.Offset
		history.Idx = to.Idx
		history.PkScriptFrom = from.PkScript
		history.PkScriptTo = to.PkScript
		history.Satoshi = to.Satoshi
		if history.Satoshi == 0 {
			history.PkScriptTo = history.PkScriptFrom
		}

	} else {
		history.TxId = from.TxId
		history.Vout = from.Vout
		history.Offset = from.Offset
		history.Idx = from.Idx
		history.PkScriptTo = from.PkScript
		history.Satoshi = from.Satoshi
	}

	if bal != nil {
		history.OverallBalance = bal.AvailableBalance.Add(bal.TransferableBalance).String()
		history.TransferableBalance = bal.TransferableBalance.String()
		history.AvailableBalance = bal.AvailableBalance.String()
	}
	return history
}
