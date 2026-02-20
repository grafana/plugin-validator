package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type DiskStore struct {
	baseDir string
}

func NewDiskStore(baseDir string) (*DiskStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("creating storage directory: %w", err)
	}
	return &DiskStore{baseDir: baseDir}, nil
}

func (d *DiskStore) Save(key string, value interface{}) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling data: %w", err)
	}

	path := filepath.Join(d.baseDir, key+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", path, err)
	}

	return nil
}

func (d *DiskStore) Load(key string, dest interface{}) error {
	path := filepath.Join(d.baseDir, key+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", path, err)
	}

	return json.Unmarshal(data, dest)
}

func (d *DiskStore) Delete(key string) error {
	path := filepath.Join(d.baseDir, key+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing file %s: %w", path, err)
	}
	return nil
}

func (d *DiskStore) List() ([]string, error) {
	entries, err := os.ReadDir(d.baseDir)
	if err != nil {
		return nil, fmt.Errorf("listing storage: %w", err)
	}

	var keys []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			keys = append(keys, e.Name()[:len(e.Name())-5])
		}
	}
	return keys, nil
}
