package loader

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func LoadBRC20InputData(fname string) ([]*model.InscriptionBRC20Data, error) {
	file, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var brc20Datas []*model.InscriptionBRC20Data
	scanner := bufio.NewScanner(file)
	max := 128 * 1024 * 1024
	buf := make([]byte, max)
	scanner.Buffer(buf, max)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, " ")

		if len(fields) != 13 {
			return nil, fmt.Errorf("invalid data format")
		}

		var data model.InscriptionBRC20Data

		sequence, err := strconv.ParseUint(fields[0], 10, 16)
		if err != nil {
			return nil, err
		}
		data.Sequence = uint16(sequence)
		data.IsTransfer = (data.Sequence > 0)

		txid, err := hex.DecodeString(fields[1])
		if err != nil {
			return nil, err
		}
		data.TxId = string(txid)

		idx, err := strconv.ParseUint(fields[2], 10, 32)
		if err != nil {
			return nil, err
		}
		data.Idx = uint32(idx)

		vout, err := strconv.ParseUint(fields[3], 10, 32)
		if err != nil {
			return nil, err
		}
		data.Vout = uint32(vout)

		offset, err := strconv.ParseUint(fields[4], 10, 64)
		if err != nil {
			return nil, err
		}
		data.Offset = uint64(offset)

		satoshi, err := strconv.ParseUint(fields[5], 10, 64)
		if err != nil {
			return nil, err
		}
		data.Satoshi = uint64(satoshi)

		pkScript, err := hex.DecodeString(fields[6])
		if err != nil {
			return nil, err
		}
		data.PkScript = string(pkScript)

		inscriptionNumber, err := strconv.ParseInt(fields[7], 10, 64)
		if err != nil {
			return nil, err
		}
		data.InscriptionNumber = int64(inscriptionNumber)

		content, err := hex.DecodeString(fields[8])
		if err != nil {
			return nil, err
		}
		data.ContentBody = content
		data.CreateIdxKey = string(fields[9])

		height, err := strconv.ParseUint(fields[10], 10, 32)
		if err != nil {
			return nil, err
		}
		data.Height = uint32(height)

		txIdx, err := strconv.ParseUint(fields[11], 10, 32)
		if err != nil {
			return nil, err
		}
		data.TxIdx = uint32(txIdx)

		blockTime, err := strconv.ParseUint(fields[12], 10, 32)
		if err != nil {
			return nil, err
		}
		data.BlockTime = uint32(blockTime)

		brc20Datas = append(brc20Datas, &data)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return brc20Datas, nil
}

func DumpTickerInfoMap(fname string,
	inscriptionsTickerInfoMap map[string]*model.BRC20TokenInfo,
	userTokensBalanceData map[string]map[string]*model.BRC20TokenBalance,
	tokenUsersBalanceData map[string]map[string]*model.BRC20TokenBalance,
	testnet bool,
) {

	netParams := &chaincfg.MainNetParams
	if testnet {
		netParams = &chaincfg.TestNet3Params
	}

	file, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Fatalf("open block index file failed, %s", err)
		return
	}
	defer file.Close()

	var allTickers []string
	for ticker := range inscriptionsTickerInfoMap {
		allTickers = append(allTickers, ticker)
	}
	sort.SliceStable(allTickers, func(i, j int) bool {
		return allTickers[i] < allTickers[j]
	})

	for _, ticker := range allTickers {
		info := inscriptionsTickerInfoMap[ticker]
		nValid := 0
		for _, h := range info.History {
			if h.Valid {
				nValid++
			}
		}

		fmt.Fprintf(file, "%s history: %d, valid: %d, minted: %s, holders: %d\n",
			info.Ticker,
			len(info.History),
			nValid,
			info.Deploy.TotalMinted.String(),
			len(tokenUsersBalanceData[ticker]),
		)

		// history
		for _, h := range info.History {
			if !h.Valid {
				continue
			}

			addressFrom, err := utils.GetAddressFromScript([]byte(h.PkScriptFrom), netParams)
			if err != nil {
				addressFrom = hex.EncodeToString([]byte(h.PkScriptFrom))
			}

			addressTo, err := utils.GetAddressFromScript([]byte(h.PkScriptTo), netParams)
			if err != nil {
				addressTo = hex.EncodeToString([]byte(h.PkScriptTo))
			}

			fmt.Fprintf(file, "%s %s %s %s %s -> %s\n",
				info.Ticker,
				utils.GetReversedStringHex(h.TxId),
				constant.BRC20_HISTORY_TYPE_NAMES[h.Type],
				h.Amount,
				addressFrom,
				addressTo,
			)
		}

		// holders
		var allHolders []string
		for holder := range tokenUsersBalanceData[ticker] {
			allHolders = append(allHolders, holder)
		}
		sort.SliceStable(allHolders, func(i, j int) bool {
			return allHolders[i] < allHolders[j]
		})

		// holders
		for _, holder := range allHolders {
			balanceData := tokenUsersBalanceData[ticker][holder]

			address, err := utils.GetAddressFromScript([]byte(balanceData.PkScript), netParams)
			if err != nil {
				address = hex.EncodeToString([]byte(balanceData.PkScript))
			}
			fmt.Fprintf(file, "%s %s history: %d, transfer: %d, balance: %s, tokens: %d\n",
				info.Ticker,
				address,
				len(balanceData.History),
				len(balanceData.ValidTransferMap),
				balanceData.OverallBalance.String(),
				len(userTokensBalanceData[string(balanceData.PkScript)]),
			)
		}
	}
}
