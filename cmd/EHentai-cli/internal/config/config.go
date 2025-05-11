package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Miuzarte/EHentai-go"
	"github.com/Miuzarte/EHentai-go/cmd/EHentai-cli/internal/errors"
	"github.com/Miuzarte/EHentai-go/internal/env"
	"github.com/Miuzarte/EHentai-go/internal/utils"
	"github.com/Miuzarte/SimpleLog"
	"github.com/spf13/viper"
)

//go:embed config.toml
var defaultConfig []byte

type logConfig struct {
	Level int
}

type accountConfig struct {
	Cookie string
	// OR
	IpbMemberId string
	IpbPassHash string
	Igneous     string
	Sk          string
}

type downloadConfig struct {
	ProgressBar    bool
	UseEnvProxy    bool
	DomainFronting bool
	Threads        int
	RetryDepth     int
	Dir            string
}

type searchConfig struct {
	DefaultSite         EHentai.Domain
	UseEhTagTranslation bool
	ShowTorrentDetails  bool
	Category            []string
	Cat                 EHentai.Category
}

var (
	Log      logConfig
	Account  accountConfig
	Download downloadConfig
	Search   searchConfig
)

var log = SimpleLog.New("[Config]", true, false)

func InitConfig() error {
	configPath := filepath.Join(env.XDir, "config.toml")

	viper.AddConfigPath(env.XDir)
	viper.SetConfigName("config")
	// Config.SetConfigType("toml") // 任意

	var err error

	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Panic("failed to read config file: ", err)
		} else {
			err = os.WriteFile(configPath, defaultConfig, 0o644)
			if err != nil {
				log.Panic("failed to create config file: ", err)
			}
			log.Infof("config.toml created at %s, please edit it", utils.HyperlinkFile(configPath))
			os.Exit(0)
		}
	}

	err = viper.UnmarshalKey("log", &Log)
	if err != nil {
		log.Warn("failed to unmarshal log config: ", err)
	}
	log.SetLevel(SimpleLog.Level(Log.Level))

	err = viper.UnmarshalKey("account", &Account)
	if err != nil {
		log.Warn("failed to unmarshal account config: ", err)
	} else {
		switch {
		case Account.Cookie != "":
			_, err := EHentai.SetCookieFromString(Account.Cookie)
			if err != nil {
				log.Error("failed to parse cookie: ", err)
				return errors.Handled
			}

		case Account.IpbMemberId != "" &&
			Account.IpbPassHash != "" &&
			Account.Igneous != "":
			if Account.Sk == "" {
				log.Warn("sk not set, the language of search results might be unexpected")
			}
			EHentai.SetCookie(
				Account.IpbMemberId,
				Account.IpbPassHash,
				Account.Igneous,
				Account.Sk,
			)

		default:
			log.Warn("cookie not set")
		}
	}

	err = viper.UnmarshalKey("download", &Download)
	if err != nil {
		log.Error("failed to unmarshal download config: ", err)
		return errors.Handled
	} else {
		if Download.Threads > 0 {
			EHentai.SetThreads(Download.Threads)
		}
		if Download.RetryDepth > 0 {
			EHentai.SetRetryDepth(Download.RetryDepth)
		}
		EHentai.SetUseEnvProxy(Download.UseEnvProxy)
		EHentai.SetDomainFronting(Download.DomainFronting)

		// 通过缓存功能实现下载
		// 同时还支持续传
		EHentai.SetAutoCacheEnabled(true)
		if Download.Dir == "" {
			Download.Dir = env.XDir
		}
		EHentai.SetCacheDir(Download.Dir)
	}

	err = viper.UnmarshalKey("search", &Search)
	if err != nil {
		log.Error("failed to unmarshal search config: ", err)
		return errors.Handled
	} else {
		if Search.DefaultSite == "" {
			Search.DefaultSite = EHentai.EHENTAI_DOMAIN
		} else {
			switch {
			case strings.Contains(Search.DefaultSite, "e-h") ||
				strings.Contains(Search.DefaultSite, "eh"):
				Search.DefaultSite = EHentai.EHENTAI_DOMAIN
			case strings.Contains(Search.DefaultSite, "ex"):
				Search.DefaultSite = EHentai.EXHENTAI_DOMAIN
			default:
				log.Errorf("invalid search default site: %s, use '%s' or '%s'", Search.DefaultSite, EHentai.EHENTAI_DOMAIN, EHentai.EXHENTAI_DOMAIN)
				if !utils.WaitAck(fmt.Sprintf("use %s to continue?", EHentai.EHENTAI_DOMAIN)) {
					return errors.Handled
				}
			}
		}
		Search.Cat, err = EHentai.ParseCategory(Search.Category...)
		if err != nil {
			log.Error("failed to parse category: ", err)
			if !utils.WaitAck("continue?") {
				return errors.Handled
			}
		}
	}

	return nil
}
