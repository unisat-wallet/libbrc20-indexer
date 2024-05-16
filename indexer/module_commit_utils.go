package indexer

import (
	"bytes"
	"container/list"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
	"github.com/unisat-wallet/libbrc20-indexer/utils/bip322"

	"github.com/btcsuite/btcd/wire"
)

// GetFunctionDataId Calculate ID hash, used for signing.
func GetFunctionDataContent(contentPrefix string, data *model.SwapFunctionData) (content string) {
	content = contentPrefix + fmt.Sprintf(`addr: %s
func: %s
params: %s
ts: %d
`, data.Address, data.Function, strings.Join(data.Params, " "), data.Timestamp)
	return content
}

func CheckFunctionSigVerify(contentPrefix string, data *model.SwapFunctionData, previous []string) (id string, ok bool) {
	if len(previous) != 0 {
		contentPrefix += fmt.Sprintf("prevs: %s\n", strings.Join(previous, " "))
	}

	content := GetFunctionDataContent(contentPrefix, data)
	// check id
	id = utils.HashString(utils.GetSha256([]byte(content)))
	message := GetFunctionDataContent(fmt.Sprintf("id: %s\n", id), data)

	signature, err := base64.StdEncoding.DecodeString(data.Signature)
	if err != nil {
		log.Println("CheckFunctionSigVerify decoding signature:", err)
		return id, false
	}

	var wit wire.TxWitness
	lenSignature := len(signature)
	if len(signature) == 66 {
		wit = wire.TxWitness{signature[2:]}
	} else if lenSignature > (2+64+34) && lenSignature <= (2+72+34) {
		wit = wire.TxWitness{signature[2 : lenSignature-34], signature[lenSignature-33 : lenSignature]}
	} else {
		fmt.Println("b64 sig:", hex.EncodeToString(signature))
		fmt.Println("pkScript:", hex.EncodeToString([]byte(data.PkScript)))
		fmt.Println("b64 sig length invalid")
		return id, false
	}

	// check sig
	if ok := bip322.VerifySignature(wit, []byte(data.PkScript), message); !ok {
		log.Printf("CheckFunctionSigVerify. content: %s", content)
		fmt.Println("sig invalid")
		return id, false
	}
	return id, true
}

// CheckAmountVerify Verify the legality of the brc20 tick amt.
func CheckAmountVerify(amtStr string, nDecimal uint8) (amt *decimal.Decimal, ok bool) {
	// check amount
	amt, err := decimal.NewDecimalFromString(amtStr, int(nDecimal))
	if err != nil {
		return nil, false
	}
	if amt.Sign() < 0 {
		return nil, false
	}

	return amt, true
}

// CheckTickVerify Verify the legality of the brc20 tick amt.
func (g *BRC20ModuleIndexer) CheckTickVerify(tick string, amtStr string) (amt *decimal.Decimal, ok bool) {
	uniqueLowerTicker := strings.ToLower(tick)
	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		return
	}

	if amtStr == "" {
		return nil, true
	}

	tinfo := tokenInfo.Deploy

	// check amount
	amt, err := decimal.NewDecimalFromString(amtStr, int(tinfo.Decimal))
	if err != nil {
		return nil, false
	}
	if amt.Sign() < 0 || amt.Cmp(tinfo.Max) > 0 {
		return nil, false
	}

	return amt, true
}

// CheckTickVerify Verify the legality of the brc20 tick amt.
func (g *BRC20ModuleIndexer) CheckTickVerifyBigInt(tick string, amtStr string) (amt *decimal.Decimal, ok bool) {
	uniqueLowerTicker := strings.ToLower(tick)
	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		return
	}

	if amtStr == "" {
		return nil, true
	}

	tinfo := tokenInfo.Deploy

	// check amount
	amt, err := decimal.NewDecimalFromString(amtStr, 0)
	if err != nil {
		return nil, false
	}
	amt.Precition = uint(tinfo.Decimal)
	if amt.Sign() < 0 || amt.Cmp(tinfo.Max) > 0 {
		return nil, false
	}

	return amt, true
}

