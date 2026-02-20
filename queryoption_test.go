package sqldb

import (
	"reflect"
	"testing"
)

func TestIgnoreColumns(t *testing.T) {
	filter := IgnoreColumns("name", "active")

	tests := []struct {
		col  ColumnInfo
		want bool
	}{
		{col: ColumnInfo{Name: "name"}, want: true},
		{col: ColumnInfo{Name: "active"}, want: true},
		{col: ColumnInfo{Name: "id"}, want: false},
		{col: ColumnInfo{Name: "email"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.col.Name, func(t *testing.T) {
			if got := filter.IgnoreColumn(&tt.col); got != tt.want {
				t.Errorf("IgnoreColumns(%q) = %v, want %v", tt.col.Name, got, tt.want)
			}
		})
	}
}

func TestOnlyColumns(t *testing.T) {
	filter := OnlyColumns("id", "name")

	tests := []struct {
		col  ColumnInfo
		want bool
	}{
		{col: ColumnInfo{Name: "id"}, want: false},       // not ignored
		{col: ColumnInfo{Name: "name"}, want: false},     // not ignored
		{col: ColumnInfo{Name: "active"}, want: true},    // ignored
		{col: ColumnInfo{Name: "created"}, want: true},   // ignored
	}
	for _, tt := range tests {
		t.Run(tt.col.Name, func(t *testing.T) {
			if got := filter.IgnoreColumn(&tt.col); got != tt.want {
				t.Errorf("OnlyColumns ignore %q = %v, want %v", tt.col.Name, got, tt.want)
			}
		})
	}
}

func TestIgnoreStructFields(t *testing.T) {
	filter := IgnoreStructFields("Name", "Active")

	tests := []struct {
		fieldName string
		want      bool
	}{
		{fieldName: "Name", want: true},
		{fieldName: "Active", want: true},
		{fieldName: "ID", want: false},
		{fieldName: "Email", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			field := reflect.StructField{Name: tt.fieldName}
			if got := filter.IgnoreField(&field); got != tt.want {
				t.Errorf("IgnoreStructFields(%q) = %v, want %v", tt.fieldName, got, tt.want)
			}
		})
	}
}

func TestOnlyStructFields(t *testing.T) {
	filter := OnlyStructFields("ID", "Name")

	tests := []struct {
		fieldName string
		want      bool
	}{
		{fieldName: "ID", want: false},       // not ignored
		{fieldName: "Name", want: false},     // not ignored
		{fieldName: "Active", want: true},    // ignored
		{fieldName: "Created", want: true},   // ignored
	}
	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			field := reflect.StructField{Name: tt.fieldName}
			if got := filter.IgnoreField(&field); got != tt.want {
				t.Errorf("OnlyStructFields ignore %q = %v, want %v", tt.fieldName, got, tt.want)
			}
		})
	}
}

func TestQueryOptionsIgnoreColumn(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		col := ColumnInfo{Name: "id"}
		if QueryOptionsIgnoreColumn(&col, nil) {
			t.Error("expected false with no options")
		}
	})

	t.Run("matching option", func(t *testing.T) {
		col := ColumnInfo{Name: "name"}
		opts := []QueryOption{IgnoreColumns("name")}
		if !QueryOptionsIgnoreColumn(&col, opts) {
			t.Error("expected true for matching IgnoreColumns")
		}
	})

	t.Run("non-matching option", func(t *testing.T) {
		col := ColumnInfo{Name: "id"}
		opts := []QueryOption{IgnoreColumns("name")}
		if QueryOptionsIgnoreColumn(&col, opts) {
			t.Error("expected false for non-matching IgnoreColumns")
		}
	})

	t.Run("multiple options first matches", func(t *testing.T) {
		col := ColumnInfo{Name: "active", PrimaryKey: false}
		opts := []QueryOption{IgnoreColumns("active"), IgnorePrimaryKey}
		if !QueryOptionsIgnoreColumn(&col, opts) {
			t.Error("expected true when first option matches")
		}
	})

	t.Run("IgnoreHasDefault", func(t *testing.T) {
		col := ColumnInfo{Name: "created_at", HasDefault: true}
		opts := []QueryOption{IgnoreHasDefault}
		if !QueryOptionsIgnoreColumn(&col, opts) {
			t.Error("expected true for HasDefault column")
		}
	})

	t.Run("IgnorePrimaryKey", func(t *testing.T) {
		col := ColumnInfo{Name: "id", PrimaryKey: true}
		opts := []QueryOption{IgnorePrimaryKey}
		if !QueryOptionsIgnoreColumn(&col, opts) {
			t.Error("expected true for PrimaryKey column")
		}
	})

	t.Run("IgnoreReadOnly", func(t *testing.T) {
		col := ColumnInfo{Name: "computed", ReadOnly: true}
		opts := []QueryOption{IgnoreReadOnly}
		if !QueryOptionsIgnoreColumn(&col, opts) {
			t.Error("expected true for ReadOnly column")
		}
	})
}

func TestQueryOptionsIgnoreStructField(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		field := reflect.StructField{Name: "ID"}
		if QueryOptionsIgnoreStructField(&field, nil) {
			t.Error("expected false with no options")
		}
	})

	t.Run("matching option", func(t *testing.T) {
		field := reflect.StructField{Name: "Name"}
		opts := []QueryOption{IgnoreStructFields("Name")}
		if !QueryOptionsIgnoreStructField(&field, opts) {
			t.Error("expected true for matching field")
		}
	})

	t.Run("non-matching option", func(t *testing.T) {
		field := reflect.StructField{Name: "ID"}
		opts := []QueryOption{IgnoreStructFields("Name")}
		if QueryOptionsIgnoreStructField(&field, opts) {
			t.Error("expected false for non-matching field")
		}
	})
}
