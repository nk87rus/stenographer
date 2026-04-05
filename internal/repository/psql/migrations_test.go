package psql

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestRollbackMigration(t *testing.T) {
	var errProvDown = fmt.Errorf("errProvDown")
	testCases := []struct {
		name      string
		wantError error
	}{
		{
			name:      "errProvDown",
			wantError: errProvDown,
		},
		{
			name:      "Correct",
			wantError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patchGPDown := monkey.PatchInstanceMethod(reflect.TypeOf(&goose.Provider{}), "Down",
				func(*goose.Provider, context.Context) (*goose.MigrationResult, error) {
					if errors.Is(tc.wantError, errProvDown) {
						return nil, tc.wantError
					}
					return &goose.MigrationResult{Source: &goose.Source{Path: "p"}}, nil
				})
			defer patchGPDown.Unpatch()
			resultError := rollbackMigration(t.Context(), &goose.Provider{})
			if tc.wantError != nil {
				require.ErrorContains(t, resultError, tc.wantError.Error())
			} else {
				require.Nil(t, resultError)
			}
		})
	}
}
