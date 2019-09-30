package finder

import "testing"

func TestLocateTerrafrom(t *testing.T) {
	type args struct {
		workdir string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "no such file",
			args:    args{workdir: "/Users/dummy/somedir/production_42/generator/output/someenv/somelayer"},
			want:    "/Users/dummy/somedir/production_42/bin/terraform",
			wantErr: true,
		},
		{name: "should be okay",
			args:    args{workdir: "/Users/maksymsh/Wix/Systems/Terraform/production_42/generator/output/100/db"},
			want:    "/Users/maksymsh/Wix/Systems/Terraform/production_42/bin/terraform",
			wantErr: false,
		},
		{name: "no production_42",
			args:    args{workdir: "/Users/dummy/somedir/someenv/somelayer"},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LocateTerrafrom(tt.args.workdir)
			if (err != nil) != tt.wantErr {
				t.Errorf("LocateTerrafrom() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("LocateTerrafrom() got = %v, want %v", got, tt.want)
			}
		})
	}
}
