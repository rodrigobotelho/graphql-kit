package graphqlkit

import "testing"

func Test_findOpName(t *testing.T) {
	type args struct {
		req string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"More than one space after bracket",
			args{"{  qn }"},
			"qn",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findOpName(tt.args.req); got != tt.want {
				t.Errorf("findOpName() = %v, want %v", got, tt.want)
			}
		})
	}
}
