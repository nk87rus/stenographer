package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsEmpty(t *testing.T) {
	testCases := []struct {
		name       string
		value      Transcription
		wantResult bool
	}{
		{
			name:       "Empty",
			value:      Transcription{},
			wantResult: true,
		},
		{
			name:       "NotEmpty",
			value:      Transcription{Id: 2},
			wantResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantResult, tc.value.IsEmpty())
		})
	}
}
