package git

import "testing"

func TestGetFullRepoName(t *testing.T) {
	type args struct {
		gitUrl string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{name: "ssh", args: args{gitUrl: "git@github.com:wix-system/rg.git"}, want: "wix-system/rg", wantErr: false},
		{name: "git", args: args{gitUrl: "git://github.com/wix-system/rg.git"}, want: "wix-system/rg", wantErr: false},
		{name: "clone", args: args{gitUrl: "https://github.com/wix-system/rg.git"}, want: "wix-system/rg", wantErr: false},
		{name: "https", args: args{gitUrl: "https://github.com/wix-system/rg"}, want: "wix-system/rg", wantErr: false},
		{name: "unknown", args: args{gitUrl: "http://github.com/wix-system/rg"}, want: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetFullRepoName(tt.args.gitUrl)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFullRepoName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetFullRepoName() got = %v, want %v", got, tt.want)
			}
		})
	}
}
