package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"github.com/cheggaaa/pb/v3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type DumpOptions struct {
	NoLocks []string
	Mysql   *MySQL
}

func dumpDB(ctx context.Context, db string, filePath string, opts DumpOptions) error {
	noLock := false
	for _, l := range opts.NoLocks {
		if l == db {
			noLock = true
			break
		}
	}

	logrus.Infof("Dumping %s -> %s (lock: %t)", db, filePath, !noLock)
	gzipFile, err := os.Create(filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to open %s", filePath)
	}
	defer logIfError(gzipFile.Close)
	gzipWriter := gzip.NewWriter(gzipFile)
	defer logIfError(gzipWriter.Close)

	pBar := pb.New64(0).Set(pb.Bytes, true)
	pBar = pBar.SetTemplateString(`{{counters . }} @{{speed . }}`)
	pBar.Start()
	defer pBar.Finish()
	pBarWriter := pBar.NewProxyWriter(gzipWriter)

	args := []string{"--single-transaction", "--quick"}
	if noLock {
		args = append(args, "--lock-tables=false")
	}
	args = append(args, db)
	proc := opts.Mysql.Exec(ctx, "mysqldump", args)
	proc.Proc.Stderr = os.Stderr
	proc.Proc.Stdout = pBarWriter
	if err := proc.Run(); err != nil {
		return errors.Wrapf(err, "failed to dump database %s", db)
	}

	pBar.Finish()
	return nil
}

func DumpAll(ctx context.Context, dest string, opts DumpOptions) error {
	if err := os.MkdirAll(dest, 0770); err != nil {
		return errors.Wrapf(err, "failed to create %s", dest)
	}

	// Get list of databases.
	proc := opts.Mysql.Exec(ctx, "mysql", []string{"-sN", "-e", `"show databases;"`})
	proc.Proc.Stderr = os.Stderr
	stdOut, err := proc.Proc.StdoutPipe()
	if err != nil {
		return err
	}
	if err := proc.Proc.Start(); err != nil {
		return errors.Wrap(err, "failed to get database list")
	}

	dbs := make([]string, 0)
	r := bufio.NewReader(stdOut)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "failed to read database list")
		}
		line = strings.TrimSpace(line)
		if line != "" && !(line == "performance_schema" || line == "information_schema") {
			dbs = append(dbs, line)
		}
	}

	if err := proc.Proc.Wait(); err != nil {
		return errors.Wrap(err, "failed to get database list")
	}
	logrus.Infof("Found %d databases", len(dbs))

	// Dump each database.
	for _, db := range dbs {
		fp := filepath.Join(dest, db+".sql.gz")
		if err := dumpDB(ctx, db, fp, opts); err != nil {
			return err
		}
	}

	return nil
}
