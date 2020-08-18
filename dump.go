package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func dumpDB(ctx context.Context, db string, filePath string, noLock bool) error {
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

	flags := []string{"--single-transaction", "--quick"}
	if noLock {
		flags = append(flags, "--lock-tables=false")
	}
	mysql := fmt.Sprintf(
		`mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" %s "%s"`,
		strings.Join(flags, " "),
		db,
	)
	proc := exec.CommandContext(ctx, "docker", "exec", "-i", "mysql", "bash", "-c", mysql)
	proc.Stderr = os.Stderr
	proc.Stdout = pBarWriter
	if err := proc.Start(); err != nil {
		return errors.Wrapf(err, "failed to dump database %s", db)
	}

	if err := proc.Wait(); err != nil {
		return errors.Wrapf(err, "failed to dump database %s", db)
	}

	pBar.Finish()
	return nil
}

func DumpAll(ctx context.Context, dest string, noLocks []string) error {
	if err := os.MkdirAll(dest, 0770); err != nil {
		return errors.Wrapf(err, "failed to create %s", dest)
	}

	// Get list of databases.
	mysql := fmt.Sprintf(`mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -sN -e "show databases;"`)
	proc := exec.CommandContext(ctx, "docker", "exec", "-i", "mysql", "bash", "-c", mysql)
	proc.Stderr = os.Stderr
	stdOut, err := proc.StdoutPipe()
	if err != nil {
		return err
	}
	if err := proc.Start(); err != nil {
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
		if line != "" {
			dbs = append(dbs, line)
		}
	}

	if err := proc.Wait(); err != nil {
		return errors.Wrap(err, "failed to get database list")
	}
	logrus.Infof("Found %d databases", len(dbs))

	// Dump each database.
	for _, db := range dbs {
		noLock := false
		for _, l := range noLocks {
			if l == db {
				noLock = true
				break
			}
		}

		fp := filepath.Join(dest, db+".sql.gz")
		if err := dumpDB(ctx, db, fp, noLock); err != nil {
			return err
		}
	}

	return nil
}
