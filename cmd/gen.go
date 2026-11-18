package cmd

import (
	"fmt"
	"strings"

	"github.com/nxkh4ng/gatekeep/core"
	"github.com/spf13/cobra"
)

// maxGenCount bounds --count so one invocation cannot flood a terminal.
const maxGenCount = 100

func newGenCmd() *cobra.Command {
	defaults := core.DefaultPolicy()
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate a password",
		Long:  "Generate random passwords that satisfy a service's character rules.",
		Example: `  gatekeep gen
  gatekeep gen -l 12 --charset lower,upper,digits
  gatekeep gen --charset symbols --symbols '!@#'`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGen(cmd)
		},
	}
	flags := cmd.Flags()
	flags.IntP("length", "l", defaults.Length, "password length")
	flags.IntP("count", "c", 1, "number of passwords to print")
	flags.String("charset", "lower,upper,digits,symbols", "character classes to use: lower, upper, digits, symbols")
	flags.Int("min-lower", defaults.MinLower, "minimum lowercase letters")
	flags.Int("min-upper", defaults.MinUpper, "minimum uppercase letters")
	flags.Int("min-digits", defaults.MinDigits, "minimum digits")
	flags.Int("min-symbols", defaults.MinSymbols, "minimum symbols")
	flags.String("symbols", "", "symbol alphabet (default: all printable punctuation)")
	flags.String("exclude", "", "characters to exclude")
	flags.Bool("no-ambiguous", false, "exclude 0, O, 1, l and I")
	flags.Bool("start-letter", false, "start with a letter")
	return cmd
}

func runGen(cmd *cobra.Command) error {
	policy, count, err := genOptions(cmd)
	if err != nil {
		return err
	}

	passwords := make([]string, 0, count)
	for range count {
		password, err := core.Generate(policy)
		if err != nil {
			return err
		}
		passwords = append(passwords, password)
	}
	for _, password := range passwords {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), password); err != nil {
			return err
		}
	}
	return nil
}

// genOptions reads the generator flags into a policy and a count.
func genOptions(cmd *cobra.Command) (core.Policy, int, error) {
	var p core.Policy
	var err error
	flags := cmd.Flags()

	if p.Length, err = flags.GetInt("length"); err != nil {
		return core.Policy{}, 0, err
	}
	count, err := flags.GetInt("count")
	if err != nil {
		return core.Policy{}, 0, err
	}
	if count < 1 || count > maxGenCount {
		return core.Policy{}, 0, fmt.Errorf("count must be between 1 and %d", maxGenCount)
	}
	charset, err := flags.GetString("charset")
	if err != nil {
		return core.Policy{}, 0, err
	}
	if p.Classes, err = parseCharset(charset); err != nil {
		return core.Policy{}, 0, err
	}
	if p.MinLower, err = flags.GetInt("min-lower"); err != nil {
		return core.Policy{}, 0, err
	}
	if p.MinUpper, err = flags.GetInt("min-upper"); err != nil {
		return core.Policy{}, 0, err
	}
	if p.MinDigits, err = flags.GetInt("min-digits"); err != nil {
		return core.Policy{}, 0, err
	}
	if p.MinSymbols, err = flags.GetInt("min-symbols"); err != nil {
		return core.Policy{}, 0, err
	}
	if p.Symbols, err = flags.GetString("symbols"); err != nil {
		return core.Policy{}, 0, err
	}
	if p.Exclude, err = flags.GetString("exclude"); err != nil {
		return core.Policy{}, 0, err
	}
	if p.NoAmbiguous, err = flags.GetBool("no-ambiguous"); err != nil {
		return core.Policy{}, 0, err
	}
	if p.StartLetter, err = flags.GetBool("start-letter"); err != nil {
		return core.Policy{}, 0, err
	}
	// Default minimums only apply to enabled classes. A minimum the user set
	// explicitly for a disabled class is still an error, so typos are not
	// silently ignored.
	if p.Classes&core.LowerChars == 0 && !flags.Changed("min-lower") {
		p.MinLower = 0
	}
	if p.Classes&core.UpperChars == 0 && !flags.Changed("min-upper") {
		p.MinUpper = 0
	}
	if p.Classes&core.DigitChars == 0 && !flags.Changed("min-digits") {
		p.MinDigits = 0
	}
	if p.Classes&core.SymbolChars == 0 && !flags.Changed("min-symbols") {
		p.MinSymbols = 0
	}
	return p, count, nil
}

// parseCharset turns a comma-separated class list into a CharClass set.
func parseCharset(value string) (core.CharClass, error) {
	var classes core.CharClass
	for part := range strings.SplitSeq(value, ",") {
		switch strings.TrimSpace(part) {
		case "lower":
			classes |= core.LowerChars
		case "upper":
			classes |= core.UpperChars
		case "digits":
			classes |= core.DigitChars
		case "symbols":
			classes |= core.SymbolChars
		default:
			return 0, fmt.Errorf(
				"unknown character class %q (want lower, upper, digits, symbols)",
				strings.TrimSpace(part),
			)
		}
	}
	return classes, nil
}
