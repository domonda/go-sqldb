package db

/*
import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/domonda/go-sqldb/mockconn"
)

func TestSerializedTransaction(t *testing.T) {
	globalConn = mockconn.New(context.Background(), os.Stdout, nil)

	expectSerialized := func(ctx context.Context) error {
		if !IsTransaction(ctx) {
			panic("not in transaction")
		}
		if ctx.Value(&serializedTransactionCtxKey) == nil {
			panic("no SerializedTransaction")
		}
		return nil
	}

	expectSerializedWithError := func(ctx context.Context) error {
		if !IsTransaction(ctx) {
			panic("not in transaction")
		}
		if ctx.Value(&serializedTransactionCtxKey) == nil {
			panic("no SerializedTransaction")
		}
		return errors.New("expected error")
	}

	nestedSerializedTransaction := func(ctx context.Context) error {
		return SerializedTransaction(ctx, expectSerialized)
	}

	okNestedTransaction := func(ctx context.Context) error {
		return Transaction(ctx, nestedSerializedTransaction)
	}

	type args struct {
		ctx    context.Context
		txFunc func(context.Context) error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "flat call", args: args{ctx: context.Background(), txFunc: expectSerialized}, wantErr: false},
		{name: "expect error", args: args{ctx: context.Background(), txFunc: expectSerializedWithError}, wantErr: true},
		{name: "nested call", args: args{ctx: context.Background(), txFunc: nestedSerializedTransaction}, wantErr: false},
		{name: "nested tx call", args: args{ctx: context.Background(), txFunc: okNestedTransaction}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SerializedTransaction(tt.args.ctx, tt.args.txFunc); (err != nil) != tt.wantErr {
				t.Errorf("SerializedTransaction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransaction(t *testing.T) {
	globalConn = mockconn.New(context.Background(), os.Stdout, nil)

	expectNonSerialized := func(ctx context.Context) error {
		if !IsTransaction(ctx) {
			panic("not in transaction")
		}
		if ctx.Value(&serializedTransactionCtxKey) != nil {
			panic("SerializedTransaction")
		}
		return nil
	}

	expectNonSerializedWithError := func(ctx context.Context) error {
		if !IsTransaction(ctx) {
			panic("not in transaction")
		}
		if ctx.Value(&serializedTransactionCtxKey) != nil {
			panic("SerializedTransaction")
		}
		return errors.New("expected error")
	}

	nestedTransaction := func(ctx context.Context) error {
		return Transaction(ctx, expectNonSerialized)
	}

	nestedSerializedTransaction := func(ctx context.Context) error {
		return SerializedTransaction(ctx, nestedTransaction)
	}

	type args struct {
		ctx    context.Context
		txFunc func(context.Context) error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "flat call", args: args{ctx: context.Background(), txFunc: expectNonSerialized}, wantErr: false},
		{name: "expected error", args: args{ctx: context.Background(), txFunc: expectNonSerializedWithError}, wantErr: true},
		{name: "nested call", args: args{ctx: context.Background(), txFunc: nestedTransaction}, wantErr: false},
		{name: "nested serialized", args: args{ctx: context.Background(), txFunc: nestedSerializedTransaction}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Transaction(tt.args.ctx, tt.args.txFunc); (err != nil) != tt.wantErr {
				t.Errorf("Transaction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
*/
