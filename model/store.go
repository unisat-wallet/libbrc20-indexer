package model

import "github.com/unisat-wallet/libbrc20-indexer/decimal"

// module state store
type BRC20ModuleSwapInfoStore struct {
	ID                string // module id
	Name              string // module name
	DeployerPkScript  string // deployer
	SequencerPkScript string // operator, sequencer
	GasToPkScript     string //
	LpFeePkScript     string //

	FeeRateSwap string
	GasTick     string

	History []*BRC20ModuleHistory // history for deploy, deposit, commit, quit

	// runtime for commit
	CommitInvalidMap map[string]struct{} // All invalid create commits
	CommitIdMap      map[string]struct{} // All valid create commits
	CommitIdChainMap map[string]struct{} // All connected commits cannot be used as parents for subsequent commits again.

	// token holders in module
	// ticker of users in module [address][tick]balanceData
	UsersTokenBalanceDataMap map[string]map[string]*BRC20ModuleTokenBalance

	// swap
	// lp token balance of address in module [pool][address]balance
	LPTokenUsersBalanceMap map[string]map[string]*decimal.Decimal

	// swap total balance
	// total balance of pool in module [pool]balanceData
	SwapPoolTotalBalanceDataMap map[string]*BRC20ModulePoolTotalBalance

	// module deposit/withdraw state [tick]balanceData
	ConditionalApproveStateBalanceDataMap map[string]*BRC20ModuleConditionalApproveStateBalance
}
