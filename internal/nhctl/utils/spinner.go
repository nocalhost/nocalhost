/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package utils

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"nocalhost/internal/nhctl/coloredoutput"

	sp "github.com/briandowns/spinner"
)

var spinnerSupport bool

type Spinner struct {
	sp *sp.Spinner
}

//NewSpinner returns a new Spinner
func NewSpinner(suffix string) *Spinner {
	spinnerSupport = !loadBoolean("DISABLE_SPINNER")
	s := sp.New(sp.CharSets[14], 100*time.Millisecond)
	//s.HideCursor = true
	s.Suffix = suffix
	return &Spinner{
		sp: s,
	}
}

func loadBoolean(k string) bool {
	v := os.Getenv(k)
	if v == "" {
		v = "false"
	}
	h, err := strconv.ParseBool(v)
	if err != nil {
		coloredoutput.Yellow("'%s' is not a valid value for environment variable %s", v, k)
	}

	return h
}

//Start starts the spinner
func (p *Spinner) Start() {
	if spinnerSupport {
		p.sp.Start()
	} else {
		fmt.Println(strings.TrimSpace(p.sp.Suffix))
	}
}

//Stop stops the spinner
func (p *Spinner) Stop() {
	if spinnerSupport {
		p.sp.Stop()
	}
}

//Update updates the spinner message
func (p *Spinner) Update(text string) {
	p.sp.Suffix = fmt.Sprintf(" %s", ucFirst(text))
	if !spinnerSupport {
		fmt.Println(strings.TrimSpace(p.sp.Suffix))
	}
}

func ucFirst(str string) string {
	for i, v := range str {
		return string(unicode.ToUpper(v)) + str[i+1:]
	}
	return ""
}
