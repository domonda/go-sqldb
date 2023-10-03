package sqldb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"
)

var (
	_ Connection   = new(ConnectionImpl)
	_ TxConnection = new(TxConnectionImpl)
)

type ConnectionImpl struct {
	StructFieldMapper
	QueryFormatter
	ArrayHandler
	Kind                   string
	DB                     *sql.DB
	Conf                   *Config
	ValidateColumnNameFunc func(string) error
	ValueConverter         driver.ValueConverter
}

func (conn *ConnectionImpl) String() string {
	return fmt.Sprintf("%s connection: %s", conn.DatabaseKind(), conn.Config().ConnectURL())
}

func (conn *ConnectionImpl) DatabaseKind() string {
	return conn.Kind
}

func (conn *ConnectionImpl) Ping(ctx context.Context, timeout time.Duration) error {
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return conn.DB.PingContext(ctx)
}

func (conn *ConnectionImpl) DBStats() sql.DBStats {
	return conn.DB.Stats()
}

func (conn *ConnectionImpl) Config() *Config {
	return conn.Conf
}

func (conn *ConnectionImpl) ValidateColumnName(name string) error {
	if conn.ValidateColumnNameFunc == nil {
		return nil
	}
	return conn.ValidateColumnNameFunc(name)
}

func (conn *ConnectionImpl) IsTransaction() bool {
	return false
}

func (conn *ConnectionImpl) Exec(ctx context.Context, query string, args ...any) (err error) {
	if conn.ValueConverter != nil {
		for i, value := range args {
			args[i], err = conn.ValueConverter.ConvertValue(value)
			if err != nil {
				return WrapErrorWithQuery(err, query, args, conn.QueryFormatter)
			}
		}
	}
	_, err = conn.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return WrapErrorWithQuery(err, query, args, conn.QueryFormatter)
	}
	return nil
}

func (conn *ConnectionImpl) Query(ctx context.Context, query string, args ...any) (rows Rows, err error) {
	if conn.ValueConverter != nil {
		for i, value := range args {
			args[i], err = conn.ValueConverter.ConvertValue(value)
			if err != nil {
				return nil, WrapErrorWithQuery(err, query, args, conn.QueryFormatter)
			}
		}
	}
	rows, err = conn.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, WrapErrorWithQuery(err, query, args, conn.QueryFormatter)
	}
	return rows, nil
}

func (conn *ConnectionImpl) Close() error {
	return conn.DB.Close()
}

type TxConnectionImpl struct {
	ConnectionImpl

	DefaultLevel sql.IsolationLevel
	Tx           *sql.Tx
	TxNo         uint64
	TxOpts       *sql.TxOptions
}

func (conn *TxConnectionImpl) Exec(ctx context.Context, query string, args ...any) (err error) {
	if conn.ValueConverter != nil {
		for i, value := range args {
			args[i], err = conn.ValueConverter.ConvertValue(value)
			if err != nil {
				return WrapErrorWithQuery(err, query, args, conn.QueryFormatter)
			}
		}
	}
	if conn.Tx != nil {
		_, err = conn.Tx.ExecContext(ctx, query, args...)
	} else {
		_, err = conn.DB.ExecContext(ctx, query, args...)
	}
	if err != nil {
		return WrapErrorWithQuery(err, query, args, conn.QueryFormatter)
	}
	return nil
}

func (conn *TxConnectionImpl) Query(ctx context.Context, query string, args ...any) (rows Rows, err error) {
	if conn.ValueConverter != nil {
		for i, value := range args {
			args[i], err = conn.ValueConverter.ConvertValue(value)
			if err != nil {
				return nil, WrapErrorWithQuery(err, query, args, conn.QueryFormatter)
			}
		}
	}
	if conn.Tx != nil {
		rows, err = conn.Tx.QueryContext(ctx, query, args...)
	} else {
		rows, err = conn.DB.QueryContext(ctx, query, args...)
	}
	if err != nil {
		return nil, WrapErrorWithQuery(err, query, args, conn.QueryFormatter)
	}
	return rows, nil
}

func (conn *TxConnectionImpl) IsTransaction() bool {
	return conn.Tx != nil
}

func (conn *TxConnectionImpl) DefaultIsolationLevel() sql.IsolationLevel {
	return conn.DefaultLevel
}

func (conn *TxConnectionImpl) TxNumber() uint64 {
	return conn.TxNo
}

func (conn *TxConnectionImpl) TxOptions() (*sql.TxOptions, bool) {
	return conn.TxOpts, conn.Tx != nil
}

func (conn *TxConnectionImpl) Begin(ctx context.Context, opts *sql.TxOptions, no uint64) (TxConnection, error) {
	if conn.Tx != nil {
		return nil, ErrWithinTransaction
	}
	tx, err := conn.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &TxConnectionImpl{
		ConnectionImpl: conn.ConnectionImpl,
		DefaultLevel:   conn.DefaultLevel,
		Tx:             tx,
		TxNo:           no,
		TxOpts:         opts,
	}, nil
}

func (conn *TxConnectionImpl) Commit() error {
	if conn.Tx == nil {
		return ErrNotWithinTransaction
	}
	return conn.Tx.Commit()
}

func (conn *TxConnectionImpl) Rollback() error {
	if conn.Tx == nil {
		return ErrNotWithinTransaction
	}
	return conn.Tx.Rollback()
}

func (conn *TxConnectionImpl) Close() error {
	if conn.Tx != nil {
		return conn.Tx.Rollback()
	}
	return conn.ConnectionImpl.Close()
}
