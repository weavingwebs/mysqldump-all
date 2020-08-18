package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type MySQL struct {
	dockerContainer string
	host            string
	user            string
	pass            string
}

type MySQLCmd struct {
	Mysql *MySQL
	Proc  *exec.Cmd
}

type NewMysqlOpts struct {
	DockerContainer string
	Host            string
	User            string
	Pass            string
}

func NewMySQL(opts NewMysqlOpts) *MySQL {
	return &MySQL{
		dockerContainer: opts.DockerContainer,
		host:            opts.DockerContainer,
		user:            opts.User,
		pass:            opts.Pass,
	}
}

func (m *MySQL) Exec(ctx context.Context, cmd string, args []string) *MySQLCmd {
	user := m.user
	if user == "" {
		user = "root"
	}
	pass := m.pass
	if pass == "" {
		pass = `"$MYSQL_ROOT_PASSWORD"`
	}
	if m.host != "" {
		args = append([]string{"-h" + m.host}, args...)
	}
	args = append([]string{"-u" + user, "-p" + pass}, args...)

	var proc *exec.Cmd
	if m.dockerContainer == "" {
		proc = exec.CommandContext(ctx, cmd, args...)
	} else {
		mysql := fmt.Sprintf(
			`%s %s`,
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
