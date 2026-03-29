package mssqlconn

import (
	"errors"
	"testing"

	mssql "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/assert"

	"github.com/domonda/go-sqldb"
)

func Test_wrapKnownErrors(t *testing.T) {
	for _, scenario := range []struct {
		name            string
		err             error
		wantNil         bool
		wantUnchanged   bool
		wantDeadlock    bool
		wantRaised      *sqldb.ErrRaisedException
		wantNotNull     *sqldb.ErrNotNullViolation
		wantUnique      *sqldb.ErrUniqueViolation
		wantForeignKey  *sqldb.ErrForeignKeyViolation
		wantCheck       *sqldb.ErrCheckViolation
		wantOriginalErr bool
	}{
		{
			name:    "nil error returns nil",
			err:     nil,
			wantNil: true,
		},
		{
			name:          "non-mssql error returned unchanged",
			err:           errors.New("some generic error"),
			wantUnchanged: true,
		},
		{
			name:          "unknown mssql error number returned unchanged",
			err:           mssql.Error{Number: 99999, Message: "Unknown error"},
			wantUnchanged: true,
		},
		{
			name:            "errCannotInsertNull (515) wraps as ErrNotNullViolation with column name",
			err:             mssql.Error{Number: 515, Message: "Cannot insert the value NULL into column 'email', table 'mydb.dbo.users'; column does not allow nulls. INSERT fails."},
			wantNotNull:     &sqldb.ErrNotNullViolation{Constraint: "email"},
			wantOriginalErr: true,
		},
		{
			name:            "errConstraintConflict (547) with FOREIGN KEY wraps as ErrForeignKeyViolation",
			err:             mssql.Error{Number: 547, Message: `The INSERT statement conflicted with the FOREIGN KEY constraint "fk_orders_user". The conflict occurred in database "mydb", table "dbo.users", column 'id'.`},
			wantForeignKey:  &sqldb.ErrForeignKeyViolation{Constraint: "fk_orders_user"},
			wantOriginalErr: true,
		},
		{
			name:            "errConstraintConflict (547) with CHECK wraps as ErrCheckViolation",
			err:             mssql.Error{Number: 547, Message: `The INSERT statement conflicted with the CHECK constraint "chk_positive_amount". The conflict occurred in database "mydb", table "dbo.orders", column 'amount'.`},
			wantCheck:       &sqldb.ErrCheckViolation{Constraint: "chk_positive_amount"},
			wantOriginalErr: true,
		},
		{
			name:            "errConstraintConflict (547) without FOREIGN KEY keyword wraps as ErrCheckViolation",
			err:             mssql.Error{Number: 547, Message: `The DELETE statement conflicted with the REFERENCE constraint "fk_ref_name". The conflict occurred in database "mydb", table "dbo.items".`},
			wantCheck:       &sqldb.ErrCheckViolation{Constraint: "fk_ref_name"},
			wantOriginalErr: true,
		},
		{
			name:            "errDeadlock (1205) wraps as ErrDeadlock",
			err:             mssql.Error{Number: 1205, Message: "Transaction (Process ID 52) was deadlocked on lock resources with another process and has been chosen as the deadlock victim. Rerun the transaction."},
			wantDeadlock:    true,
			wantOriginalErr: true,
		},
		{
			name:            "errRaisedException (50000) wraps as ErrRaisedException",
			err:             mssql.Error{Number: 50000, Message: "Custom application error: validation failed"},
			wantRaised:      &sqldb.ErrRaisedException{Message: "Custom application error: validation failed"},
			wantOriginalErr: true,
		},
		{
			name:            "errDupKeyRow (2601) wraps as ErrUniqueViolation with index name",
			err:             mssql.Error{Number: 2601, Message: "Cannot insert duplicate key row in object 'dbo.users' with unique index 'ix_users_email'. The duplicate key value is (john@example.com)."},
			wantUnique:      &sqldb.ErrUniqueViolation{Constraint: "ix_users_email"},
			wantOriginalErr: true,
		},
		{
			name:            "errUniqueConstraint (2627) wraps as ErrUniqueViolation with constraint name",
			err:             mssql.Error{Number: 2627, Message: "Violation of UNIQUE KEY constraint 'ux_users_email'. Cannot insert duplicate key in object 'dbo.users'. The duplicate key value is (john@example.com)."},
			wantUnique:      &sqldb.ErrUniqueViolation{Constraint: "ux_users_email"},
			wantOriginalErr: true,
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result := wrapKnownErrors(scenario.err)

			// then
			if scenario.wantNil {
				assert.Nil(t, result)
				return
			}
			if scenario.wantUnchanged {
				assert.Equal(t, scenario.err, result)
				return
			}
			if scenario.wantDeadlock {
				assert.ErrorIs(t, result, sqldb.ErrDeadlock)
			}
			if scenario.wantRaised != nil {
				var target sqldb.ErrRaisedException
				assert.ErrorAs(t, result, &target)
				assert.Equal(t, scenario.wantRaised.Message, target.Message)
			}
			if scenario.wantNotNull != nil {
				var target sqldb.ErrNotNullViolation
				assert.ErrorAs(t, result, &target)
				assert.Equal(t, scenario.wantNotNull.Constraint, target.Constraint)
			}
			if scenario.wantUnique != nil {
				var target sqldb.ErrUniqueViolation
				assert.ErrorAs(t, result, &target)
				assert.Equal(t, scenario.wantUnique.Constraint, target.Constraint)
			}
			if scenario.wantForeignKey != nil {
				var target sqldb.ErrForeignKeyViolation
				assert.ErrorAs(t, result, &target)
				assert.Equal(t, scenario.wantForeignKey.Constraint, target.Constraint)
			}
			if scenario.wantCheck != nil {
				var target sqldb.ErrCheckViolation
				assert.ErrorAs(t, result, &target)
				assert.Equal(t, scenario.wantCheck.Constraint, target.Constraint)
			}
			if scenario.wantOriginalErr {
				var mssqlErr mssql.Error
				assert.ErrorAs(t, result, &mssqlErr)
			}
		})
	}
}

