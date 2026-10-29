package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/duxlabs-dev/dux-files/core/internal/sshfs"
)

var sshRelayCmd = &cobra.Command{
	Use:                "sshrelay",
	Short:              "Runs ssh for sshfs with the sudo password sent first (not for direct use)",
	Hidden:             true,
	DisableFlagParsing: true,
	Args:               cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		os.Exit(sshfs.Relay(args[0], args[1:], os.Getenv(sshfs.PasswordEnv), sshfs.PollableStdin(), os.Stdout, os.Stderr))
	},
}
