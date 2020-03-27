package storer

import (
	"fmt"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"log"
	"os"
	"tfChek/misc"
)

func GetTaskPath(id int) (string, error) {
	dir := viper.GetString(misc.OutDirKey)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		log.Printf("tasks output directory does not exist")
		return "", err
	}
	return getTaskPath(dir, id), nil
}

func getTaskPath(dir string, id int) string {
	return fmt.Sprintf("%s/task-%d", dir, id)
}

func GetTaskFileWriteCloser(id int) (io.WriteCloser, error) {
	dir := viper.GetString(misc.OutDirKey)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			log.Printf("Cannot create directory %s Error %s", dir, err)
		}
	}
	file, err := os.Create(getTaskPath(dir, id))
	if err != nil {
		log.Printf("Cannot create file task-%d Error %s", id, err)
		return nil, err

	}
	return file, nil
}

func ReadTask(id int) ([]byte, error) {
	dir := viper.GetString(misc.OutDirKey)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		misc.Debugf("Directory with stored tasks does not exist. Error: %s", err)
		return nil, err
	}
	f, err := os.Open(getTaskPath(dir, id))
	if err != nil {
		misc.Debugf("Cannot find task %d log. Error: %s", id, err)
		return nil, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		misc.Debugf("Cannot read data from the task log. Error %s", err)
	}
	return data, err
}
