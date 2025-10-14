package collector

import "testing"

func Test_getCelciusFromFixedPointSignedInteger(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		dataOut []string
		want    int
		wantErr bool
	}{
		{
			name:    "25 degrees",
			dataOut: []string{"0x00", "0x19", "0x00", "0x00"},
			want:    25,
			wantErr: false,
		},
		{
			name:    "28 degrees",
			dataOut: []string{"0x00", "0x1C", "0x00", "0x00"},
			want:    28,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := getCelciusFromFixedPointSignedInteger(tt.dataOut)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("getCelciusFromFixedPointSignedInteger() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("getCelciusFromFixedPointSignedInteger() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if tt.want != got {
				t.Errorf("getCelciusFromFixedPointSignedInteger() = %v, want %v", got, tt.want)
			}
		})
	}
}
