// Copyright © 2019 NVIDIA Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
