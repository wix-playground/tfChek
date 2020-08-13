package github

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

func DownloadRevision(manager *Manager, ref, dest string) error {
	link, err := manager.GetClient().GetArchiveLink(ref)
	if err != nil {
		return err
	}
	resp, err := http.Get(link.String())
	if err != nil {
		return fmt.Errorf("failed to download repo archive by link %s. Error: %w", link.String(), err)
	}
	base := path.Base(dest)
	stat, err := os.Stat(base)
	if os.IsNotExist(err) {
		return fmt.Errorf("destination directory %s does not exist", base)
	}
	if !stat.IsDir() {
		return fmt.Errorf("destination directory %s is not a directory", base)
	}
	//TODO create a temp file
	err = unzip(resp.Body, dest)
	if err != nil {
		return fmt.Errorf("failed to extract archive from the web stream. Error: %w", err)
	}
	return nil
}

func unzip(src io.ReadCloser, dest string) error {
	//It appeared that zip format requires a random access for unarchiving process
	zip.NewReader(src, 0)
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	err := os.MkdirAll(dest, 0755)
	if err != nil {
		return fmt.Errorf("cannot create a directory to unzip a stream. Error %w", err)
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
			os.MkdirAll(path, f.Mode())
		} else {
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
			return err
		}
	}

	return nil
}
