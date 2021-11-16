package internal

import (
	"fmt"
	"os"
	"strconv"
	"sync"
)

// Those suggest value are all set according to
// https://github.com/shadowsocks/shadowsocks-org/issues/44#issuecomment-281021054
// Due to this package contains various internal implementation so const named with DefaultBR prefix
const (
	DefaultSFCapacity = 1e6
	// FalsePositiveRate
	DefaultSFFPR  = 1e-6
	DefaultSFSlot = 10
)

const EnvironmentPrefix = "SHADOWSOCKS_"

// A shared instance used for checking salt repeat
var saltfilter *BloomRing

// Used to initialize the saltfilter singleton only once.
var initSaltfilterOnce sync.Once

// GetSaltFilterSingleton returns the BloomRing singleton,
// initializing it on first call.
func getSaltFilterSingleton() *BloomRing {
	initSaltfilterOnce.Do(func() {
		var (
			finalCapacity = DefaultSFCapacity
			finalFPR      = DefaultSFFPR
			finalSlot     = float64(DefaultSFSlot)
		)
		for _, opt := range []struct {
			ENVName string
			Target  *float64
		}{
			{
				ENVName: "CAPACITY",
				Target:  &finalCapacity,
			},
			{
				ENVName: "FPR",
				Target:  &finalFPR,
			},
			{
				ENVName: "SLOT",
				Target:  &finalSlot,
			},
		} {
			envKey := EnvironmentPrefix + "SF_" + opt.ENVName
			env := os.Getenv(envKey)
			if env != "" {
				p, err := strconv.ParseFloat(env, 64)
				if err != nil {
					panic(fmt.Sprintf("Invalid envrionment `%s` setting in saltfilter: %s", envKey, env))
				}
				*opt.Target = p
			}
		}
		// Support disable saltfilter by given a negative capacity
		if finalCapacity <= 0 {
			return
		}
		saltfilter = NewBloomRing(int(finalSlot), int(finalCapacity), finalFPR)
	})
	return saltfilter
}

// TestSalt returns true if salt is repeated
func TestSalt(b []byte) bool {
	return getSaltFilterSingleton().Test(b)
}

// AddSalt salt to filter
func AddSalt(b []byte) {
	getSaltFilterSingleton().Add(b)
}

func CheckSalt(b []byte) bool {
	return getSaltFilterSingleton().Test(b)
}
