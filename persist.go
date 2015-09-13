package cookiejar

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"
)

type atomicFile struct {
	filename string
	file     *os.File
	ctime    time.Time
}

var erratomicFileRetry = errors.New("original file newer than new contents")

// Write file to temp and atomically move when everything else succeeds.
func (a *atomicFile) create(filename string) (f *os.File, err error) {
	a.filename = filename
	dir, name := path.Split(filepath.ToSlash(filename))
	a.file, err = ioutil.TempFile(dir, name)
	if err == nil {
		a.ctime = time.Now()
	}
	return a.file, err
}

func (a *atomicFile) isRetry(err error) bool {
	return err == erratomicFileRetry
}

func (a *atomicFile) cancel() error {
	if a.file == nil {
		return nil
	}
	a.file.Close()
	err := os.Remove(a.file.Name())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (a *atomicFile) reset() error {
	if a.file == nil {
		return nil
	}
	a.ctime = time.Now()
	return a.file.Truncate(0)
}

func (a *atomicFile) commit() error {
	if a.file == nil {
		return nil
	}
	err := a.file.Sync()
	if closeErr := a.file.Close(); err == nil {
		err = closeErr
	}
	fi, err := os.Stat(a.filename)
	if err == nil || os.IsNotExist(err) {
		// File was modified after we started writing to the new version.
		if fi != nil && fi.ModTime().After(a.ctime) {
			err = erratomicFileRetry
		} else {
			err = os.Rename(a.file.Name(), a.filename)
		}
	}
	// Any err should result in full cleanup.
	if err != nil {
		a.cancel()
	} else {
		a.file.Close()
	}
	return err
}
