package main

import (
	"path/filepath"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:     "show",
	Short:   "Show the file manager window, launching dfiles if it is not running",
	PreRunE: shellApp.ResolveConfig,
	RunE: func(_ *cobra.Command, _ []string) error {
		return shellApp.CallOrLaunch("ui.show", nil)
	},
}

var toggleCmd = &cobra.Command{
	Use:     "toggle",
	Short:   "Toggle file manager window visibility, launching dfiles if it is not running",
	PreRunE: shellApp.ResolveConfig,
	RunE: func(_ *cobra.Command, _ []string) error {
		return shellApp.CallOrLaunch("ui.toggle", nil)
	},
}

var openCmd = &cobra.Command{
	Use:     "open [path]",
	Short:   "Open a directory in the file manager (no argument just shows the window)",
	PreRunE: shellApp.ResolveConfig,
	Args:    cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 || args[0] == "" {
			return shellApp.CallOrLaunch("ui.show", nil)
		}
		path, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		return shellApp.CallOrLaunch("ui.open", map[string]any{"path": path})
	},
}
