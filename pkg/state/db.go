package state

import (
	"encoding/json"
	"time"

	"go.etcd.io/bbolt"
)

type FileMetadata struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	ETag    string    `json:"etag,omitempty"`
}

type DB struct {
	db *bbolt.DB
}

func Open(path string) (*DB, error) {
	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("files"))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte("hits"))
		return err
	})

	if err != nil {
		db.Close()
		return nil, err
	}

	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

// ShouldScan returns true if the file has changed since last scan
func (d *DB) ShouldScan(path string, size int64, modTime time.Time) bool {
	var meta FileMetadata
	err := d.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("files"))
		v := b.Get([]byte(path))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &meta)
	})

	if err != nil || meta.Path == "" {
		return true
	}

	// If size or modtime changed, we should re-scan
	return meta.Size != size || !meta.ModTime.Equal(modTime)
}

func (d *DB) MarkScanned(path string, size int64, modTime time.Time) error {
	meta := FileMetadata{
		Path:    path,
		Size:    size,
		ModTime: modTime,
	}
	data, _ := json.Marshal(meta)

	return d.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("files"))
		return b.Put([]byte(path), data)
	})
}

// IsDuplicate returns true if the hit has been seen before (globally)
func (d *DB) IsDuplicate(hash string) bool {
	exists := false
	_ = d.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("hits"))
		if b.Get([]byte(hash)) != nil {
			exists = true
		}
		return nil
	})
	return exists
}

func (d *DB) MarkHit(hash string) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("hits"))
		return b.Put([]byte(hash), []byte("1"))
	})
}