func GetLowerInnerPairNameByToken(token0, token1 string) (poolPair string) {
	token0 = strings.ToLower(token0)
	token1 = strings.ToLower(token1)

	if token0 > token1 {
		poolPair = fmt.Sprintf("%s%s%s", string([]byte{uint8(len(token1))}), token1, token0)
	} else {
		poolPair = fmt.Sprintf("%s%s%s", string([]byte{uint8(len(token0))}), token0, token1)
	}
	return poolPair
}

// GetEachItemLengthOfCommitJsonData Get the actual number of bytes occupied by obj in the data list
func GetEachItemLengthOfCommitJsonData(body []byte) (results []uint64, err error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	const (
		TOKEN_TYPE_OBJ = iota
		TOKEN_TYPE_ARR
	)
	curType := -1
	const (
		TOKEN_VALUE_MAPKEY = iota
		TOKEN_VALUE_MAPVALUE
		TOKEN_VALUE_ARRAY_ELEMENT
	)
	curEle := -1

	indentLevel := 0

	stack := list.New()

	setEleType := func() {
		switch curType {
		case TOKEN_TYPE_OBJ:
			curEle = TOKEN_VALUE_MAPKEY
		case TOKEN_TYPE_ARR:
			curEle = TOKEN_VALUE_ARRAY_ELEMENT
		}
	}

	readyDataProcess := false
	startDataProcess := false
	var lastPos uint64

	for {
		tok, err := decoder.Token()
		// Return the next unprocessed token.
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		offset := decoder.InputOffset()

		switch tok := tok.(type) {
		// Based on the token type, appropriate processing is performed.
		case json.Delim:

			switch tok {
			case '{':

				if indentLevel == 2 && readyDataProcess && startDataProcess {
					// Step 3: Record start offset at '{' character.
					lastPos = uint64(offset)
				}

				stack.PushBack(TOKEN_TYPE_OBJ)
				curType = TOKEN_TYPE_OBJ
				setEleType()
				indentLevel += 1
			case '}':

				if indentLevel == 3 && readyDataProcess && startDataProcess {
					// Step 4: Record length at '}' character.
					results = append(results, uint64(offset)-lastPos+1)
				}

				stack.Remove(stack.Back())
				if stack.Len() > 0 {
					curType = stack.Back().Value.(int)
					setEleType()
				}
				indentLevel -= 1
			case '[':

				if indentLevel == 1 && readyDataProcess && !startDataProcess {
					// Step 2: Start formally counting after '['.
					results = nil
					startDataProcess = true
				}

				stack.PushBack(TOKEN_TYPE_ARR)
				curType = TOKEN_TYPE_ARR
				setEleType()
				indentLevel += 1
			case ']':

				if indentLevel == 2 && readyDataProcess && startDataProcess {
					// Step 5: End the statistics after ']'.
					readyDataProcess = false
					startDataProcess = false
				}

				stack.Remove(stack.Back())
				if stack.Len() > 0 {
					curType = stack.Back().Value.(int)
					setEleType()
				}
				indentLevel -= 1
			}

		default:
			switch curType {
			case TOKEN_TYPE_OBJ:
				switch curEle {
				case TOKEN_VALUE_MAPKEY:

					if indentLevel == 1 {
						if tok == "data" {
							// Step 1: Mark the data start, and initialize the marker and result variables.
							results = nil
							readyDataProcess = true
							startDataProcess = false
						} else {
							// Step 6: Mark complete.
							readyDataProcess = false
							startDataProcess = false
						}
					}

					curEle = TOKEN_VALUE_MAPVALUE
				case TOKEN_VALUE_MAPVALUE:
					curEle = TOKEN_VALUE_MAPKEY
				}
			}
		}
	}

	return results, nil
}
