// Copyright © 2019 NVIDIA Corporation
package vdisc_cli

import (
	"reflect"

	"github.com/alecthomas/kong"
	"github.com/google/uuid"
)

type Globals struct {
	LogLevel string `help:"Set the logging level (debug|info|warn|error)" default:"info"`
}

type CLI struct {
	Globals

	Burn    BurnCmd    `cmd help:"Burn creates a new vdisc"`
	Mount   MountCmd   `cmd help:"Mount a vdisc"`
	Inspect InspectCmd `cmd help:"Inspect a vdisc"`
	Tree    TreeCmd    `cmd help:"Print the file system hierarchy as a tree"`
	Ls      LsCmd      `cmd help:"List directory contents"`
	Cp      CpCmd      `cmd help:"Copy a file from a vdisc to a local path"`
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
