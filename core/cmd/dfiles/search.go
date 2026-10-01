package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type searchEvent struct {
	SearchID   string            `json:"searchId"`
	Kind       string            `json:"kind"`
	Entries    []json.RawMessage `json:"entries"`
	Count      int               `json:"count"`
	Truncated  bool              `json:"truncated"`
	Cancelled  bool              `json:"cancelled"`
	Unreadable int               `json:"unreadable"`
}

var (
	searchHiddenFlag      bool
	searchCrossDeviceFlag bool
	searchLimitFlag       int
	searchTypesFlag       []string
)

var searchCmd = &cobra.Command{
	Use:   "search <root> <query>",
	Short: "Search a folder tree by name, printing matching paths as they are found",
	Args:  cobra.ExactArgs(2),
	ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return nil, cobra.ShellCompDirectiveFilterDirs
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(_ *cobra.Command, args []string) error {
		root, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		params := map[string]any{"root": root, "query": args[1], "includeHidden": searchHiddenFlag, "crossDevice": searchCrossDeviceFlag}
		if searchLimitFlag > 0 {
			params["limit"] = searchLimitFlag
		}
		if len(searchTypesFlag) > 0 {
			params["types"] = searchTypesFlag
		}
		return withSession(searchServices, func(s *session) error {
			found := []json.RawMessage{}
			done, err := runSearch(s, params, func(raw json.RawMessage) {
				if jsonOutput {
					found = append(found, raw)
					return
				}
				var entry struct {
					Path string `json:"path"`
				}
				if json.Unmarshal(raw, &entry) == nil {
					fmt.Println(entry.Path)
				}
			})
			if err != nil {
				return err
			}
			if jsonOutput {
				return printJSON(map[string]any{"entries": found, "count": done.Count, "truncated": done.Truncated, "unreadable": done.Unreadable})
			}
			if done.Truncated {
				fmt.Fprintf(os.Stderr, "dfiles: stopped at %d results; raise --limit to see more\n", done.Count)
			}
			if done.Unreadable > 0 {
				fmt.Fprintf(os.Stderr, "dfiles: %d folders could not be read\n", done.Unreadable)
			}
			return nil
		})
	},
}

func runSearch(s *session, params map[string]any, onEntry func(json.RawMessage)) (searchEvent, error) {
	var started struct {
		SearchID string `json:"searchId"`
		Topic    string `json:"topic"`
	}
	if err := s.call("search.start", params, &started); err != nil {
		return searchEvent{}, err
	}
	if err := s.subscribe(started.Topic); err != nil {
		return searchEvent{}, err
	}

	for ev := range s.events {
		if ev.Topic != started.Topic {
			continue
		}
		var batch searchEvent
		if json.Unmarshal(ev.Data, &batch) != nil || batch.SearchID != started.SearchID {
			continue
		}
		switch batch.Kind {
		case "results":
			for _, raw := range batch.Entries {
				onEntry(raw)
			}
		case "done":
			return batch, nil
		}
	}
	return searchEvent{}, errors.New("lost the connection while searching")
}

func init() {
	searchCmd.Flags().BoolVar(&searchHiddenFlag, "hidden", false, "Descend into and report hidden entries")
	searchCmd.Flags().BoolVar(&searchCrossDeviceFlag, "cross-device", false, "Follow into other filesystems")
	searchCmd.Flags().IntVar(&searchLimitFlag, "limit", 0, "Stop after this many results (at most and by default 10000)")
	searchCmd.Flags().StringSliceVar(&searchTypesFlag, "type", nil, "Only documents, images, video, audio, archives or folders (repeatable)")
}
