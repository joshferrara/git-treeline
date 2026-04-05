package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/git-treeline/git-treeline/internal/format"
	"github.com/git-treeline/git-treeline/internal/registry"
	"github.com/spf13/cobra"
)

var portJSON bool

func init() {
	portCmd.Flags().BoolVar(&portJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(portCmd)
}

var portCmd = &cobra.Command{
	Use:   "port",
	Short: "Print the allocated port for the current worktree",
	Long:  `Prints the primary allocated port for the current directory's worktree. Useful for scripts, agents, and CI that need the port without parsing status output.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		absPath, _ := filepath.Abs(cwd)

		reg := registry.New("")
		entry := reg.Find(absPath)
		if entry == nil {
			fmt.Fprintf(os.Stderr, "No allocation found for %s\nRun `gtl setup` first.\n", absPath)
			os.Exit(1)
		}

		ports := format.GetPorts(format.Allocation(entry))
		if len(ports) == 0 {
			fmt.Fprintln(os.Stderr, "Allocation exists but has no ports.")
			os.Exit(1)
		}

		if portJSON {
			data, _ := json.MarshalIndent(map[string]any{
				"port":  ports[0],
				"ports": ports,
			}, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Println(ports[0])
		return nil
	},
}
