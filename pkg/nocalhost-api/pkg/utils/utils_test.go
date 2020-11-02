/*
Copyright 2020 The Nocalhost Authors.
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

package utils

import (
	"testing"
)

func TestGenShortID(t *testing.T) {
	shortID, err := GenShortID()
	if shortID == "" || err != nil {
		t.Error("GenShortID failed!")
	}

	t.Log("GenShortID test pass")
}

func BenchmarkGenShortID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenShortID()
	}
}

func BenchmarkGenShortIDTimeConsuming(b *testing.B) {
	b.StopTimer() //调用该函数停止压力测试的时间计数

	shortID, err := GenShortID()
	if shortID == "" || err != nil {
		b.Error(err)
	}

	b.StartTimer() //重新开始时间

	for i := 0; i < b.N; i++ {
		GenShortID()
	}
}

func TestRandomStr(t *testing.T) {
	test := RandomStr(8)
	t.Log(test)
}
