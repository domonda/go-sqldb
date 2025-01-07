package sqldb

import "context"

type QueryRecording struct {
	Execs   []QueryData
	Queries []QueryData
}

type RecordingMockConn struct {
	MockConn
	QueryRecording
	Normalize bool
}

func NewRecordingMockConn(placeholderPosPrefix string, normalize bool) *RecordingMockConn {
	return &RecordingMockConn{
		MockConn: MockConn{
			QueryFormatter: StdQueryFormatter{PlaceholderPosPrefix: placeholderPosPrefix},
		},
		Normalize: normalize,
	}
}

func (c *RecordingMockConn) Exec(ctx context.Context, query string, args ...any) error {
	queryData, err := NewQueryData(query, args, c.Normalize)
	if err != nil {
		return err
	}
	c.Execs = append(c.Execs, queryData)
	return c.MockConn.Exec(ctx, query, args...)
}

func (c *RecordingMockConn) Query(ctx context.Context, query string, args ...any) Rows {
	queryData, err := NewQueryData(query, args, c.Normalize)
	if err != nil {
		return NewErrRows(err)
	}
	c.Queries = append(c.Queries, queryData)
	return c.MockConn.Query(ctx, query, args...)
}
