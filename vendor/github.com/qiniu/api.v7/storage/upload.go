package storage

import (
	"bytes"
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"sync"
	"time"

	"github.com/qiniu/api.v7"
	"github.com/qiniu/api.v7/internal/log"
)

const (
	// 上传一个分片失败
	ErrUploadChunkFailed = "ErrUploadChunkFailed"

	// 获取下一个分片Reader失败
	ErrNextReader = "ErrNextReader"

	// 超过了最大的重试上传次数
	ErrMaxUpRetry = "ErrMaxUpRetry"

	// 取消了分片的上传
	ErrChunkUpCanceled = "ErrChunkUpCanceled"
)

type defaultChunkRetryer struct {
	// 最大的重试次数
	maxCount int

	// 重试的间隔秒
	delay int

	uploader *uploader
}

func (r *defaultChunkRetryer) Retry(ck *Chunk) {
	var i = 0
	for ; i < r.maxCount; i++ {
		if ck.ShouldRetry() {
			log.Debug("Retrying to upload chunk: ", ck.Index)
			if err := ck.Reset(); err != nil {
				log.Warn(fmt.Sprintf("Reset chunk body error: %q, chunk index: %d", err, ck.Index))
				ck.Err = err
				continue
			}
			time.Sleep(time.Second * time.Duration(r.delay))
			r.uploader.chunkPut(ck)
		} else {
			return
		}
	}
	if i >= r.maxCount {
		ck.Err = api.NewError(ErrMaxUpRetry, "exceed max retry times")
	}
}

// ChunkPutRetryer 上传分片失败时候重试接口
type ChunkPutRetryer interface {
	Retry(ck *Chunk)
}

// Chunk表示要上传的数据块, 该片的大小不能大于4M
// 上传块的过程： 1. 调用接口在七牛后端创建块 2. 上传数据到该块
// 详细可以参考 https://developer.qiniu.com/kodo/api/1286/mkblk
type Chunk struct {
	// 要上传的块数据
	Body io.ReadSeeker

	// 该片数据所属的块的大小
	BlkSize int

	// 将大的数据切成4M一块，每个块都有一个index
	// 该片数据所属的块的index
	Index int

	// 上传该片发生的错误
	Err error

	// 是否调用了mkblk接口在后台创建了该片所属的块
	Created bool

	// 上传块的返回值
	Ret *BlkputRet
}

// ShouldRetry 是否需要重新上传
func (b *Chunk) ShouldRetry() bool {
	return b.Err != nil
}

