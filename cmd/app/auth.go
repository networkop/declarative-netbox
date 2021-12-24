package app

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

const (
	authFile = "config"
	authDir  = ".netbox"
)

type AuthData struct {
	Token      string `json:"token"`
	Server     string `json:"server"`
	configFile string
}

func NewAuthData() *AuthData {

	dirPath := filepath.Join(homeDir(), authDir)

	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err = os.MkdirAll(dirPath, 0755); err != nil {
			return nil
		}
	}

	filePath := filepath.Join(dirPath, authFile)
	return readAuth(filePath)
}

func readAuth(path string) *AuthData {
	result := &AuthData{
		configFile: path,
	}

	f, err := os.Open(result.configFile)
	if os.IsNotExist(err) {
		return result
	}

	raw, err := ioutil.ReadAll(f)
	if err != nil {
		log.Infof("failed to read file %s: %s", path, err)
		return result
	}

	err = json.Unmarshal(raw, result)
	if err != nil {
		log.Infof("failed to parse JSON %s: %s", path, err)
		return result
	}

	return result
}

func (c *AuthData) SaveAuth(s, t string) error {
	c.Token = t
	c.Server = s

	log.Debugf("Saving token data in %s", c.configFile)
	bytes, err := json.Marshal(c)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(c.configFile, bytes, 0600)
}

func homeDir() string {
	return os.Getenv("HOME")
}
