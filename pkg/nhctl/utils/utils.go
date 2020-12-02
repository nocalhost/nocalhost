package utils

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"io"
	"path/filepath"
	"strings"
	"sync"
)

func Mush(err error) {
	if err != nil {
		fmt.Printf("%v\n", err)
		panic(err)
	}
}

func MushWithErrInfo(info string, err error) {
	if err != nil {
		fmt.Printf("%s, err : %v\n", info, err)
		panic(err)
	}
}

type ProgressBar struct {
	lock sync.Mutex
}

func (cpb *ProgressBar) TrackProgress(src string, currentSize, totalSize int64, stream io.ReadCloser) io.ReadCloser {
	cpb.lock.Lock()
	defer cpb.lock.Unlock()

	newPb := pb.New64(totalSize)
	newPb.Set("prefix", fmt.Sprintf("%s ", filepath.Base(src)))
	newPb.SetCurrent(currentSize)
	newPb.Start()
	reader := newPb.NewProxyReader(stream)

	return &readCloser{
		Reader: reader,
		close: func() error {
			cpb.lock.Lock()
			defer cpb.lock.Unlock()

			newPb.Finish()
			return nil
		},
	}
}

type readCloser struct {
	io.Reader
	close func() error
}

func (c *readCloser) Close() error { return c.close() }

func RenderProgressBar(prefix string, current, scalingFactor float64) string {
	var sb strings.Builder
	_, _ = sb.WriteString(prefix)
	_, _ = sb.WriteString("[")

	scaledMax := int(100 * scalingFactor)
	scaledCurrent := int(current * scalingFactor)

	switch {
	case scaledCurrent == 0:
		_, _ = sb.WriteString(strings.Repeat("_", scaledMax))
	case scaledCurrent >= scaledMax:
		_, _ = sb.WriteString(strings.Repeat("-", scaledMax))
	default:
		_, _ = sb.WriteString(strings.Repeat("-", scaledCurrent-1))
		_, _ = sb.WriteString(">")
		_, _ = sb.WriteString(strings.Repeat("_", scaledMax-scaledCurrent))
	}

	_, _ = sb.WriteString("]")
	_, _ = sb.WriteString(fmt.Sprintf(" %3v%%", int(current)))
	return sb.String()
}
