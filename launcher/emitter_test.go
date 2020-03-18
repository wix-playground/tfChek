package launcher

import (
	"github.com/fsnotify/fsnotify"
	"os"
	"reflect"
	"testing"
)

func Test_follower_Follow(t *testing.T) {
	type fields struct {
		watcher  *fsnotify.Watcher
		reader   *os.File
		filePath string
		control  chan controlFlag
	}
	type args struct {
		lines chan<- string
		errs  chan<- error
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &follower{
				watcher:  tt.fields.watcher,
				reader:   tt.fields.reader,
				filePath: tt.fields.filePath,
				control:  tt.fields.control,
			}
		})
	}
}
