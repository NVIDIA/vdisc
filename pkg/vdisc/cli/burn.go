// Copyright © 2019 NVIDIA Corporation
package vdisc_cli

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/NVIDIA/vdisc/pkg/storage"
	"github.com/NVIDIA/vdisc/pkg/vdisc"
)

type IsoOptions struct {
	NameValidation              string `help:"Restrictions on file names" enum:"portable,extended" default:"portable"`
	SystemIdentifier            string `help:"The name of the system that can act upon sectors 0x00-0x0F for the volume" default:"LINUX"`
	VolumeIdentifier            string `help:"Identification of this volume"`
	VolumeSetIdentifier         string `help:"Identifier of the volume set of which this volume is a member"`
	PublisherIdentifier         string `help:"The volume publisher"`
	DataPreparerIdentifier      string `help:"The identifier of the person(s) who prepared the data for this volume"`
	ApplicationIdentifier       string `help:"Identifies how the data are recorded on this volume" default:"NVIDIA VDISC"`
	CopyrightFileIdentifier     string `help:"Filename of a file in the root directory that contains copyright information for this volume set"`
	AbstractFileIdentifier      string `help:"Filename of a file in the root directory that contains abstract information for this volume set"`
	BibliographicFileIdentifier string `help:"Filename of a file in the root directory that contains bibliographic information for this volume set"`
}

type BurnCmd struct {
	Url string     `short:"o" help:"VDisc output URL" required:"true"`
	Csv string     `short:"i" help:"Path to a CSV" required:"true"`
	Iso IsoOptions `embed prefix:"iso9660-"`
}

func (cmd *BurnCmd) Run(globals *Globals) error {
	input, err := storage.Open(cmd.Csv)
	if err != nil {
		zap.L().Fatal("opening csv", zap.Error(err))
	}
	defer input.Close()

	r := csv.NewReader(input)
	r.ReuseRecord = true

	var b vdisc.Builder
	switch cmd.Iso.NameValidation {
	case "portable":
		b = vdisc.NewPosixPortableISO9660Builder(vdisc.BuilderConfig{
			URL: cmd.Url,
		})
	case "extended":
		b = vdisc.NewExtendedISO9660Builder(vdisc.BuilderConfig{
			URL: cmd.Url,
		})
	default:
		panic("never")
	}

	zap.L().Info("Burning visc...")

	// Set the iso9660 metadata
	if cmd.Iso.VolumeIdentifier == "" {
		id := uuid.NewSHA1(uuid.Nil, []byte(cmd.Url))
		b.SetVolumeIdentifier(fmt.Sprintf("%x", id))
	} else {
		b.SetVolumeIdentifier(cmd.Iso.VolumeIdentifier)
	}

	b.SetSystemIdentifier(cmd.Iso.SystemIdentifier)
	b.SetVolumeSetIdentifier(cmd.Iso.VolumeSetIdentifier)
	b.SetPublisherIdentifier(cmd.Iso.PublisherIdentifier)
	b.SetDataPreparerIdentifier(cmd.Iso.DataPreparerIdentifier)
	b.SetApplicationIdentifier(cmd.Iso.ApplicationIdentifier)
	b.SetCopyrightFileIdentifier(cmd.Iso.CopyrightFileIdentifier)
	b.SetAbstractFileIdentifier(cmd.Iso.AbstractFileIdentifier)
	b.SetBibliographicFileIdentifier(cmd.Iso.BibliographicFileIdentifier)

	// Add all the files from the CSV
	for {
		record, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			zap.L().Fatal("reading csv line", zap.Error(err))
		}

		size, err := strconv.ParseInt(record[2], 10, 64)
		if err != nil {
			zap.L().Fatal("parsing size", zap.Error(err))
		}

		if err := b.AddFile(record[0], record[1], size); err != nil {
			zap.L().Fatal("adding file", zap.Error(err))
		}
		zap.L().Debug("added file", zap.String("path", record[0]), zap.String("url", record[1]), zap.Int64("size", size))
	}

	url, err := b.Build()
	if err != nil {
		zap.L().Fatal("burning vdisc", zap.Error(err))
	}

	zap.L().Info("complete", zap.String("url", url))
	return nil
}
