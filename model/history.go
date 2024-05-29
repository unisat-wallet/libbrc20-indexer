package model

import (
	"encoding/binary"

	scriptDecoder "github.com/unisat-wallet/libbrc20-indexer/utils/script"
)

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

	Inscription InscriptionBRC20TickInfoResp

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
			Fee:       to.Fee,
		},
		Inscription: InscriptionBRC20TickInfoResp{
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
	} else {
		history.OverallBalance = "0"
		history.TransferableBalance = "0"
		history.AvailableBalance = "0"
	}
	return history
}

func (h *BRC20History) Marshal() (result []byte) {
	var buf [1024]byte

	// type
	buf[0] = h.Type
	// valid
	if h.Valid {
		buf[1] = 1
	} else {
		buf[1] = 0
	}
	// txid
	copy(buf[2:2+32], h.TxId[:])

	offset := 34

	offset += scriptDecoder.PutVLQ(buf[offset:], uint64(h.Idx))
	offset += scriptDecoder.PutVLQ(buf[offset:], uint64(h.Vout))
	offset += scriptDecoder.PutVLQ(buf[offset:], uint64(h.Offset))

	offset += scriptDecoder.PutCompressedScript(buf[offset:], []byte(h.PkScriptFrom))
	offset += scriptDecoder.PutCompressedScript(buf[offset:], []byte(h.PkScriptTo))

	offset += scriptDecoder.PutVLQ(buf[offset:], uint64(h.Satoshi))
	offset += scriptDecoder.PutVLQ(buf[offset:], uint64(h.Fee))

	binary.LittleEndian.PutUint32(buf[offset:], h.Height) // 4
	offset += 4
	offset += scriptDecoder.PutVLQ(buf[offset:], uint64(h.TxIdx))
	binary.LittleEndian.PutUint32(buf[offset:], h.BlockTime) // 4
	offset += 4

	// Amount
	n := len(h.Amount)
	if n < 40 {
		buf[offset] = uint8(n)
		offset += 1
		copy(buf[offset:offset+n], h.Amount[:])
		offset += n
	} else {
		buf[offset] = 0
		offset += 1
	}

	// OverallBalance
	n = len(h.OverallBalance)
	if n < 40 {
		buf[offset] = uint8(n)
		offset += 1
		copy(buf[offset:offset+n], h.OverallBalance[:])
		offset += n
	} else {
		buf[offset] = 0
		offset += 1
	}

	// TransferableBalance
	n = len(h.TransferableBalance)
	if n < 40 {
		buf[offset] = uint8(n)
		offset += 1
		copy(buf[offset:offset+n], h.TransferableBalance[:])
		offset += n
	} else {
		buf[offset] = 0
		offset += 1
	}

	// AvailableBalance
	n = len(h.AvailableBalance)
	if n < 40 {
		buf[offset] = uint8(n)
		offset += 1
		copy(buf[offset:offset+n], h.AvailableBalance[:])
		offset += n
	} else {
		buf[offset] = 0
		offset += 1
	}

	// Inscription
	binary.LittleEndian.PutUint32(buf[offset:], h.Inscription.Height) // 4
	offset += 4
	offset += scriptDecoder.PutVLQ(buf[offset:], uint64(h.Inscription.InscriptionNumber))
	offset += scriptDecoder.PutVLQ(buf[offset:], uint64(h.Inscription.Satoshi))
	// inscriptionId
	n = len(h.Inscription.InscriptionId)
	if n < 70 {
		buf[offset] = uint8(n)
		offset += 1
		copy(buf[offset:offset+n], h.Inscription.InscriptionId[:])
		offset += n
	} else {
		buf[offset] = 0
		offset += 1
	}

	// data
	data := h.Inscription.Data
	if data == nil {
		result = make([]byte, offset)
		copy(result, buf[:offset])
		return result
	}

	// BRC20Tick
	n = len(data.BRC20Tick)
	if n < 16 {
		buf[offset] = uint8(n)
		offset += 1
		copy(buf[offset:offset+n], data.BRC20Tick[:])
		offset += n
	} else {
		buf[offset] = 0
		offset += 1
	}

	// BRC20Max
	n = len(data.BRC20Max)
	if n < 40 {
		buf[offset] = uint8(n)
		offset += 1
		copy(buf[offset:offset+n], data.BRC20Max[:])
		offset += n
	} else {
		buf[offset] = 0
		offset += 1
	}
	// BRC20Limit
	n = len(data.BRC20Limit)
	if n < 40 {
		buf[offset] = uint8(n)
		offset += 1
		copy(buf[offset:offset+n], data.BRC20Limit[:])
		offset += n
	} else {
		buf[offset] = 0
		offset += 1
	}
	// BRC20Amount
	n = len(data.BRC20Amount)
	if n < 40 {
		buf[offset] = uint8(n)
		offset += 1
		copy(buf[offset:offset+n], data.BRC20Amount[:])
		offset += n
	} else {
		buf[offset] = 0
		offset += 1
	}
	// BRC20Decimal
	n = len(data.BRC20Decimal)
	if n < 8 {
		buf[offset] = uint8(n)
		offset += 1
		copy(buf[offset:offset+n], data.BRC20Decimal[:])
		offset += n
	} else {
		buf[offset] = 0
		offset += 1
	}
	// BRC20Minted
	n = len(data.BRC20Minted)
	if n < 40 {
		buf[offset] = uint8(n)
		offset += 1
		copy(buf[offset:offset+n], data.BRC20Minted[:])
		offset += n
	} else {
		buf[offset] = 0
		offset += 1
	}
	// BRC20SelfMint
	n = len(data.BRC20SelfMint)
	if n < 8 {
		buf[offset] = uint8(n)
		offset += 1
		copy(buf[offset:offset+n], data.BRC20SelfMint[:])
		offset += n
	} else {
		buf[offset] = 0
		offset += 1
	}
	result = make([]byte, offset)
	copy(result, buf[:offset])
	return result
}

