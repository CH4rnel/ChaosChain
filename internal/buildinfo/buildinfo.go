package buildinfo

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}

func Current() Info {
	return Info{Version: Version, Commit: Commit, BuildTime: BuildTime}
}

func Command() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print build identity",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := Current()
			switch output {
			case "text":
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "version=%s\ncommit=%s\nbuild_time=%s\n", info.Version, info.Commit, info.BuildTime)
				return err
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(info)
			default:
				return fmt.Errorf("unsupported output format %q", output)
			}
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format: text or json")
	return cmd
}
