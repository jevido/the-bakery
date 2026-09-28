package moderation

import (
	"bufio"
	"context"
	"errors"
	"os"
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
)

// Commands are the artisan commands that make operators: there is no
// sign-up for the console.
func Commands() []console.Command {
	return []console.Command{operatorCreate{}, operatorConfirm{}}
}

// operatorCreate makes an operator and prints the otpauth:// URL for their
// authenticator app. It asks for the password, or reads it from stdin when
// stdin is not a terminal.
type operatorCreate struct{}

func (operatorCreate) Signature() string { return "operator:create" }
func (operatorCreate) Description() string {
	return "Make a console operator (then confirm their authenticator with operator:confirm)"
}
func (operatorCreate) Extend() command.Extend { return command.Extend{ArgsUsage: "<email>", Category: "operator"} }

func (operatorCreate) Handle(ctx console.Context) error {
	email := strings.TrimSpace(ctx.Argument(0))
	if email == "" {
		return errors.New("usage: operator:create <email>")
	}
	password, err := readSecret(ctx, "Password (at least 12 characters)")
	if err != nil {
		return err
	}
	_, url, err := service.CreateOperator(context.Background(), email, password)
	if err != nil {
		return err
	}
	ctx.Info("Operator " + email + " made. Add this to an authenticator app:")
	ctx.Line(url)
	ctx.Line("Then confirm with a first code: go run . artisan operator:confirm " + email + " <code>")
	return nil
}

// operatorConfirm checks an operator's first code; until then they cannot
// sign in.
type operatorConfirm struct{}

func (operatorConfirm) Signature() string   { return "operator:confirm" }
func (operatorConfirm) Description() string { return "Confirm a console operator's authenticator with a first code" }
func (operatorConfirm) Extend() command.Extend {
	return command.Extend{ArgsUsage: "<email> <code>", Category: "operator"}
}

func (operatorConfirm) Handle(ctx console.Context) error {
	email, code := strings.TrimSpace(ctx.Argument(0)), strings.TrimSpace(ctx.Argument(1))
	if email == "" || code == "" {
		return errors.New("usage: operator:confirm <email> <code>")
	}
	if err := service.ConfirmOperator(context.Background(), email, code); err != nil {
		return err
	}
	ctx.Info("Confirmed. " + email + " can sign in to the console.")
	return nil
}

func readSecret(ctx console.Context, question string) (string, error) {
	if st, err := os.Stdin.Stat(); err == nil && st.Mode()&os.ModeCharDevice == 0 {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	}
	return ctx.Secret(question)
}
