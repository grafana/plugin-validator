package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func readArchive(archiveURL string) ([]byte, error) {
	if strings.HasPrefix(archiveURL, "https://") || strings.HasPrefix(archiveURL, "http://") {
		resp, err := http.Get(archiveURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusNotFound {
				return nil, errors.New("plugin not found")
			}
			return nil, fmt.Errorf("unexpected status: %s", resp.Status)
		}

		return ioutil.ReadAll(resp.Body)
	}

	return ioutil.ReadFile(archiveURL)
}

func extractPlugin(body io.Reader) (string, func(), error) {
	// Create a file for the zipball.
	zipball, err := ioutil.TempFile("", "")
	if err != nil {
		return "", nil, err
	}
	defer zipball.Close()
	defer os.Remove(zipball.Name())

	if _, err := io.Copy(zipball, body); err != nil {
		return "", nil, err
	}

	// Create a directory where we'll extract the archive.
	output, err := ioutil.TempDir("", "")
	if err != nil {
		return "", nil, err
	}

	cleanup := func() {
		os.RemoveAll(output)
	}

	if _, err := unzip(zipball.Name(), output); err != nil {
		cleanup()
		return "", nil, err
	}

	return output, cleanup, nil
}

func unzip(src string, dest string) ([]string, error) {
	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {
		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}
