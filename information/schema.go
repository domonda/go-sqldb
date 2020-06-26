package information

import (
	"github.com/domonda/go-sqldb"
)

func NewDatabase(conn sqldb.Connection) Database {
	return Database{conn}
}

type Database struct {
	sqldb.Connection
}

func (db Database) GetTable(name string) (table *Table, err error) {
	err = db.QueryRow("select * from information_schema.tables where table_name = $1", name).ScanStruct(&table)
	if err != nil {
		return nil, err
	}
	return table, nil
}

func (db Database) GetTables() (tables []*Table, err error) {
	err = db.QueryRows("select * from information_schema.tables").ScanStructSlice(&tables)
	if err != nil {
		return nil, err
	}
	return tables, nil
}
