package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"github.com/cheggaaa/pb/v3"
	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type DumpOptions struct {
	NoLock  bool
	NoLocks []string
	Mysql   *MySQL
}

func dumpDB(ctx context.Context, db string, filePath string, opts DumpOptions) error {
	noLock := opts.NoLock
	if !noLock {
		for _, l := range opts.NoLocks {
			if l == db {
				noLock = true
				break
			}
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

	var pBar *pb.ProgressBar
	if isatty.IsTerminal(os.Stdout.Fd()) {
		pBar = pb.New64(0).Set(pb.Bytes, true)
		pBar = pBar.SetTemplateString(`{{counters . }} @{{speed . }}`)
		pBar.Start()
		defer pBar.Finish()
	}

	args := []string{"--single-transaction", "--quick"}
	if noLock {
		args = append(args, "--lock-tables=false")
	}
	args = append(args, db)
	proc := opts.Mysql.Exec(ctx, "mysqldump", args)
	proc.Proc.Stderr = os.Stderr
	if pBar != nil {
		proc.Proc.Stdout = pBar.NewProxyWriter(gzipWriter)
	} else {
		proc.Proc.Stdout = gzipWriter
	}
	if err := proc.Run(); err != nil {
		return errors.Wrapf(err, "failed to dump database %s", db)
	}

	if pBar != nil {
		pBar.Finish()
	}
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
	start := time.Now()
	timings := make(Timings, 0)
	for _, db := range dbs {
		fp := filepath.Join(dest, db+".sql.gz")
		dbStart := time.Now()
		if err := dumpDB(ctx, db, fp, opts); err != nil {
			return err
		}
		timings = append(timings, Timing{Label: db, Duration: time.Since(dbStart)})
	}

	logrus.Infof("Dumped %d databases in %s 👍", len(dbs), time.Since(start))

	sort.Sort(sort.Reverse(timings))
	max := math.Min(float64(len(timings)), 5)
	msg := "Slowest dbs:\n"
	for i := 0; i < int(max); i++ {
		msg += "- " + timings[i].String() + "\n"
	}
	msg += "\n"
	logrus.Info(msg)

	return nil
}
