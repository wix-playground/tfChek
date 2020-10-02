package github

import (
	"archive/zip"
	"fmt"
	"github.com/wix-playground/tfChek/misc"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

func DownloadRevision(manager *Manager, ref, dest string) (string, error) {
	link, err := manager.GetClient().GetArchiveLink(ref)
	if err != nil {
		return "", err
	}
	resp, err := http.Get(link.String())
	if err != nil {
		return "", fmt.Errorf("failed to download repo archive by link %s. Error: %w", link.String(), err)
	}
	base := path.Base(dest)
	stat, err := os.Stat(base)
	if os.IsNotExist(err) {
		misc.Debugf("%s directory does not exist. Trying to create one...", base)
		err := os.MkdirAll(base, 0755)
		if err != nil {
			misc.Debugf("error directory %s does not exist", base)
			return "", fmt.Errorf("destination directory %s does not exist", base)
		} else {
			misc.Debugf("directory %s has been successfully created", base)
			stat, err = os.Stat(base)
			if err != nil {
				return "", fmt.Errorf("Cannot stat %s directory. Error: %w", base, err)
			}
		}
	}
	if !stat.IsDir() {
		return "", fmt.Errorf("destination directory %s is not a directory", base)
	}
	zipFile, err := os.OpenFile(fmt.Sprintf("%s-%s.zip", dest, ref), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file to store zip stream. Error: %w", err)
	}
	zipAbs, err := filepath.Abs(zipFile.Name())
	if err != nil {
		return "", fmt.Errorf("cannot get absolute path of zipfile %s Error: %w", zipFile.Name(), err)
	}
	n, err := io.Copy(zipFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save zip stream to a file. Error: %w", err)
	}
	misc.Debugf("saved reposoritory snapshot (ref: %s) %d bytes long zip archive to %s", ref, n, dest+".zip")
	container, err := unzip(zipAbs, dest)
	if err != nil {
		return "", fmt.Errorf("failed to extract archive from the web stream. Error: %w", err)
	}
	misc.Debugf("extracted reposoritory snapshot (ref: %s) to %s", ref, dest)
	return container, nil
}

//TODO: Mark repo with a timestamp file for further deletion (rotation)

//This function should return correct path to the repository root after unzipping
//Deprecated
func checkPath(destination string) (string, error) {
	ds, err := os.Stat(destination)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("destination directory does not exist. Error: %w", err)
		} else {
			return "", fmt.Errorf("cannot stat directory %s. Error: %w", destination, err)
		}
	}
	if !ds.IsDir() {
		return "", fmt.Errorf("destination is not a directory")
	}
	list, err := ioutil.ReadDir(destination)
	if err != nil {
		return "", fmt.Errorf("cannot list directory contents. Error: %s", err)
	}
	if len(list) == 1 && list[0].IsDir() {
		misc.Debugf("%s will be added to a destination as a result of path checking", list[0].Name())
		return path.Join(destination, list[0].Name()), nil
	}
	misc.Debugf("destination destination was left unmodified after checking")
	return destination, nil
}

func unzip(src, dest string) (string, error) {
	//It appeared that zip format requires a random access for unarchiving process

	firstDirectory := ""

	r, err := zip.OpenReader(src)
	if err != nil {
		return "", fmt.Errorf("cannot open zip file %s Error: %w", src, err)
	}
	//Perhaps panic is not needed here
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	err = os.MkdirAll(dest, 0755)
	if err != nil {
		return "", fmt.Errorf("cannot create a directory to unzip a stream. Error %w", err)
	}

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			//Add error checking here
			if firstDirectory == "" {
				firstDirectory = path
			}
			os.MkdirAll(path, f.Mode())
		} else {
			//Add error checking here
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return firstDirectory, err
		}
	}

	return firstDirectory, nil
}
