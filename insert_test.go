package sqldb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInsert(t *testing.T) {
	var queryBuf QueryBuffer
	conn := LogConnection(NullConnection(defaultQueryFormatter{}, DefaultStructFieldMapping), &queryBuf)
	ctx := ContextWithConnection(context.Background(), conn)

	err := Insert(ctx, "test_table", Values{
		"a": "Hello",
		"b": true,
		"c": 666,
	})
	require.NoError(t, err)
	expectedQuery := `INSERT INTO test_table("a","b","c")
VALUES($1,$2,$3);
`
	require.Equal(t, expectedQuery, queryBuf.String())
}
