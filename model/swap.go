package model

import (
	"fmt"
	"log"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/decimal"
)

// decode data
type InscriptionBRC20ModuleDeploySwapContent struct {
	Proto     string            `json:"p,omitempty"`
	Operation string            `json:"op,omitempty"`
	Name      string            `json:"name,omitempty"`
	Source    string            `json:"source,omitempty"`
	Init      map[string]string `json:"init,omitempty"`
}

type InscriptionBRC20ModuleSwapApproveContent struct {
	Proto     string `json:"p,omitempty"`
	Operation string `json:"op,omitempty"`
	Module    string `json:"module,omitempty"`
	Tick      string `json:"tick,omitempty"`
	Amount    string `json:"amt,omitempty"`
}

type InscriptionBRC20ModuleSwapQuitContent struct {
	Proto     string `json:"p,omitempty"`
	Operation string `json:"op,omitempty"`
	Module    string `json:"module,omitempty"`
}

// check state
type SwapFunctionResultCheckStateForUser struct {
	Address string `json:"address,omitempty"`
	Tick    string `json:"tick,omitempty"`
	Balance string `json:"balance,omitempty"`
}

type SwapFunctionResultCheckStateForPool struct {
	Pair           string `json:"pair,omitempty"`
	ReserveAmount0 string `json:"reserve0,omitempty"`
	ReserveAmount1 string `json:"reserve1,omitempty"`
	LPAmount       string `json:"lp,omitempty"`
}

type SwapFunctionResultCheckState struct {
	Users       []SwapFunctionResultCheckStateForUser `json:"users,omitempty"`
	Pools       []SwapFunctionResultCheckStateForPool `json:"pools,omitempty"`
	CommitId    string                                `json:"commit,omitempty"`
	FunctionIdx int                                   `json:"function,omitempty"`
}

// load events
type BRC20ModuleHistoryInfoEvent struct {
	Type  string `json:"type"` // inscribe-deploy/inscribe-mint/inscribe-transfer/transfer/send/receive
	Valid bool   `json:"valid"`

	TxIdHex           string `json:"txid"`
	Idx               uint32 `json:"idx"` // inscription index
	Vout              uint32 `json:"vout"`
	Offset            uint64 `json:"offset"`
	InscriptionNumber int64  `json:"inscriptionNumber"`
	InscriptionId     string `json:"inscriptionId"`

	ContentType string `json:"contentType"`
	ContentBody string `json:"contentBody"`

	AddressFrom string `json:"from"`
	AddressTo   string `json:"to"`
	Satoshi     uint64 `json:"satoshi"`

	Data *BRC20SwapHistoryCondApproveData `json:"data"`

	Height       uint32 `json:"height"`
	TxIdx        uint32 `json:"txidx"` // txidx in block
	BlockHashHex string `json:"blockhash"`
	BlockTime    uint32 `json:"blocktime"`
}

// commit function data
type SwapFunctionData struct {
	Address   string   `json:"addr,omitempty"`
	Function  string   `json:"func,omitempty"`
	Params    []string `json:"params,omitempty"`
	Timestamp uint     `json:"ts,omitempty"`
	Signature string   `json:"sig,omitempty"`

	ID       string `json:"-"`
	PkScript string `json:"-"`
}

type InscriptionBRC20ModuleSwapCommitContent struct {
	Proto     string `json:"p,omitempty"`
	Operation string `json:"op,omitempty"`
	Module    string `json:"module,omitempty"`

	Parent   string              `json:"parent,omitempty"`
	GasPrice string              `json:"gas_price,omitempty"`
	Data     []*SwapFunctionData `json:"data,omitempty"`
}

type TransferStateForConditionalApprove struct {
	Tick    string
	From    string
	To      string
	Balance *decimal.Decimal

	InscriptionId string
	Max           string // origin amount
}

func (d *TransferStateForConditionalApprove) DeepCopy() (copy *TransferStateForConditionalApprove) {
	copy = &TransferStateForConditionalApprove{
		Tick:          d.Tick,
		From:          d.From,
		To:            d.To,
		Balance:       decimal.NewDecimalCopy(d.Balance), // maybe no need copy
		InscriptionId: d.InscriptionId,
		Max:           d.Max,
	}
	return copy
}

