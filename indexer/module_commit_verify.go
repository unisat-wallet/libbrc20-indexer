package indexer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/conf"
	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

var GResultsExternal []*model.SwapFunctionResultCheckState

func InitResultDataFromFile(fname string) (err error) {
	// Open our jsonFile
	jsonFile, err := os.Open(fname)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
		return err
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(byteValue), &GResultsExternal)
	if err != nil {
		return err
	}
	return nil
}

func (g *BRC20ModuleIndexer) BRC20ModulePrepareSwapCommitContent(
	commitsStr []string,
	commitsObj []*model.InscriptionBRC20ModuleSwapCommitContent) {

	total := len(commitsStr)
	if total < 2 {
		return
	}

	for idx, commitStr := range commitsStr[:total-1] {
		nextCommitObj := commitsObj[idx+1]

		if _, ok := g.InscriptionsValidCommitMapById[nextCommitObj.Parent]; ok {
			continue
		}

		data := &model.InscriptionBRC20Data{
			InscriptionId: nextCommitObj.Parent,
			ContentBody:   []byte(commitStr),
		}
		g.InscriptionsValidCommitMapById[nextCommitObj.Parent] = data
	}
}

func (g *BRC20ModuleIndexer) BRC20ModuleVerifySwapCommitContent(
	commitStr string,
	commitObj *model.InscriptionBRC20ModuleSwapCommitContent,
	results []*model.SwapFunctionResultCheckState) (idx int, critical bool, err error) {

	if len(commitObj.Data) != len(results) {
		return -1, false, errors.New("commit verify, function results different size")
	}

	idx, err = g.ProcessInscribeCommitPreVerify(commitObj)
	if err != nil {
		log.Printf("commit verify failed: inscribe pre function[%d] %s", idx, err)
		return idx, true, err
	}

	// Verifying commit that was not moved in the middle
	// check module exist
	moduleInfo, ok := g.ModulesInfoMap[commitObj.Module]
	if !ok {
		return -1, true, errors.New("commit, module not exist")
	}

	parentId := commitObj.Parent
	// invalid if parent commit not exist
	commitIdsToCheck := []string{}
	for parentId != "" {
		if _, ok := moduleInfo.CommitIdMap[parentId]; ok {
			break
		}
		commitIdsToCheck = append([]string{parentId}, commitIdsToCheck...)

		parentCommitData, ok := g.InscriptionsValidCommitMapById[parentId]
		if !ok {
			return -1, false, errors.New("commit, parent body missing")
		}

		parentId, err = GetCommitParentFromData(parentCommitData)
		if err != nil {
			return -1, true, errors.New("commit, parent json invalid")
		}
	}

	for _, parentId := range commitIdsToCheck {
		parentCommitData, ok := g.InscriptionsValidCommitMapById[parentId]
		if !ok {
			return -1, false, errors.New("commit, parent body not ready")
		}

		if idx, err := g.ProcessCommitCheck(parentCommitData); err != nil {
			return idx, true, err
		}
	}

	// verify current commit
	eachFuntionSize, err := GetEachItemLengthOfCommitJsonData([]byte(commitStr))
	if err != nil {
		return -1, true, errors.New("commit, get function size failed")
	}
	idx, critical, err = g.ProcessCommitVerify("", commitObj, eachFuntionSize, results)
	if err != nil {
		log.Printf("commit verify failed, send function[%d] invalid", idx)
		return idx, critical, err
	}
	return 0, false, nil
}

