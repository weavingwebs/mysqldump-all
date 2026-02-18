package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

type MySQL struct {
	dockerContainer string
	host            string
	user            string
	pass            string
	client          string
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
	Client          string
}

func NewMySQL(opts NewMysqlOpts) *MySQL {
	if opts.DockerContainer == "" && opts.Pass == "" {
		fmt.Print("Password: ")
		pass, _ := terminal.ReadPassword(syscall.Stdin)
		opts.Pass = strings.TrimSpace(string(pass))
	}
	return &MySQL{
		dockerContainer: opts.DockerContainer,
		host:            opts.Host,
		user:            opts.User,
		pass:            opts.Pass,
		client:          opts.Client,
	}
}

func (m *MySQL) Exec(ctx context.Context, cmd string, args []string) *MySQLCmd {
	if m.client == "mariadb" {
		if cmd == "mysqldump" {
			cmd = "mariadb-dump --skip-opt"
		} else {
			cmd = strings.ReplaceAll(cmd, "mysql", "mariadb")
		}
	}

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
		// Unquote values, exec will escape for us.
		cleanArgs := make([]string, len(args))
		for i, a := range args {
			cleanArgs[i] = strings.Trim(a, `"`)
		}
		proc = exec.CommandContext(ctx, cmd, cleanArgs...)
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