type ApproveStateForConditionalApprove struct {
	Module string
	Tick   string

	Owner   string
	Balance *decimal.Decimal

	ApproveInfo *InscriptionBRC20SwapConditionalApproveInfo
	ToData      *InscriptionBRC20Data
}

func (d *ApproveStateForConditionalApprove) DeepCopy() (copy *ApproveStateForConditionalApprove) {
	copy = &ApproveStateForConditionalApprove{
		Module: d.Module,
		Tick:   d.Tick,

		Owner:   d.Owner,
		Balance: decimal.NewDecimalCopy(d.Balance), // maybe no need copy

		ApproveInfo: d.ApproveInfo.DeepCopy(),
		ToData:      d.ToData,
	}
	return copy
}

type ConditionalApproveEvent struct {
	Module string
	Tick   string

	TransferInscriptionId string
	TransferMax           string

	From    string
	To      string
	Amount  *decimal.Decimal
	Balance *decimal.Decimal

	FromData InscriptionBRC20Data
	ToData   InscriptionBRC20Data

	ApproveInfo *InscriptionBRC20SwapConditionalApproveInfo
}

func NewConditionalApproveEvent(senderPkScript, receiverPkScript string, amount, balance *decimal.Decimal,
	data *InscriptionBRC20Data, approveInfo *InscriptionBRC20SwapConditionalApproveInfo,
	transferInscriptionId, transferMax string) (event *ConditionalApproveEvent) {

	fromData := *approveInfo.Data
	fromData.PkScript = senderPkScript
	toData := *data
	toData.PkScript = receiverPkScript

	// 一开始直接退回
	return &ConditionalApproveEvent{
		Module:                approveInfo.Module,
		Tick:                  approveInfo.Tick,
		TransferInscriptionId: transferInscriptionId,
		TransferMax:           transferMax,
		From:                  senderPkScript,
		To:                    receiverPkScript,
		Amount:                amount,
		Balance:               balance,

		FromData: fromData,
		ToData:   toData,

		ApproveInfo: approveInfo,
	}
}

// module state
type BRC20ModuleSwapInfo struct {
	UpdateHeight uint32

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
	// token balance of address in module [tick][address]balanceData
	TokenUsersBalanceDataMap map[string]map[string]*BRC20ModuleTokenBalance

	// swap
	// lp token balance of address in module [pool][address]balance
	LPTokenUsersBalanceMap        map[string]map[string]*decimal.Decimal
	LPTokenUsersBalanceUpdatedMap map[string]struct{} // set if update

	// lp token of users in module [address][pool]balance
	UsersLPTokenBalanceMap map[string]map[string]*decimal.Decimal

	// swap total balance
	// total balance of pool in module [pool]balanceData
	SwapPoolTotalBalanceDataMap map[string]*BRC20ModulePoolTotalBalance

	// module deposit/withdraw state [tick]balanceData
	ConditionalApproveStateBalanceDataMap map[string]*BRC20ModuleConditionalApproveStateBalance
	// runtime for approve
	ThisTxId                            string
	TransferStatesForConditionalApprove []*TransferStateForConditionalApprove
	ApproveStatesForConditionalApprove  []*ApproveStateForConditionalApprove
}

