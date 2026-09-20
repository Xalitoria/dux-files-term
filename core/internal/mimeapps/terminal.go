package mimeapps

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

type terminalSpec struct {
	command string
	exec    []string
}

// knownTerminals is ordered by preference; exec holds the flags that make the
// terminal run the rest of the argv as a command.
var knownTerminals = []terminalSpec{
	{command: "ptyxis", exec: []string{"--"}},
	{command: "kgx", exec: []string{"--"}},
	{command: "ghostty", exec: []string{"-e"}},
	{command: "kitty", exec: nil},
	{command: "foot", exec: nil},
	{command: "alacritty", exec: []string{"-e"}},
	{command: "wezterm", exec: []string{"start", "--"}},
	{command: "konsole", exec: []string{"-e"}},
	{command: "gnome-terminal", exec: []string{"--"}},
	{command: "xfce4-terminal", exec: []string{"-x"}},
	{command: "tilix", exec: []string{"-e"}},
	{command: "terminator", exec: []string{"-x"}},
	{command: "urxvt", exec: []string{"-e"}},
	{command: "st", exec: []string{"-e"}},
	{command: "xterm", exec: []string{"-e"}},
	{command: "x-terminal-emulator", exec: []string{"-e"}},
}

// terminal resolves the terminal to use: an explicit preference, then
// $TERMINAL, then the first known emulator on PATH.
func (s *Service) terminal(preferred string) (terminalSpec, error) {
	for _, candidate := range []string{preferred, os.Getenv("TERMINAL")} {
		if candidate == "" {
			continue
		}
		if spec, ok := lookupTerminal(candidate); ok {
			return spec, nil
		}
	}

	for _, spec := range knownTerminals {
		if _, err := exec.LookPath(spec.command); err == nil {
			return spec, nil
		}
	}
	return terminalSpec{}, errors.New("no terminal emulator found")
}

func lookupTerminal(command string) (terminalSpec, bool) {
	if _, err := exec.LookPath(command); err != nil {
		return terminalSpec{}, false
	}
	base := command
	if cut := strings.LastIndexByte(base, '/'); cut >= 0 {
		base = base[cut+1:]
	}
	for _, spec := range knownTerminals {
		if spec.command == base {
			spec.command = command
			return spec, true
		}
	}
	return terminalSpec{command: command, exec: []string{"-e"}}, true
}

// Terminals lists the emulators actually installed, for a settings picker.
func Terminals() []string {
	var out []string
	for _, spec := range knownTerminals {
		if _, err := exec.LookPath(spec.command); err == nil {
			out = append(out, spec.command)
		}
	}
	return out
}

func (s *Service) wrapTerminal(argv []string, preferred string) ([]string, error) {
	spec, err := s.terminal(preferred)
	if err != nil {
		return nil, err
	}
	wrapped := append([]string{spec.command}, spec.exec...)
	return append(wrapped, argv...), nil
}
