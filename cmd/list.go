package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"cast-desktop-go/pkg/cast"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"devices", "ls"},
	Short:   "List discoverable Chromecast devices on the network",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Scanning local network for Chromecast devices...")
		devices := cast.DiscoverDevices(true)

		if len(devices) == 0 {
			fmt.Println("No Chromecast devices found.")
			return
		}

		fmt.Printf("\nFound %d Chromecast device(s):\n\n", len(devices))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "#\tName\tIP Address\tModel\tPort")
		fmt.Fprintln(w, "-\t----\t----------\t-----\t----")

		for i, d := range devices {
			fmt.Fprintf(w, "[%d]\t%s\t%s\t%s\t%d\n", i+1, d.Name, d.IP, d.Model, d.Port)
		}
		w.Flush()
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
