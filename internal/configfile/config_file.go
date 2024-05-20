package configfile

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/qiniu/go-sdk/v7/internal/env"
)

type profileConfig struct {
	AccessKey             string      `toml:"access_key"`
	SecretKey             string      `toml:"secret_key"`
	BucketURL             interface{} `toml:"bucket_url"`
	DisableSecureProtocol bool        `toml:"disable_secure_protocol"`
}

var (
	profileConfigs      map[string]*profileConfig
	profileConfigsError error
	profileConfigsOnce  sync.Once
	ErrInvalidBucketUrl = errors.New("invalid bucket url")
)

func CredentialsFromConfigFile() (string, string, error) {
	profile, err := getProfile()
	if err != nil || profile == nil {
		return "", "", err
	} else if profile.AccessKey == "" || profile.SecretKey == "" {
		return "", "", nil
	}
	return profile.AccessKey, profile.SecretKey, nil
}

func BucketURLsFromConfigFile() ([]string, error) {
	profile, err := getProfile()
	if err != nil || profile == nil {
		return nil, err
	} else if profile.BucketURL == "" {
		return nil, nil
	}
	switch u := profile.BucketURL.(type) {
	case string:
		return strings.Split(u, ","), nil
	case []interface{}:
		var bucketUrls []string
		for _, v := range u {
			if s, ok := v.(string); ok {
				bucketUrls = append(bucketUrls, s)
			} else {
				return nil, ErrInvalidBucketUrl
			}
		}
		return bucketUrls, nil
	}
	return nil, ErrInvalidBucketUrl
}

func DisableSecureProtocolFromConfigFile() (bool, error) {
	profile, err := getProfile()
	if err != nil || profile == nil {
		return false, err
	}
	return profile.DisableSecureProtocol, nil
}

func getProfile() (*profileConfig, error) {
	if err := load(); err != nil {
		return nil, err
	}
	profileName := env.ProfileFromEnvironment()
	if profileName == "" {
		profileName = "default"
	}
	profile, ok := profileConfigs[profileName]
	if !ok || profile == nil {
		return nil, nil
	}
	return profile, nil
}

func load() error {
	profileConfigsOnce.Do(func() {
		profileConfigsError = _load()
	})
	return profileConfigsError
}

func _load() error {
	configFilePath := env.ConfigFileFromEnvironment()
	if configFilePath == "" {
		configFilePath = getDefaultConfigFilePath()
	}
	_, err := toml.DecodeFile(configFilePath, &profileConfigs)
	return err
}

func getDefaultConfigFilePath() string {
	homeDir, err := userHomeDir()
	if err != nil {
		homeDir = ""
	}
	return filepath.Join(homeDir, ".qiniu", "config.toml")
}
