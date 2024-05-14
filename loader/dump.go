package loader

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/conf"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func DumpBRC20InputData(fname string, brc20Datas chan interface{}, hexBody bool) {
	file, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Fatalf("open block index file failed, %s", err)
		return
	}
	defer file.Close()

	for dataIn := range brc20Datas {
		data := dataIn.(*model.InscriptionBRC20Data)

		var body, address string
		if hexBody {
			body = hex.EncodeToString(data.ContentBody)
			address = hex.EncodeToString([]byte(data.PkScript))
		} else {
			body = strings.ReplaceAll(string(data.ContentBody), "\n", " ")
			address, err = utils.GetAddressFromScript([]byte(data.PkScript), conf.GlobalNetParams)
			if err != nil {
				address = hex.EncodeToString([]byte(data.PkScript))
			}
		}

		fmt.Fprintf(file, "%t %s %d %d %d %d %s %d %s %s %d %d %d %d\n",
			data.IsTransfer,

			hex.EncodeToString([]byte(data.TxId)),
			data.Idx,
			data.Vout,
			data.Offset,
			data.Satoshi,
			address,
			data.InscriptionNumber,
			body,
			hex.EncodeToString([]byte(data.CreateIdxKey)),
			data.Height,
			data.TxIdx,
			data.BlockTime,
			data.Sequence,
		)
	}
}
