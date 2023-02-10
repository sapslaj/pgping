package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/jackc/pgx/v5"
)

var (
	count   = kingpin.Flag("count", "stop after N pings").Default("-1").Short('c').Int()
	wait    = kingpin.Flag("wait", "wait time between sending each ping").Default("1s").Short('i').Duration()
	timeout = kingpin.Flag("timeout", "timeout for connections to the DB").Default("5s").Short('t').Duration()
	query   = kingpin.Flag("query", "Test query to execute on database").Default("SELECT 1").String()

	pgHost     = kingpin.Flag("pg-host", "").Envar("PGHOST").String()
	pgPort     = kingpin.Flag("pg-port", "").Envar("PGPORT").String()
	pgDatabase = kingpin.Flag("pg-database", "").Envar("PGDATABASE").String()
	pgUser     = kingpin.Flag("pg-user", "").Envar("PGUSER").String()
	pgPassword = kingpin.Flag("pg-password", "").Envar("PGPASSWORD").String()
	pgAppName  = kingpin.Flag("pg-app-name", "").Default("pgping/0.1.0").Envar("PGAPPNAME").String()
)

func kv(key string, value any) string {
	switch v := value.(type) {
	case string:
		value = fmt.Sprintf("%q", v)
	}
	return fmt.Sprintf("%s=%v", key, value)
}

func result(i int, start time.Time, kvs ...string) time.Duration {
	if kvs == nil {
		kvs = make([]string, 2)
	}
	duration := time.Since(start)
	kvs = append(kvs, kv("i", i))
	kvs = append(kvs, kv("duration", duration))
	var format strings.Builder
	format.WriteString(time.Now().Format("2006-01-02T15:04:05.999Z") + " ")
	for i, label := range kvs {
		if i > 0 {
			format.WriteString(" ")
		}
		format.WriteString(label)
	}
	fmt.Println(format.String())
	return duration
}

func ping(ctx context.Context, connConfig *pgx.ConnConfig, i int) time.Duration {
	start := time.Now()
	conn, err := pgx.ConnectConfig(ctx, connConfig)
	if err != nil {
		return result(i, start, kv("msg", "error connecting"), kv("err", err))
	}
	rows, err := conn.Query(ctx, *query)
	if err != nil {
		return result(i, start, kv("msg", "error querying"), kv("err", err))
	}
	err = conn.Close(ctx)
	if err != nil {
		return result(i, start, kv("msg", "error closing"), kv("err", err))
	}
	if rows.Next() {
		return result(i, start, kv("status", "OK"), kv("host", connConfig.Host))
	}
	return result(i, start, kv("status", "FAIL"), kv("host", connConfig.Host))
}

func main() {
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Parse()
	ctx := context.Background()
	var connString strings.Builder
	connString.WriteString("postgres://")
	if pgUser != nil {
		connString.WriteString(*pgUser)
		if pgPassword != nil {
			connString.WriteString(":")
			connString.WriteString(*pgPassword)
		}
		connString.WriteString("@")
	}
	if pgHost != nil {
		connString.WriteString(*pgHost)
	}
	if pgPort != nil {
		connString.WriteString(":")
		connString.WriteString(*pgPort)
	}
	if pgDatabase != nil {
		connString.WriteString("/")
		connString.WriteString(*pgDatabase)
	}
	connString.WriteString("?application_name=")
	connString.WriteString(*pgAppName)

	connConfig, err := pgx.ParseConfig(connString.String())
	if err != nil {
		panic(err)
	}
	connConfig.Config.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, *timeout)
	}

	for i := 1; *count == -1 || i <= *count; i++ {
		duration := ping(ctx, connConfig, i)
		timeUntilNext := *wait - duration
		if timeUntilNext > 0 {
			time.Sleep(timeUntilNext)
		}
	}
}
