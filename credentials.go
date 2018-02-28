package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/defaults"
)

type cachedCredentials struct {
	credentials.Value
	Expiration time.Time
}

func (c *cachedCredentials) isExpired() bool {
	return c.Expiration.Before(time.Now().UTC())
}

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
	cacheFile := path.Join(c.Dir(), fmt.Sprintf("profile-%s.json", c.Profile))

	// create config directory
	os.MkdirAll(c.Dir(), 0755)

	// attempt to read cached credentials
	if _, readerr := os.Stat(cacheFile); readerr == nil {
		var creds *cachedCredentials
		content, err := ioutil.ReadFile(cacheFile)
		if err != nil {
			return credentials.Value{}, err
		}
		err = json.Unmarshal(content, &creds)

		if creds.isExpired() {
			c.Creds.Expire()
		} else {
			return creds.Value, nil
		}
	}

	creds, err := c.Creds.Get()
	if err != nil {
		return creds, err
	}

	switch creds.ProviderName {
	case stscreds.ProviderName:
		cache := &cachedCredentials{creds, time.Now().UTC().Add(stscreds.DefaultDuration)}
		content, err := json.Marshal(cache)
		if err != nil {
			return creds, err
		}

		return creds, ioutil.WriteFile(cacheFile, content, 0600)
	}

	return creds, nil
}

func (c *CredentialCacheProvider) IsExpired() bool {
	return c.Creds.IsExpired()
}
