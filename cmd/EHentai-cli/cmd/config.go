package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	_ "embed"

	"github.com/Miuzarte/EHentai-go"
	"github.com/Miuzarte/EHentai-go/internal/env"
	"github.com/Miuzarte/EHentai-go/internal/utils"
	"github.com/Miuzarte/SimpleLog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configLog = SimpleLog.New("[Config]", true, false)

//go:embed defaultConfig.toml
var defaultConfig []byte

type configStruct struct {
	Log struct {
		Level int // '--trace' / '--debug'
	}

	Account struct {
		Cookie string // '--cookie'
		// OR
		IpbMemberId string // '--ipbm'
		IpbPassHash string // '--ipbh'
		Igneous     string // '--ig'
		Sk          string // '--sk'
	}

	Download struct {
		Threads        int    // '--threads'
		ProgressBar    bool   // '--progress'
		RetryDepth     int    // '--retry'
		EnvProxy       bool   // '--env-proxy'
		DomainFronting bool   // '--domain-fronting'
		Dir            string // '--dir'
	}

	Search struct {
		Site             EHentai.Domain   // '-e' / '-x' / '--site'
		EhTagTranslation bool             // '-t' '--eh-tag'
		Category         []string         // '--cat'
		Detail           bool             // '-d' '--detail'
		TorrentDetail    bool             // '--torrent-detail'
		category         EHentai.Category // parsed
	}
}

var (
	config     configStruct
	configPath = filepath.Join(env.XDir, "config.toml")
)

func initConfig(cmd *cobra.Command, _ []string) (err error) {
	configLog.Debug("using config file: ", configPath)
	viper.SetConfigFile(configPath)

	err = viper.ReadInConfig()
	if err != nil {
		// if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		if _, ok := err.(*fs.PathError); !ok {
			// configLog.Debugf("type of error: %T", err)
			configLog.Fatal("failed to read config file: ", err)
		} else {
			err = os.WriteFile(configPath, defaultConfig, 0o644)
			if err != nil {
				configLog.Fatal("failed to create config file: ", err)
			}
			configLog.Infof("file created at %s, please edit it", utils.HyperlinkFile(configPath))
			const logLen = len(` [INFO][15:04-|01/02][Config] file created at `)
			spaces := strings.Repeat(" ", logLen-len("Hyperlink: "))
			arrows := strings.Repeat("^", len(configPath))
			fmt.Println(spaces + "Hyperlink: " + arrows)
			// os.Exit(0)
			return ErrConfigCreated
		}
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		configLog.Fatal("failed to unmarshal config file: ", err)
	}
	configLog.Debug("config file loaded: ", configPath)

	// log
	flagTrace, _ := cmd.PersistentFlags().GetBool("trace")
	flagDebug, _ := cmd.PersistentFlags().GetBool("debug")
	switch {
	case flagTrace:
		configLog.SetLevel(SimpleLog.TraceLevel)
	case flagDebug:
		configLog.SetLevel(SimpleLog.DebugLevel)
	default:
		configLog.SetLevel(SimpleLog.Level(config.Log.Level))
	}

	// account
	switch {
	case config.Account.Cookie != "":
		_, err := EHentai.SetCookieFromString(config.Account.Cookie)
		if err != nil {
			configLog.Error("failed to parse cookie: ", err)
			return ErrHandled
		}

	case config.Account.IpbMemberId != "" &&
		config.Account.IpbPassHash != "" &&
		config.Account.Igneous != "":
		if config.Account.Sk == "" {
			configLog.Warn("sk not set, the title language of search results might be unexpected")
		}
		EHentai.SetCookie(
			config.Account.IpbMemberId,
			config.Account.IpbPassHash,
			config.Account.Igneous,
			config.Account.Sk,
		)

	default:
		configLog.Warn("cookie not set")
	}

	// download
	if config.Download.Threads > 0 {
		EHentai.SetThreads(config.Download.Threads)
	}
	if config.Download.RetryDepth >= 0 {
		EHentai.SetRetryDepth(config.Download.RetryDepth)
	}
	EHentai.SetUseEnvProxy(config.Download.EnvProxy)
	EHentai.SetDomainFronting(config.Download.DomainFronting)
	if config.Download.Dir == "" {
		config.Download.Dir = env.XDir
	}
	EHentai.SetCacheDir(config.Download.Dir)

	// search
	config.Search.category, err = EHentai.ParseCategory(config.Search.Category...)
	if err != nil {
		configLog.Error("failed to parse category: ", err)
		if !utils.WaitAck("continue?") {
			return ErrAborted
		}
	}

	return nil
}
