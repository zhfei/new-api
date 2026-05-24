package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateOneCardRequiredGroupRatios(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{
			name: "valid free plus pro ratios",
			raw:  `{"default":1,"free":1,"plus":1.2,"pro":1.5}`,
		},
		{
			name:    "missing plus ratio",
			raw:     `{"default":1,"free":1,"pro":1.5}`,
			wantErr: true,
		},
		{
			name:    "zero free ratio",
			raw:     `{"default":1,"free":0,"plus":1.2,"pro":1.5}`,
			wantErr: true,
		},
		{
			name:    "negative pro ratio",
			raw:     `{"default":1,"free":1,"plus":1.2,"pro":-1}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			raw:     `{`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOneCardRequiredGroupRatios(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
