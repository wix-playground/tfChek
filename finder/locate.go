package finder

import (
	"errors"
	"fmt"
	"github.com/wix-playground/tfChek/misc"
	"log"
	"os"
	"strings"
)

func LocateRepo(workdir string) (string, error) {
	if !strings.HasPrefix(workdir, "/") {
		return "", errors.New("path should be absolute and starts from /")
	}
	chunks := strings.Split(workdir, "/")
	var p []string
	for i, f := range chunks {
		if f == misc.PROD42 {
			p = chunks[:i+1]
			break
		}
	}
	if len(p) == 0 {
		return "", errors.New("could not locate repo root")
	}
	tp := strings.Join(p, "/")
	return tp, nil
}

func LocateTerrafrom(workdir string) (string, error) {
	repo, err := LocateRepo(workdir)
	if err != nil {
		return "", err
	}
	tp := repo + "/bin/terraform"
	ext, _ := checkExecutable(tp)
	if !ext {
		return tp, errors.New(fmt.Sprintf("Terrafrom binary is not executable"))
	}
	return tp, nil
}

func checkExecutable(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if !os.IsExist(err) {
			log.Printf("file does not exist at %s", path)
			return false, err
		} else {
			log.Printf("Cannot check file info. Error: %s", err)
		}
	}
	if info.Mode()|0111 == 0 {
		log.Printf("file is not executable. Check premissions for info %s", path)
		return false, nil
	}
	return true, nil
}
func LocateRunSh(workdir string) (string, error) {
	repo, err := LocateRepo(workdir)
	if err != nil {
		return "", err
	}
	tp := repo + "/run.sh"
	ext, _ := checkExecutable(tp)
	if !ext {
		return tp, errors.New(fmt.Sprintf("Run.sh script is not executable. Please check permissions"))
	}
	return tp, nil
}
