package storer

import (
	"github.com/spf13/viper"
	"testing"
	"tfChek/misc"
)

func Test_createSequenceTable(t *testing.T) {
	type args struct {
		name string
		wait bool
	}
	viper.SetDefault(misc.AWSSequenceTable, "tfChek-sequence-test")
	viper.SetDefault(misc.AWSRegion, "us-east-1")
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "Create table test",
			args:    args{name: "tfChek-sequence-test", wait: true},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := createSequenceTable(tt.args.name, tt.args.wait); (err != nil) != tt.wantErr {
				t.Errorf("createSequenceTable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_listSequenceTable(t *testing.T) {
	type args struct {
		name string
	}
	viper.SetDefault(misc.AWSSequenceTable, "tfChek-sequence-test")
	viper.SetDefault(misc.AWSRegion, "us-east-1")
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{name: "List table error test",
			args:    args{name: "tfChek-sequence-test"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := listSequenceTable(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("listSequenceTable() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !r {
				t.Errorf("listSequenceTable() failed to found table")
			}
		})
	}
}

func Test_updateSequence(t *testing.T) {
	type args struct {
		seq  int
		name string
	}
	viper.SetDefault(misc.AWSSequenceTable, "tfChek-sequence-test")
	viper.SetDefault(misc.AWSRegion, "us-east-1")
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{name: "Put sequence test",
			args:    args{seq: 101, name: "tfChek-sequence-test"},
			wantErr: false},
		{name: "Update sequence test",
			args:    args{seq: 111, name: "tfChek-sequence-test"},
			wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := updateSequence(tt.args.seq, tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("updateSequence() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getSequence(t *testing.T) {
	type args struct {
		name string
	}
	viper.SetDefault(misc.AWSSequenceTable, "tfChek-sequence-test")
	viper.SetDefault(misc.AWSRegion, "us-east-1")
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		// TODO: Add test cases.
		{name: "Get sequence test",
			args:    args{name: "tfChek-sequence-test"},
			wantErr: false,
			want:    111,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getSequence(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSequence() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getSequence() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deleteSequenceTable(t *testing.T) {
	type args struct {
		name string
	}
	viper.SetDefault(misc.AWSSequenceTable, "tfChek-sequence-test")
	viper.SetDefault(misc.AWSRegion, "us-east-1")
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "Delete table test",
			args:    args{name: "tfChek-sequence-test"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := deleteSequenceTable(tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("deleteSequenceTable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
