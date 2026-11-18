package core

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// ErrInvalidPolicy reports a generation policy that cannot be satisfied.
var ErrInvalidPolicy = errors.New("invalid generation policy")

// maxPasswordLength bounds a generated password so a bad policy cannot ask
// for an unbounded amount of memory or output.
const maxPasswordLength = 1024

// DefaultSymbols is the printable punctuation alphabet used when a policy
// does not provide its own.
const DefaultSymbols = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"

const (
	lowerChars     = "abcdefghijklmnopqrstuvwxyz"
	upperChars     = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars     = "0123456789"
	ambiguousChars = "0O1lI"
)

// CharClass is a bit set of the character categories a password may draw
// from.
type CharClass uint8

const (
	// LowerChars is lowercase ASCII letters.
	LowerChars CharClass = 1 << iota
	// UpperChars is uppercase ASCII letters.
	UpperChars
	// DigitChars is the digits 0-9.
	DigitChars
	// SymbolChars is printable punctuation.
	SymbolChars
)

// Policy describes the character rules for a generated password. Start from
// DefaultPolicy or set every field explicitly.
type Policy struct {
	Length      int       // total number of characters
	Classes     CharClass // categories the password may use
	MinLower    int       // required lowercase letters
	MinUpper    int       // required uppercase letters
	MinDigits   int       // required digits
	MinSymbols  int       // required symbols
	Symbols     string    // symbol alphabet; empty means DefaultSymbols
	Exclude     string    // characters removed from every category
	NoAmbiguous bool      // remove 0, O, 1, l and I
	StartLetter bool      // the first character must be a letter
}

// DefaultPolicy generates a 20-character password with at least one character
// from every class.
func DefaultPolicy() Policy {
	return Policy{
		Length:     20,
		Classes:    LowerChars | UpperChars | DigitChars | SymbolChars,
		MinLower:   1,
		MinUpper:   1,
		MinDigits:  1,
		MinSymbols: 1,
		Symbols:    DefaultSymbols,
	}
}

// Validate reports whether p can be satisfied by Generate.
func (p Policy) Validate() error {
	_, err := p.prepare()
	return err
}

// Generate returns a random password that satisfies p. It draws exclusively
// from crypto/rand and returns ErrInvalidPolicy when p cannot be satisfied.
func Generate(p Policy) (string, error) {
	s, err := p.prepare()
	if err != nil {
		return "", err
	}

	buf := make([]byte, 0, p.Length)
	if p.StartLetter {
		first, err := randomChar(p.startLetterPool(s))
		if err != nil {
			return "", err
		}
		buf = append(buf, first)
		// A reserved character satisfies a minimum of its own class.
		switch {
		case p.MinLower > 0 && strings.IndexByte(s.lower, first) >= 0:
			p.MinLower--
		case p.MinUpper > 0 && strings.IndexByte(s.upper, first) >= 0:
			p.MinUpper--
		}
	}

	for _, need := range []struct {
		alphabet string
		min      int
	}{
		{s.lower, p.MinLower},
		{s.upper, p.MinUpper},
		{s.digits, p.MinDigits},
		{s.symbols, p.MinSymbols},
	} {
		for range need.min {
			ch, err := randomChar(need.alphabet)
			if err != nil {
				return "", err
			}
			buf = append(buf, ch)
		}
	}

	pool := s.union(p.Classes)
	for len(buf) < p.Length {
		ch, err := randomChar(pool)
		if err != nil {
			return "", err
		}
		buf = append(buf, ch)
	}

	start := 0
	if p.StartLetter {
		start = 1
	}
	if err := shuffle(buf[start:]); err != nil {
		return "", err
	}

	password := string(buf)
	clear(buf)
	return password, nil
}

// charSets holds the alphabets a policy draws from after exclusions.
type charSets struct {
	lower   string
	upper   string
	digits  string
	symbols string
}

// prepare validates p and returns its filtered alphabets.
func (p Policy) prepare() (charSets, error) {
	if err := p.validate(); err != nil {
		return charSets{}, err
	}
	s := p.charSets()
	if err := s.validate(p); err != nil {
		return charSets{}, err
	}
	return s, nil
}

