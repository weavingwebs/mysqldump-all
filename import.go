package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

func importDb(ctx context.Context, db string, filePath string) error {
	logrus.Infof("Importing %s -> %s", filePath, db)

	// Create DB.
	if db != "mysql" {
		mysql := fmt.Sprintf(
			`mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -e "DROP DATABASE IF EXISTS %s; CREATE DATABASE %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"`,
			db,
			db,
		)
		proc := exec.CommandContext(ctx, "docker", "exec", "-i", "mysql", "bash", "-c", mysql)
		proc.Stderr = os.Stderr
		if err := proc.Start(); err != nil {
			return errors.Wrapf(err, "failed to create database: %s", db)
		}
		if err := proc.Wait(); err != nil {
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

	mysql := fmt.Sprintf(
		`mysql -uroot -p"$MYSQL_ROOT_PASSWORD" "%s"`,
		db,
	)
	proc := exec.CommandContext(ctx, "docker", "exec", "-i", "mysql", "bash", "-c", mysql)
	proc.Stderr = os.Stderr
	proc.Stdin = pBarReader
	if err := proc.Start(); err != nil {
		return errors.Wrapf(err, "failed to import database %s", db)
	}

	if err := proc.Wait(); err != nil {
		return errors.Wrapf(err, "failed to import database %s", db)
	}

	pBar.Finish()
	return nil
}

func ImportAll(ctx context.Context, src string) error {
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

		if err := importDb(ctx, db, filePath); err != nil {
			return err
		}
	}

	return nil
}