func (m *BRC20ModuleSwapInfo) DeepCopy() (copy *BRC20ModuleSwapInfo) {
	copy = &BRC20ModuleSwapInfo{
		ID:                m.ID,
		Name:              m.Name,
		DeployerPkScript:  m.DeployerPkScript,  // deployer
		SequencerPkScript: m.SequencerPkScript, // Sequencer
		GasToPkScript:     m.GasToPkScript,
		LpFeePkScript:     m.LpFeePkScript,

		FeeRateSwap: m.FeeRateSwap,
		GasTick:     m.GasTick,

		History: make([]*BRC20ModuleHistory, 0),

		// runtime for commit
		CommitInvalidMap: make(map[string]struct{}, 0),
		CommitIdChainMap: make(map[string]struct{}, 0),
		CommitIdMap:      make(map[string]struct{}, 0),

		// runtime for holders
		// token holders in module
		// ticker of users in module [address][tick]balanceData
		UsersTokenBalanceDataMap: make(map[string]map[string]*BRC20ModuleTokenBalance, 0),

		// token balance of address in module [tick][address]balanceData
		TokenUsersBalanceDataMap: make(map[string]map[string]*BRC20ModuleTokenBalance, 0),

		// swap
		// lp token balance of address in module [pair][address]balance
		LPTokenUsersBalanceMap:        make(map[string]map[string]*decimal.Decimal, 0),
		LPTokenUsersBalanceUpdatedMap: make(map[string]struct{}, 0),

		// lp token of users in module [address][pair]balance
		UsersLPTokenBalanceMap: make(map[string]map[string]*decimal.Decimal, 0),

		// swap total balance
		// total balance of pool in module [pair]balanceData
		SwapPoolTotalBalanceDataMap: make(map[string]*BRC20ModulePoolTotalBalance, 0),

		ConditionalApproveStateBalanceDataMap: make(map[string]*BRC20ModuleConditionalApproveStateBalance, 0),
	}

	for _, h := range m.History {
		copy.History = append(copy.History, h)
		// fix: more history
	}

	// invalid commit
	for k := range m.CommitInvalidMap {
		copy.CommitInvalidMap[k] = struct{}{}
	}

	for k := range m.CommitIdChainMap {
		copy.CommitIdChainMap[k] = struct{}{}
	}
	for k := range m.CommitIdMap {
		copy.CommitIdMap[k] = struct{}{}
	}

	// user/tick: balance
	for address, dataMap := range m.UsersTokenBalanceDataMap {
		dataMapCopy := make(map[string]*BRC20ModuleTokenBalance, 0)
		for tick, balance := range dataMap {
			dataMapCopy[tick] = balance.DeepCopy()
		}
		copy.UsersTokenBalanceDataMap[address] = dataMapCopy
	}
	// tick/user: balance
	for tick, dataMap := range m.TokenUsersBalanceDataMap {
		dataMapCopy := make(map[string]*BRC20ModuleTokenBalance, 0)
		for address := range dataMap {
			dataMapCopy[address] = copy.UsersTokenBalanceDataMap[address][tick]
		}
		copy.TokenUsersBalanceDataMap[tick] = dataMapCopy
	}

	// user/pair: lpbalance
	for address, dataMap := range m.UsersLPTokenBalanceMap {
		dataMapCopy := make(map[string]*decimal.Decimal, 0)
		for pair, balance := range dataMap {
			dataMapCopy[pair] = decimal.NewDecimalCopy(balance)
		}
		copy.UsersLPTokenBalanceMap[address] = dataMapCopy
	}
	// pair/user: lpbalance
	for pair, dataMap := range m.LPTokenUsersBalanceMap {
		dataMapCopy := make(map[string]*decimal.Decimal, 0)
		for address := range dataMap {
			dataMapCopy[address] = copy.UsersLPTokenBalanceMap[address][pair]
		}
		copy.LPTokenUsersBalanceMap[pair] = dataMapCopy
	}

	// swap total balance
	for pair, balance := range m.SwapPoolTotalBalanceDataMap {
		copy.SwapPoolTotalBalanceDataMap[pair] = balance.DeepCopy()
	}

	// swap deposit/approve state balance
	for tick, balance := range m.ConditionalApproveStateBalanceDataMap {
		copy.ConditionalApproveStateBalanceDataMap[tick] = balance.DeepCopy()
	}

	// runtime for approve
	copy.ThisTxId = m.ThisTxId
	for _, v := range m.TransferStatesForConditionalApprove {
		copy.TransferStatesForConditionalApprove = append(copy.TransferStatesForConditionalApprove, v.DeepCopy())
	}
	for _, v := range m.ApproveStatesForConditionalApprove {
		copy.ApproveStatesForConditionalApprove = append(copy.ApproveStatesForConditionalApprove, v.DeepCopy())
	}
	return copy
}

