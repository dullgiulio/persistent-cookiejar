// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cookiejar

import (
	"encoding/json"
	"errors"
	"io"
	"os"
)

// Save uses j.WriteTo to save the cookies in j to a file at the path
// they were loaded from with Load. Note that there is no locking
// of the file, so concurrent calls to Save and Load can yield
// corrupted or missing cookies.
//
// It returns an error if Load was not called.
func (j *Jar) Save() error {
	if j.filename == "" {
		return errors.New("save called on non-loaded cookie jar")
	}
	var af AtomicFile
	f, err := af.Create(j.filename)
	if err != nil {
		return err
	}
	err = j.WriteTo(f)
	if err != nil {
		af.Cancel()
		return err
	}
	err = af.Close()
	if af.IsRetry(err) {
		// TODO: Read again, merge, save.
	}
	return err
}

// Load uses j.ReadFrom to read cookies
// from the file at the given path. If the file does not exist,
// no error will be returned and no cookies
// will be loaded.
//
// The path will be stored in the jar and
// used when j.Save is next called.
func (j *Jar) Load(path string) error {
	j.filename = path
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	return j.ReadFrom(f)
}

// WriteTo writes all the cookies in the jar to w
// as a JSON object.
func (j *Jar) WriteTo(w io.Writer) error {
	// TODO don't store non-persistent cookies.
	encoder := json.NewEncoder(w)
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := encoder.Encode(j.entries); err != nil {
		return err
	}
	return nil
}

// ReadFrom reads all the cookies from r
// and stores them in the Jar.
func (j *Jar) ReadFrom(r io.Reader) error {
	// TODO verification and expiry on read.
	decoder := json.NewDecoder(r)
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := decoder.Decode(&j.entries); err != nil {
		return err
	}
	return nil
}
