package main_test

import (
	"net"
	"reflect"
	"testing"

	main "github.com/tonymet/dnstrace"
)

func TestScanRecords(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		input   string
		want    main.AnswerList
		wantErr bool
	}{
		{
			name:  "first",
			input: "CNAME:cloudfront.aws.com,A:34.33.22.33",
			want: main.AnswerList{main.ExpectedAnswer{Type: "CNAME", Address: "cloudfront.aws.com", NetAddr: net.IP{}},
				main.ExpectedAnswer{Type: "A", Address: "34.33.22.33", NetAddr: net.ParseIP("34.33.22.33")}},
			wantErr: false,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := main.ScanRecords(tt.input)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ScanRecords() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("ScanRecords() succeeded unexpectedly")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ScanRecords() = %v, want %v", got, tt.want)
			}
		})
	}
}