func (m *BRC20ModuleSwapInfo) CherryPick(pickUsersPkScript, pickTokensTick, pickPoolsPair map[string]bool) (copy *BRC20ModuleSwapInfo) {
	copy = &BRC20ModuleSwapInfo{
		ID:                m.ID,
		Name:              m.Name,
		DeployerPkScript:  m.DeployerPkScript,  // deployer
		SequencerPkScript: m.SequencerPkScript, // Sequencer
		GasToPkScript:     m.GasToPkScript,
		LpFeePkScript:     m.LpFeePkScript,

		FeeRateSwap: m.FeeRateSwap,
		GasTick:     m.GasTick,

		// runtime for commit
		CommitInvalidMap: make(map[string]struct{}, 0),
		CommitIdChainMap: make(map[string]struct{}, 0),
		CommitIdMap:      make(map[string]struct{}, 0),

		// runtime for holders
		// token holders in module
		// ticker of users in module [address][tick]balanceData
		UsersTokenBalanceDataMap: make(map[string]map[string]*BRC20ModuleTokenBalance, 0),

		// token balance of address in module [tick][address]balanceData
		TokenUsersBalanceDataMap: make(map[string]map[string]*BRC20ModuleTokenBalance, 0),

		// swap
		// lp token balance of address in module [pair][address]balance
		LPTokenUsersBalanceMap:        make(map[string]map[string]*decimal.Decimal, 0),
		LPTokenUsersBalanceUpdatedMap: make(map[string]struct{}, 0),

		// lp token of users in module [address][pair]balance
		UsersLPTokenBalanceMap: make(map[string]map[string]*decimal.Decimal, 0),

		// swap total balance
		// total balance of pool in module [pair]balanceData
		SwapPoolTotalBalanceDataMap: make(map[string]*BRC20ModulePoolTotalBalance, 0),

		ConditionalApproveStateBalanceDataMap: make(map[string]*BRC20ModuleConditionalApproveStateBalance, 0),
	}

	for k := range m.CommitIdChainMap {
		copy.CommitIdChainMap[k] = struct{}{}
	}
	for k := range m.CommitIdMap {
		copy.CommitIdMap[k] = struct{}{}
	}

	// user/tick: balance
	for address, dataMap := range m.UsersTokenBalanceDataMap {
		dataMapCopy := make(map[string]*BRC20ModuleTokenBalance, 0)
		for tick, balance := range dataMap {
			dataMapCopy[tick] = balance.CherryPick()
		}
		copy.UsersTokenBalanceDataMap[address] = dataMapCopy
	}
	// tick/user: balance
	for tick, dataMap := range m.TokenUsersBalanceDataMap {
		dataMapCopy := make(map[string]*BRC20ModuleTokenBalance, 0)
		for address := range dataMap {
			dataMapCopy[address] = copy.UsersTokenBalanceDataMap[address][tick]
		}
		copy.TokenUsersBalanceDataMap[tick] = dataMapCopy
	}

	// user/pair: lpbalance
	for address, dataMap := range m.UsersLPTokenBalanceMap {
		dataMapCopy := make(map[string]*decimal.Decimal, 0)
		for pair, balance := range dataMap {
			dataMapCopy[pair] = decimal.NewDecimalCopy(balance)
		}
		copy.UsersLPTokenBalanceMap[address] = dataMapCopy
	}
	// pair/user: lpbalance
	for pair, dataMap := range m.LPTokenUsersBalanceMap {
		dataMapCopy := make(map[string]*decimal.Decimal, 0)
		for address := range dataMap {
			dataMapCopy[address] = copy.UsersLPTokenBalanceMap[address][pair]
		}
		copy.LPTokenUsersBalanceMap[pair] = dataMapCopy
	}

	// swap total balance
	for pair, balance := range m.SwapPoolTotalBalanceDataMap {
		copy.SwapPoolTotalBalanceDataMap[pair] = balance.CherryPick()
	}

	// swap deposit/approve state balance
	// no need
	return copy
}

func (moduleInfo *BRC20ModuleSwapInfo) GetTickConditionalApproveStateBalance(ticker string) (tokenBalance *BRC20ModuleConditionalApproveStateBalance) {
	uniqueLowerTicker := strings.ToLower(ticker)

	stateBalance, ok := moduleInfo.ConditionalApproveStateBalanceDataMap[uniqueLowerTicker]
	if !ok {
		stateBalance = &BRC20ModuleConditionalApproveStateBalance{Tick: ticker}
		moduleInfo.ConditionalApproveStateBalanceDataMap[uniqueLowerTicker] = stateBalance
	}
	return stateBalance
}

