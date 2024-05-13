package decimal

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
)

const MAX_PRECISION = 18

var MAX_PRECISION_STRING = "18"

var precisionFactor [19]*big.Int = [19]*big.Int{
	new(big.Int).Exp(big.NewInt(10), big.NewInt(0), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(1), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(2), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(3), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(4), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(5), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(7), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(8), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(10), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(11), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(13), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(14), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(16), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
}

// Decimal represents a fixed-point decimal number with 18 decimal places
type Decimal struct {
	Precition uint
	Value     *big.Int
}

func NewDecimal(v uint64, p uint) *Decimal {
	if p > MAX_PRECISION {
		p = MAX_PRECISION
	}
	return &Decimal{Precition: p, Value: new(big.Int).SetUint64(v)}
}

func NewDecimalCopy(other *Decimal) *Decimal {
	if other == nil {
		return nil
	}
	return &Decimal{Precition: other.Precition, Value: new(big.Int).Set(other.Value)}
}

// NewDecimalFromString creates a Decimal instance from a string
func NewDecimalFromString(s string, maxPrecision int) (*Decimal, error) {
	if s == "" {
		return nil, errors.New("empty string")
	}

	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid decimal format: %s", s)
	}

	integerPartStr := parts[0]
	if integerPartStr == "" || integerPartStr[0] == '+' {
		return nil, errors.New("empty integer")
	}

	integerPart, ok := new(big.Int).SetString(parts[0], 10)
	if !ok {
		return nil, fmt.Errorf("invalid integer format: %s", parts[0])
	}

	currPrecision := 0
	decimalPart := big.NewInt(0)
	if len(parts) == 2 {
		decimalPartStr := parts[1]
		if decimalPartStr == "" || decimalPartStr[0] == '-' || decimalPartStr[0] == '+' {
			return nil, errors.New("empty decimal")
		}

		currPrecision = len(decimalPartStr)
		if currPrecision > maxPrecision {
			return nil, fmt.Errorf("decimal exceeds maximum precision: %s", s)
		}
		n := maxPrecision - currPrecision
		for i := 0; i < n; i++ {
			decimalPartStr += "0"
		}
		decimalPart, ok = new(big.Int).SetString(decimalPartStr, 10)
		if !ok || decimalPart.Sign() < 0 {
			return nil, fmt.Errorf("invalid decimal format: %s", parts[0])
		}
	}

	value := new(big.Int).Mul(integerPart, precisionFactor[maxPrecision])
	if value.Sign() < 0 {
		value = value.Sub(value, decimalPart)
	} else {
		value = value.Add(value, decimalPart)
	}

	return &Decimal{Precition: uint(maxPrecision), Value: value}, nil
}

// String returns the string representation of a Decimal instance
func (d *Decimal) String() string {
	if d == nil {
		return "0"
	}
	value := new(big.Int).Abs(d.Value)
	quotient, remainder := new(big.Int).QuoRem(value, precisionFactor[d.Precition], new(big.Int))
	sign := ""
	if d.Value.Sign() < 0 {
		sign = "-"
	}
	if remainder.Sign() == 0 {
		return fmt.Sprintf("%s%s", sign, quotient.String())
	}
	decimalPart := fmt.Sprintf("%0*d", d.Precition, remainder)
	decimalPart = strings.TrimRight(decimalPart, "0")
	return fmt.Sprintf("%s%s.%s", sign, quotient.String(), decimalPart)
}

// Add adds two Decimal instances and returns a new Decimal instance
func (d *Decimal) Add(other *Decimal) *Decimal {
	if d == nil && other == nil {
		return nil
	}
	if other == nil {
		value := new(big.Int).Set(d.Value)
		return &Decimal{Precition: d.Precition, Value: value}
	}
	if d == nil {
		value := new(big.Int).Set(other.Value)
		return &Decimal{Precition: other.Precition, Value: value}
	}
	if d.Precition != other.Precition {
		panic("precition not match")
	}
	value := new(big.Int).Add(d.Value, other.Value)
	return &Decimal{Precition: d.Precition, Value: value}
}

