package impl

import (
	"reflect"
	"testing"

	"github.com/domonda/go-sqldb"
	"github.com/stretchr/testify/require"
)

func TestReflectStructColumnPointers(t *testing.T) {
	type DeepEmbeddedStruct struct {
		DeeperEmbInt int `db:"deep_emb_int"`
	}
	type embeddedStruct struct {
		DeepEmbeddedStruct
		EmbInt int `db:"emb_int"`
	}
	type Struct struct {
		ID     string `db:"id,pk"`
		Int    int    `db:"int"`
		Ignore int    `db:"-"`
		embeddedStruct
		UntaggedField int
		Struct        struct {
			InlineStructInt int `db:"inline_struct_int"`
		} `db:"-"` // TODO enable access to named embedded fields?
		NilPtr *byte `db:"nil_ptr"`
	}
	var (
		structPtr       = new(Struct)
		structFieldPtrs = []any{
			&structPtr.ID,
			&structPtr.Int,
			&structPtr.DeeperEmbInt,
			&structPtr.EmbInt,
			// &structPtr.Struct.InlineStructInt,
			&structPtr.NilPtr,
		}
		structCols = []string{"id", "int", "deep_emb_int", "emb_int" /*"inline_struct_int",*/, "nil_ptr"}
	)

	type args struct {
		structVal reflect.Value
		mapper    sqldb.StructFieldMapper
		columns   []string
	}
	tests := []struct {
		name         string
		args         args
		wantPointers []any
		wantErr      bool
	}{
		{
			name: "ok",
			args: args{
				structVal: reflect.ValueOf(structPtr).Elem(),
				mapper:    sqldb.NewTaggedStructFieldMapping(),
				columns:   structCols,
			},
			wantPointers: structFieldPtrs,
		},

		// Errors:
		{
			name: "no columns",
			args: args{
				structVal: reflect.ValueOf(structPtr).Elem(),
				mapper:    sqldb.NewTaggedStructFieldMapping(),
				columns:   []string{},
			},
			wantErr: true,
		},
		{
			name: "not a struct",
			args: args{
				structVal: reflect.ValueOf(structPtr),
				mapper:    sqldb.NewTaggedStructFieldMapping(),
				columns:   structCols,
			},
			wantErr: true,
		},
		{
			name: "extra columns",
			args: args{
				structVal: reflect.ValueOf(structPtr).Elem(),
				mapper:    sqldb.NewTaggedStructFieldMapping(),
				columns:   append(structCols, "some_column_not_found_at_struct"),
			},
			wantErr: true,
		},
		{
			name: "not enough columns",
			args: args{
				structVal: reflect.ValueOf(structPtr).Elem(),
				mapper:    sqldb.NewTaggedStructFieldMapping(),
				columns:   structCols[1:],
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPointers, err := ReflectStructColumnPointers(tt.args.structVal, tt.args.mapper, tt.args.columns)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantPointers, gotPointers)
		})
	}
}
