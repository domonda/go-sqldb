package sqldb

import (
	"database/sql"
	"testing"
)

func TestCheckTxOptionsCompatibility(t *testing.T) {
	type args struct {
		parent           *sql.TxOptions
		child            *sql.TxOptions
		defaultIsolation sql.IsolationLevel
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "nil, nil",
			args: args{
				parent: nil,
				child:  nil,
			},
		},
		{
			name: "nil, default",
			args: args{
				parent: nil,
				child:  &sql.TxOptions{},
			},
		},
		{
			name: "default, nil",
			args: args{
				parent: &sql.TxOptions{},
				child:  nil,
			},
		},
		{
			name: "default, default",
			args: args{
				parent: &sql.TxOptions{},
				child:  &sql.TxOptions{},
			},
		},
		{
			name: "nil, ReadOnly",
			args: args{
				parent: nil,
				child:  &sql.TxOptions{ReadOnly: true},
			},
		},
		{
			name: "ReadOnly, ReadOnly",
			args: args{
				parent: nil,
				child:  &sql.TxOptions{ReadOnly: true},
			},
			wantErr: false,
		},
		{
			name: "ReadOnly, nil",
			args: args{
				parent: &sql.TxOptions{ReadOnly: true},
				child:  nil,
			},
			wantErr: true,
		},
		{
			name: "ReadCommitted, ReadCommitted",
			args: args{
				parent: &sql.TxOptions{Isolation: sql.LevelReadCommitted},
				child:  &sql.TxOptions{Isolation: sql.LevelReadCommitted},
			},
		},
		{
			name: "Serializable, ReadCommitted",
			args: args{
				parent: &sql.TxOptions{Isolation: sql.LevelSerializable},
				child:  &sql.TxOptions{Isolation: sql.LevelReadCommitted},
			},
		},
		{
			name: "ReadCommitted, Serializable",
			args: args{
				parent: &sql.TxOptions{Isolation: sql.LevelReadCommitted},
				child:  &sql.TxOptions{Isolation: sql.LevelSerializable},
			},
			wantErr: true,
		},
		{
			name: "ReadCommitted, Serializable/ReadOnly",
			args: args{
				parent: &sql.TxOptions{Isolation: sql.LevelReadCommitted},
				child:  &sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: true},
			},
			wantErr: true,
		},
		{
			name: "ReadCommitted/ReadOnly, ReadCommitted/ReadOnly",
			args: args{
				parent: &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: true},
				child:  &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: true},
			},
		},
		{
			name: "Serializable/ReadOnly, ReadCommitted/ReadOnly",
			args: args{
				parent: &sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: true},
				child:  &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckTxOptionsCompatibility(tt.args.parent, tt.args.child, tt.args.defaultIsolation); (err != nil) != tt.wantErr {
				t.Errorf("CheckTxOptionsCompatibility() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNextTransactionNo(t *testing.T) {
	// Always returns >= 1
	if NextTransactionNo() < 1 {
		t.Fatal("NextTransactionNo() < 1")
	}
}
