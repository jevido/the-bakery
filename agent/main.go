// Command bakery is the Bakery host agent CLI. `bakery daemon` runs the
// check-in loop against a control plane; other subcommands (setup, join,
// bootstrap — added across this phase) prepare a host to run it in the
// first place.
package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]

	subcommand := "daemon"
	if len(args) > 0 && !looksLikeFlag(args[0]) {
		subcommand = args[0]
		args = args[1:]
	}

	switch subcommand {
	case "daemon":
		cmdDaemon(args)
	case "setup":
		cmdSetup(args)
	case "join":
		cmdJoin(args)
	default:
		fmt.Fprintf(os.Stderr, "bakery: unknown subcommand %q\n", subcommand)
		os.Exit(2)
	}
}

// looksLikeFlag reports whether arg is a flag (e.g. -token=...) rather than
// a subcommand name — lets the pre-subcommand invocation shape (`bakery
// -token=... -url=...`) keep working without every already-deployed host's
// systemd unit needing to say `bakery daemon` explicitly first.
func looksLikeFlag(arg string) bool {
	return len(arg) > 0 && arg[0] == '-'
}
