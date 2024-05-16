package indexer

import (
	"errors"
	"log"

	"github.com/unisat-wallet/libbrc20-indexer/conf"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func (g *BRC20ModuleIndexer) ProcessCommitFunctionSend(moduleInfo *model.BRC20ModuleSwapInfo, f *model.SwapFunctionData) error {
	addressTo := f.Params[0]
	pkScriptTo, _ := utils.GetPkScriptByAddress(addressTo, conf.GlobalNetParams)

	tokenOrPair := f.Params[1]
	tokenAmtStr := f.Params[2]

	tokenAmt, _ := g.CheckTickVerify(tokenOrPair, tokenAmtStr)
	tokenBalanceFrom := moduleInfo.GetUserTokenBalance(tokenOrPair, f.PkScript)

	// fixme: Must use the confirmed amount
	if tokenBalanceFrom.SwapAccountBalance.Cmp(tokenAmt) < 0 {
		log.Printf("token[%s] user[%s], balance %s", tokenOrPair, f.Address, tokenBalanceFrom)
		return errors.New("send: token balance insufficient")
	}

	tokenBalanceTo := moduleInfo.GetUserTokenBalance(tokenOrPair, string(pkScriptTo))

	// User Real-time Balance Update
	tokenBalanceFrom.SwapAccountBalance = tokenBalanceFrom.SwapAccountBalance.Sub(tokenAmt)
	tokenBalanceTo.SwapAccountBalance = tokenBalanceTo.SwapAccountBalance.Add(tokenAmt)

	tokenBalanceFrom.UpdateHeight = g.BestHeight
	tokenBalanceTo.UpdateHeight = g.BestHeight

	log.Printf("pool send [%s] swappable: %s -> %s", tokenOrPair, tokenBalanceFrom.SwapAccountBalance, tokenBalanceTo.SwapAccountBalance)

	return nil
}
