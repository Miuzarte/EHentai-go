package env

import (
	"os"
	"path/filepath"
	"strings"
)

var (
	_       = initEnv() // 在全局变量中初始化
	XDir    string
	WorkDir string
	NoBuild bool // unit test or go build
	Testing bool // unit test
)

func initEnv() (_ struct{}) {
	XPath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	WorkDir, err = os.Getwd()
	if err != nil {
		panic(err)
	}
	NoBuild = strings.Contains(XPath, "go-build")
	Testing = strings.Contains(XPath, ".test")
	XDir = filepath.Dir(XPath)
	return
}
