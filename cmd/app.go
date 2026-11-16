package cmd

// app holds the dependencies shared by all commands.
type app struct {
	vault    string
	prompter Prompter
}
