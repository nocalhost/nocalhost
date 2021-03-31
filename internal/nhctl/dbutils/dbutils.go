/*
Copyright 2021 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dbutils

import (
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type LevelDBUtils struct {
	readonly bool
	db       *leveldb.DB
}

// It is safe to close a no-open LevelDBUtils
func (l *LevelDBUtils) Close() error {
	if l != nil && l.db != nil {
		return errors.Wrap(l.db.Close(), "")
	}
	return nil
}

func (l *LevelDBUtils) Get(key []byte) (value []byte, err error) {
	return l.db.Get(key, nil)
}

func (l *LevelDBUtils) Put(key []byte, val []byte) error {
	return errors.Wrap(l.db.Put(key, val, nil), "")
}

func (l *LevelDBUtils) ListAll() (map[string]string, error) {
	result := make(map[string]string, 0)
	iter := l.db.NewIterator(nil, nil)
	for iter.Next() {
		result[string(iter.Key())] = string(iter.Value())
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return nil, errors.Wrap(err, "")
	}
	return result, nil
}

func (l *LevelDBUtils) CompactKey(key []byte) error {
	return errors.Wrap(l.db.CompactRange(*util.BytesPrefix(key)), "")
}

// Get db's total size
func (l *LevelDBUtils) GetSize() (int, error) {
	kvMap, err := l.ListAll()
	if err != nil {
		return 0, err
	}
	ranges := make([]util.Range, 0)
	for key := range kvMap {
		ranges = append(ranges, *util.BytesPrefix([]byte(key)))
	}
	s, err := l.db.SizeOf(ranges)
	if err != nil {
		return 0, errors.Wrap(err, "")
	}
	return int(s.Sum()), nil
}