// BlockLength 返回实际要上传的数据的长度
func (b *Chunk) ChunkLength() (int, error) {
	n, err := api.SeekerLen(b.Body)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// ResetBody 重置Body到开头
func (b *Chunk) ResetBody() error {
	_, err := b.Body.Seek(0, io.SeekStart)
	return err
}

// Reset 重置Body和Err
func (b *Chunk) Reset() error {
	err := b.ResetBody()
	if err != nil {
		return err
	}
	b.Err = nil
	return nil
}

type uploader struct {
	*ResumeUploader
	workers   int
	blkSize   int64 // 块大小
	body      io.Reader
	readerPos int64 // current reader position
	totalSize int64 // set to -1 if the size is not known

	bufferPool sync.Pool

	// 每个块重试上传逻辑
	retryer ChunkPutRetryer

	// 上传token
	upToken string

	// 上传Host
	upHost string

	// 当一个片上传失败的时候发送到这个channel
	errChan chan struct{}

	success chan *Chunk

	ctx context.Context

	Notify    func(blkIdx int, blkSize int, ret *BlkputRet) // 可选。进度提示（注意多个block是并行传输的）
	NotifyErr func(blkIdx int, blkSize int, err error)

	wg sync.WaitGroup
}

func (u *uploader) init() {
	u.bufferPool = sync.Pool{
		New: func() interface{} { return make([]byte, u.blkSize) },
	}
	u.totalSize = -1

	switch r := u.body.(type) {
	case io.Seeker:
		n, err := api.SeekerLen(r)
		if err != nil {
			return
		}
		u.totalSize = n
	}
	u.retryer = &defaultChunkRetryer{
		maxCount: defaultTryTimes,
		delay:    3,
	}
	if u.ctx == nil {
		u.ctx = context.Background()
	}
	if u.errChan == nil {
		u.errChan = make(chan struct{})
	}
	if u.workers <= 0 {
		u.workers = 10
	}
	if u.success == nil {
		u.success = make(chan *Chunk)
	}
}

func (u *uploader) upload(ret interface{}, key string, hasKey bool, extra *RputExtra) error {
	u.init()

	defer close(u.success)
	ctx, cancelFunc := context.WithCancel(u.ctx)
	u.ctx = ctx

	go func() {
		<-u.errChan
		cancelFunc()
	}()

	cks := make(map[int]*Chunk)
	go func() {
		for c := range u.success {
			if _, ok := cks[c.Index]; ok {
				panic("chunk with same index")
			}
			if c.Index <= 0 {
				panic("index less than 1")
			}
			cks[c.Index] = c

			u.wg.Done()
		}
	}()

	sema := make(chan struct{}, u.workers)

	var (
		index     int
		totalSize int64
		n         int
		part      []byte
		nerr      error
		reader    io.ReadSeeker
	)
	for nerr == nil {
		reader, n, part, nerr = u.nextReader()
		sema <- struct{}{}
		u.wg.Add(1)

		index++
		chunk := Chunk{
			Body:    reader,
			Index:   index,
			Created: false,
			BlkSize: n,
		}
		totalSize += int64(n)
		go u.uploadChunk(sema, &chunk, part)
	}
	if nerr != nil && nerr != io.EOF {
		cancelFunc()
		//  等待所有的worker结束
		u.wg.Wait()
		return api.NewError(ErrNextReader, nerr.Error())
	}
	u.wg.Wait()
	if u.totalSize == -1 {
		u.totalSize = totalSize
	}
	//log.Debug(cks)
	extra.Progresses = make([]BlkputRet, index)
	for ind, c := range cks {
		if ind < 0 {
			panic("negative index count")
		}
		extra.Progresses[ind-1] = *c.Ret
	}
	return u.Mkfile(ctx, u.upToken, u.upHost, ret, key, hasKey, u.totalSize, extra)
}

func (u *uploader) chunkPut(chunk *Chunk) {
	h := crc32.NewIEEE()
	body := io.TeeReader(chunk.Body, h)
	chunkLength, berr := chunk.ChunkLength()
	if berr != nil {
		chunk.Err = berr
		return
	}
	if !chunk.Created {
		merr := u.Mkblk(u.ctx, u.upToken, u.upHost, chunk.Ret, chunk.BlkSize, body, chunkLength)
		if merr != nil {
			chunk.Err = merr
			return
		}
		if chunk.Ret.Crc32 != h.Sum32() || int(chunk.Ret.Offset) != chunkLength {
			chunk.Err = ErrUnmatchedChecksum
			return
		}
		chunk.Created = true
	} else {
		// 上传分片要么成功要么失败，不存在一部分成功的情况
		err := u.Bput(u.ctx, u.upToken, chunk.Ret, body, chunkLength)
		if err != nil {
			chunk.Err = err
			return
		}
		if chunk.Ret.Crc32 != h.Sum32() {
			chunk.Err = ErrUnmatchedChecksum
			return
		}
	}
	u.Notify(chunk.Index, chunk.BlkSize, chunk.Ret)
}

func (u *uploader) uploadChunk(sema chan struct{}, chunk *Chunk, part []byte) {
	if chunk.Ret == nil {
		chunk.Ret = new(BlkputRet)
	}

	u.chunkPut(chunk)
	if u.retryer != nil && chunk.Err != context.Canceled {
		u.retryer.Retry(chunk)
	}
	if chunk.Err != nil {
		u.errChan <- struct{}{}
		u.NotifyErr(chunk.Index, chunk.BlkSize, chunk.Err)
	}

	u.bufferPool.Put(part)
	<-sema
	u.success <- chunk
}

func (u *uploader) nextReader() (io.ReadSeeker, int, []byte, error) {
	type readerAtSeeker interface {
		io.ReaderAt
		io.ReadSeeker
	}
	switch r := u.body.(type) {
	case readerAtSeeker:
		var err error

		n := u.blkSize
		if u.totalSize >= 0 {
			bytesLeft := u.totalSize - u.readerPos

			if bytesLeft <= u.blkSize {
				err = io.EOF
				n = bytesLeft
			}
		}

		reader := io.NewSectionReader(r, u.readerPos, n)
		u.readerPos += n

		return reader, int(n), nil, err

	default:
		part := u.bufferPool.Get().([]byte)
		n, err := readFillBuf(r, part)
		u.readerPos += int64(n)

		return bytes.NewReader(part[0:n]), n, part, err
	}
}

func readFillBuf(r io.Reader, b []byte) (offset int, err error) {
	for offset < len(b) && err == nil {
		var n int
		n, err = r.Read(b[offset:])
		offset += n
	}

	return offset, err
}
