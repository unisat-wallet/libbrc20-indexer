package decimal_test

import (
	"testing"

	"github.com/unisat-wallet/libbrc20-indexer/decimal"
)

func TestNewDecimalFromString(t *testing.T) {
	testCases := []struct {
		input string
		want  string
		err   bool
	}{
		{"123456789.123456789", "123456789.123456789", false},
		{"123456789.123", "123456789.123", false},
		{"123456789", "123456789", false},
		{"-123456789.123456789", "-123456789.123456789", false},
		{"-123456789.123", "-123456789.123", false},
		{"-123456789", "-123456789", false},
		{"", "", true},
		{" ", "", true},
		{".", "", true},
		{" 123.456", "", true},
		{".456", "", true},
		{".456 ", "", true},
		{" .456 ", "", true},
		{" 456", "", true},
		{"456 ", "", true},
		{"45 6", "", true},
		{"123. 456", "", true},
		{"123.-456", "", true},
		{"123.+456", "", true},
		{"+123.456", "", true},
		{"123.456.789", "", true},
		{"123456789.", "123456789", true},
		{"123456789.12345678901234567891", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			got, _, err := decimal.NewDecimalFromString(tc.input)
			if (err != nil) != tc.err {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && got.String() != tc.want {
				t.Errorf("got %s, want %s", got.String(), tc.want)
			}
		})
	}
}

func TestDecimal_Add(t *testing.T) {
	testCases := []struct {
		a, b string
		want string
	}{
		{"123456789.123456789", "987654321.987654321", "1111111111.11111111"},
		{"123456789.123", "987654321.987", "1111111111.11"},
		{"123456789", "987654321", "1111111110"},
		{"-123456789.123456789", "987654321.987654321", "864197532.864197532"},
		{"-123456789.123", "987654321.987", "864197532.864"},
		{"-123456789", "987654321", "864197532"},
	}

	for _, tc := range testCases {
		t.Run(tc.a+"+"+tc.b, func(t *testing.T) {
			da, _, _ := decimal.NewDecimalFromString(tc.a)
			db, _, _ := decimal.NewDecimalFromString(tc.b)
			got := da.Add(db)
			if got.String() != tc.want {
				t.Errorf("got %s, want %s", got.String(), tc.want)
			}
		})
	}
}

func TestDecimal_Sub(t *testing.T) {
	testCases := []struct {
		a, b string
		want string
	}{
		{"123456789.123456789", "987654321.987654321", "-864197532.864197532"},
		{"123456789.123", "987654321.987", "-864197532.864"},
		{"123456789", "987654321", "-864197532"},
		{"-123456789.123456789", "987654321.987654321", "-1111111111.11111111"},
		{"-123456789.123", "987654321.987", "-1111111111.11"},
		{"-123456789", "987654321", "-1111111110"},
	}

	for _, tc := range testCases {
		t.Run(tc.a+"-"+tc.b, func(t *testing.T) {
			da, _, _ := decimal.NewDecimalFromString(tc.a)
			db, _, _ := decimal.NewDecimalFromString(tc.b)
			got := da.Sub(db)
			if got.String() != tc.want {
				t.Errorf("got %s, want %s", got.String(), tc.want)
			}
		})
	}

}

func TestDecimal_String(t *testing.T) {
	testCases := []struct {
		input string
		want  string
	}{
		{"123456789.123456789", "123456789.123456789"},
		{"123456789", "123456789"},
		{"-987654321.987654321", "-987654321.987654321"},
		{"0.123456789", "0.123456789"},
		{"0.123", "0.123"},
		{"123456789", "123456789"},
		{"-123456789", "-123456789"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			d, _, _ := decimal.NewDecimalFromString(tc.input)
			got := d.String()
			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func BenchmarkAdd(b *testing.B) {
	d1, _, _ := decimal.NewDecimalFromString("123456789.123456789")
	d2, _, _ := decimal.NewDecimalFromString("987654321.987654321")
	for n := 0; n < b.N; n++ {
		d1.Add(d2)
	}
}

func BenchmarkSub(b *testing.B) {
	d1, _, _ := decimal.NewDecimalFromString("123456789.123456789")
	d2, _, _ := decimal.NewDecimalFromString("987654321.987654321")
	for n := 0; n < b.N; n++ {
		d1.Sub(d2)
	}
}
