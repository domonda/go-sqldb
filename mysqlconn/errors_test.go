package mysqlconn

import (
	"errors"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
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
			name:          "non-MySQL error returned unchanged",
			err:           errors.New("some generic error"),
			wantUnchanged: true,
		},
		{
			name: "unknown MySQL error number returned unchanged",
			err: &mysqldriver.MySQLError{
				Number:  9999,
				Message: "Unknown error",
			},
			wantUnchanged: true,
		},
		{
			name: "errDeadlock (1213) wraps as ErrDeadlock",
			err: &mysqldriver.MySQLError{
				Number:  1213,
				Message: "Deadlock found when trying to get lock; try restarting transaction",
			},
			wantDeadlock:    true,
			wantOriginalErr: true,
		},
		{
			name: "errSignal (1644) wraps as ErrRaisedException",
			err: &mysqldriver.MySQLError{
				Number:  1644,
				Message: "Custom application error occurred",
			},
			wantRaised:      &sqldb.ErrRaisedException{Message: "Custom application error occurred"},
			wantOriginalErr: true,
		},
		{
			name: "errBadNullError (1048) wraps as ErrNotNullViolation with column name",
			err: &mysqldriver.MySQLError{
				Number:  1048,
				Message: "Column 'email' cannot be null",
			},
			wantNotNull:     &sqldb.ErrNotNullViolation{Constraint: "email"},
			wantOriginalErr: true,
		},
		{
			name: "errDupEntry (1062) wraps as ErrUniqueViolation with key name stripped of table prefix",
			err: &mysqldriver.MySQLError{
				Number:  1062,
				Message: "Duplicate entry 'john@example.com' for key 'users.ux_users_email'",
			},
			wantUnique:      &sqldb.ErrUniqueViolation{Constraint: "ux_users_email"},
			wantOriginalErr: true,
		},
		{
			name: "errDupEntry (1062) without table prefix in key name",
			err: &mysqldriver.MySQLError{
				Number:  1062,
				Message: "Duplicate entry '42' for key 'PRIMARY'",
			},
			wantUnique:      &sqldb.ErrUniqueViolation{Constraint: "PRIMARY"},
			wantOriginalErr: true,
		},
		{
			name: "errNoReferencedRow (1216) wraps as ErrForeignKeyViolation",
			err: &mysqldriver.MySQLError{
				Number:  1216,
				Message: "Cannot add or update a child row: a foreign key constraint fails (`mydb`.`orders`, CONSTRAINT `fk_orders_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`))",
			},
			wantForeignKey:  &sqldb.ErrForeignKeyViolation{Constraint: "fk_orders_user"},
			wantOriginalErr: true,
		},
		{
			name: "errNoReferencedRow2 (1452) wraps as ErrForeignKeyViolation",
			err: &mysqldriver.MySQLError{
				Number:  1452,
				Message: "Cannot add or update a child row: a foreign key constraint fails (`mydb`.`orders`, CONSTRAINT `fk_orders_customer` FOREIGN KEY (`customer_id`) REFERENCES `customers` (`id`))",
			},
			wantForeignKey:  &sqldb.ErrForeignKeyViolation{Constraint: "fk_orders_customer"},
			wantOriginalErr: true,
		},
		{
			name: "errRowIsReferenced (1217) wraps as ErrForeignKeyViolation",
			err: &mysqldriver.MySQLError{
				Number:  1217,
				Message: "Cannot delete or update a parent row: a foreign key constraint fails (`mydb`.`orders`, CONSTRAINT `fk_orders_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`))",
			},
			wantForeignKey:  &sqldb.ErrForeignKeyViolation{Constraint: "fk_orders_user"},
			wantOriginalErr: true,
		},
		{
			name: "errRowIsReferenced2 (1451) wraps as ErrForeignKeyViolation",
			err: &mysqldriver.MySQLError{
				Number:  1451,
				Message: "Cannot delete or update a parent row: a foreign key constraint fails (`mydb`.`invoices`, CONSTRAINT `fk_invoices_order` FOREIGN KEY (`order_id`) REFERENCES `orders` (`id`))",
			},
			wantForeignKey:  &sqldb.ErrForeignKeyViolation{Constraint: "fk_invoices_order"},
			wantOriginalErr: true,
		},
		{
			name: "errCheckViolated (3819) wraps as ErrCheckViolation",
			err: &mysqldriver.MySQLError{
				Number:  3819,
				Message: "Check constraint 'chk_positive_amount' is violated.",
			},
			wantCheck:       &sqldb.ErrCheckViolation{Constraint: "chk_positive_amount"},
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
				var mysqlErr *mysqldriver.MySQLError
				assert.ErrorAs(t, result, &mysqlErr)
			}
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
			s:    "Column 'email' cannot be null",
			n:    0,
			want: "email",
		},
		{
			name: "second quoted value",
			s:    "Duplicate entry 'john@example.com' for key 'users.ux_users_email'",
			n:    1,
			want: "users.ux_users_email",
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
			name: "empty quoted value",
			s:    "empty '' value",
			n:    0,
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

func Test_nthBacktickQuoted(t *testing.T) {
	for _, scenario := range []struct {
		name string
		s    string
		n    int
		want string
	}{
		{
			name: "first backtick quoted value",
			s:    "CONSTRAINT `fk_name` FOREIGN KEY (`col`) REFERENCES `other` (`id`)",
			n:    0,
			want: "fk_name",
		},
		{
			name: "third backtick quoted value extracts constraint name from FK message",
			s:    "Cannot add or update a child row: a foreign key constraint fails (`mydb`.`orders`, CONSTRAINT `fk_orders_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`))",
			n:    2,
			want: "fk_orders_user",
		},
		{
			name: "no backticks in string",
			s:    "no backticks here",
			n:    0,
			want: "",
		},
		{
			name: "index out of range",
			s:    "only `one` backtick value",
			n:    1,
			want: "",
		},
		{
			name: "unclosed backtick",
			s:    "unclosed `backtick",
			n:    0,
			want: "",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result := nthBacktickQuoted(scenario.s, scenario.n)

			// then
			assert.Equal(t, scenario.want, result)
		})
	}
}
