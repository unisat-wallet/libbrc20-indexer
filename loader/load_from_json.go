package loader

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func LoadBRC20InputJsonData(fname string) ([]*model.InscriptionBRC20Data, error) {
	file, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var brc20Datas []*model.InscriptionBRC20Data
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, " ")

		if len(fields) != 13 {
			return nil, fmt.Errorf("invalid data format")
		}

		var data model.InscriptionBRC20Data
		data.IsTransfer, err = strconv.ParseBool(fields[0])
		if err != nil {
			return nil, err
		}

		txid, err := hex.DecodeString(fields[1])
		if err != nil {
			return nil, err
		}
		data.TxId = string(utils.ReverseBytes([]byte(txid)))

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

		data.ContentBody = []byte(fields[8])

		createIdxKey, err := hex.DecodeString(fields[9])
		if err != nil {
			return nil, err
		}

		data.CreateIdxKey = string(createIdxKey)

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
