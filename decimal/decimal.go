package decimal

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
)

const MAX_PRECISION = 18

var precisionFactor = new(big.Int).Exp(big.NewInt(10), big.NewInt(MAX_PRECISION), nil)

// Decimal represents a fixed-point decimal number with 18 decimal places
type Decimal struct {
	Value *big.Int
}

func NewDecimal() *Decimal {
	return &Decimal{Value: new(big.Int).SetUint64(0)}
}

func NewDecimalCopy(other *Decimal) *Decimal {
	if other == nil {
		return nil
	}
	return &Decimal{Value: new(big.Int).Set(other.Value)}
}

// NewDecimalFromString creates a Decimal instance from a string
func NewDecimalFromString(s string) (*Decimal, int, error) {
	if s == "" {
		return nil, 0, errors.New("empty string")
	}

	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return nil, 0, fmt.Errorf("invalid decimal format: %s", s)
	}

	integerPartStr := parts[0]
	if integerPartStr == "" || integerPartStr[0] == '+' {
		return nil, 0, errors.New("empty integer")
	}

	integerPart, ok := new(big.Int).SetString(parts[0], 10)
	if !ok {
		return nil, 0, fmt.Errorf("invalid integer format: %s", parts[0])
	}

	currPrecision := 0
	decimalPart := big.NewInt(0)
	if len(parts) == 2 {
		decimalPartStr := parts[1]
		if decimalPartStr == "" || decimalPartStr[0] == '-' || decimalPartStr[0] == '+' {
			return nil, 0, errors.New("empty decimal")
		}

		currPrecision = len(decimalPartStr)
		if currPrecision > MAX_PRECISION {
			return nil, 0, fmt.Errorf("decimal exceeds maximum precision: %s", s)
		}
		n := MAX_PRECISION - currPrecision
		for i := 0; i < n; i++ {
			decimalPartStr += "0"
		}
		decimalPart, ok = new(big.Int).SetString(decimalPartStr, 10)
		if !ok || decimalPart.Sign() < 0 {
			return nil, 0, fmt.Errorf("invalid decimal format: %s", parts[0])
		}
	}

	value := new(big.Int).Mul(integerPart, precisionFactor)
	if value.Sign() < 0 {
		value = value.Sub(value, decimalPart)
	} else {
		value = value.Add(value, decimalPart)
	}

	return &Decimal{Value: value}, currPrecision, nil
}

// String returns the string representation of a Decimal instance
func (d *Decimal) String() string {
	if d == nil {
		return "0"
	}
	value := new(big.Int).Abs(d.Value)
	quotient, remainder := new(big.Int).QuoRem(value, precisionFactor, new(big.Int))
	sign := ""
	if d.Value.Sign() < 0 {
		sign = "-"
	}
	if remainder.Sign() == 0 {
		return fmt.Sprintf("%s%s", sign, quotient.String())
	}
	decimalPart := fmt.Sprintf("%0*d", MAX_PRECISION, remainder)
	decimalPart = strings.TrimRight(decimalPart, "0")
	return fmt.Sprintf("%s%s.%s", sign, quotient.String(), decimalPart)
}

// Add adds two Decimal instances and returns a new Decimal instance
func (d *Decimal) Add(other *Decimal) *Decimal {
	if d == nil && other == nil {
		value := new(big.Int).SetUint64(0)
		return &Decimal{Value: value}
	}
	if other == nil {
		value := new(big.Int).Set(d.Value)
		return &Decimal{Value: value}
	}
	if d == nil {
		value := new(big.Int).Set(other.Value)
		return &Decimal{Value: value}
	}
	value := new(big.Int).Add(d.Value, other.Value)
	return &Decimal{Value: value}
}

// Sub subtracts two Decimal instances and returns a new Decimal instance
func (d *Decimal) Sub(other *Decimal) *Decimal {
	if d == nil && other == nil {
		value := new(big.Int).SetUint64(0)
		return &Decimal{Value: value}
	}
	if other == nil {
		value := new(big.Int).Set(d.Value)
		return &Decimal{Value: value}
	}
	if d == nil {
		value := new(big.Int).Neg(other.Value)
		return &Decimal{Value: value}
	}
	value := new(big.Int).Sub(d.Value, other.Value)
	return &Decimal{Value: value}
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
	value := new(big.Int).Mul(integerPart, precisionFactor)
	if d.Value.Cmp(value) > 0 {
		return true
	}
	return false
}

func (d *Decimal) Float64() float64 {
	if d == nil {
		return 0
	}
	value := new(big.Int).Abs(d.Value)
	quotient, remainder := new(big.Int).QuoRem(value, precisionFactor, new(big.Int))
	f := float64(quotient.Uint64()) + float64(remainder.Uint64())/math.MaxFloat64
	if d.Value.Sign() < 0 {
		return -f
	}
	return f
}
