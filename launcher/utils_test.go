package launcher

import (
	"reflect"
	"testing"
)

func Test_normalizeGitRemotes(t *testing.T) {
	in2 := []string{"b", "d", "a", "c", "b", "c", "d"}
	out2 := []string{"a", "b", "c", "d"}
	in0 := []string{"a", "b"}
	out0 := in0
	in1 := []string{"b", "a", "b"}
	out1 := in0
	type args struct {
		remotes *[]string
	}
	tests := []struct {
		name string
		args args
		want *[]string
	}{
		{"Left as is", args{&in0}, &out0},
		{"Pop first", args{&in1}, &out1},
		// TODO: Add test cases.
		{name: "Remove duplicates, respect last", args: args{
			remotes: &in2,
		}, want: &out2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeGitRemotes(tt.args.remotes); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("normalizeGitRemotes() = %v, want %v", got, tt.want)
			}
		})
	}
}
