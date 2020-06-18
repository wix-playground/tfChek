package git

import (
	"gopkg.in/src-d/go-git.v4"
	"os"
	"testing"
)

func TestBuiltInManager_Clone(t *testing.T) {
	type fields struct {
		remote   string
		repoPath string
		repo     *git.Repository
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
		{name: "ssh clone",
			wantErr: false,
			fields:  fields{remote: "git@github.com:wix-system/tfChek-testrepo.git", repoPath: "/tmp/testchekrepo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BuiltInManager{
				remoteUrl: tt.fields.remote,
				repoPath:  tt.fields.repoPath,
				repo:      tt.fields.repo,
			}
			if err := b.Clone(); (err != nil) != tt.wantErr {
				t.Errorf("Clone() error = %v, wantErr %v", err, tt.wantErr)
			}
			err := os.RemoveAll(tt.fields.repoPath)
			if err != nil {
				t.Errorf("Cannot clean up cloned repository Error: %s", err)
			}
		})
	}
}

//func TestBuiltInManager_Checkout(t *testing.T) {
//	type fields struct {
//		remote   string
//		repoPath string
//		repo     *git.Repository
//	}
//	type args struct {
//		ref string
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//		{name: "ssh clone",
//			wantErr: false,
//			fields:  fields{remote: "git@github.com:wix-system/tfChek-testrepo.git", repoPath: "/tmp/testchekrepo"},
//			args:    args{ref: "test"}},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			b := &BuiltInManager{
//				remoteUrl: tt.fields.remote,
//				repoPath:  tt.fields.repoPath,
//				repo:      tt.fields.repo,
//			}
//			if err := b.Clone(); (err != nil) != tt.wantErr {
//				t.Errorf("Clone() error = %v, wantErr %v", err, tt.wantErr)
//			}
//
//			if err := b.Checkout(tt.args.ref); (err != nil) != tt.wantErr {
//				t.Errorf("Checkout() error = %v, wantErr %v", err, tt.wantErr)
//			}
//			err := os.RemoveAll(tt.fields.repoPath)
//			if err != nil {
//				t.Errorf("Cannot clean up cloned repository Error: %s", err)
//			}
//		})
//	}
//}

//func TestBuiltInManager_Pull(t *testing.T) {
//	type fields struct {
//		remote   string
//		repoPath string
//		repo     *git.Repository
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//		{name: "ssh clone",
//			wantErr: false,
//			fields:  fields{remote: "git@github.com:wix-system/tfChek-testrepo.git", repoPath: "/tmp/testchekrepo"},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			b := &BuiltInManager{
//				remoteUrl: tt.fields.remote,
//				repoPath:  tt.fields.repoPath,
//				repo:      tt.fields.repo,
//			}
//			if err := b.Clone(); (err != nil) != tt.wantErr {
//				t.Errorf("Clone() error = %v, wantErr %v", err, tt.wantErr)
//			}
//			if err := b.Pull(); (err != nil) != tt.wantErr {
//				t.Errorf("Pull() error = %v, wantErr %v", err, tt.wantErr)
//			}
//			err := os.RemoveAll(tt.fields.repoPath)
//			if err != nil {
//				t.Errorf("Cannot clean up cloned repository Error: %s", err)
//			}
//		})
//	}
//}
