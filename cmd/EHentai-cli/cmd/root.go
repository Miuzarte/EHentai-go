package cmd

import (
	"os"

	"github.com/Miuzarte/EHentai-go/cmd/EHentai-cli/internal/errors"
	"github.com/Miuzarte/SimpleLog"
	"github.com/spf13/cobra"
)

var rootLog = SimpleLog.New("[EHcli]", true, false)

var rootCmd = &cobra.Command{
	Use:   "EHcli",
	Short: "A command line tool for E(x)Hentai search/gallery download/pages download",
	Long:  "A command line tool for E(x)Hentai search/gallery download/pages download",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func Execute() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	if err := rootCmd.Execute(); err != nil {
		switch err {
		case errors.Handled:
			os.Exit(1)
		case errors.Aborted:
			rootLog.Warn(err)
			return
		default:
			rootLog.Fatal(err)
		}
	}
}
