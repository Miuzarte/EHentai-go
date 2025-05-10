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
	var err error
	XDir, err = os.Executable()
	if err != nil {
		panic(err)
	}
	WorkDir, err = os.Getwd()
	if err != nil {
		panic(err)
	}
	NoBuild = strings.Contains(XDir, "go-build")
	Testing = strings.Contains(XDir, ".test")
	XDir = filepath.Dir(XDir)
	return
}
