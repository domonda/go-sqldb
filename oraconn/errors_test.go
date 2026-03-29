package oraconn

import (
	"errors"
	"testing"

	"github.com/sijms/go-ora/v2/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func Test_extractConstraintName(t *testing.T) {
	for _, scenario := range []struct {
		name string
		msg  string
		want string
	}{
		{
			name: "extracts constraint from unique violation message",
			msg:  "ORA-00001: unique constraint (SCHEMA.UK_NAME) violated",
			want: "SCHEMA.UK_NAME",
		},
		{
			name: "extracts constraint from FK parent not found message",
			msg:  "ORA-02291: integrity constraint (HR.FK_DEPT_ID) violated - parent key not found",
			want: "HR.FK_DEPT_ID",
		},
		{
			name: "returns empty string when no parentheses",
			msg:  "ORA-00060: deadlock detected while waiting for resource",
			want: "",
		},
		{
			name: "returns empty string when only opening parenthesis",
			msg:  "ORA-00001: unique constraint (SCHEMA.UK_NAME violated",
			want: "",
		},
		{
			name: "returns empty string for empty message",
			msg:  "",
			want: "",
		},
		{
			name: "extracts first parenthesized group only",
			msg:  "ORA-00001: (FIRST) and (SECOND)",
			want: "FIRST",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result := extractConstraintName(scenario.msg)

			// then
			assert.Equal(t, scenario.want, result)
		})
	}
}

