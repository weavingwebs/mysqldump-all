package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"regexp"
)

type ImportOptions struct {
	Mysql             *MySQL
	IncludeMysqlTable bool
	NoDrop            bool
}

func importDb(ctx context.Context, db string, filePath string, opts ImportOptions) error {
	logrus.Infof("Importing %s -> %s", filePath, db)

	// Create DB.
	if !(db == "mysql" || db == "performance_schema" || db == "information_schema") {
		sql := `"`
		if !opts.NoDrop {
			sql += fmt.Sprintf(
				`DROP DATABASE IF EXISTS %s;`,
				db,
			)
		}
		sql += fmt.Sprintf(
			`CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;`,
			db,
		)
		sql += `"`

		proc := opts.Mysql.Exec(ctx, "mysql", []string{"-e", sql})
		proc.Proc.Stderr = os.Stderr
		if err := proc.Run(); err != nil {
			return errors.Wrapf(err, "failed to create database: %s", db)
		}
	}

	gzipFile, err := os.Open(filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to open %s", filePath)
	}
	defer logIfError(gzipFile.Close)

	pBar := pb.New64(0).Set(pb.Bytes, true)
	pBar = pBar.SetTemplateString(`{{counters . }} @{{speed . }}`)
	pBar.Start()
	defer pBar.Finish()

	gzipReader, err := gzip.NewReader(gzipFile)
	if err != nil {
		return errors.Wrapf(err, "failed to open reader for %s", filePath)
	}
	defer logIfError(gzipReader.Close)
	pBarReader := pBar.NewProxyReader(gzipReader)

	proc := opts.Mysql.Exec(ctx, "mysql", []string{db})
	proc.Proc.Stderr = os.Stderr
	proc.Proc.Stdin = pBarReader
	if err := proc.Run(); err != nil {
		return errors.Wrapf(err, "failed to import database %s", db)
	}

	pBar.Finish()
	return nil
}

func ImportAll(ctx context.Context, src string, opts ImportOptions) error {
	fRegex := regexp.MustCompile(`^(.+)\.sql\.gz`)

	dir, err := os.Open(src)
	if err != nil {
		return errors.Wrapf(err, "failed to read directory %s", src)
	}
	files, err := dir.Readdirnames(0)
	if err != nil {
		return errors.Wrapf(err, "failed to read directory contents of %s", src)
	}

	for _, file := range files {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		filePath := filepath.Join(src, file)
		matches := fRegex.FindStringSubmatch(file)
		if len(matches) < 2 {
			logrus.Debugf("Ignoring file %s", filePath)
			continue
		}
		db := matches[1]

		if db == "performance_schema" || db == "information_schema" {
			logrus.Debugf("Ignoring %s table", db)
		}
		if db == "mysql" && !opts.IncludeMysqlTable {
			logrus.Debug("Ignoring mysql table")
			continue
		}

		if err := importDb(ctx, db, filePath, opts); err != nil {
			return err
		}
	}

	return nil
}
