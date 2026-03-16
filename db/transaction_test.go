package db

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestSerializedTransaction(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	expectSerialized := func(ctx context.Context) error {
		if !IsTransaction(ctx) {
			panic("not in transaction")
		}
		if ctx.Value(serializedTransactionCtxKey{}) == nil {
			panic("no SerializedTransaction")
		}
		return nil
	}

	expectSerializedWithError := func(ctx context.Context) error {
		if !IsTransaction(ctx) {
			panic("not in transaction")
		}
		if ctx.Value(serializedTransactionCtxKey{}) == nil {
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
		{name: "flat call", args: args{ctx: ctx, txFunc: expectSerialized}, wantErr: false},
		{name: "expect error", args: args{ctx: ctx, txFunc: expectSerializedWithError}, wantErr: true},
		{name: "nested call", args: args{ctx: ctx, txFunc: nestedSerializedTransaction}, wantErr: false},
		{name: "nested tx call", args: args{ctx: ctx, txFunc: okNestedTransaction}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SerializedTransaction(tt.args.ctx, tt.args.txFunc)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTransaction(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	expectNonSerialized := func(ctx context.Context) error {
		if !IsTransaction(ctx) {
			panic("not in transaction")
		}
		if ctx.Value(serializedTransactionCtxKey{}) != nil {
			panic("SerializedTransaction")
		}
		return nil
	}

	expectNonSerializedWithError := func(ctx context.Context) error {
		if !IsTransaction(ctx) {
			panic("not in transaction")
		}
		if ctx.Value(serializedTransactionCtxKey{}) != nil {
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
		{name: "flat call", args: args{ctx: ctx, txFunc: expectNonSerialized}, wantErr: false},
		{name: "expected error", args: args{ctx: ctx, txFunc: expectNonSerializedWithError}, wantErr: true},
		{name: "nested call", args: args{ctx: ctx, txFunc: nestedTransaction}, wantErr: false},
		{name: "nested serialized", args: args{ctx: ctx, txFunc: nestedSerializedTransaction}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Transaction(tt.args.ctx, tt.args.txFunc)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
