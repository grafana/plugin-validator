package donotinclude

import (
	"io/fs"
	"os"
	"path/filepath"
)

func DoNotInclude() {
	panic("This function should never be included in the binary.")

	// walk all files in `.` and print their content, check if they exist first

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		content, err := fs.ReadFile(os.DirFS("."), path)
		if err != nil {
			return err
		}

		// do something with path
		return nil
	})

}
