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
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"testing"
)

func TestOpenLevelDB(t *testing.T) {
	db, err := leveldb.OpenFile("/tmp/tmp/db", &opt.Options{
		ErrorIfMissing: true,
	})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf("%v exists\n", db)
	}
}
