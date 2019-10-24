package github

import (
	"os"
	"testing"
)

var num int = 1

func TestClientRunSH_CreatePR(t *testing.T) {
	type fields struct {
		Repository string
		Owner      string
		Token      string
	}
	type args struct {
		branch string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{name: "Pull request test",
			args:    args{branch: "test"},
			wantErr: false,
			fields: fields{Repository: "tfChek-testrepo",
				Owner: "wix-system",
				Token: os.Getenv("TFCHEK_TOKEN"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//c := &ClientRunSH{
			//	Repository: tt.fields.Repository,
			//	Owner:      tt.fields.Owner,
			//	Token:      tt.fields.Token,
			//}
			c := NewClientRunSH(tt.fields.Repository, tt.fields.Owner, tt.fields.Token)
			n, err := c.CreatePR(tt.args.branch)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePR() error = %v, wantErr %v", err, tt.wantErr)
			}
			if n != nil {
				num = *n
			}
		})
	}
}

func TestClientRunSH_RequestReview(t *testing.T) {
	type fields struct {
		Repository string
		Owner      string
		Token      string
	}
	type args struct {
		number    int
		reviewers *[]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{name: "Pull request test",
			args: args{number: num, reviewers: &[]string{"system-bot"}},
			//Author cannot request a pull review of itself
			wantErr: true,
			fields: fields{Repository: "tfChek-testrepo",
				Owner: "wix-system",
				Token: os.Getenv("TFCHEK_TOKEN"),
			},
		},
		{name: "Pull request test",
			args:    args{number: num, reviewers: &[]string{"maskimko", "remm"}},
			wantErr: false,
			fields: fields{Repository: "tfChek-testrepo",
				Owner: "wix-system",
				Token: os.Getenv("TFCHEK_TOKEN"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClientRunSH(tt.fields.Repository, tt.fields.Owner, tt.fields.Token)
			if err := c.RequestReview(tt.args.number, tt.args.reviewers); (err != nil) != tt.wantErr {
				t.Errorf("Review() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientRunSH_Review(t *testing.T) {
	type fields struct {
		Repository string
		Owner      string
		Token      string
	}
	type args struct {
		number  int
		comment string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{name: "Pull request test",
			args:    args{number: num, comment: "test tfChek"},
			wantErr: false,
			fields: fields{Repository: "tfChek-testrepo",
				Owner: "wix-system",
				Token: os.Getenv("TFCHEK_TOKEN"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClientRunSH(tt.fields.Repository, tt.fields.Owner, tt.fields.Token)
			if err := c.Review(tt.args.number, tt.args.comment); (err != nil) != tt.wantErr {
				t.Errorf("Review() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientRunSH_Comment(t *testing.T) {
	cmnt := "test tfChek ------------ long comment"
	type fields struct {
		Repository string
		Owner      string
		Token      string
	}
	type args struct {
		number  int
		comment *string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{name: "Pull request test",
			args:    args{number: num, comment: &cmnt},
			wantErr: false,
			fields: fields{Repository: "tfChek-testrepo",
				Owner: "wix-system",
				Token: os.Getenv("TFCHEK_TOKEN"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClientRunSH(tt.fields.Repository, tt.fields.Owner, tt.fields.Token)
			if err := c.Comment(tt.args.number, tt.args.comment); (err != nil) != tt.wantErr {
				t.Errorf("Review() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientRunSH_Close(t *testing.T) {
	type fields struct {
		Repository string
		Owner      string
		Token      string
	}
	type args struct {
		number int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{name: "Pull request test",
			args:    args{number: num},
			wantErr: false,
			fields: fields{Repository: "tfChek-testrepo",
				Owner: "wix-system",
				Token: os.Getenv("TFCHEK_TOKEN"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClientRunSH(tt.fields.Repository, tt.fields.Owner, tt.fields.Token)
			if err := c.Close(tt.args.number); (err != nil) != tt.wantErr {
				t.Errorf("Review() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
