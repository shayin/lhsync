package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"fmt"
	"strings"
	"path/filepath"
)

var (
	SConf *ServerConfig
	CConf *ClientConfig
)

type ClientConfig struct {
	Dest     string `yaml:"dest"`
	WatchDir map[string]string `yaml:"watch_dir"`
}

type ServerConfig struct {
	Listen    string            `yaml:"listen"`
	WhiteList []string          `yaml:"white_list"`
	SyncPath  map[string]string `yaml:"sync_path"`
}

func InitServerConfig(confFile string) error {
	SConf = &ServerConfig{}
	f, err := ioutil.ReadFile(confFile)
	if err != nil {
		return fmt.Errorf("read config file error: %s", err.Error())
	}

	err = yaml.Unmarshal(f, SConf)
	if err != nil {
		return fmt.Errorf("decode config file error: %s", err.Error())
	}
	return nil
}

func InitClientConfig(confFile string) error {
	CConf = &ClientConfig{}
	f, err := ioutil.ReadFile(confFile)
	if err != nil {
		return fmt.Errorf("read config file error: %s", err.Error())
	}

	err = yaml.Unmarshal(f, CConf)
	if err != nil {
		return fmt.Errorf("decode config file error: %s", err.Error())
	}
	return nil
}

func GetClientPathKey(path string) (string, error) {
	for kName, pathName := range CConf.WatchDir {
		if strings.Contains(filepath.Dir(path), pathName) {
			return kName, nil
		}
	}
	return "", fmt.Errorf("cannot find path key: <%+v> in <%s>", CConf.WatchDir, filepath.Dir(path))
}