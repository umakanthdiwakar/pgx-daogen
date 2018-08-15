package main

import (
	"fmt"
	"time"

	"github.com/jackc/pgx"
)

// DBase - common structure for both individual and pooled connections
type DBase struct {
	ConnPool *pgx.ConnPool
}

// CreateConnection - create connection pool
func CreateConnection(hostname string, dbname string, userName string,
	password string, numConnections int) (*DBase, error) {
	e := DBase{}
	var err error
	connPoolConfig := pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			Host:     hostname,
			User:     userName,
			Password: password,
			Database: dbname,
		},
		MaxConnections: numConnections,
		AcquireTimeout: 5 * time.Second,
	}
	pool, err := pgx.NewConnPool(connPoolConfig)
	if err != nil {
		fmt.Println("***ERROR***", "Unable to create connection pool", err)
		return nil, err
	}
	fmt.Println("* Success *", "Connection pool established successfully ...")
	e.ConnPool = pool
	return &e, nil
}

//Close - to close the connection
func (dbconn *DBase) Close() {
	dbconn.ConnPool.Close()
}