// Sub subtracts two Decimal instances and returns a new Decimal instance
func (d *Decimal) Sub(other *Decimal) *Decimal {
	if d == nil && other == nil {
		return nil
	}
	if other == nil {
		value := new(big.Int).Set(d.Value)
		return &Decimal{Precition: d.Precition, Value: value}
	}
	if d == nil {
		value := new(big.Int).Neg(other.Value)
		return &Decimal{Precition: other.Precition, Value: value}
	}
	if d.Precition != other.Precition {
		panic(fmt.Sprintf("precition not match, (%d != %d)", d.Precition, other.Precition))
	}
	value := new(big.Int).Sub(d.Value, other.Value)
	return &Decimal{Precition: d.Precition, Value: value}
}

// Mul muls two Decimal instances and returns a new Decimal instance
func (d *Decimal) Mul(other *Decimal) *Decimal {
	if d == nil || other == nil {
		return nil
	}
	value := new(big.Int).Mul(d.Value, other.Value)
	// value := new(big.Int).Div(value0, precisionFactor[other.Precition])
	return &Decimal{Precition: d.Precition, Value: value}
}

// Sqrt muls two Decimal instances and returns a new Decimal instance
func (d *Decimal) Sqrt() *Decimal {
	if d == nil {
		return nil
	}
	// value0 := new(big.Int).Mul(d.Value, precisionFactor[d.Precition])
	value := new(big.Int).Sqrt(d.Value)
	return &Decimal{Precition: MAX_PRECISION, Value: value}
}

// Div divs two Decimal instances and returns a new Decimal instance
func (d *Decimal) Div(other *Decimal) *Decimal {
	if d == nil || other == nil {
		return nil
	}
	// value0 := new(big.Int).Mul(d.Value, precisionFactor[other.Precition])
	value := new(big.Int).Div(d.Value, other.Value)
	return &Decimal{Precition: d.Precition, Value: value}
}

func (d *Decimal) Cmp(other *Decimal) int {
	if d == nil && other == nil {
		return 0
	}
	if other == nil {
		return d.Value.Sign()
	}
	if d == nil {
		return -other.Value.Sign()
	}
	if d.Precition != other.Precition {
		panic(fmt.Sprintf("precition not match, (%d != %d)", d.Precition, other.Precition))
	}
	return d.Value.Cmp(other.Value)
}

func (d *Decimal) CmpAlign(other *Decimal) int {
	if d == nil && other == nil {
		return 0
	}
	if other == nil {
		return d.Value.Sign()
	}
	if d == nil {
		return -other.Value.Sign()
	}
	return d.Value.Cmp(other.Value)
}

func (d *Decimal) Sign() int {
	if d == nil {
		return 0
	}
	return d.Value.Sign()
}

func (d *Decimal) IsOverflowUint64() bool {
	if d == nil {
		return false
	}

	integerPart := new(big.Int).SetUint64(math.MaxUint64)
	value := new(big.Int).Mul(integerPart, precisionFactor[d.Precition])
	if d.Value.Cmp(value) > 0 {
		return true
	}
	return false
}

func (d *Decimal) GetMaxUint64() *Decimal {
	if d == nil {
		return nil
	}
	integerPart := new(big.Int).SetUint64(math.MaxUint64)
	value := new(big.Int).Mul(integerPart, precisionFactor[d.Precition])
	return &Decimal{Precition: d.Precition, Value: value}
}

func (d *Decimal) Float64() float64 {
	if d == nil {
		return 0
	}
	value := new(big.Int).Abs(d.Value)
	quotient, remainder := new(big.Int).QuoRem(value, precisionFactor[d.Precition], new(big.Int))
	f := float64(quotient.Uint64()) + float64(remainder.Uint64())/math.MaxFloat64
	if d.Value.Sign() < 0 {
		return -f
	}
	return f
}
