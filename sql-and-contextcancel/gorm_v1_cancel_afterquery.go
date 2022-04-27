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
	db, err := gorm.Open("postgres", "host=127.0.0.1 user=xxxxx_user dbname=xxxxx_local password=xxxx_pass port=5433  sslmode=disable")
	if err != nil {
		println(err.Error())
	}
	db.LogMode(true)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	tx := db.BeginTx(ctx, nil)
	fmt.Println("start exec")
	err = tx.Exec("select pg_sleep(5);").Error
	if err != nil {
		fmt.Println("sql failed")
		fmt.Println(err.Error())
	} else {
		fmt.Println("sql succeeded")
	}
	time.Sleep(8 * time.Second)
	if err := ctx.Err(); err != nil {
		fmt.Println(err.Error())
	}
	// context already canceled, but application try to commit.
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
[2021-04-07 00:00:23]  [5002.64ms]  select pg_sleep(5);
[1 rows affected or returned ]
sql succeeded
context deadline exceeded
[2021-04-07 00:00:31]  sql: transaction has already been committed or rolled back
sql: transaction has already been committed or rolled back
commit failed
already transaction ended
finished
*/