func (h *BRC20History) Unmarshal(buf []byte) {
	h.Type = buf[0]
	h.Valid = (buf[1] == 1)

	h.TxId = string(buf[2 : 2+32])

	offset := 34

	idx, bytesRead := scriptDecoder.DeserializeVLQ(buf[offset:])
	if bytesRead >= len(buf[offset:]) {
		return
	}
	h.Idx = uint32(idx)
	offset += bytesRead

	vout, bytesRead := scriptDecoder.DeserializeVLQ(buf[offset:])
	if bytesRead >= len(buf[offset:]) {
		return
	}
	h.Vout = uint32(vout)
	offset += bytesRead

	nftOffset, bytesRead := scriptDecoder.DeserializeVLQ(buf[offset:])
	if bytesRead >= len(buf[offset:]) {
		return
	}
	h.Offset = nftOffset
	offset += bytesRead

	// Decode the compressed script size and ensure there are enough bytes
	// left in the slice for it.
	scriptSize := scriptDecoder.DecodeCompressedScriptSize(buf[offset:])
	if len(buf[offset:]) < scriptSize {
		return
	}
	h.PkScriptFrom = string(scriptDecoder.DecompressScript(buf[offset : offset+scriptSize]))
	offset += scriptSize

	scriptSize = scriptDecoder.DecodeCompressedScriptSize(buf[offset:])
	if len(buf[offset:]) < scriptSize {
		return
	}
	h.PkScriptTo = string(scriptDecoder.DecompressScript(buf[offset : offset+scriptSize]))
	offset += scriptSize

	satoshi, bytesRead := scriptDecoder.DeserializeVLQ(buf[offset:])
	if bytesRead >= len(buf[offset:]) {
		return
	}
	h.Satoshi = satoshi
	offset += bytesRead

	fee, bytesRead := scriptDecoder.DeserializeVLQ(buf[offset:])
	if bytesRead >= len(buf[offset:]) {
		return
	}
	h.Fee = int64(fee)
	offset += bytesRead

	h.Height = binary.LittleEndian.Uint32(buf[offset:]) // 4
	offset += 4

	txidx, bytesRead := scriptDecoder.DeserializeVLQ(buf[offset:])
	if bytesRead >= len(buf[offset:]) {
		return
	}
	h.TxIdx = uint32(txidx)
	offset += bytesRead

	h.BlockTime = binary.LittleEndian.Uint32(buf[offset:]) // 4
	offset += 4

	// Amount
	n := int(buf[offset])
	offset += 1
	if n > 0 {
		h.Amount = string(buf[offset : offset+n])
		offset += n
	}

	// OverallBalance
	n = int(buf[offset])
	offset += 1
	if n > 0 {
		h.OverallBalance = string(buf[offset : offset+n])
		offset += n
	}

	// TransferableBalance
	n = int(buf[offset])
	offset += 1
	if n > 0 {
		h.TransferableBalance = string(buf[offset : offset+n])
		offset += n
	}

	// AvailableBalance
	n = int(buf[offset])
	offset += 1
	if n > 0 {
		h.AvailableBalance = string(buf[offset : offset+n])
		offset += n
	}

	// Inscription
	h.Inscription.Height = binary.LittleEndian.Uint32(buf[offset:]) // 4
	offset += 4

	number, bytesRead := scriptDecoder.DeserializeVLQ(buf[offset:])
	if bytesRead >= len(buf[offset:]) {
		return
	}
	h.Inscription.InscriptionNumber = int64(number)
	offset += bytesRead

	nftSatoshi, bytesRead := scriptDecoder.DeserializeVLQ(buf[offset:])
	if bytesRead >= len(buf[offset:]) {
		return
	}
	h.Inscription.Satoshi = nftSatoshi
	offset += bytesRead

	// inscriptionId
	n = int(buf[offset])
	offset += 1
	if n > 0 {
		h.Inscription.InscriptionId = string(buf[offset : offset+n])
		offset += n
	}

	// data
	if len(buf[offset:]) == 0 {
		return
	}
	data := &InscriptionBRC20InfoResp{}
	h.Inscription.Data = data

	// BRC20Tick
	n = int(buf[offset])
	offset += 1
	if n > 0 {
		data.BRC20Tick = string(buf[offset : offset+n])
		offset += n
	}

	// BRC20Max
	n = int(buf[offset])
	offset += 1
	if n > 0 {
		data.BRC20Max = string(buf[offset : offset+n])
		offset += n
	}

	// BRC20Limit
	n = int(buf[offset])
	offset += 1
	if n > 0 {
		data.BRC20Limit = string(buf[offset : offset+n])
		offset += n
	}

	// BRC20Amount
	n = int(buf[offset])
	offset += 1
	if n > 0 {
		data.BRC20Amount = string(buf[offset : offset+n])
		offset += n
	}

	// BRC20Decimal
	n = int(buf[offset])
	offset += 1
	if n > 0 {
		data.BRC20Decimal = string(buf[offset : offset+n])
		offset += n
	}

	// BRC20Minted
	n = int(buf[offset])
	offset += 1
	if n > 0 {
		data.BRC20Minted = string(buf[offset : offset+n])
		offset += n
	}

	// BRC20SelfMint
	n = int(buf[offset])
	offset += 1
	if n > 0 {
		data.BRC20SelfMint = string(buf[offset : offset+n])
		offset += n
	}
}
