package model

import (
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
)

// state of approve for each tick, (balance and history)
type BRC20ModuleConditionalApproveStateBalance struct {
	Tick            string
	BalanceDeposite *decimal.Decimal // Direct Charging Total amount

	BalanceApprove       *decimal.Decimal // Total amount of successful withdrawal matches
	BalanceNewApprove    *decimal.Decimal // Initiate total withdrawal amount
	BalanceCancelApprove *decimal.Decimal // Total amount of withdrawals cancelled
	BalanceWaitApprove   *decimal.Decimal // Total amount waiting for withdrawal matching

	// BalanceNewApprove - BalanceCancelApprove - BalanceApprove == BalanceWaitApprove
}

func (in *BRC20ModuleConditionalApproveStateBalance) DeepCopy() *BRC20ModuleConditionalApproveStateBalance {
	tb := &BRC20ModuleConditionalApproveStateBalance{
		Tick:            in.Tick,
		BalanceDeposite: decimal.NewDecimalCopy(in.BalanceDeposite),

		BalanceApprove:       decimal.NewDecimalCopy(in.BalanceApprove),
		BalanceNewApprove:    decimal.NewDecimalCopy(in.BalanceNewApprove),
		BalanceCancelApprove: decimal.NewDecimalCopy(in.BalanceCancelApprove),
		BalanceWaitApprove:   decimal.NewDecimalCopy(in.BalanceWaitApprove),
	}
	return tb
}