func (moduleInfo *BRC20ModuleSwapInfo) GetUserTokenBalance(ticker, userPkScript string) (tokenBalance *BRC20ModuleTokenBalance) {
	uniqueLowerTicker := strings.ToLower(ticker)
	// get user's tokens to update
	var usersTokens map[string]*BRC20ModuleTokenBalance
	if tokens, ok := moduleInfo.UsersTokenBalanceDataMap[userPkScript]; !ok {
		usersTokens = make(map[string]*BRC20ModuleTokenBalance, 0)
		moduleInfo.UsersTokenBalanceDataMap[userPkScript] = usersTokens
	} else {
		usersTokens = tokens
	}
	// get tokenBalance to update
	if tb, ok := usersTokens[uniqueLowerTicker]; !ok {
		tokenBalance = &BRC20ModuleTokenBalance{Tick: ticker, PkScript: userPkScript}
		usersTokens[uniqueLowerTicker] = tokenBalance
	} else {
		tokenBalance = tb
		return tokenBalance // fixme: may missing tokenUsers
	}

	// set token's users
	tokenUsers, ok := moduleInfo.TokenUsersBalanceDataMap[uniqueLowerTicker]
	if !ok {
		tokenUsers = make(map[string]*BRC20ModuleTokenBalance, 0)
		moduleInfo.TokenUsersBalanceDataMap[uniqueLowerTicker] = tokenUsers
	}
	tokenUsers[userPkScript] = tokenBalance
	return tokenBalance
}

// Generate matching approve events within the transaction when a transfer inscription transfer event occurs.
func (moduleInfo *BRC20ModuleSwapInfo) GenerateApproveEventsByTransfer(transState *TransferStateForConditionalApprove) (events []*ConditionalApproveEvent) {

	balanceAmt := transState.Balance
	for _, apprState := range moduleInfo.ApproveStatesForConditionalApprove {
		// skip tick miss match
		if apprState.Tick != transState.Tick {
			continue
		}
		// skip to other user
		if apprState.Owner != transState.To {
			continue
		}
		// skip 0 approve balance
		if apprState.Balance.Sign() == 0 {
			continue
		}
		// skip 0 transfer balance
		if balanceAmt.Sign() == 0 {
			break
		}

		if apprState.Balance.Cmp(balanceAmt) <= 0 {
			balanceAmt = balanceAmt.Sub(apprState.Balance)

			senderPkScript := apprState.Owner
			receiverPkScript := transState.From
			event := NewConditionalApproveEvent(senderPkScript, receiverPkScript, apprState.Balance, nil, apprState.ToData, apprState.ApproveInfo, transState.InscriptionId, transState.Max)
			events = append(events, event)
			log.Printf("generate new approve event by transfer. rest match. id: %s", transState.InscriptionId)
			log.Printf("generate new approve event. amt: %s", apprState.Balance.String())
			apprState.Balance = nil
			continue
		} else {
			apprState.Balance = apprState.Balance.Sub(balanceAmt)

			senderPkScript := apprState.Owner
			receiverPkScript := transState.From
			event := NewConditionalApproveEvent(senderPkScript, receiverPkScript, balanceAmt, apprState.Balance, apprState.ToData, apprState.ApproveInfo, transState.InscriptionId, transState.Max)
			events = append(events, event)
			log.Printf("generate new approve event by transfer. partial match. id: %s", transState.InscriptionId)
			log.Printf("generate new approve event. amt: %s", balanceAmt.String())
			balanceAmt = nil
			break
		}
	}
	if balanceAmt.Sign() > 0 {
		transState.Balance = balanceAmt
		moduleInfo.TransferStatesForConditionalApprove = append(moduleInfo.TransferStatesForConditionalApprove, transState)
	}

	return events
}

