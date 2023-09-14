package utils

import (
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

func GetReversedStringHex(data string) (result string) {
	return hex.EncodeToString(ReverseBytes([]byte(data)))
}

func ReverseBytes(data []byte) (result []byte) {
	for _, b := range data {
		result = append([]byte{b}, result...)
	}
	return result
}

// GetAddressFromScript 。
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