func (g *BRC20ModuleIndexer) BRC20ResultsPreVerify(moduleInfo *model.BRC20ModuleSwapInfo, result *model.SwapFunctionResultCheckState) (err error) {
	// check user amt format
	for idxUser, user := range result.Users {
		userPkScript := constant.ZERO_ADDRESS_PKSCRIPT
		// format check
		if user.Address != "0" {
			if pk, err := utils.GetPkScriptByAddress(user.Address, conf.GlobalNetParams); err != nil {
				return errors.New(fmt.Sprintf("result users[%d] addr(%s) invalid", idxUser, user.Address))
			} else {
				userPkScript = string(pk)
			}
		}
		if len(user.Tick) == 4 {
			tokenAmt, ok := g.CheckTickVerify(user.Tick, user.Balance)
			if !ok {
				return errors.New(fmt.Sprintf("result users[%d] balance invalid", idxUser))
			}

			// balance check
			tokenBalance := moduleInfo.GetUserTokenBalance(user.Tick, userPkScript)
			if tokenBalance.SwapAccountBalance.Cmp(tokenAmt) != 0 {
				return errors.New(fmt.Sprintf("result users[%d] %s amount not match (%s != %s)",
					idxUser, user.Tick,
					tokenAmt.String(),
					tokenBalance.SwapAccountBalance.String(),
				))
			}

		} else {
			token0, token1, err := utils.DecodeTokensFromSwapPair(user.Tick)
			if err != nil {
				return errors.New(fmt.Sprintf("result users[%d] tick invalid", idxUser))
			}

			if _, ok := g.CheckTickVerify(token0, ""); !ok {
				return errors.New(fmt.Sprintf("result users[%d] tick/0 invalid", idxUser))
			}

			if _, ok := g.CheckTickVerify(token1, ""); !ok {
				return errors.New(fmt.Sprintf("result users[%d] tick/1 invalid", idxUser))
			}
			lpAmt, ok := CheckAmountVerify(user.Balance, 18)
			if !ok {
				return errors.New(fmt.Sprintf("result users[%d] Lp Amount invalid", idxUser))
			}

			// balance check
			poolPair := GetLowerInnerPairNameByToken(token0, token1)
			usersLpBalanceInPool, ok := moduleInfo.LPTokenUsersBalanceMap[poolPair]
			if !ok {
				return errors.New(fmt.Sprintf("result users[%d] Pair invalid", idxUser))
			}
			lpBalance := usersLpBalanceInPool[userPkScript]
			if lpBalance.Cmp(lpAmt) != 0 {
				return errors.New(fmt.Sprintf("result users[%d] %s lp balance not match", idxUser, poolPair))
			}
		}
	}

	// check pool amt format
	for idxPool, poolResult := range result.Pools {
		// format check
		token0, token1, err := utils.DecodeTokensFromSwapPair(poolResult.Pair)
		if err != nil {
			return errors.New(fmt.Sprintf("result pools[%d] Pair invalid", idxPool))
		}
		token0Amt, ok := g.CheckTickVerify(token0, poolResult.ReserveAmount0)
		if !ok {
			return errors.New(fmt.Sprintf("result pools[%d] Amount0 invalid", idxPool))
		}

		token1Amt, ok := g.CheckTickVerify(token1, poolResult.ReserveAmount1)
		if !ok {
			return errors.New(fmt.Sprintf("result pools[%d] Amount1 invalid", idxPool))
		}

		lpAmt, ok := CheckAmountVerify(poolResult.LPAmount, 18)
		if !ok {
			return errors.New(fmt.Sprintf("result pools[%d] Lp Amount invalid", idxPool))
		}

		// balance check
		poolPair := GetLowerInnerPairNameByToken(token0, token1)
		pool, ok := moduleInfo.SwapPoolTotalBalanceDataMap[poolPair]
		if !ok {
			return errors.New(fmt.Sprintf("result pools[%d] missing pair[%s]", idxPool, poolPair))
		}
		// Determine the token order id of the pool
		var token0Idx, token1Idx int
		if token0 == pool.Tick[0] {
			token0Idx = 0
			token1Idx = 1
		} else {
			token0Idx = 1
			token1Idx = 0
		}

		if token0Amt.Cmp(pool.TickBalance[token0Idx]) != 0 {
			return errors.New(fmt.Sprintf("result pool[%d] %s balance not match", idxPool, pool.Tick[token0Idx]))
		}

		if token1Amt.Cmp(pool.TickBalance[token1Idx]) != 0 {
			return errors.New(fmt.Sprintf("result pool[%d] %s balance not match", idxPool, pool.Tick[token1Idx]))
		}

		lpAmt.Precition = 18
		if lpAmt.Cmp(pool.LpBalance) != 0 {
			return errors.New(fmt.Sprintf("result pool[%d] %s lpbalance not match", idxPool, poolPair))
		}
	}

	return nil
}

