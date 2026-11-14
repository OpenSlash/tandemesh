// Package pgdriver adapts pgx so the store can keep writing "?" placeholders.
//
// The control plane's SQL is written for SQLite. Instead of rewriting every
// query, this driver converts "?" placeholders to "$1..$n" at the driver
// boundary, leaving the store readable and portable.
package pgdriver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/stdlib"
)

func init() {
	sql.Register("orca-postgres", &Driver{base: stdlib.GetDefaultDriver()})
}

// Driver wraps the pgx driver.
type Driver struct{ base driver.Driver }

// Open implements driver.Driver.
func (d *Driver) Open(name string) (driver.Conn, error) {
	c, err := d.base.Open(name)
	if err != nil {
		return nil, err
	}
	return &conn{Conn: c}, nil
}

type conn struct{ driver.Conn }

func (c *conn) Prepare(query string) (driver.Stmt, error) {
	return c.Conn.Prepare(Rebind(query))
}

func (c *conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if pc, ok := c.Conn.(driver.ConnPrepareContext); ok {
		return pc.PrepareContext(ctx, Rebind(query))
	}
	return c.Prepare(query)
}

func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if q, ok := c.Conn.(driver.QueryerContext); ok {
		return q.QueryContext(ctx, Rebind(query), args)
	}
	return nil, driver.ErrSkip
}

func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if e, ok := c.Conn.(driver.ExecerContext); ok {
		return e.ExecContext(ctx, Rebind(query), args)
	}
	return nil, driver.ErrSkip
}

func (c *conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if b, ok := c.Conn.(driver.ConnBeginTx); ok {
		return b.BeginTx(ctx, opts)
	}
	return nil, driver.ErrSkip
}

func (c *conn) Ping(ctx context.Context) error {
	if p, ok := c.Conn.(driver.Pinger); ok {
		return p.Ping(ctx)
	}
	return nil
}

func (c *conn) ResetSession(ctx context.Context) error {
	if r, ok := c.Conn.(driver.SessionResetter); ok {
		return r.ResetSession(ctx)
	}
	return nil
}

func (c *conn) IsValid() bool {
	if v, ok := c.Conn.(driver.Validator); ok {
		return v.IsValid()
	}
	return true
}

// Rebind converts "?" placeholders to "$n", skipping single-quoted literals.
func Rebind(query string) string {
	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 1
	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '\'' {
			b.WriteByte(ch)
			i++
			for ; i < len(query); i++ {
				b.WriteByte(query[i])
				if query[i] == '\'' {
					break
				}
			}
			continue
		}
		if ch == '?' {
			fmt.Fprintf(&b, "$%d", n)
			n++
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}
