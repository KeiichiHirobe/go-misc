package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

func main() {
	db, err := gorm.Open("postgres", "host=127.0.0.1 user=xxxxx_user dbname=xxxxx_local password=xxxxx_pass port=5433  sslmode=disable")
	if err != nil {
		println(err.Error())
	}
	db.LogMode(true)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	tx := db.BeginTx(ctx, nil)
	fmt.Println("start exec")
	// query canceled at https://github.com/lib/pq/blob/e7751f584844fbf92a5a18b13a0af1c855e34460/conn_go18.go#L90-L149
	err = tx.Exec("select pg_sleep(15);").Error
	if err != nil {
		fmt.Println("sql failed")
		fmt.Println(err.Error())
	} else {
		fmt.Println("sql succeeded")
	}
	if err := ctx.Err(); err != nil {
		fmt.Println(err.Error())
	}
	if err != nil {
		err = tx.Rollback().Error
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println("rollback failed")
			if errors.Is(err, sql.ErrTxDone) {
				fmt.Println("already transaction ended")
			}
		} else {
			fmt.Println("rollback succeeded")
		}
	} else {
		err = tx.Commit().Error
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println("commit failed")
			if errors.Is(err, sql.ErrTxDone) {
				// https://github.com/golang/go/blob/b345a306a0258085b65081cf2dadc238dc7e26ee/src/database/sql/sql.go#L2112
				// ErrTxDone is returned by any operation that is performed on a transaction
				// that has already been committed or rolled back.
				// var ErrTxDone = errors.New("sql: transaction has already been committed or rolled back")
				fmt.Println("already transaction ended")
			}
		} else {
			fmt.Println("commit succeeded")
		}
	}
	fmt.Println("finished")
}

/*
start exec
[2021-04-07 00:08:41]  pq: canceling statement due to user request
[2021-04-07 00:08:41]  [10002.84ms]  select pg_sleep(15);
[0 rows affected or returned ]
sql failed
pq: canceling statement due to user request
context deadline exceeded
rollback succeeded
finished
*/
