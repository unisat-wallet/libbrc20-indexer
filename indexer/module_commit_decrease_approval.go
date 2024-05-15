package indexer

import (
	"errors"
	"log"

	"github.com/unisat-wallet/libbrc20-indexer/model"
)

func (g *BRC20ModuleIndexer) ProcessCommitFunctionDecreaseApproval(moduleInfo *model.BRC20ModuleSwapInfo, f *model.SwapFunctionData) error {

	token := f.Params[0]
	tokenAmtStr := f.Params[1]
	tokenAmt, _ := g.CheckTickVerify(token, tokenAmtStr)

	tokenBalance := moduleInfo.GetUserTokenBalance(token, f.PkScript)

	// fixme: Must use the confirmed amount
	if tokenBalance.SwapAccountBalance.Cmp(tokenAmt) < 0 {
		log.Printf("token[%s] user[%s], balance %s", token, f.Address, tokenBalance)
		return errors.New("decreaseApproval: token balance insufficient")
	}

	// User Real-time Balance Update
	tokenBalance.SwapAccountBalance = tokenBalance.SwapAccountBalance.Sub(tokenAmt)
	tokenBalance.AvailableBalance = tokenBalance.AvailableBalance.Add(tokenAmt)

	tokenBalance.UpdateHeight = g.BestHeight

	log.Printf("pool decreaseApproval [%s] available: %s, swappable: %s", token, tokenBalance.AvailableBalance, tokenBalance.SwapAccountBalance)
	return nil
}