func Test_firstSingleQuoted(t *testing.T) {
	for _, scenario := range []struct {
		name string
		s    string
		want string
	}{
		{
			name: "extracts first single-quoted value",
			s:    "Cannot insert the value NULL into column 'email', table 'mydb.dbo.users'",
			want: "email",
		},
		{
			name: "no quotes returns empty",
			s:    "no quotes here",
			want: "",
		},
		{
			name: "empty quoted value",
			s:    "empty '' value",
			want: "",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result := firstSingleQuoted(scenario.s)

			// then
			assert.Equal(t, scenario.want, result)
		})
	}
}

func Test_firstDoubleQuoted(t *testing.T) {
	for _, scenario := range []struct {
		name string
		s    string
		want string
	}{
		{
			name: "extracts first double-quoted constraint name",
			s:    `The INSERT statement conflicted with the FOREIGN KEY constraint "fk_orders_user". The conflict occurred in database "mydb".`,
			want: "fk_orders_user",
		},
		{
			name: "no double quotes returns empty",
			s:    "no double quotes here",
			want: "",
		},
		{
			name: "unclosed double quote returns empty",
			s:    `unclosed "quote`,
			want: "",
		},
		{
			name: "empty double-quoted value",
			s:    `empty "" value`,
			want: "",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result := firstDoubleQuoted(scenario.s)

			// then
			assert.Equal(t, scenario.want, result)
		})
	}
}

func Test_nthSingleQuoted(t *testing.T) {
	for _, scenario := range []struct {
		name string
		s    string
		n    int
		want string
	}{
		{
			name: "first quoted value",
			s:    "Violation of UNIQUE KEY constraint 'ux_users_email'. Cannot insert duplicate key in object 'dbo.users'.",
			n:    0,
			want: "ux_users_email",
		},
		{
			name: "second quoted value for duplicate key index name",
			s:    "Cannot insert duplicate key row in object 'dbo.users' with unique index 'ix_users_email'.",
			n:    1,
			want: "ix_users_email",
		},
		{
			name: "no quotes in string",
			s:    "no quotes here",
			n:    0,
			want: "",
		},
		{
			name: "index out of range",
			s:    "only 'one' quoted value",
			n:    1,
			want: "",
		},
		{
			name: "unclosed quote",
			s:    "unclosed 'quote",
			n:    0,
			want: "",
		},
		{
			name: "third quoted value",
			s:    "'first' and 'second' and 'third'",
			n:    2,
			want: "third",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result := nthSingleQuoted(scenario.s, scenario.n)

			// then
			assert.Equal(t, scenario.want, result)
		})
	}
}
