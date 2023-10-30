package sqldb

type TableName struct{}

func (TableName) TableNameMarker() {}

type RowWithTableName interface {
	TableNameMarker()
}
