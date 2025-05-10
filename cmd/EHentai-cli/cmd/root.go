package cmd

import (
	"github.com/Miuzarte/EHentai-go/cmd/EHentai-cli/internal/errors"
	"github.com/Miuzarte/EHentai-go/cmd/EHentai-cli/internal/log"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "EHentai-cli",
	Short: "A command line tool for E(x)Hentai search/gallery download/pages download",
	Long:  "A command line tool for E(x)Hentai search/gallery download/pages download",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		switch err {
		case errors.ErrAborted:
			log.Warn("aborted")
		default:
			log.Fatal(err)
		}
	}
}
