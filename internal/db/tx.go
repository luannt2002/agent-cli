package db

import (
	"context"
	"database/sql"
	"errors"
)

// BeginTx mở transaction trên kết nối mà q dùng (q = New(conn)), để service
// chỉ giữ *Queries vẫn ghi được nhiều câu trong một lần, như session.Delete
// làm với *sql.DB (BeginTx rồi WithTx).
func (q *Queries) BeginTx(ctx context.Context) (*sql.Tx, error) {
	b, ok := q.db.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return nil, errors.New("db: queries are not bound to a database connection")
	}
	return b.BeginTx(ctx, nil)
}
