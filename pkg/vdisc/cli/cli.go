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
package vdisc_cli

import (
	"reflect"

	"github.com/alecthomas/kong"
	"github.com/google/uuid"
)

type Globals struct {
	LogLevel string      `help:"Set the logging level (debug|info|warn|error)" default:"info"`
	Cache    CacheConfig `embed prefix:"cache-"`
}

type CLI struct {
	Globals

	Burn    BurnCmd    `cmd help:"Burn creates a new vdisc"`
	Cache   CacheCmd   `cmd help:"Cache management"`
	Cp      CpCmd      `cmd help:"Copy a file from a vdisc to a local path"`
	Inspect InspectCmd `cmd help:"Inspect a vdisc"`
	Ls      LsCmd      `cmd help:"List directory contents"`
	Mount   MountCmd   `cmd help:"Mount a vdisc"`
	Tree    TreeCmd    `cmd help:"Print the file system hierarchy as a tree"`
	Version VersionCmd `cmd help:"Print the client version information"`
}

func UUIDDecoder(ctx *kong.DecodeContext, target reflect.Value) error {
	var value string
	if err := ctx.Scan.PopValueInto("uuid", &value); err != nil {
		return err
	}

	var u uuid.UUID
	if value != "" {
		var err error
		u, err = uuid.Parse(value)
		if err != nil {
			return err
		}
	}
	target.Set(reflect.ValueOf(u))
	return nil
}

func UUIDTypeMapper() kong.Option {
	var u uuid.UUID
	return kong.TypeMapper(reflect.TypeOf(u), kong.MapperFunc(UUIDDecoder))
}