func Test_wrapKnownErrors(t *testing.T) {
	const testConstraint = "SCHEMA.UK_NAME"

	for _, scenario := range []struct {
		name           string
		inputErr       error
		wantNil        bool
		wantUnchanged  bool
		wantSentinel   error
		wantAsType     any
		wantConstraint string
		wantMessage    string
	}{
		{
			name:    "nil input returns nil",
			wantNil: true,
		},
		{
			name:          "non-oracle error is returned unchanged",
			inputErr:      errors.New("some other error"),
			wantUnchanged: true,
		},
		{
			name:           "ORA-00001 unique violation wraps to ErrUniqueViolation",
			inputErr:       &network.OracleError{ErrCode: 1, ErrMsg: "ORA-00001: unique constraint (SCHEMA.UK_NAME) violated"},
			wantAsType:     &sqldb.ErrUniqueViolation{},
			wantConstraint: testConstraint,
		},
		{
			name:           "ORA-01400 cannot insert null wraps to ErrNotNullViolation",
			inputErr:       &network.OracleError{ErrCode: 1400, ErrMsg: "ORA-01400: cannot insert NULL into (SCHEMA.UK_NAME)"},
			wantAsType:     &sqldb.ErrNotNullViolation{},
			wantConstraint: testConstraint,
		},
		{
			name:           "ORA-02291 FK parent not found wraps to ErrForeignKeyViolation",
			inputErr:       &network.OracleError{ErrCode: 2291, ErrMsg: "ORA-02291: integrity constraint (SCHEMA.UK_NAME) violated - parent key not found"},
			wantAsType:     &sqldb.ErrForeignKeyViolation{},
			wantConstraint: testConstraint,
		},
		{
			name:           "ORA-02292 FK child record found wraps to ErrForeignKeyViolation",
			inputErr:       &network.OracleError{ErrCode: 2292, ErrMsg: "ORA-02292: integrity constraint (SCHEMA.UK_NAME) violated - child record found"},
			wantAsType:     &sqldb.ErrForeignKeyViolation{},
			wantConstraint: testConstraint,
		},
		{
			name:           "ORA-02290 check violation wraps to ErrCheckViolation",
			inputErr:       &network.OracleError{ErrCode: 2290, ErrMsg: "ORA-02290: check constraint (SCHEMA.UK_NAME) violated"},
			wantAsType:     &sqldb.ErrCheckViolation{},
			wantConstraint: testConstraint,
		},
		{
			name:         "ORA-00060 deadlock wraps to ErrDeadlock",
			inputErr:     &network.OracleError{ErrCode: 60, ErrMsg: "ORA-00060: deadlock detected while waiting for resource"},
			wantSentinel: sqldb.ErrDeadlock,
		},
		{
			name:         "ORA-01013 query canceled wraps to ErrQueryCanceled",
			inputErr:     &network.OracleError{ErrCode: 1013, ErrMsg: "ORA-01013: user requested cancel of current operation"},
			wantSentinel: sqldb.ErrQueryCanceled,
		},
		{
			name:        "ORA-20000 raised exception wraps to ErrRaisedException",
			inputErr:    &network.OracleError{ErrCode: 20000, ErrMsg: "custom raised message"},
			wantAsType:  &sqldb.ErrRaisedException{},
			wantMessage: "custom raised message",
		},
		{
			name:        "ORA-20500 mid-range raised exception wraps to ErrRaisedException",
			inputErr:    &network.OracleError{ErrCode: 20500, ErrMsg: "another raised message"},
			wantAsType:  &sqldb.ErrRaisedException{},
			wantMessage: "another raised message",
		},
		{
			name:        "ORA-20999 upper bound raised exception wraps to ErrRaisedException",
			inputErr:    &network.OracleError{ErrCode: 20999, ErrMsg: "upper bound message"},
			wantAsType:  &sqldb.ErrRaisedException{},
			wantMessage: "upper bound message",
		},
		{
			name:          "unrecognized Oracle error code is returned unchanged",
			inputErr:      &network.OracleError{ErrCode: 904, ErrMsg: "ORA-00904: invalid identifier"},
			wantUnchanged: true,
		},
		{
			name:          "ORA-21000 outside raised exception range is returned unchanged",
			inputErr:      &network.OracleError{ErrCode: 21000, ErrMsg: "ORA-21000: error number argument to raise_application_error"},
			wantUnchanged: true,
		},
		{
			name:           "unique violation without constraint in message extracts empty constraint",
			inputErr:       &network.OracleError{ErrCode: 1, ErrMsg: "ORA-00001: unique constraint violated"},
			wantAsType:     &sqldb.ErrUniqueViolation{},
			wantConstraint: "",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			inputErr := scenario.inputErr

			// when
			result := wrapKnownErrors(inputErr)

			// then
			if scenario.wantNil {
				require.NoError(t, result, "expected nil result for nil input")
				return
			}

			if scenario.wantUnchanged {
				assert.Equal(t, inputErr, result, "expected error to be returned unchanged")
				return
			}

			require.Error(t, result, "expected a non-nil wrapped error")

			// Verify the original OracleError is still accessible
			var oraErr *network.OracleError
			require.True(t, errors.As(result, &oraErr), "original OracleError should be accessible via errors.As")
			assert.Equal(t, inputErr, oraErr, "unwrapped OracleError should match the original input")

			// Verify sentinel errors
			if scenario.wantSentinel != nil {
				assert.True(t, errors.Is(result, scenario.wantSentinel), "expected errors.Is to match %v", scenario.wantSentinel)
			}

			// Verify struct error types with constraint
			switch scenario.wantAsType.(type) {
			case *sqldb.ErrUniqueViolation:
				var target sqldb.ErrUniqueViolation
				require.True(t, errors.As(result, &target), "expected errors.As to match ErrUniqueViolation")
				assert.Equal(t, scenario.wantConstraint, target.Constraint, "constraint name should be preserved")

			case *sqldb.ErrNotNullViolation:
				var target sqldb.ErrNotNullViolation
				require.True(t, errors.As(result, &target), "expected errors.As to match ErrNotNullViolation")
				assert.Equal(t, scenario.wantConstraint, target.Constraint, "constraint name should be preserved")

			case *sqldb.ErrForeignKeyViolation:
				var target sqldb.ErrForeignKeyViolation
				require.True(t, errors.As(result, &target), "expected errors.As to match ErrForeignKeyViolation")
				assert.Equal(t, scenario.wantConstraint, target.Constraint, "constraint name should be preserved")

			case *sqldb.ErrCheckViolation:
				var target sqldb.ErrCheckViolation
				require.True(t, errors.As(result, &target), "expected errors.As to match ErrCheckViolation")
				assert.Equal(t, scenario.wantConstraint, target.Constraint, "constraint name should be preserved")

			case *sqldb.ErrRaisedException:
				var target sqldb.ErrRaisedException
				require.True(t, errors.As(result, &target), "expected errors.As to match ErrRaisedException")
				assert.Equal(t, scenario.wantMessage, target.Message, "raised exception message should be preserved")
			}
		})
	}
}

func TestIsUniqueViolation(t *testing.T) {
	for _, scenario := range []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "returns true for ORA-00001",
			err:  &network.OracleError{ErrCode: 1, ErrMsg: "ORA-00001: unique constraint violated"},
			want: true,
		},
		{
			name: "returns false for different Oracle error",
			err:  &network.OracleError{ErrCode: 1400, ErrMsg: "ORA-01400: cannot insert NULL"},
			want: false,
		},
		{
			name: "returns false for non-Oracle error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "returns false for nil",
			err:  nil,
			want: false,
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result := IsUniqueViolation(scenario.err)

			// then
			assert.Equal(t, scenario.want, result)
		})
	}
}

