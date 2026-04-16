package cfsmtpcmd

import (
	"testing"

	"github.com/urfave/cli/v3"
)

func TestRootCommandRegistersExpectedCommands(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	commandNames := map[string]bool{}
	for _, subcommand := range cmd.Commands {
		commandNames[subcommand.Name] = true
	}

	for _, name := range []string{"serve", "healthcheck", "send-hello"} {
		if !commandNames[name] {
			t.Fatalf("missing command %q", name)
		}
	}
}

func TestHelloFlagsUseNeutralDefaults(t *testing.T) {
	t.Parallel()

	var fromValue string
	var toValue string

	for _, flag := range helloFlags() {
		stringFlag, ok := flag.(*cli.StringFlag)
		if !ok {
			continue
		}

		switch stringFlag.Name {
		case "from":
			fromValue = stringFlag.Value
		case "to":
			toValue = stringFlag.Value
		}
	}

	if fromValue != "CF SMTP Example <sender@example.com>" {
		t.Fatalf("from default = %q", fromValue)
	}
	if toValue != "recipient@example.com" {
		t.Fatalf("to default = %q", toValue)
	}
}
