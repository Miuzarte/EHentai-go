package config

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/Miuzarte/EHentai-go"
	"github.com/Miuzarte/EHentai-go/cmd/EHentai-cli/internal/log"
	"github.com/Miuzarte/EHentai-go/internal/env"
	"github.com/Miuzarte/EHentai-go/internal/utils"
	"github.com/spf13/viper"
)

//go:embed config.toml
var defaultConfig []byte

type config struct {
	Account struct {
		Cookie string
		// OR
		IpbMemberId string
		IpbPassHash string
		Igneous     string
		Sk          string
	}
	Download struct {
		ProgressBar    bool
		UseEnvProxy    bool
		DomainFronting bool
		Threads        int
		RetryDepth     int
		Dir            string
	}

	*viper.Viper
}

var C = config{Viper: viper.New()}

func init() {
	configPath := filepath.Join(env.XDir, "config.toml")

	C.AddConfigPath(env.XDir)
	C.SetConfigName("config")
	// Config.SetConfigType("toml") // 任意

	var err error

	err = C.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatal("failed to read config file: ", err)
		} else {
			err = os.WriteFile(configPath, defaultConfig, 0o644)
			if err != nil {
				log.Fatal("failed to create config file: ", err)
			}
			log.Infof("config.toml created at %s, please edit it", utils.HyperlinkFile(configPath))
			os.Exit(0)
		}
	}

	err = C.UnmarshalKey("account", &C.Account)
	if err != nil {
		log.Warn("failed to unmarshal account config: ", err)
	} else {
		switch {
		case C.Account.Cookie != "":
			_, err := EHentai.SetCookieFromString(C.Account.Cookie)
			if err != nil {
				log.Fatal("failed to set cookie: ", err)
			}

		case C.Account.IpbMemberId != "" &&
			C.Account.IpbPassHash != "" &&
			C.Account.Igneous != "":
			if C.Account.Sk == "" {
				log.Warn("sk not set, the language of search results might be unexpected")
			}
			EHentai.SetCookie(
				C.Account.IpbMemberId,
				C.Account.IpbPassHash,
				C.Account.Igneous,
				C.Account.Sk,
			)

		default:
			log.Warn("cookie not set")
		}
	}

	err = C.UnmarshalKey("download", &C.Download)
	if err != nil {
		log.Fatal("failed to unmarshal download config: ", err)
	} else {
		if C.Download.Threads > 0 {
			EHentai.SetThreads(C.Download.Threads)
		}
		if C.Download.RetryDepth > 0 {
			EHentai.SetRetryDepth(C.Download.RetryDepth)
		}
		EHentai.SetUseEnvProxy(C.Download.UseEnvProxy)
		EHentai.SetDomainFronting(C.Download.DomainFronting)

		// 通过缓存功能实现下载
		// 同时还支持续传
		EHentai.SetAutoCacheEnabled(true)
		if C.Download.Dir == "" {
			C.Download.Dir = env.XDir
		}
		EHentai.SetCacheDir(C.Download.Dir)
	}
}
