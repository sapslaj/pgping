package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jdxcode/netrc"
)

type Target struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	AppName  string
}

func (t *Target) FromConnString(s string) error {
	if !strings.Contains(s, "postgres://") {
		s = "postgres://" + s
	}
	parsed, err := pgx.ParseConfig(s)
	if err != nil {
		return err
	}
	t.Host = parsed.Host
	t.Port = int(parsed.Port)
	t.Database = parsed.Database
	t.User = parsed.User
	t.Password = parsed.Password
	return nil
}

func (t *Target) FromNetrc(path string) error {
	if path == "" {
		if env := os.Getenv("NETRC"); env != "" {
			path = env
		} else {
			base := ".netrc"
			if runtime.GOOS == "windows" {
				base = "_netrc"
			}
			usr, err := user.Current()
			if err != nil {
				return err
			}
			path = filepath.Join(usr.HomeDir, base)
		}
	}
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if stat.IsDir() {
		return nil
	}
	n, err := netrc.Parse(path)
	if err != nil {
		return err
	}
	machine := n.Machine(t.Host)
	if machine == nil {
		return nil
	}
	if t.User == "" {
		if username := machine.Get("login"); username != "" {
			t.User = username
		}
	}
	if t.Password == "" {
		if password := machine.Get("password"); password != "" {
			t.Password = password
		}
	}
	return nil
}

func (t *Target) FromFlags() error {
	if pgHost != nil && *pgHost != "" {
		t.Host = *pgHost
	}
	if pgPort != nil && *pgPort != "" {
		port, err := strconv.Atoi(*pgPort)
		if err != nil {
			return err
		}
		t.Port = port
	}
	if pgDatabase != nil && *pgDatabase != "" {
		t.Database = *pgDatabase
	}
	if pgUser != nil && *pgUser != "" {
		t.User = *pgUser
	}
	if pgPassword != nil && *pgPassword != "" {
		t.Password = *pgPassword
	}
	if pgAppName != nil && *pgAppName != "" {
		t.AppName = *pgAppName
	}
	return nil
}

func (t *Target) ToConnConfig() (*pgx.ConnConfig, error) {
	var connString strings.Builder
	connString.WriteString("postgres://")
	if t.User != "" {
		connString.WriteString(t.User)
		if t.Password != "" {
			connString.WriteString(":")
			connString.WriteString(t.Password)
		}
		connString.WriteString("@")
	}
	if t.Host != "" {
		connString.WriteString(t.Host)
	}
	if t.Port != 0 {
		connString.WriteString(":")
		connString.WriteString(fmt.Sprint(t.Port))
	}
	if t.Database != "" {
		connString.WriteString("/")
		connString.WriteString(t.Database)
	}
	connString.WriteString("?application_name=")
	if t.AppName == "" {
		connString.WriteString("pgping/" + VERSION)
	} else {
		connString.WriteString(t.AppName)
	}
	return pgx.ParseConfig(connString.String())
}
