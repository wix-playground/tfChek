package storer

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"io/ioutil"
	"os"
	"testing"
	"tfChek/misc"
	"time"
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
	items := 60
	duration := time.Duration(500) //milliseconds
	lines := make(chan string)
	errs := make(chan error)

	tf, err := ioutil.TempFile("/tmp", "follow_test")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	go func() {
		for i := 1; i <= items; i++ {
			line := fmt.Sprintf("#########%d#######\n", i)
			tf.WriteString(line)
			//t.Logf("Wrote line: %s",line)
			time.Sleep(duration * time.Millisecond)
		}
		err := tf.Close()
		if err != nil {
			t.Log(err)
			t.Fail()
		}
	}()
	tfn := tf.Name()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	t.Logf("Creating watcher for file %s", tfn)
	err = watcher.Add(tf.Name())
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	_, err = os.Stat(tfn)
	if os.IsNotExist(err) {
		misc.Debug(fmt.Sprintf("file %s does not exist. Error: %s", tfn, err.Error()))
		t.Log(err)
		t.Fail()
	}
	f, err := os.Open(tfn)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	cc := make(chan controlFlag)

	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{name: "Simple followin test",
			fields: fields{watcher, f, tfn, cc},
			args:   args{lines, errs}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flwr := &follower{
				watcher:  tt.fields.watcher,
				reader:   tt.fields.reader,
				filePath: tt.fields.filePath,
				control:  tt.fields.control,
			}
			itemCounter := 0

			go func(ic *int) {
				for {
					select {
					case line, ok := <-lines:
						if !ok {
							return
						}
						*ic++
						t.Logf("Read line %d: %s", *ic, line)
					}
				}

			}(&itemCounter)
			go func() {
				time.Sleep(duration*time.Duration(items)*time.Millisecond + time.Second)
				flwr.Stop()
				t.Log("Follower has been stopped")
			}()
			flwr.Follow(lines, errs)
			t.Logf("Items: %d\tCounter: %d", items, itemCounter)
			if itemCounter != items {
				t.Log("Item quantity differs")
				t.Fail()
			}
		})
	}
	err = os.Remove(tfn)
	if err != nil {
		t.Logf("Cannot remove temp file %s", tfn)
	}
}
