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
		Use: "dump",
	}

	var noLock []string
	dumpCmd := &cobra.Command{
		Use: "dump",
		RunE: func(cmd *cobra.Command, args []string) error {
			dest := "dumps"
			if len(args) > 0 {
				dest = args[0]
			}

			return DumpAll(cmd.Context(), dest, noLock)
		},
	}
	dumpCmd.Flags().StringSliceVar(&noLock, "no-lock", []string{}, "comma separated list of database names that should not be locked during dump")
	rootCmd.AddCommand(dumpCmd)

	importCmd := &cobra.Command{
		Use: "import",
		RunE: func(cmd *cobra.Command, args []string) error {
			src := "dumps"
			if len(args) > 0 {
				src = args[0]
			}

			return ImportAll(cmd.Context(), src)
		},
	}
	rootCmd.AddCommand(importCmd)

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}
