# contextとsql実行の関係を理解する
ポイントは、レイヤが低い順に、

github.com/lib/pq -> database/sql -> github.com/jinzhu/gorm 

とあり、どのレイヤでどのようなハンドリングが行われているかを理解すること

# database/sql
重要なエラーとしては、https://github.com/golang/go/blob/24b570354caee33d4fb3934ce7ef1cc97fb403fd/src/database/sql/sql.go#L2206

```
// ErrTxDone is returned by any operation that is performed on a transaction
// that has already been committed or rolled back.
var ErrTxDone = errors.New("sql: transaction has already been committed or rolled back")
```

がある。

https://github.com/golang/go/blob/24b570354caee33d4fb3934ce7ef1cc97fb403fd/src/database/sql/sql.go#L1831-L1841
に

```
// BeginTx starts a transaction.
//
// The provided context is used until the transaction is committed or rolled back.
// If the context is canceled, the sql package will roll back
// the transaction. Tx.Commit will return an error if the context provided to
// BeginTx is canceled.
//
// The provided TxOptions is optional and may be nil if defaults should be used.
// If a non-default isolation level is used that the driver doesn't support,
// an error will be returned.
func (db *DB) BeginTx(ctx context.Context, opts *TxOptions) (*Tx, error) {
```

とある。
具体的な処理は
https://github.com/golang/go/blob/24b570354caee33d4fb3934ce7ef1cc97fb403fd/src/database/sql/sql.go#L2183-L2198

```
// awaitDone blocks until the context in Tx is canceled and rolls back
// the transaction if it's not already done.
func (tx *Tx) awaitDone() {
	// Wait for either the transaction to be committed or rolled
	// back, or for the associated context to be closed.
	<-tx.ctx.Done()

	// Discard and close the connection used to ensure the
	// transaction is closed and the resources are released.  This
	// rollback does nothing if the transaction has already been
	// committed or rolled back.
	// Do not discard the connection if the connection knows
	// how to reset the session.
	discardConnection := !tx.keepConnOnRollback
	tx.rollback(discardConnection)
}
```
で、`awaitDone`は BeginTxの際に専用goroutineで実行される。

要するに、`BeginTx`に渡したコンテキストがキャンセルされた後に、`commit`/`rollback`を呼ぶと、`ErrTxDone`に遭遇する。

# lib/pq 
https://github.com/lib/pq/blob/006a3f492338e7f74b87a2c16d2c4be10cc04ae6/conn_go18.go#L102-L181
を見るとわかるように、Beginの際のcontextがキャンセルされるとDBサーバに実行中のクエリをキャンセルよう依頼する。これはおそらく`pg_cancel`と同じ。

実行中のクエリは以下のエラーで返却される。

`pq: canceling statement due to user request`

ただし、この挙動はdriverに依存するはずで、例えば https://github.com/jackc/pgx はサポートしてなかったはず。

# gorm
https://github.com/jinzhu/gorm/blob/5c235b72a414e448d1f441aba24a47fd6eb976f4/main.go#L581-L602

commit -> `ErrTxDone`の場合そのまま返す

rollback -> `ErrTxDone`の場合、エラーがなかったことにする

という奇妙な挙動になっている。

これは、contextがすでにキャンセル済でrollbackをcallしてもエラーが帰らないという意味では良いのかもしれないが、
commit済なのにrollbackをcallしても本当はrollbackできていないのに成功したような挙動になると思われ、それはいいのだろうか。

gorm V2の挙動は調べてないので、そこは注意


