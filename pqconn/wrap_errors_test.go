package pqconn

import (
	"errors"
	"testing"

	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func Test_wrapKnownErrors(t *testing.T) {
	const testConstraint = "test_constraint"
	const testMessage = "custom raised message"

	for _, scenario := range []struct {
		name           string
		inputErr       error
		wantNil        bool
		wantUnchanged  bool
		wantSentinel   error
		wantAsType     any
		wantConstraint string
	}{
		{
			name:    "nil input returns nil",
			wantNil: true,
		},
		{
			name:          "non-pq error is returned unchanged",
			inputErr:      errors.New("some other error"),
			wantUnchanged: true,
		},
		{
			name:         "NullValueNotAllowed wraps to ErrNullValueNotAllowed",
			inputErr:     &pq.Error{Code: pqerror.NullValueNotAllowed, Constraint: testConstraint},
			wantSentinel: sqldb.ErrNullValueNotAllowed,
		},
		{
			name:           "IntegrityConstraintViolation wraps to ErrIntegrityConstraintViolation",
			inputErr:       &pq.Error{Code: pqerror.IntegrityConstraintViolation, Constraint: testConstraint},
			wantAsType:     &sqldb.ErrIntegrityConstraintViolation{},
			wantConstraint: testConstraint,
		},
		{
			name:           "RestrictViolation wraps to ErrRestrictViolation",
			inputErr:       &pq.Error{Code: pqerror.RestrictViolation, Constraint: testConstraint},
			wantAsType:     &sqldb.ErrRestrictViolation{},
			wantConstraint: testConstraint,
		},
		{
			name:           "NotNullViolation wraps to ErrNotNullViolation",
			inputErr:       &pq.Error{Code: pqerror.NotNullViolation, Constraint: testConstraint},
			wantAsType:     &sqldb.ErrNotNullViolation{},
			wantConstraint: testConstraint,
		},
		{
			name:           "ForeignKeyViolation wraps to ErrForeignKeyViolation",
			inputErr:       &pq.Error{Code: pqerror.ForeignKeyViolation, Constraint: testConstraint},
			wantAsType:     &sqldb.ErrForeignKeyViolation{},
			wantConstraint: testConstraint,
		},
		{
			name:           "UniqueViolation wraps to ErrUniqueViolation",
			inputErr:       &pq.Error{Code: pqerror.UniqueViolation, Constraint: testConstraint},
			wantAsType:     &sqldb.ErrUniqueViolation{},
			wantConstraint: testConstraint,
		},
		{
			name:           "CheckViolation wraps to ErrCheckViolation",
			inputErr:       &pq.Error{Code: pqerror.CheckViolation, Constraint: testConstraint},
			wantAsType:     &sqldb.ErrCheckViolation{},
			wantConstraint: testConstraint,
		},
		{
			name:         "TRDeadlockDetected wraps to ErrDeadlock",
			inputErr:     &pq.Error{Code: pqerror.TRDeadlockDetected},
			wantSentinel: sqldb.ErrDeadlock,
		},
		{
			name:         "QueryCanceled wraps to ErrQueryCanceled",
			inputErr:     &pq.Error{Code: pqerror.QueryCanceled},
			wantSentinel: sqldb.ErrQueryCanceled,
		},
		{
			name:           "ExclusionViolation wraps to ErrExclusionViolation",
			inputErr:       &pq.Error{Code: pqerror.ExclusionViolation, Constraint: testConstraint},
			wantAsType:     &sqldb.ErrExclusionViolation{},
			wantConstraint: testConstraint,
		},
		{
			name:     "RaiseException wraps to ErrRaisedException with message",
			inputErr: &pq.Error{Code: pqerror.RaiseException, Message: testMessage},
		},
		{
			name:          "unrecognized pq error code is returned unchanged",
			inputErr:      &pq.Error{Code: pqerror.InsufficientPrivilege},
			wantUnchanged: true,
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

			// Verify the original pq error is still accessible
			var pqError *pq.Error
			require.True(t, errors.As(result, &pqError), "original pq.Error should be accessible via errors.As")
			assert.Equal(t, inputErr, pqError, "unwrapped pq.Error should match the original input")

			// Verify sentinel errors
			if scenario.wantSentinel != nil {
				assert.True(t, errors.Is(result, scenario.wantSentinel), "expected errors.Is to match %v", scenario.wantSentinel)
			}

			// Verify struct error types with constraint
			switch scenario.wantAsType.(type) {
			case *sqldb.ErrIntegrityConstraintViolation:
				var target sqldb.ErrIntegrityConstraintViolation
				require.True(t, errors.As(result, &target), "expected errors.As to match ErrIntegrityConstraintViolation")
				assert.Equal(t, scenario.wantConstraint, target.Constraint, "constraint name should be preserved")

			case *sqldb.ErrRestrictViolation:
				var target sqldb.ErrRestrictViolation
				require.True(t, errors.As(result, &target), "expected errors.As to match ErrRestrictViolation")
				assert.Equal(t, scenario.wantConstraint, target.Constraint, "constraint name should be preserved")

			case *sqldb.ErrNotNullViolation:
				var target sqldb.ErrNotNullViolation
				require.True(t, errors.As(result, &target), "expected errors.As to match ErrNotNullViolation")
				assert.Equal(t, scenario.wantConstraint, target.Constraint, "constraint name should be preserved")

			case *sqldb.ErrForeignKeyViolation:
				var target sqldb.ErrForeignKeyViolation
				require.True(t, errors.As(result, &target), "expected errors.As to match ErrForeignKeyViolation")
				assert.Equal(t, scenario.wantConstraint, target.Constraint, "constraint name should be preserved")

			case *sqldb.ErrUniqueViolation:
				var target sqldb.ErrUniqueViolation
				require.True(t, errors.As(result, &target), "expected errors.As to match ErrUniqueViolation")
				assert.Equal(t, scenario.wantConstraint, target.Constraint, "constraint name should be preserved")

			case *sqldb.ErrCheckViolation:
				var target sqldb.ErrCheckViolation
				require.True(t, errors.As(result, &target), "expected errors.As to match ErrCheckViolation")
				assert.Equal(t, scenario.wantConstraint, target.Constraint, "constraint name should be preserved")

			case *sqldb.ErrExclusionViolation:
				var target sqldb.ErrExclusionViolation
				require.True(t, errors.As(result, &target), "expected errors.As to match ErrExclusionViolation")
				assert.Equal(t, scenario.wantConstraint, target.Constraint, "constraint name should be preserved")
			}

			// Special case: RaiseException verifies message
			if scenario.inputErr != nil {
				if pqe, ok := scenario.inputErr.(*pq.Error); ok && pqe.Code == pqerror.RaiseException {
					var target sqldb.ErrRaisedException
					require.True(t, errors.As(result, &target), "expected errors.As to match ErrRaisedException")
					assert.Equal(t, testMessage, target.Message, "raised exception message should be preserved")
				}
			}
		})
	}
}