func TestIsNotNullViolation(t *testing.T) {
	for _, scenario := range []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "returns true for ORA-01400",
			err:  &network.OracleError{ErrCode: 1400, ErrMsg: "ORA-01400: cannot insert NULL"},
			want: true,
		},
		{
			name: "returns false for different Oracle error",
			err:  &network.OracleError{ErrCode: 1, ErrMsg: "ORA-00001: unique constraint violated"},
			want: false,
		},
		{
			name: "returns false for non-Oracle error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "returns false for nil",
			err:  nil,
			want: false,
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result := IsNotNullViolation(scenario.err)

			// then
			assert.Equal(t, scenario.want, result)
		})
	}
}

func TestIsForeignKeyViolation(t *testing.T) {
	for _, scenario := range []struct {
		name                string
		err                 error
		violatedConstraints []string
		want                bool
	}{
		{
			name: "returns true for ORA-02291 without constraint filter",
			err:  &network.OracleError{ErrCode: 2291, ErrMsg: "ORA-02291: integrity constraint (HR.FK_DEPT) violated - parent key not found"},
			want: true,
		},
		{
			name: "returns true for ORA-02292 without constraint filter",
			err:  &network.OracleError{ErrCode: 2292, ErrMsg: "ORA-02292: integrity constraint (HR.FK_DEPT) violated - child record found"},
			want: true,
		},
		{
			name:                "returns true when constraint matches filter",
			err:                 &network.OracleError{ErrCode: 2291, ErrMsg: "ORA-02291: integrity constraint (HR.FK_DEPT) violated - parent key not found"},
			violatedConstraints: []string{"HR.FK_DEPT"},
			want:                true,
		},
		{
			name:                "returns true when constraint matches one of multiple filters",
			err:                 &network.OracleError{ErrCode: 2291, ErrMsg: "ORA-02291: integrity constraint (HR.FK_DEPT) violated - parent key not found"},
			violatedConstraints: []string{"HR.FK_OTHER", "HR.FK_DEPT"},
			want:                true,
		},
		{
			name:                "returns false when constraint does not match filter",
			err:                 &network.OracleError{ErrCode: 2291, ErrMsg: "ORA-02291: integrity constraint (HR.FK_DEPT) violated - parent key not found"},
			violatedConstraints: []string{"HR.FK_OTHER"},
			want:                false,
		},
		{
			name: "returns false for different Oracle error",
			err:  &network.OracleError{ErrCode: 1, ErrMsg: "ORA-00001: unique constraint violated"},
			want: false,
		},
		{
			name: "returns false for non-Oracle error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "returns false for nil",
			err:  nil,
			want: false,
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result := IsForeignKeyViolation(scenario.err, scenario.violatedConstraints...)

			// then
			assert.Equal(t, scenario.want, result)
		})
	}
}

func TestIsCheckViolation(t *testing.T) {
	for _, scenario := range []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "returns true for ORA-02290",
			err:  &network.OracleError{ErrCode: 2290, ErrMsg: "ORA-02290: check constraint (SCHEMA.CK_NAME) violated"},
			want: true,
		},
		{
			name: "returns false for different Oracle error",
			err:  &network.OracleError{ErrCode: 1, ErrMsg: "ORA-00001: unique constraint violated"},
			want: false,
		},
		{
			name: "returns false for non-Oracle error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "returns false for nil",
			err:  nil,
			want: false,
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result := IsCheckViolation(scenario.err)

			// then
			assert.Equal(t, scenario.want, result)
		})
	}
}

func TestIsDeadlockDetected(t *testing.T) {
	for _, scenario := range []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "returns true for ORA-00060",
			err:  &network.OracleError{ErrCode: 60, ErrMsg: "ORA-00060: deadlock detected while waiting for resource"},
			want: true,
		},
		{
			name: "returns false for different Oracle error",
			err:  &network.OracleError{ErrCode: 1, ErrMsg: "ORA-00001: unique constraint violated"},
			want: false,
		},
		{
			name: "returns false for non-Oracle error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "returns false for nil",
			err:  nil,
			want: false,
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result := IsDeadlockDetected(scenario.err)

			// then
			assert.Equal(t, scenario.want, result)
		})
	}
}

func TestIsQueryCanceled(t *testing.T) {
	for _, scenario := range []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "returns true for ORA-01013",
			err:  &network.OracleError{ErrCode: 1013, ErrMsg: "ORA-01013: user requested cancel of current operation"},
			want: true,
		},
		{
			name: "returns false for different Oracle error",
			err:  &network.OracleError{ErrCode: 1, ErrMsg: "ORA-00001: unique constraint violated"},
			want: false,
		},
		{
			name: "returns false for non-Oracle error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "returns false for nil",
			err:  nil,
			want: false,
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result := IsQueryCanceled(scenario.err)

			// then
			assert.Equal(t, scenario.want, result)
		})
	}
}
