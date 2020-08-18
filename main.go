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

	var noLock []string
	var dockerContainer string
	var includeMysql bool
	dumpCmd := &cobra.Command{
		Use: "dump",
		RunE: func(cmd *cobra.Command, args []string) error {
			dest := "dumps"
			if len(args) > 0 {
				dest = args[0]
			}

			return DumpAll(cmd.Context(), dest, DumpOptions{
				Mysql:   NewMySQL(dockerContainer),
				NoLocks: noLock,
			})
		},
	}
	dumpCmd.Flags().StringSliceVar(&noLock, "no-lock", []string{}, "comma separated list of database names that should not be locked during dump")
	dumpCmd.Flags().StringVar(&dockerContainer, "docker", "", "docker container name to use for mysql")
	rootCmd.AddCommand(dumpCmd)

	importCmd := &cobra.Command{
		Use: "import",
		RunE: func(cmd *cobra.Command, args []string) error {
			src := "dumps"
			if len(args) > 0 {
				src = args[0]
			}

			return ImportAll(cmd.Context(), src, ImportOptions{
				Mysql:             NewMySQL(dockerContainer),
				IncludeMysqlTable: includeMysql,
			})
		},
	}
	importCmd.Flags().StringVar(&dockerContainer, "docker", "", "docker container name to use for mysql")
	importCmd.Flags().BoolVar(&includeMysql, "include-mysql", false, "import the mysql table as well if present (WARNING: Only do this when restoring to the same version of MySQL or MariaDB.")
	rootCmd.AddCommand(importCmd)

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}
