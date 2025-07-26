package sqldb

import (
	"fmt"
	"io"
	"strings"
)

func DefaultQueryBuilder() QueryBuilder {
	return defaultQueryBuilder{}
}

type defaultQueryBuilder struct{}

func (defaultQueryBuilder) QueryForRowWithPK(w io.Writer, table string, pkColumns []string, f QueryFormatter) (err error) {
	table, err = f.FormatTableName(table)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, `SELECT * FROM %s WHERE %s = %s`, table, pkColumns[0], f.FormatPlaceholder(0))
	if err != nil {
		return err
	}
	for i := 1; i < len(pkColumns); i++ {
		_, err = fmt.Fprintf(w, ` AND %s = %s`, pkColumns[i], f.FormatPlaceholder(i))
		if err != nil {
			return err
		}
	}
	return nil
}

func (defaultQueryBuilder) Insert(w io.Writer, table string, columns []ColumnInfo, f QueryFormatter) (err error) {
	table, err = f.FormatTableName(table)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, `INSERT INTO %s(`, table)
	for i := range columns {
		column := columns[i].Name
		column, err = f.FormatColumnName(column)
		if err != nil {
			return err
		}
		if i > 0 {
			_, err = w.Write([]byte{','})
			if err != nil {
				return err
			}
		}
		_, err = io.WriteString(w, column)
		if err != nil {
			return err
		}
	}
	_, err = io.WriteString(w, `) VALUES(`)
	if err != nil {
		return err
	}
	for i := range columns {
		if i > 0 {
			_, err = w.Write([]byte{','})
			if err != nil {
				return err
			}
		}
		_, err = io.WriteString(w, f.FormatPlaceholder(i))
		if err != nil {
			return err
		}
	}
	_, err = w.Write([]byte{')'})
	if err != nil {
		return err
	}
	return nil
}

func (b defaultQueryBuilder) InsertUnique(w io.Writer, table string, columns []ColumnInfo, onConflict string, f QueryFormatter) error {
	err := b.Insert(w, table, columns, f)
	if err != nil {
		return err
	}
	if strings.HasPrefix(onConflict, "(") && strings.HasSuffix(onConflict, ")") {
		onConflict = onConflict[1 : len(onConflict)-1]
	}
	_, err = fmt.Fprintf(w, " ON CONFLICT (%s) DO NOTHING RETURNING TRUE", onConflict)
	return err
}

func (b defaultQueryBuilder) Upsert(w io.Writer, table string, columns []ColumnInfo, f QueryFormatter) (err error) {
	err = b.Insert(w, table, columns, f)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, ` ON CONFLICT(`)
	if err != nil {
		return err
	}
	first := true
	for i := range columns {
		if !columns[i].PrimaryKey {
			continue
		}
		if first {
			first = false
		} else {
			_, err = w.Write([]byte{','})
			if err != nil {
				return err
			}
		}
		columnName, err := f.FormatColumnName(columns[i].Name)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, columnName)
		if err != nil {
			return err
		}
	}
	_, err = io.WriteString(w, `) DO UPDATE SET`)
	if err != nil {
		return err
	}
	first = true
	for i := range columns {
		if columns[i].PrimaryKey {
			continue
		}
		if first {
			first = false
		} else {
			_, err = w.Write([]byte{','})
			if err != nil {
				return err
			}
		}
		columnName, err := f.FormatColumnName(columns[i].Name)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, ` %s=%s`, columnName, f.FormatPlaceholder(i))
	}
	return nil
}

func (b defaultQueryBuilder) UpdateValues(w io.Writer, table string, values Values, where string, args []any, f QueryFormatter) (vals []any, err error) {
	table, err = f.FormatTableName(table)
	if err != nil {
		return nil, err
	}

	_, err = fmt.Fprintf(w, `UPDATE %s SET`, table)
	if err != nil {
		return nil, err
	}

	columns, vals := values.SortedColumnsAndValues()
	for i := range columns {
		column, err := f.FormatColumnName(columns[i].Name)
		if err != nil {
			return nil, err
		}
		if i > 0 {
			_, err = w.Write([]byte{','})
			if err != nil {
				return nil, err
			}
		}
		_, err = fmt.Fprintf(w, ` %s=%s`, column, f.FormatPlaceholder(len(args)+i))
		if err != nil {
			return nil, err
		}
	}
	_, err = fmt.Fprintf(w, ` WHERE %s`, where)
	if err != nil {
		return nil, err
	}

	return append(args, vals...), nil
}

func (b defaultQueryBuilder) UpdateColumns(w io.Writer, table string, columns []ColumnInfo, f QueryFormatter) error {
	table, err := f.FormatTableName(table)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, `UPDATE %s SET`, table)
	if err != nil {
		return err
	}

	first := true
	for i := range columns {
		if columns[i].PrimaryKey {
			continue
		}
		if first {
			first = false
		} else {
			_, err = w.Write([]byte{','})
			if err != nil {
				return err
			}
		}
		columnName, err := f.FormatColumnName(columns[i].Name)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, ` %s=%s`, columnName, f.FormatPlaceholder(i))
	}

	_, err = io.WriteString(w, ` WHERE `)
	if err != nil {
		return err
	}

	first = true
	for i := range columns {
		if !columns[i].PrimaryKey {
			continue
		}
		if first {
			first = false
		} else {
			_, err = io.WriteString(w, ` AND `)
			if err != nil {
				return err
			}
		}
		columnName, err := f.FormatColumnName(columns[i].Name)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, `%s = %s`, columnName, f.FormatPlaceholder(i))
	}

	return nil
}