// validate checks the policy fields that do not depend on the built alphabets.
func (p Policy) validate() error {
	if p.Length < 1 || p.Length > maxPasswordLength {
		return fmt.Errorf("%w: length must be between 1 and %d", ErrInvalidPolicy, maxPasswordLength)
	}
	if p.Classes == 0 {
		return fmt.Errorf("%w: at least one character class is required", ErrInvalidPolicy)
	}

	minimums := []struct {
		name  string
		value int
		class CharClass
	}{
		{"lowercase", p.MinLower, LowerChars},
		{"uppercase", p.MinUpper, UpperChars},
		{"digit", p.MinDigits, DigitChars},
		{"symbol", p.MinSymbols, SymbolChars},
	}
	total := 0
	for _, m := range minimums {
		if m.value < 0 {
			return fmt.Errorf("%w: %s minimum cannot be negative", ErrInvalidPolicy, m.name)
		}
		if m.value > 0 && p.Classes&m.class == 0 {
			return fmt.Errorf("%w: %s minimum is set but the class is not enabled", ErrInvalidPolicy, m.name)
		}
		total += m.value
	}
	if total > p.Length {
		return fmt.Errorf("%w: minimums need %d characters but length is %d", ErrInvalidPolicy, total, p.Length)
	}
	if p.Symbols != "" && p.Classes&SymbolChars == 0 {
		return fmt.Errorf("%w: symbols are set but the symbol class is not enabled", ErrInvalidPolicy)
	}
	for _, r := range p.Symbols {
		if r < '!' || r > '~' {
			return fmt.Errorf("%w: symbols must be printable ASCII without spaces", ErrInvalidPolicy)
		}
	}
	return nil
}

// charSets builds the alphabets from the policy, applying exclusions.
func (p Policy) charSets() charSets {
	symbols := p.Symbols
	if symbols == "" {
		symbols = DefaultSymbols
	}
	return charSets{
		lower:   filterChars(lowerChars, p.Exclude, p.NoAmbiguous),
		upper:   filterChars(upperChars, p.Exclude, p.NoAmbiguous),
		digits:  filterChars(digitChars, p.Exclude, p.NoAmbiguous),
		symbols: filterChars(symbols, p.Exclude, p.NoAmbiguous),
	}
}

// validate checks that the alphabets can meet the policy's minimums.
func (s charSets) validate(p Policy) error {
	needs := []struct {
		name     string
		alphabet string
		min      int
	}{
		{"lowercase", s.lower, p.MinLower},
		{"uppercase", s.upper, p.MinUpper},
		{"digit", s.digits, p.MinDigits},
		{"symbol", s.symbols, p.MinSymbols},
	}
	for _, n := range needs {
		if n.min > 0 && n.alphabet == "" {
			return fmt.Errorf("%w: %s minimum cannot be met after exclusions", ErrInvalidPolicy, n.name)
		}
	}
	if s.union(p.Classes) == "" {
		return fmt.Errorf("%w: no characters left after exclusions", ErrInvalidPolicy)
	}
	if p.StartLetter && p.startLetterPool(s) == "" {
		return fmt.Errorf("%w: start-letter needs an enabled letter class with characters left", ErrInvalidPolicy)
	}
	return nil
}

// union concatenates the alphabets of the enabled classes.
func (s charSets) union(classes CharClass) string {
	var out []byte
	if classes&LowerChars != 0 {
		out = append(out, s.lower...)
	}
	if classes&UpperChars != 0 {
		out = append(out, s.upper...)
	}
	if classes&DigitChars != 0 {
		out = append(out, s.digits...)
	}
	if classes&SymbolChars != 0 {
		out = append(out, s.symbols...)
	}
	return string(out)
}

// startLetterPool returns the letters the first character may use. When every
// slot is already required by a minimum, the first character must count
// toward one of those minimums.
func (p Policy) startLetterPool(s charSets) string {
	if !p.StartLetter {
		return ""
	}
	slack := p.MinLower+p.MinUpper+p.MinDigits+p.MinSymbols < p.Length
	var pool []byte
	if p.Classes&LowerChars != 0 && (slack || p.MinLower > 0) {
		pool = append(pool, s.lower...)
	}
	if p.Classes&UpperChars != 0 && (slack || p.MinUpper > 0) {
		pool = append(pool, s.upper...)
	}
	return string(pool)
}

// filterChars removes excluded and ambiguous characters from an alphabet.
func filterChars(chars, exclude string, noAmbiguous bool) string {
	out := make([]byte, 0, len(chars))
	for i := range len(chars) {
		ch := chars[i]
		if strings.IndexByte(exclude, ch) >= 0 {
			continue
		}
		if noAmbiguous && strings.IndexByte(ambiguousChars, ch) >= 0 {
			continue
		}
		out = append(out, ch)
	}
	return string(out)
}

// randomChar picks one character uniformly from alphabet.
func randomChar(alphabet string) (byte, error) {
	if alphabet == "" {
		return 0, fmt.Errorf("%w: empty character set", ErrInvalidPolicy)
	}
	i, err := randomInt(len(alphabet))
	if err != nil {
		return 0, err
	}
	return alphabet[i], nil
}

// randomInt returns a uniform value in [0, n) using crypto/rand.
func randomInt(n int) (int, error) {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, fmt.Errorf("generating random number: %w", err)
	}
	return int(v.Int64()), nil
}

// shuffle permutes b in place using crypto/rand.
func shuffle(b []byte) error {
	for i := len(b) - 1; i > 0; i-- {
		j, err := randomInt(i + 1)
		if err != nil {
			return err
		}
		b[i], b[j] = b[j], b[i]
	}
	return nil
}
