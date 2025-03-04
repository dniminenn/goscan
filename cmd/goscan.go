package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	Execute()
}

func Execute() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Modify NewRootCmd() in goscan.go to add the subcommands and show flag
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "goscan",
		Short: "Goscan is a network scanner using ICMP to detect active hosts",
		Long: `A network scanner using ICMP echo requests to detect active hosts on a network.
Root command without subcommands will run in CLI mode.`,
		Run: runCLI,
	}

	rootCmd.PersistentFlags().IntP("timeout", "t", 500, "Timeout in milliseconds")
	rootCmd.PersistentFlags().StringP("interface", "i", "", "Specify network interface name")
	rootCmd.PersistentFlags().BoolP("measure", "m", false, "Measure execution time")
	rootCmd.PersistentFlags().StringP("show", "s", "all", "Show mode: all, alive, or available")
	rootCmd.PersistentFlags().BoolP("scriptable", "q", false, "Scriptable output (no headers, no extra text)")

	aliveCmd := &cobra.Command{
		Use:     "alive",
		Aliases: []string{"online", "used", "taken"},
		Short:   "Show only alive hosts in a scriptable format",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Flags().Set("show", "alive")
			cmd.Flags().Set("scriptable", "true")
			runCLI(cmd, args)
		},
	}

	availableCmd := &cobra.Command{
		Use:     "available",
		Aliases: []string{"offline", "unused", "free"},
		Short:   "Show only available IPs in a scriptable format",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Flags().Set("show", "available")
			cmd.Flags().Set("scriptable", "true")
			runCLI(cmd, args)
		},
	}

	aliveCmd.PersistentFlags().IntP("timeout", "t", 500, "Timeout in milliseconds")
	aliveCmd.PersistentFlags().StringP("interface", "i", "", "Specify network interface name")
	aliveCmd.PersistentFlags().BoolP("measure", "m", false, "Measure execution time")

	availableCmd.PersistentFlags().IntP("timeout", "t", 500, "Timeout in milliseconds")
	availableCmd.PersistentFlags().StringP("interface", "i", "", "Specify network interface name")
	availableCmd.PersistentFlags().BoolP("measure", "m", false, "Measure execution time")

	rootCmd.AddCommand(aliveCmd)
	rootCmd.AddCommand(availableCmd)
	rootCmd.AddCommand(NewServerCmd())

	return rootCmd
}

func NewServerCmd() *cobra.Command {
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Run goscan in server mode with web interface",
		Long:  `Run goscan in server mode with a web interface for enumerating alive hosts.`,
		Run:   runServer,
	}

	serverCmd.Flags().StringP("listen-address", "l", "0.0.0.0", "IP address for the server")
	serverCmd.Flags().StringP("listen-port", "p", "8080", "Port number")
	serverCmd.Flags().String("ssl-cert", "", "SSL certificate file")
	serverCmd.Flags().String("ssl-key", "", "SSL key file")
	serverCmd.Flags().Int("max-subnet-size", 1024, "Maximum subnet size to scan")

	return serverCmd
}
