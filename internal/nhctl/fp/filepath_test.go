package fp

import (
	"os"
	"testing"
)

const (
	FP01 = "fp-01"
	FP02 = "fp-02"
	FP03 = "fp-03"
)

func TestFilePath01(t *testing.T) {
	path, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	fp01 := NewFilePath(path).RelOrAbs("testdata/tmp1/fp-01.test")

	if fp01.ReadFile() != FP01 {
		t.Error("Not matched!!")
	}
}

func TestFilePath02(t *testing.T) {
	path, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	fp02 := NewFilePath(path).RelOrAbs("testdata/tmp2/fp-02.test")
	if err != nil {
		t.Error(err)
	}

	if fp02.ReadFile() != FP02 {
		t.Error("Not matched!!")
	}
}

func TestFilePath03(t *testing.T) {
	path, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	fp03 := NewFilePath(path).RelOrAbs("testdata/tmp1/subtemp1/subtemp2/fp-03.test")
	if err != nil {
		t.Error(err)
	}

	if fp03.ReadFile() != FP03 {
		t.Error("Not matched!!")
	}
}

func TestFilePathComplex(t *testing.T) {
	path, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	fp02 := NewFilePath(path).RelOrAbs("testdata/../testdata/////////././././tmp1/././././subtemp1/subtemp2/../../../tmp2/fp-02.test")
	if err != nil {
		t.Error(err)
	}

	if fp02.ReadFile() != FP02 {
		t.Error("Not matched!!")
	}
}
