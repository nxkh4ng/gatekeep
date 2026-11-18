package core

import (
	"errors"
	"strings"
	"testing"
)

// countIn returns how many bytes of password appear in chars.
func countIn(password, chars string) int {
	n := 0
	for i := range len(password) {
		if strings.IndexByte(chars, password[i]) >= 0 {
			n++
		}
	}
	return n
}

func isLetter(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

func TestGenerateDefault(t *testing.T) {
	p := DefaultPolicy()
	allowed := lowerChars + upperChars + digitChars + DefaultSymbols
	for range 200 {
		password, err := Generate(p)
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if len(password) != p.Length {
			t.Fatalf("len(%q) = %d, want %d", password, len(password), p.Length)
		}
		for _, class := range []string{lowerChars, upperChars, digitChars, DefaultSymbols} {
			if countIn(password, class) < 1 {
				t.Fatalf("password %q misses a character from %q", password, class)
			}
		}
		if countIn(password, allowed) != len(password) {
			t.Fatalf("password %q contains an unexpected character", password)
		}
	}
}

func TestGenerateAlnum(t *testing.T) {
	p := Policy{
		Length:    12,
		Classes:   LowerChars | UpperChars | DigitChars,
		MinLower:  1,
		MinUpper:  1,
		MinDigits: 2,
	}
	allowed := lowerChars + upperChars + digitChars
	for range 100 {
		password, err := Generate(p)
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if len(password) != p.Length {
			t.Fatalf("len(%q) = %d, want %d", password, len(password), p.Length)
		}
		if countIn(password, digitChars) < p.MinDigits {
			t.Fatalf("password %q has fewer than %d digits", password, p.MinDigits)
		}
		if countIn(password, DefaultSymbols) != 0 {
			t.Fatalf("password %q contains a symbol", password)
		}
		if countIn(password, allowed) != len(password) {
			t.Fatalf("password %q contains an unexpected character", password)
		}
	}
}

func TestGenerateCustomSymbols(t *testing.T) {
	p := Policy{
		Length:     16,
		Classes:    LowerChars | UpperChars | DigitChars | SymbolChars,
		MinLower:   1,
		MinUpper:   1,
		MinDigits:  1,
		MinSymbols: 1,
		Symbols:    "!@#",
	}
	allowed := lowerChars + upperChars + digitChars + p.Symbols
	for range 100 {
		password, err := Generate(p)
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if countIn(password, p.Symbols) < p.MinSymbols {
			t.Fatalf("password %q has no custom symbol", password)
		}
		if countIn(password, allowed) != len(password) {
			t.Fatalf("password %q contains an unexpected character", password)
		}
	}
}

func TestGenerateExcludeAndAmbiguous(t *testing.T) {
	p := DefaultPolicy()
	p.Exclude = "abcABC"
	p.NoAmbiguous = true
	forbidden := p.Exclude + ambiguousChars
	for range 100 {
		password, err := Generate(p)
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if strings.ContainsAny(password, forbidden) {
			t.Fatalf("password %q contains a forbidden character", password)
		}
	}
}

func TestGenerateStartLetter(t *testing.T) {
	t.Run("slack", func(t *testing.T) {
		p := Policy{Length: 8, Classes: LowerChars | DigitChars, StartLetter: true}
		for range 100 {
			password, err := Generate(p)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if !isLetter(password[0]) {
				t.Fatalf("password %q does not start with a letter", password)
			}
		}
	})
	t.Run("no slack", func(t *testing.T) {
		p := Policy{
			Length:      4,
			Classes:     LowerChars | DigitChars,
			MinLower:    2,
			MinDigits:   2,
			StartLetter: true,
		}
		for range 100 {
			password, err := Generate(p)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if !isLetter(password[0]) {
				t.Fatalf("password %q does not start with a letter", password)
			}
			if got := countIn(password, lowerChars); got != p.MinLower {
				t.Fatalf("password %q has %d lowercase letters, want %d", password, got, p.MinLower)
			}
			if got := countIn(password, digitChars); got != p.MinDigits {
				t.Fatalf("password %q has %d digits, want %d", password, got, p.MinDigits)
			}
		}
	})
}

func TestGenerateDistinct(t *testing.T) {
	p := DefaultPolicy()
	seen := make(map[string]bool, 100)
	for range 100 {
		password, err := Generate(p)
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if seen[password] {
			t.Fatalf("duplicate password %q", password)
		}
		seen[password] = true
	}
}

func TestValidateDefaultPolicy(t *testing.T) {
	if err := DefaultPolicy().Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestGenerateInvalidPolicy(t *testing.T) {
	tests := []struct {
		name string
		p    Policy
	}{
		{"zero length", Policy{Length: 0, Classes: LowerChars}},
		{"too long", Policy{Length: maxPasswordLength + 1, Classes: LowerChars}},
		{"no classes", Policy{Length: 10}},
		{"negative minimum", Policy{Length: 10, Classes: LowerChars, MinLower: -1}},
		{"minimum without class", Policy{Length: 10, Classes: DigitChars, MinLower: 1}},
		{
			"minimums exceed length",
			Policy{Length: 3, Classes: LowerChars | DigitChars, MinLower: 2, MinDigits: 2},
		},
		{"non-ascii symbol", Policy{Length: 10, Classes: SymbolChars, Symbols: "✓"}},
		{"symbols without class", Policy{Length: 10, Classes: LowerChars, Symbols: "!@#"}},
		{"empty alphabet", Policy{Length: 10, Classes: LowerChars, Exclude: lowerChars}},
		{
			"empty symbol alphabet",
			Policy{Length: 10, Classes: SymbolChars, MinSymbols: 1, Symbols: "!@", Exclude: "!@"},
		},
		{"start letter without letters", Policy{Length: 10, Classes: DigitChars, StartLetter: true}},
		{
			"start letter without slack",
			Policy{Length: 2, Classes: LowerChars | DigitChars, MinDigits: 2, StartLetter: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.p.Validate(); !errors.Is(err, ErrInvalidPolicy) {
				t.Fatalf("Validate() error = %v, want ErrInvalidPolicy", err)
			}
			if _, err := Generate(tt.p); !errors.Is(err, ErrInvalidPolicy) {
				t.Fatalf("Generate() error = %v, want ErrInvalidPolicy", err)
			}
		})
	}
}
