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
	for {
		// Create a temporary file
		var af atomicFile
		f, err := af.create(j.filename)
		if err != nil {
			return err
		}
		// Write out to the temporary file
		err = j.WriteTo(f)
		if err != nil {
			// On write error, remove the temp file
			af.cancel()
			return err
		}
		// Try replacing the original file with our temporary one.
		// If the file to be replaced is newer, close() fails.
		err = af.close()
		// Success.
		if err == nil {
			return nil
		}
		// We failed, remove the temporary file.
		af.cancel()
		// Some error occurred, not related to retrying mechanism.
		if !af.isRetry(err) {
			return err
		}
		// Load the entries from the file to overwrite.
		m := make(map[string]map[string]entry)
		if err := loadJSON(j.filename, m); err != nil {
			continue
		}
		// Merge them on top of ours (they are newer).
		j.mu.Lock()
		j.mergeEntries(m)
		j.mu.Unlock()
	}
}

// Load uses j.ReadFrom to read cookies
// from the file at the given path. If the file does not exist,
// no error will be returned and no cookies
// will be loaded.
//
// The path will be stored in the jar and
// used when j.Save is next called.
func (j *Jar) Load(path string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := loadJSON(path, j.entries); err != nil {
		return err
	}
	j.filename = path
	return nil
}

// WriteTo writes all the cookies in the jar to w
// as a JSON object.
func (j *Jar) WriteTo(w io.Writer) error {
	// TODO don't store non-persistent cookies.
	j.mu.Lock()
	defer j.mu.Unlock()
	return encodeJSON(w, j.entries)
}

// ReadFrom reads all the cookies from r
// and stores them in the Jar.
func (j *Jar) ReadFrom(r io.Reader) error {
	// TODO verification and expiry on read.
	j.mu.Lock()
	defer j.mu.Unlock()
	return decodeJSON(r, j.entries)
}

func (j *Jar) mergeEntries(m map[string]map[string]entry) {
	for k0 := range m {
		if _, ok := j.entries[k0]; !ok {
			j.entries[k0] = make(map[string]entry)
		}
		for k1 := range m[k0] {
			j.entries[k0][k1] = m[k0][k1]
		}
	}
}

func loadJSON(path string, m map[string]map[string]entry) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	return decodeJSON(f, m)
}

func encodeJSON(w io.Writer, m map[string]map[string]entry) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(m)
}

func decodeJSON(r io.Reader, m map[string]map[string]entry) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(&m)
}
