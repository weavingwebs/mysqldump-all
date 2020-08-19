package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func logIfError(fn func() error) {
	if err := fn(); err != nil {
		logrus.Error(err)
	}
}

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	rootCmd := &cobra.Command{
		Use: "mysqldump-all",
	}
	mysqlOpts := NewMysqlOpts{}
	rootCmd.PersistentFlags().StringVar(&mysqlOpts.DockerContainer, "docker", "", "docker container name to use for mysql")
	rootCmd.PersistentFlags().StringVar(&mysqlOpts.Host, "host", "", "mysql host")
	rootCmd.PersistentFlags().StringVar(&mysqlOpts.User, "user", "root", "mysql username")
	rootCmd.PersistentFlags().StringVar(&mysqlOpts.Pass, "pass", "", "mysql password")

	var noLock bool
	var noLocks []string
	var includeMysql bool
	var noDrop bool
	dumpCmd := &cobra.Command{
		Use: "dump",
		RunE: func(cmd *cobra.Command, args []string) error {
			dest := "dumps"
			if len(args) > 0 {
				dest = args[0]
			}

			return DumpAll(cmd.Context(), dest, DumpOptions{
				Mysql:   NewMySQL(mysqlOpts),
				NoLock:  noLock,
				NoLocks: noLocks,
			})
		},
	}
	dumpCmd.Flags().BoolVar(&noLock, "no-lock", false, "do not lock any databases while dumping")
	dumpCmd.Flags().StringSliceVar(&noLocks, "no-locks", []string{}, "comma separated list of database names that should not be locked during dump")
	rootCmd.AddCommand(dumpCmd)

	importCmd := &cobra.Command{
		Use: "import",
		RunE: func(cmd *cobra.Command, args []string) error {
			src := "dumps"
			if len(args) > 0 {
				src = args[0]
			}

			return ImportAll(cmd.Context(), src, ImportOptions{
				Mysql:             NewMySQL(mysqlOpts),
				IncludeMysqlTable: includeMysql,
				NoDrop:            noDrop,
			})
		},
	}
	importCmd.Flags().BoolVar(&includeMysql, "include-mysql", false, "import the mysql table as well if present (WARNING: Only do this when restoring to the same version of MySQL or MariaDB).")
	importCmd.Flags().BoolVar(&noDrop, "no-drop", false, "do not drop databases before importing")
	rootCmd.AddCommand(importCmd)

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}