// Generate a matching approve event within the transaction when an approve inscription transfer event occurs.
func (moduleInfo *BRC20ModuleSwapInfo) GenerateApproveEventsByApprove(owner string, balance *decimal.Decimal,
	data *InscriptionBRC20Data, approveInfo *InscriptionBRC20SwapConditionalApproveInfo) (events []*ConditionalApproveEvent) {

	balanceAmt := decimal.NewDecimalCopy(balance)
	apprState := &ApproveStateForConditionalApprove{
		Tick:    approveInfo.Tick,
		Owner:   owner,
		Balance: balanceAmt,

		// fixme: object
		Module:      approveInfo.Module,
		ApproveInfo: approveInfo,
		ToData:      data,
	}

	for _, transState := range moduleInfo.TransferStatesForConditionalApprove {
		// approve balance
		if balanceAmt.Sign() == 0 {
			break
		}

		if transState.Tick != apprState.Tick {
			continue
		}
		if transState.To != apprState.Owner {
			continue
		}

		if transState.Balance.Sign() == 0 {
			continue
		}

		if transState.Balance.Cmp(balanceAmt) <= 0 {
			balanceAmt = balanceAmt.Sub(transState.Balance)

			senderPkScript := owner
			receiverPkScript := transState.From
			event := NewConditionalApproveEvent(senderPkScript, receiverPkScript, transState.Balance, balanceAmt, data, approveInfo, transState.InscriptionId, transState.Max)
			events = append(events, event)
			log.Printf("generate new approve event by cond. rest match. id: %s", transState.InscriptionId)
			log.Printf("generate new approve event. amt: %s", transState.Balance.String())
			transState.Balance = nil
			continue
		} else {
			transState.Balance = transState.Balance.Sub(balanceAmt)

			senderPkScript := owner
			receiverPkScript := transState.From
			event := NewConditionalApproveEvent(senderPkScript, receiverPkScript, balanceAmt, nil, data, approveInfo, transState.InscriptionId, transState.Max)

			events = append(events, event)
			log.Printf("generate new approve event by cond. partial match. id: %s", transState.InscriptionId)
			log.Printf("generate new approve event. amt: %s", balanceAmt.String())
			balanceAmt = nil
			break
		}
	}
	// Remaining approve, written into state waiting for subsequent deduction
	if balanceAmt.Sign() > 0 {
		apprState.Balance = balanceAmt
		moduleInfo.ApproveStatesForConditionalApprove = append(moduleInfo.ApproveStatesForConditionalApprove, apprState)
	}

	return events
}

// state of address for each tick, (balance and history)
type BRC20ModuleTokenBalance struct {
	UpdateHeight uint32

	Tick     string
	PkScript string

	// confirmed safe
	SwapAccountBalanceSafe   *decimal.Decimal
	ModuleAccountBalanceSafe *decimal.Decimal

	// with unconfirmed balance
	SwapAccountBalance     *decimal.Decimal
	AvailableBalanceSafe   *decimal.Decimal
	AvailableBalance       *decimal.Decimal
	ApproveableBalance     *decimal.Decimal
	CondApproveableBalance *decimal.Decimal
	ReadyToWithdrawAmount  *decimal.Decimal

	ValidConditionalApproveMap map[string]*InscriptionBRC20Data
	ValidApproveMap            map[string]*InscriptionBRC20Data
	ReadyToWithdrawMap         map[string]*InscriptionBRC20Data // ready to use, but inscription may invalid(depends on available b)

	History []*BRC20ModuleHistory
}

func (b *BRC20ModuleTokenBalance) String() string {
	return fmt.Sprintf("%s", b.SwapAccountBalance.String())
}

func (bal *BRC20ModuleTokenBalance) ModuleBalance() *decimal.Decimal {
	return bal.AvailableBalance.Add(
		bal.ApproveableBalance).Add(
		bal.CondApproveableBalance)
}

func (in *BRC20ModuleTokenBalance) DeepCopy() *BRC20ModuleTokenBalance {
	tb := &BRC20ModuleTokenBalance{
		Tick:     in.Tick,
		PkScript: in.PkScript,

		SwapAccountBalanceSafe:   decimal.NewDecimalCopy(in.SwapAccountBalanceSafe),
		ModuleAccountBalanceSafe: decimal.NewDecimalCopy(in.ModuleAccountBalanceSafe),

		SwapAccountBalance: decimal.NewDecimalCopy(in.SwapAccountBalance),

		AvailableBalanceSafe: decimal.NewDecimalCopy(in.AvailableBalanceSafe),
		AvailableBalance:     decimal.NewDecimalCopy(in.AvailableBalance),

		ApproveableBalance:     decimal.NewDecimalCopy(in.ApproveableBalance),
		CondApproveableBalance: decimal.NewDecimalCopy(in.CondApproveableBalance),
		ReadyToWithdrawAmount:  decimal.NewDecimalCopy(in.ReadyToWithdrawAmount),

		ValidConditionalApproveMap: make(map[string]*InscriptionBRC20Data, len(in.ValidConditionalApproveMap)),
		ValidApproveMap:            make(map[string]*InscriptionBRC20Data, len(in.ValidApproveMap)),
		ReadyToWithdrawMap:         make(map[string]*InscriptionBRC20Data, len(in.ReadyToWithdrawMap)),
	}

	for k, v := range in.ValidConditionalApproveMap {
		data := *v
		tb.ValidConditionalApproveMap[k] = &data
	}
	for k, v := range in.ValidApproveMap {
		data := *v
		tb.ValidApproveMap[k] = &data
	}
	for k, v := range in.ReadyToWithdrawMap {
		data := *v
		tb.ReadyToWithdrawMap[k] = &data
	}

	for _, h := range in.History {
		tb.History = append(tb.History, h)
		// fix: more history
	}
	// tb.History = make([]BRC20History, len(in.History))
	// copy(tb.History, in.History)
	return tb
}

