package utils

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

func DecodeTokensFromSwapPair(tickPair string) (token0, token1 string, err error) {
	if len(tickPair) != 9 || tickPair[4] != '/' {
		return "", "", errors.New("func: removeLiq tickPair invalid")
	}
	token0 = tickPair[:4]
	token1 = tickPair[5:]

	return token0, token1, nil
}

func GetValidUniqueLowerTickerTicker(ticker string) (lowerTicker string, err error) {
	if len(ticker) != 4 && len(ticker) != 5 {
		return "", errors.New("ticker len invalid")
	}

	lowerTicker = strings.ToLower(ticker)
	return lowerTicker, nil
}

// single sha256 hash
func GetSha256(data []byte) (hash []byte) {
	sha := sha256.New()
	sha.Write(data[:])
	hash = sha.Sum(nil)
	return
}

func GetHash256(data []byte) (hash []byte) {
	sha := sha256.New()
	sha.Write(data[:])
	tmp := sha.Sum(nil)
	sha.Reset()
	sha.Write(tmp)
	hash = sha.Sum(nil)
	return
}

func HashString(data []byte) (res string) {
	length := 32
	var reverseData [32]byte

	// need reverse
	for i := 0; i < length; i++ {
		reverseData[i] = data[length-i-1]
	}
	return hex.EncodeToString(reverseData[:])
}

func ReverseBytes(data []byte) (result []byte) {
	for _, b := range data {
		result = append([]byte{b}, result...)
	}
	return result
}

// PayToTaprootScript creates a pk script for a pay-to-taproot output key.
func PayToTaprootScript(taprootKey *btcec.PublicKey) ([]byte, error) {
	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_1).
		AddData(schnorr.SerializePubKey(taprootKey)).
		Script()
}

// PayToWitnessScript creates a pk script for a pay-to-wpkh output key.
func PayToWitnessScript(pubkey *btcec.PublicKey) ([]byte, error) {
	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_0).
		AddData(btcutil.Hash160(pubkey.SerializeCompressed())).
		Script()
}

func GetPkScriptByAddress(addr string, netParams *chaincfg.Params) (pk []byte, err error) {
	if len(addr) == 0 {
		return nil, errors.New("decoded address empty")
	}

	addressObj, err := btcutil.DecodeAddress(addr, netParams)
	if err != nil {
		if len(addr) != 68 || !strings.HasPrefix(addr, "6a20") {
			return nil, errors.New("decoded address is of unknown format")
		}
		// check full hex
		pkHex, err := hex.DecodeString(addr)
		if err != nil {
			return nil, errors.New("decoded address is of unknown format")
		}
		return pkHex, nil
	}
	addressPkScript, err := txscript.PayToAddrScript(addressObj)
	if err != nil {
		return nil, errors.New("decoded address is of unknown format")
	}
	return addressPkScript, nil
}

// GetAddressFromScript Use btcsuite to get address
func GetAddressFromScript(script []byte, params *chaincfg.Params) (string, error) {
	scriptClass, addresses, _, err := txscript.ExtractPkScriptAddrs(script, params)
	if err != nil {
		return "", fmt.Errorf("failed to get address: %v", err)
	}

	if len(addresses) == 0 {
		return "", fmt.Errorf("noaddress")
	}

	if scriptClass == txscript.NonStandardTy {
		return "", fmt.Errorf("non-standard")
	}

	return addresses[0].EncodeAddress(), nil
}

func GetModuleFromScript(script []byte) (module string, ok bool) {
	if len(script) < 34 || len(script) > 38 {
		return "", false
	}
	if script[0] != 0x6a {
		return "", false
	}
	if int(script[1])+2 != len(script) {
		return "", false
	}

	var idx uint32
	if script[1] <= 32 {
		idx = uint32(0)
	} else if script[1] <= 33 {
		idx = uint32(script[34])
	} else if script[1] <= 34 {
		idx = uint32(binary.LittleEndian.Uint16(script[34:36]))
	} else if script[1] <= 35 {
		idx = uint32(script[34]) | uint32(script[35])<<8 | uint32(script[36])<<16
	} else if script[1] <= 36 {
		idx = binary.LittleEndian.Uint32(script[34:38])
	}

	module = fmt.Sprintf("%si%d", HashString(script[2:34]), idx)
	return module, true
}

func DecodeInscriptionFromBin(script []byte) (id string) {
	n := len(script)
	if n < 32 || n > 36 {
		return ""
	}

	var idx uint32
	if n == 32 {
		idx = uint32(0)
	} else if n <= 33 {
		idx = uint32(script[32])
	} else if n <= 34 {
		idx = uint32(binary.LittleEndian.Uint16(script[32:34]))
	} else if n <= 35 {
		idx = uint32(script[32]) | uint32(script[33])<<8 | uint32(script[34])<<16
	} else if n <= 36 {
		idx = binary.LittleEndian.Uint32(script[32:36])
	}

	id = fmt.Sprintf("%si%d", HashString(script[:32]), idx)
	return id
}
