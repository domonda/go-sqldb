package reflection

// StructFieldFlags is a bitmask for special properties
// of how struct fields relate to database columns.
type StructFieldFlags uint

// PrimaryKey indicates if FlagPrimaryKey is set
func (f StructFieldFlags) PrimaryKey() bool { return f&FlagPrimaryKey != 0 }

// ReadOnly indicates if FlagReadOnly is set
func (f StructFieldFlags) ReadOnly() bool { return f&FlagReadOnly != 0 }

// HasDefault indicates if FlagHasDefault is set
func (f StructFieldFlags) HasDefault() bool { return f&FlagHasDefault != 0 }

const (
	// FlagPrimaryKey marks a field as primary key
	FlagPrimaryKey StructFieldFlags = 1 << iota

	// FlagReadOnly marks a field as read-only
	FlagReadOnly

	// FlagHasDefault marks a field as having a column default value
	FlagHasDefault
)