func (in *BRC20ModuleTokenBalance) CherryPick() *BRC20ModuleTokenBalance {
	tb := &BRC20ModuleTokenBalance{
		Tick:     in.Tick,
		PkScript: in.PkScript,

		SwapAccountBalanceSafe:   decimal.NewDecimalCopy(in.SwapAccountBalanceSafe),
		ModuleAccountBalanceSafe: decimal.NewDecimalCopy(in.ModuleAccountBalanceSafe),

		SwapAccountBalance: decimal.NewDecimalCopy(in.SwapAccountBalance),

		AvailableBalanceSafe: decimal.NewDecimalCopy(in.AvailableBalanceSafe),
		AvailableBalance:     decimal.NewDecimalCopy(in.AvailableBalance),

		ApproveableBalance:     decimal.NewDecimalCopy(in.ApproveableBalance),
		CondApproveableBalance: decimal.NewDecimalCopy(in.CondApproveableBalance),
		ReadyToWithdrawAmount:  decimal.NewDecimalCopy(in.ReadyToWithdrawAmount),
	}
	return tb
}

// state of address for each tick, (balance and history)
type BRC20ModulePoolTotalBalance struct {
	UpdateHeight uint32

	Tick        [2]string
	TickBalance [2]*decimal.Decimal
	LpBalance   *decimal.Decimal
	LastRootK   *decimal.Decimal

	// history
	History []*BRC20ModuleHistory
}

func (in *BRC20ModulePoolTotalBalance) DeepCopy() *BRC20ModulePoolTotalBalance {
	tb := &BRC20ModulePoolTotalBalance{
		Tick:        in.Tick,
		TickBalance: in.TickBalance,

		LpBalance: decimal.NewDecimalCopy(in.LpBalance),
		LastRootK: decimal.NewDecimalCopy(in.LastRootK),
	}

	for _, h := range in.History {
		tb.History = append(tb.History, h)
		// fix: more history
	}
	return tb
}

func (in *BRC20ModulePoolTotalBalance) CherryPick() *BRC20ModulePoolTotalBalance {
	tb := &BRC20ModulePoolTotalBalance{
		Tick:        in.Tick,
		TickBalance: in.TickBalance,

		LpBalance: decimal.NewDecimalCopy(in.LpBalance),
		LastRootK: decimal.NewDecimalCopy(in.LastRootK),
	}
	return tb
}

type InscriptionBRC20SwapInfo struct {
	Module string
	Tick   string
	Amount *decimal.Decimal
	Data   *InscriptionBRC20Data
}

type InscriptionBRC20SwapConditionalApproveInfo struct {
	UpdateHeight uint32

	Module            string
	Tick              string
	Amount            *decimal.Decimal // current amt
	Balance           *decimal.Decimal // current balance
	HasMoved          bool             // has moved
	OwnerPkScript     string           // owner
	DelegatorPkScript string           // delegator
	Data              *InscriptionBRC20Data
}

func (d *InscriptionBRC20SwapConditionalApproveInfo) DeepCopy() (copy *InscriptionBRC20SwapConditionalApproveInfo) {
	copy = &InscriptionBRC20SwapConditionalApproveInfo{
		Module:            d.Module,
		Tick:              d.Tick,
		Amount:            decimal.NewDecimalCopy(d.Amount),  // maybe no need copy
		Balance:           decimal.NewDecimalCopy(d.Balance), // maybe no need copy
		HasMoved:          d.HasMoved,
		OwnerPkScript:     d.OwnerPkScript,
		DelegatorPkScript: d.DelegatorPkScript,
		Data:              d.Data,
	}
	return copy
}

// history inscription info
type InscriptionBRC20SwapInfoResp struct {
	ContentBody       []byte `json:"content"`
	InscriptionNumber int64  `json:"inscriptionNumber"`
	InscriptionId     string `json:"inscriptionId"`

	Height        uint32 `json:"-"`
	Confirmations int    `json:"confirmations"`
}
