package clickhouse

import (
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// NewClient 创建并返回一个 ClickHouse 客户端连接
func NewClient() (clickhouse.Conn, error) {
	host := os.Getenv("CLICKHOUSE_HOST")
	port := os.Getenv("CLICKHOUSE_PORT")
	user := os.Getenv("CLICKHOUSE_USER")
	password := os.Getenv("CLICKHOUSE_PASSWORD")
	secure := os.Getenv("CLICKHOUSE_SECURE") == "true"

	// 可选：支持指定默认数据库环境变量
	database := os.Getenv("CLICKHOUSE_DATABASE")
	if database == "" {
		database = "default"
	}

	options := clickhouse.Options{
		Addr:        []string{fmt.Sprintf("%s:%s", host, port)},
		Auth:        clickhouse.Auth{Database: database, Username: user, Password: password},
		DialTimeout: time.Second * 30,
	}
	if secure {
		// 如果需要跳过证书校验
		options.TLS = &tls.Config{InsecureSkipVerify: os.Getenv("CLICKHOUSE_VERIFY") != "true"}
	}

	conn, err := clickhouse.Open(&options)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
