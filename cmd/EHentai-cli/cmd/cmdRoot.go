package cmd

import (
	"github.com/Miuzarte/SimpleLog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const ehCliDesc = "A command line tool for E(x)Hentai search/gallery download/pages download"

const ehCliDescLong = ehCliDesc +
	"\n" + `"EHcli search <keyword>" to search gallery` +
	"\n" + `"EHcli download <gallery/page url>..." to download gallery or pages`

var rootLog = SimpleLog.New("[EHcli]", true, false)

var rootCmd = &cobra.Command{
	Use:   "EHcli",
	Short: ehCliDesc,
	Long:  ehCliDescLong,
	PreRun: func(cmd *cobra.Command, args []string) {
		// 如果直接绑定 viper,
		// 会导致 viper 的值被 flag 的默认值覆盖,
		// 因此手动检查 flag 是否被修改
		rootFlagChanged(cmd.Flags())
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

const (
	FLAG_CONFIG = "config"

	FLAG_TRACE = "trace"
	FLAG_DEBUG = "debug"

	FLAG_COOKIE = "cookie"

	FLAG_IPBM = "ipbm"
	FLAG_IPBH = "ipbh"
	FLAG_IG   = "ig"
	FLAG_SK   = "sk"
)

func init() {
	rootCmd.PersistentFlags().StringP(FLAG_CONFIG, "c", "", "path to config file")

	rootCmd.PersistentFlags().Bool(FLAG_TRACE, false, "set log level to trace")
	rootCmd.PersistentFlags().Bool(FLAG_DEBUG, false, "set log level to debug")
	rootCmd.MarkFlagsMutuallyExclusive(FLAG_TRACE, FLAG_DEBUG)

	rootCmd.PersistentFlags().String(FLAG_COOKIE, "", "well... cookie")

	rootCmd.PersistentFlags().String(FLAG_IPBM, "", "ipb_member_id")
	rootCmd.PersistentFlags().String(FLAG_IPBH, "", "ipb_pass_hash")
	rootCmd.PersistentFlags().String(FLAG_IG, "", "igneous")
	// 表站不需要 igneous
	rootCmd.MarkFlagsRequiredTogether(FLAG_IPBM, FLAG_IPBH)
	// sk 留空时搜索结果标题只有英文
	rootCmd.PersistentFlags().String(FLAG_SK, "", "sk")
}

func Execute() int {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	if err := rootCmd.Execute(); err != nil {
		switch err {
		case ErrAborted:
			rootLog.Warn(err)
			return 0
		case ErrHandled:
			return 1
		case ErrConfigCreated:
			return 0
		default:
			rootLog.Fatal(err)
		}
	}
	return 0
}

func rootFlagChanged(rootFlags *pflag.FlagSet) {
	if rootFlags.Changed(FLAG_CONFIG) {
		configPath, _ = rootFlags.GetString(FLAG_CONFIG)
	}
	if rootFlags.Changed(FLAG_TRACE) {
		trace, _ := rootFlags.GetBool(FLAG_TRACE)
		if trace {
			viper.Set("log.level", SimpleLog.TraceLevel)
		}
	}
	if rootFlags.Changed(FLAG_DEBUG) {
		debug, _ := rootFlags.GetBool(FLAG_DEBUG)
		if debug {
			viper.Set("log.level", SimpleLog.DebugLevel)
		}
	}
	if rootFlags.Changed(FLAG_COOKIE) {
		cookie, _ := rootFlags.GetString(FLAG_COOKIE)
		viper.Set("account.cookie", cookie)
	}
	if rootFlags.Changed(FLAG_IPBM) {
		ipbm, _ := rootFlags.GetString(FLAG_IPBM)
		viper.Set("account.ipbmemberid", ipbm)
	}
	if rootFlags.Changed(FLAG_IPBH) {
		ipbh, _ := rootFlags.GetString(FLAG_IPBH)
		viper.Set("account.ipbpasshash", ipbh)
	}
	if rootFlags.Changed(FLAG_IG) {
		ig, _ := rootFlags.GetString(FLAG_IG)
		viper.Set("account.igneous", ig)
	}
	if rootFlags.Changed(FLAG_SK) {
		sk, _ := rootFlags.GetString(FLAG_SK)
		viper.Set("account.sk", sk)
	}
}
