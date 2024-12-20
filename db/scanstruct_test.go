package db

// func TestGetStructFieldIndices(t *testing.T) {
// 	type DeepEmbeddedStruct struct {
// 		DeeperEmbInt int `db:"deep_emb_int"`
// 	}
// 	type embeddedStruct struct {
// 		DeepEmbeddedStruct
// 		EmbInt int `db:"emb_int"`
// 	}
// 	type Struct struct {
// 		ID     string `db:"id,pk"`
// 		Int    int    `db:"int"`
// 		Ignore int    `db:"-"`
// 		embeddedStruct
// 		UntaggedField int
// 		Struct        struct {
// 			InlineStructInt int `db:"inline_struct_int"`
// 		}
// 		NilPtr *byte `db:"nil_ptr"`
// 	}

// 	fieldIndices := map[string][]int{
// 		"id":           {0},
// 		"int":          {1},
// 		"deep_emb_int": {3, 0, 0},
// 		"emb_int":      {3, 1},
// 		"nil_ptr":      {6},
// 	}
// 	naming := sqldb.StructFieldTagNaming{NameTag: "db", IgnoreName: "-", UntaggedNameFunc: sqldb.IgnoreStructField}

// 	type args struct {
// 		t     reflect.Type
// 		namer sqldb.StructFieldMapper
// 	}
// 	tests := []struct {
// 		name             string
// 		args             args
// 		wantFieldIndices map[string][]int
// 		wantErr          bool
// 	}{
// 		{name: "embedd", args: args{t: reflect.TypeOf(Struct{}), namer: naming}, wantFieldIndices: fieldIndices},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			gotFieldIndices, err := GetStructFieldIndices(tt.args.t, tt.args.namer)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("GetStructFieldIndices() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if !reflect.DeepEqual(gotFieldIndices, tt.wantFieldIndices) {
// 				t.Errorf("GetStructFieldIndices() = %v, want %v", gotFieldIndices, tt.wantFieldIndices)
// 			}
// 		})
// 	}
// }
