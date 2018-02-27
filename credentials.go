package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/defaults"
)

type CredentialCacheProvider struct {
	Creds   *credentials.Credentials
	Profile string
}

func (c *CredentialCacheProvider) Dir() string {
	return path.Join(
		path.Dir(defaults.SharedConfigFilename()),
		"ecs-local", "cache",
	)
}

func (c *CredentialCacheProvider) Retrieve() (credentials.Value, error) {
	cacheFile := path.Join(c.Dir(), fmt.Sprintf("%s_cache.json", c.Profile))

	// create config directory
	os.MkdirAll(c.Dir(), 0755)

	// attempt to read cached credentials
	if _, readerr := os.Stat(cacheFile); readerr == nil {
		var creds credentials.Value
		content, err := ioutil.ReadFile(cacheFile)
		if err != nil {
			return creds, err
		}
		err = json.Unmarshal(content, &creds)
		return creds, err
	}

	creds, err := c.Creds.Get()
	if err != nil {
		return creds, err
	}

	content, err := json.Marshal(creds)
	if err != nil {
		return creds, err
	}

	// only cache assumed roles for now
	if creds.ProviderName == "AssumeRoleProvider" {
		return creds, ioutil.WriteFile(cacheFile, content, 0600)
	}

	return creds, nil
}

func (c *CredentialCacheProvider) IsExpired() bool {
	return c.Creds.IsExpired()
}

func (c *CredentialCacheProvider) Remove() error {
	return os.Remove(path.Join(c.Dir(), fmt.Sprintf("%s_cache.json", c.Profile)))
}
