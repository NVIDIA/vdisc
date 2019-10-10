// Copyright © 2019 NVIDIA Corporation
package swiftdriver

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

const (
	SwiftCtxProviderName = "SwiftCtxProvider"
	SwiftEnvProviderName = "SwiftEnvProvider"
)

const (
	SwiftAccessKeyId     = "SWIFT_ACCESS_KEY_ID"
	SwiftSecretAccessKey = "SWIFT_SECRET_ACCESS_KEY"
)

var (
	ErrSwiftAccessKeyIDNotFound     = fmt.Errorf("%s not found in environment", SwiftAccessKeyId)
	ErrSwiftSecretAccessKeyNotFound = fmt.Errorf("%s not found in environment", SwiftSecretAccessKey)
)

type SwiftEnvProvider struct {
	account   string
	retrieved bool
}

func NewSwiftEnvProvider(account string) *SwiftEnvProvider {
	return &SwiftEnvProvider{account, false}
}

func (e *SwiftEnvProvider) IsExpired() bool {
	return !e.retrieved
}

func (e *SwiftEnvProvider) Retrieve() (credentials.Value, error) {
	e.retrieved = false

	id := os.Getenv(SwiftAccessKeyId)
	secret := os.Getenv(SwiftSecretAccessKey)

	if id == "" {
		return credentials.Value{ProviderName: SwiftEnvProviderName}, ErrSwiftAccessKeyIDNotFound
	}

	if secret == "" {
		return credentials.Value{ProviderName: SwiftEnvProviderName}, ErrSwiftSecretAccessKeyNotFound
	}

	if id != e.account {
		id = fmt.Sprintf("%s:AUTH_%s", id, e.account)
	}

	e.retrieved = true
	return credentials.Value{
		AccessKeyID:     id,
		SecretAccessKey: secret,
		SessionToken:    "",
		ProviderName:    SwiftEnvProviderName,
	}, nil
}

type SwiftCreds struct {
	AccessKeyId     string
	SecretAccessKey string
}

type SwiftCtxProvider struct {
	account string
	creds   *SwiftCreds
}

func NewSwiftCtxProvider(account string, creds *SwiftCreds) *SwiftCtxProvider {
	return &SwiftCtxProvider{account, creds}
}

func (e *SwiftCtxProvider) IsExpired() bool {
	return false
}

func (e *SwiftCtxProvider) Retrieve() (credentials.Value, error) {
	id := e.creds.AccessKeyId
	if id != e.account {
		id = fmt.Sprintf("%s:AUTH_%s", id, e.account)
	}

	return credentials.Value{
		AccessKeyID:     id,
		SecretAccessKey: e.creds.SecretAccessKey,
		SessionToken:    "",
		ProviderName:    SwiftCtxProviderName,
	}, nil
}
