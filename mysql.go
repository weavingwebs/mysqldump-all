package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type MySQL struct {
	dockerContainer string
}

type MySQLCmd struct {
	Mysql *MySQL
	Proc  *exec.Cmd
}

func NewMySQL(dockerContainer string) *MySQL {
	return &MySQL{
		dockerContainer: dockerContainer,
	}
}

func (m *MySQL) Exec(ctx context.Context, cmd string, args []string) *MySQLCmd {
	var proc *exec.Cmd
	if m.dockerContainer == "" {
		args = append([]string{"-uroot", `-p"$MYSQL_ROOT_PASSWORD"`}, args...)
		proc = exec.CommandContext(ctx, cmd, args...)
	} else {
		mysql := fmt.Sprintf(
			`%s -uroot -p"$MYSQL_ROOT_PASSWORD" %s`,
			cmd,
			strings.Join(args, " "),
		)
		proc = exec.CommandContext(ctx, "docker", "exec", "-i", m.dockerContainer, "bash", "-c", mysql)
	}

	return &MySQLCmd{
		Mysql: m,
		Proc:  proc,
	}
}

func (mc *MySQLCmd) Run() error {
	if err := mc.Proc.Start(); err != nil {
		return err
	}
	return mc.Proc.Wait()
}
