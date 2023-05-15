package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/jackc/pgx/v5"
	"golang.org/x/term"
)

var (
	count   = kingpin.Flag("count", "stop after N pings").Default("-1").Short('c').Int()
	wait    = kingpin.Flag("wait", "wait time between sending each ping").Default("1s").Short('i').Duration()
	timeout = kingpin.Flag("timeout", "timeout for connections to the DB").Default("5s").Short('t').Duration()
	query   = kingpin.Flag("query", "Test query to execute on database").Default("SELECT 1").String()

	pgHost     = kingpin.Flag("pg-host", "").String()
	pgPort     = kingpin.Flag("pg-port", "").String()
	pgDatabase = kingpin.Flag("pg-database", "").String()
	pgUser     = kingpin.Flag("pg-user", "").String()
	pgPassword = kingpin.Flag("pg-password", "").String()
	pgAppName  = kingpin.Flag("pg-app-name", "").Default("pgping/" + VERSION).String()

	promptPassword = kingpin.Flag("prompt-password", "prompt for password").Short('p').Bool()

	target = kingpin.Arg("target", "").String()
)

func kv(key string, value any) string {
	switch v := value.(type) {
	case string, error:
		value = fmt.Sprintf("%q", v)
	}
	return fmt.Sprintf("%s=%v", key, value)
}

func result(i int, start time.Time, kvs ...string) time.Duration {
	if kvs == nil {
		kvs = make([]string, 0)
	}
	duration := time.Since(start)
	kvs = append(kvs, kv("i", i))
	kvs = append(kvs, kv("duration", duration))
	var format strings.Builder
	format.WriteString(fmt.Sprintf("%-25s", time.Now().Format("2006-01-02T15:04:05.999Z")))
	for i, label := range kvs {
		if i > 0 {
			format.WriteString(" ")
		}
		format.WriteString(label)
	}
	fmt.Println(format.String())
	return duration
}

func ping(parent context.Context, connConfig *pgx.ConnConfig, i int) (bool, time.Duration) {
	ctx, cancel := context.WithTimeout(parent, *timeout)
	defer cancel()
	start := time.Now()
	conn, err := pgx.ConnectConfig(ctx, connConfig)
	if err != nil {
		return false, result(i, start, kv("status", "ERR"), kv("msg", "error connecting"), kv("err", err))
	}
	rows, err := conn.Query(ctx, *query)
	if err != nil {
		return false, result(i, start, kv("status", "ERR"), kv("msg", "error querying"), kv("err", err))
	}
	err = conn.Close(ctx)
	if err != nil {
		return false, result(i, start, kv("status", "ERR"), kv("msg", "error closing"), kv("err", err))
	}
	if rows.Next() {
		return true, result(i, start, kv("status", "OK"), kv("host", connConfig.Host))
	}
	return false, result(i, start, kv("status", "FAIL"), kv("host", connConfig.Host), kv("msg", "0 rows returned"))
}

func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytepw, err := term.ReadPassword(syscall.Stdin)
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(bytepw), nil
}

func buildTarget() *Target {
	t := &Target{}
	err := t.FromNetrc("")
	if err != nil {
		panic(err)
	}
	err = t.FromEnv(os.Getenv)
	if err != nil {
		panic(err)
	}
	err = t.FromFlags()
	if err != nil {
		panic(err)
	}
	if target != nil && *target != "" {
		err = t.FromConnString(*target)
		if err != nil {
			panic(err)
		}
	}
	if promptPassword != nil && *promptPassword {
		password, err := readPassword("Password: ")
		if err != nil {
			panic(err)
		}
		t.Password = password
	}
	return t
}

func main() {
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Parse()
	ctx := context.Background()
	t := buildTarget()

	connConfig, err := t.ToConnConfig()
	if err != nil {
		panic(err)
	}

	for i := 1; *count == -1 || i <= *count; i++ {
		pass, duration := ping(ctx, connConfig, i)
		if i == *count {
			if pass {
				os.Exit(0)
			} else {
				os.Exit(1)
			}
		}
		timeUntilNext := *wait - duration
		if timeUntilNext > 0 {
			time.Sleep(timeUntilNext)
		}
	}
}