// ProcessInscribeCommit Created a commit, but it has not yet taken effect.
func (g *BRC20ModuleIndexer) ProcessInscribeCommitPreVerify(body *model.InscriptionBRC20ModuleSwapCommitContent) (index int, err error) {
	if body.Module != strings.ToLower(body.Module) {
		return -1, errors.New("module id invalid")
	}

	// check module exist
	moduleInfo, ok := g.ModulesInfoMap[body.Module]
	if !ok {
		return -1, errors.New("module invalid")
	}

	// check gasPrice
	if _, ok := g.CheckTickVerify(moduleInfo.GasTick, body.GasPrice); !ok {
		log.Printf("ProcessInscribeCommit commit gas err: %s", body.GasPrice)
		return -1, errors.New("gas price invalid")
	}

	// common content
	content := fmt.Sprintf("module: %s\n", moduleInfo.ID)
	if body.Parent != "" {
		content += fmt.Sprintf("parent: %s\n", body.Parent)
	}
	if body.GasPrice != "" {
		content += fmt.Sprintf("gas_price: %s\n", body.GasPrice)
	}

	paramOffset := 0
	if g.BestHeight >= conf.ENABLE_SWAP_WITHDRAW_HEIGHT {
		paramOffset = 1
	}

	// for previous id
	functionsByAddressMap := make(map[string][]string)
	for idx, f := range body.Data {
		if pkScript, err := utils.GetPkScriptByAddress(f.Address, conf.GlobalNetParams); err != nil {
			return idx, errors.New("addr invalid")
		} else {
			f.PkScript = string(pkScript)
		}

		// log.Printf("ProcessInscribeCommitPreVerify func[%d] %s(%s)", idx, f.Function, strings.Join(f.Params, ", "))

		// get prevouse function id by user
		previous := functionsByAddressMap[f.Address]
		if id, ok := CheckFunctionSigVerify(content, f, previous); !ok {
			return idx, errors.New(fmt.Sprintf("function[%d]%s sig invalid", idx, id))
		} else {
			// update previous id list
			previous = append(previous, id)
			functionsByAddressMap[f.Address] = previous
		}

		// function process
		if f.Function == constant.BRC20_SWAP_FUNCTION_DEPLOY_POOL {
			if len(f.Params) != 2 {
				return idx, errors.New("func: deploy params invalid")
			}
			token0 := f.Params[0]
			token1 := f.Params[1]
			if token0 == token1 {
				return idx, errors.New("func: deploy same tokens")
			}
			if _, ok := g.InscriptionsTickerInfoMap[strings.ToLower(token0)]; !ok {
				return idx, errors.New("func: deploy tick0 invalid")
			}
			if _, ok := g.InscriptionsTickerInfoMap[strings.ToLower(token1)]; !ok {
				return idx, errors.New("func: deploy tick1 invalid")
			}

			// Check for duplicate pairs when the Commit inscription effect is applied.

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_ADD_LIQ {
			if len(f.Params) != 5+paramOffset {
				return idx, errors.New("func: addLiq params invalid")
			}

			token0, token1 := f.Params[0], f.Params[1]
			if g.BestHeight < conf.ENABLE_SWAP_WITHDRAW_HEIGHT {
				token0, token1, err = utils.DecodeTokensFromSwapPair(f.Params[0])
				if err != nil {
					return idx, errors.New("func: addLiq poolPair invalid")
				}
			}

			token0AmtStr := f.Params[1+paramOffset]
			token1AmtStr := f.Params[2+paramOffset]
			tokenLpAmtStr := f.Params[3+paramOffset]
			slippage := f.Params[4+paramOffset]

			if _, ok := g.CheckTickVerify(token0, token0AmtStr); !ok {
				return idx, errors.New("func: addLiq amt0 invalid")
			}

			if _, ok := g.CheckTickVerify(token1, token1AmtStr); !ok {
				return idx, errors.New("func: addLiq amt1 invalid")
			}

			if _, ok := CheckAmountVerify(tokenLpAmtStr, 18); !ok {
				return idx, errors.New("func: addLiq amtLp invalid")
			}

			if _, ok := CheckAmountVerify(slippage, 3); !ok {
				return idx, errors.New("func: addLiq slippage invalid")
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_REMOVE_LIQ {
			if len(f.Params) != 5+paramOffset {
				return idx, errors.New("func: removeLiq params invalid")
			}

			token0, token1 := f.Params[0], f.Params[1]
			if g.BestHeight < conf.ENABLE_SWAP_WITHDRAW_HEIGHT {
				token0, token1, err = utils.DecodeTokensFromSwapPair(f.Params[0])
				if err != nil {
					return idx, errors.New("func: removeLiq poolPair invalid")
				}
			}
			tokenLpAmtStr := f.Params[1+paramOffset]
			token0AmtStr := f.Params[2+paramOffset]
			token1AmtStr := f.Params[3+paramOffset]
			slippage := f.Params[4+paramOffset]

			if _, ok := CheckAmountVerify(tokenLpAmtStr, 18); !ok {
				return idx, errors.New(fmt.Sprintf("func: removeLiq amtLp invalid, %s/%s", token0, token1))
			}

			if _, ok := g.CheckTickVerify(token0, token0AmtStr); !ok {
				return idx, errors.New("func: removeLiq amt0 invalid")
			}

			if _, ok := g.CheckTickVerify(token1, token1AmtStr); !ok {
				return idx, errors.New("func: removeLiq amt1 invalid")
			}

			if _, ok := CheckAmountVerify(slippage, 3); !ok {
				return idx, errors.New("func: removeLiq slippage invalid")
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_SWAP {
			if len(f.Params) != 6+paramOffset {
				return idx, errors.New("func: swap params invalid")
			}

			token0, token1 := f.Params[0], f.Params[1]
			if g.BestHeight < conf.ENABLE_SWAP_WITHDRAW_HEIGHT {
				token0, token1, err = utils.DecodeTokensFromSwapPair(f.Params[0])
				if err != nil {
					return idx, errors.New("func: swap poolPair invalid")
				}
			}

			// Check that the first parameter must be one of the token pairs.
			if token := f.Params[1+paramOffset]; token != token0 && token != token1 {
				return idx, errors.New("func: swap token invalid")
			}

			derection := f.Params[3+paramOffset]
			if derection != "exactIn" && derection != "exactOut" {
				return idx, errors.New("func: swap derection invalid")
			}

			var tokenIn, tokenInAmtStr, tokenOut, tokenOutAmtStr string
			if derection == "exactIn" {
				tokenIn = f.Params[1+paramOffset]
				tokenInAmtStr = f.Params[2+paramOffset]

				if tokenIn == token0 {
					tokenOut = token1
				} else {
					tokenOut = token0
				}
				tokenOutAmtStr = f.Params[4+paramOffset]
			} else if derection == "exactOut" {
				tokenOut = f.Params[1+paramOffset]
				tokenOutAmtStr = f.Params[2+paramOffset]

				if tokenOut == token0 {
					tokenIn = token1
				} else {
					tokenIn = token0
				}
				tokenInAmtStr = f.Params[4+paramOffset]
			}

			if _, ok := g.CheckTickVerify(tokenIn, tokenInAmtStr); !ok {
				return idx, errors.New("func: swap token amount invalid")
			}

			if _, ok := g.CheckTickVerify(tokenOut, tokenOutAmtStr); !ok {
				return idx, errors.New("func: swap amt1 invalid")
			}

			slippage := f.Params[5+paramOffset]
			if _, ok := CheckAmountVerify(slippage, 3); !ok {
				return idx, errors.New("func: swap slippage invalid")
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_DECREASE_APPROVAL {
			if len(f.Params) != 2 {
				return idx, errors.New("func: decrease approval params invalid")
			}

			token := f.Params[0]
			tokenAmtStr := f.Params[1]

			if _, ok := g.CheckTickVerify(token, tokenAmtStr); !ok {
				return idx, errors.New("func: decrease approval amt invalid")
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_SEND {
			if len(f.Params) != 3 {
				return idx, errors.New("func: send params invalid")
			}

			addressTo := f.Params[0]
			if _, err := utils.GetPkScriptByAddress(addressTo, conf.GlobalNetParams); err != nil {
				return idx, errors.New("send addr invalid")
			}

			token := f.Params[1]
			tokenAmtStr := f.Params[2]
			if _, ok := g.CheckTickVerify(token, tokenAmtStr); !ok {
				return idx, errors.New("func: send amt invalid")
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_SENDLP {
			if len(f.Params) != 4 {
				return idx, errors.New("func: send params invalid")
			}

			addressTo := f.Params[0]
			if _, err := utils.GetPkScriptByAddress(addressTo, conf.GlobalNetParams); err != nil {
				return idx, errors.New("send addr invalid")
			}

			token0, token1 := f.Params[1], f.Params[2]
			tokenOrPair := GetLowerInnerPairNameByToken(token0, token1)
			tokenAmtStr := f.Params[3]
			if _, _, err := utils.DecodeTokensFromSwapPair(tokenOrPair); err != nil {
				return idx, errors.New("func: send lp invalid")
			}
			if _, ok := CheckAmountVerify(tokenAmtStr, 18); !ok {
				return idx, errors.New(fmt.Sprintf("func: send amtLp invalid, %s", tokenAmtStr))
			}

		} else {
			log.Printf("ProcessInscribeCommit commit[%d] invalid function: %s. id: %s", idx, f.Function, f.ID)
			return idx, errors.New("func invalid")
		}
	}

	return 0, nil
}

func (g *BRC20ModuleIndexer) ProcessCommitVerify(commitId string, body *model.InscriptionBRC20ModuleSwapCommitContent,
	eachFuntionSize []uint64, results []*model.SwapFunctionResultCheckState) (index int, critical bool, err error) {

	// check module exist
	moduleInfo, ok := g.ModulesInfoMap[body.Module]
	if !ok {
		return -1, true, errors.New("commit, module not exist")
	}

	// check empty parent
	if body.Parent == "" {
		if len(moduleInfo.CommitIdMap) > 0 {
			return -1, true, errors.New("commit, missing parent")
		}
	} else {
		// invalid if reusing 'parent'
		if _, ok := moduleInfo.CommitIdChainMap[body.Parent]; ok {
			return -1, true, errors.New("commit, parent already sattled")
		}

		// invalid if parent commit not exist
		if _, ok := moduleInfo.CommitIdMap[body.Parent]; !ok {
			return -1, true, errors.New("commit, parent invalid")
		}
	}

	gasPriceAmt, _ := g.CheckTickVerify(moduleInfo.GasTick, body.GasPrice)

	if len(body.Data) != len(eachFuntionSize) {
		return -1, true, errors.New("commit, function size not match data")
	}
	for idx, f := range body.Data {
		if pkScript, err := utils.GetPkScriptByAddress(f.Address, conf.GlobalNetParams); err != nil {
			return idx, true, errors.New("commit, addr invalid")
		} else {
			f.PkScript = string(pkScript)
		}

		// gas fee
		if gasPriceAmt.Sign() > 0 {
			size := eachFuntionSize[idx]
			if g.BestHeight >= conf.ENABLE_SWAP_WITHDRAW_HEIGHT {
				size = 1
			}
			gasAmt := gasPriceAmt.Mul(decimal.NewDecimal(size, 3))
			// log.Printf("process commit[%d] size: %d, gas fee: %s, module[%s]", idx, size, gasAmt.String(), body.Module)
			if err := g.ProcessCommitFunctionGasFee(moduleInfo, f.PkScript, gasAmt); err != nil { // has update
				log.Printf("process commit[%d] gas failed: %s", idx, err)
				return idx, true, err
			}
		}

		// functions
		if f.Function == constant.BRC20_SWAP_FUNCTION_DEPLOY_POOL {
			if err := g.ProcessCommitFunctionDeployPool(moduleInfo, f); err != nil {
				log.Printf("process commit[%d] deploy pool failed: %s, module[%s]", idx, err, body.Module)
				return idx, true, err
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_ADD_LIQ {
			if err := g.ProcessCommitFunctionAddLiquidity(moduleInfo, f); err != nil {
				log.Printf("process commit[%d] add liq failed: %s, module[%s]", idx, err, body.Module)
				return idx, true, err
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_REMOVE_LIQ {
			if err := g.ProcessCommitFunctionRemoveLiquidity(moduleInfo, f); err != nil {
				log.Printf("process commit[%d] remove liq failed: %s, module[%s]", idx, err, body.Module)
				return idx, true, err
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_SWAP {
			if err := g.ProcessCommitFunctionSwap(moduleInfo, f); err != nil {
				log.Printf("process commit[%d] swap failed: %s, module[%s]", idx, err, body.Module)
				return idx, true, err
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_DECREASE_APPROVAL {
			if err := g.ProcessCommitFunctionDecreaseApproval(moduleInfo, f); err != nil {
				log.Printf("process commit[%d] decrease approval failed: %s, module[%s]", idx, err, body.Module)
				return idx, true, err
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_SEND {
			if err := g.ProcessCommitFunctionSend(moduleInfo, f); err != nil {
				log.Printf("process commit[%d] send failed: %s, module[%s]", idx, err, body.Module)
				return idx, true, err
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_SENDLP {
			if err := g.ProcessCommitFunctionSendLp(moduleInfo, f); err != nil {
				log.Printf("process commit[%d] sendlp failed: %s, module[%s]", idx, err, body.Module)
				return idx, true, err
			}
		}

		// instant verify
		if len(results) == len(body.Data) {
			if err = g.BRC20ResultsPreVerify(moduleInfo, results[idx]); err != nil {
				log.Printf("commit verify failed: result[%d] %s", idx, err)
				return idx, false, err
			}
		}

		// verify test result
		if GResultsExternal == nil {
			continue
		}
		for _, result := range GResultsExternal {
			if result.CommitId != commitId {
				continue
			}

			if result.FunctionIdx != idx {
				continue
			}

			if err = g.BRC20ResultsPreVerify(moduleInfo, result); err != nil {
				log.Printf("commit verify failed: result[%d] %s", idx, err)
				return idx, false, err
			}
		}
	}

	return 0, false, nil
}

func (g *BRC20ModuleIndexer) InitCherryPickFilter(body *model.InscriptionBRC20ModuleSwapCommitContent, pickUsersPkScript, pickTokensTick, pickPoolsPair map[string]bool) (index int, err error) {
	// check module exist
	moduleInfo, ok := g.ModulesInfoMap[body.Module]
	if !ok {
		return -1, errors.New("module invalid")
	}

	pickUsersPkScript[string(moduleInfo.GasToPkScript)] = true
	pickUsersPkScript[string(moduleInfo.LpFeePkScript)] = true
	pickUsersPkScript[string(moduleInfo.SequencerPkScript)] = true
	pickUsersPkScript[string(moduleInfo.DeployerPkScript)] = true

	pickTokensTick[moduleInfo.GasTick] = true

	paramOffset := 0
	if g.BestHeight >= conf.ENABLE_SWAP_WITHDRAW_HEIGHT {
		paramOffset = 1
	}

	for idx, f := range body.Data {
		if pkScript, err := utils.GetPkScriptByAddress(f.Address, conf.GlobalNetParams); err != nil {
			return idx, errors.New("addr invalid")
		} else {
			pickUsersPkScript[string(pkScript)] = true
		}

		// function process
		if f.Function == constant.BRC20_SWAP_FUNCTION_DEPLOY_POOL {
			if len(f.Params) != 2 {
				return idx, errors.New("func: deploy params invalid")
			}
			token0 := f.Params[0]
			token1 := f.Params[1]

			// pair
			poolPair := GetLowerInnerPairNameByToken(token0, token1)
			pickPoolsPair[poolPair] = true

			// tick
			token0 = strings.ToLower(token0)
			token1 = strings.ToLower(token1)
			pickTokensTick[token0] = true
			pickTokensTick[token1] = true

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_ADD_LIQ {
			if len(f.Params) != 5+paramOffset {
				return idx, errors.New("func: addLiq params invalid")
			}

			token0, token1 := f.Params[0], f.Params[1]
			if g.BestHeight < conf.ENABLE_SWAP_WITHDRAW_HEIGHT {
				token0, token1, err = utils.DecodeTokensFromSwapPair(f.Params[0])
				if err != nil {
					return idx, errors.New("func: addLiq poolPair invalid")
				}
			}
			// pair
			poolPair := GetLowerInnerPairNameByToken(token0, token1)
			pickPoolsPair[poolPair] = true

			// tick
			token0 = strings.ToLower(token0)
			token1 = strings.ToLower(token1)
			pickTokensTick[token0] = true
			pickTokensTick[token1] = true

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_REMOVE_LIQ {
			if len(f.Params) != 5+paramOffset {
				return idx, errors.New("func: removeLiq params invalid")
			}

			token0, token1 := f.Params[0], f.Params[1]
			if g.BestHeight < conf.ENABLE_SWAP_WITHDRAW_HEIGHT {
				token0, token1, err = utils.DecodeTokensFromSwapPair(f.Params[0])
				if err != nil {
					return idx, errors.New("func: removeLiq poolPair invalid")
				}
			}
			// pair
			poolPair := GetLowerInnerPairNameByToken(token0, token1)
			pickPoolsPair[poolPair] = true

			// tick
			token0 = strings.ToLower(token0)
			token1 = strings.ToLower(token1)
			pickTokensTick[token0] = true
			pickTokensTick[token1] = true

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_SWAP {
			if len(f.Params) != 6+paramOffset {
				return idx, errors.New("func: swap params invalid")
			}

			token0, token1 := f.Params[0], f.Params[1]
			if g.BestHeight < conf.ENABLE_SWAP_WITHDRAW_HEIGHT {
				token0, token1, err = utils.DecodeTokensFromSwapPair(f.Params[0])
				if err != nil {
					return idx, errors.New("func: swap poolPair invalid")
				}
			}
			// pair
			poolPair := GetLowerInnerPairNameByToken(token0, token1)
			pickPoolsPair[poolPair] = true

			// tick
			token0 = strings.ToLower(token0)
			token1 = strings.ToLower(token1)
			pickTokensTick[token0] = true
			pickTokensTick[token1] = true

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_DECREASE_APPROVAL {
			if len(f.Params) != 2 {
				return idx, errors.New("func: decrease approval params invalid")
			}

			token0 := f.Params[0]
			token0 = strings.ToLower(token0)
			pickTokensTick[token0] = true

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_SEND {
			if len(f.Params) != 3 {
				return idx, errors.New("func: send params invalid")
			}

			addressTo := f.Params[0]
			if pk, err := utils.GetPkScriptByAddress(addressTo, conf.GlobalNetParams); err != nil {
				return idx, errors.New("send addr invalid")
			} else {
				pickUsersPkScript[string(pk)] = true
			}

			token0 := f.Params[1]
			token0 = strings.ToLower(token0)
			pickTokensTick[token0] = true

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_SENDLP {
			if len(f.Params) != 4 {
				return idx, errors.New("func: send params invalid")
			}

			addressTo := f.Params[0]
			if pk, err := utils.GetPkScriptByAddress(addressTo, conf.GlobalNetParams); err != nil {
				return idx, errors.New("send addr invalid")
			} else {
				pickUsersPkScript[string(pk)] = true
			}

			token0, token1 := f.Params[1], f.Params[2]
			poolPair := GetLowerInnerPairNameByToken(token0, token1)
			pickPoolsPair[poolPair] = true

			// tick
			token0 = strings.ToLower(token0)
			token1 = strings.ToLower(token1)
			pickTokensTick[token0] = true
			pickTokensTick[token1] = true

		} else {
			log.Printf("ProcessInscribeCommit commit[%d] invalid function: %s. id: %s", idx, f.Function, f.ID)
			return idx, errors.New("func invalid")
		}
	}

	return 0, nil
}
