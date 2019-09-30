package kodo

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/defs"
	"github.com/qiniu/go-sdk/qiniu/qerr"
	"github.com/qiniu/go-sdk/qiniu/request"
)

const (
	// DefaultUploadPartSize 定义了分片上传每一个片的默认大小
	DefaultUploadPartSize = 4 * defs.MB

	// DefaultUploadConcurrency 定义了分片上传默认的goroutines数目
	DefaultUploadConcurrency = 5

	// DefaultFormSize 定义了默认最大的可以用表单上传的数据大小
	DefaultFormSize = 10 * defs.MB

	// DefaultResumeSize 定义了最小的可以开启断点续传的文件大小
	DefaultResumeSize = 100 * defs.MB

	// DefaultStoreNumber 定义了上传多少个块后，保存目前上传的块的信息到文件，以支持断点续传
	// 该值最小为1， 如果设置的很小，那么没上传一个分片都要写一次文件，如果文件很大，写入的文件次数较多
	// 如果设置的过大， 那么断点续传的时候可能保存的记录信息比较少，导致大部分文件内容重新上传
	//
	// 默认是上传10 * DefaultUploadPartSize 数据保存一次记录, 如果上传的文件大小小于该值相当于没有断点续传的功能
	DefaultStoreNumber = 10
)

const (
	// ErrMd5Written 计算数据的md5值出错
	ErrMd5Written = "Md5WrittenError"
)

type part struct {
	// uploadID 唯一地标识一个文件分片上传的过程
	uploadID string

	// index 是要上传的块的索引值
	index int

	// data 要上传的数据块的数据
	data []byte
}

// partUploadOutput 上传块接口的返回结构
type partUploadOutput struct {
	Etag string `json:"etag"`
	Md5  string `json:"md5"`
}

type partInitOutput struct {
	UploadID string `json:"uploadId,omitempty"`
	ExpireAt int64  `json:"expireAt,omitempty"`
}

// 如果expireAt是空串或者非法的字符串，则返回time.Time{}
func (o *partInitOutput) expiredAt() time.Time {
	if o.ExpireAt <= 0 {
		return time.Time{}
	}
	return time.Unix(o.ExpireAt, 0)
}

func computeMd5(data []byte) (string, error) {
	hasher := md5.New()
	n, err := hasher.Write(data)
	if n != len(data) || err != nil {
		return "", qerr.New(ErrMd5Written, "failed to write data to md5 hasher", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func debugLogMultipartUpload(loglevel *qiniu.LogLevelType, logger qiniu.Logger, part *part) {
	if loglevel.Matches(qiniu.LogDebugMultipartUpload) {
		logger.Log(fmt.Sprintf("Uploading part %d with uploadID: %s\n", part.index, part.uploadID))
	}
}

// ProgressRecorder 是上传进度条的接口
// 默认的实现会输出上传进度
// 如果需要使用自己的实现， 可以在qiniu.Config中配置ProgressRecorder字段
//
// 如果上传的数据源Reader是不可Seekable的，那么我们上传完成之前并不知道要上传数据的总的大小
// 这个时候， totalSize为-1
//
// 如果上传的数据并不是来自于文件， 那么filename为空字符串
type ProgressRecorder interface {
	// Key 为要保存在七牛存储空间中的名字
	// filename 是本地上传的文件名
	// bucket  是存储空间的名字
	Progress(bucket, filename, Key string, totalSize int64, uploadedSize int64)
}

// ProgressRecorder的默认实现
type recorder struct {
	logger qiniu.Logger
}

func defaultRecorder(logger qiniu.Logger) ProgressRecorder {
	return &recorder{logger: logger}
}

// Progress 输出上传进度信息， 如果filename为空或者totalSize未知， 那么什么也不做
func (r *recorder) Progress(bucket, filename, key string, totalSize, uploadedSize int64) {
	if filename == "" || totalSize == -1 {
		return
	}
	r.logger.Log(fmt.Sprintf("Uploading file `%s` => `%s:%s` [%.2f%%|%s/%s]", filename, bucket, key,
		float64(uploadedSize)/float64(totalSize)*100, defs.Size(uploadedSize).String(), defs.Size(totalSize).String()))
}

// UploadMultipart 使用v2版本分片上传上传数据到七牛存储
// v2版本分片上传的过程：
// 1. 使用init接口在服务端创建相应的数据结构， 返回一个UploadID
// 2. 把数据切成一块一块的数据， 分别使用uploadPart接口上传每一块，默认块的大小为DefaultUploadPartSize
// 3. 调用complete接口，表示文件上传完毕
//
// 返回的UploadID有一个过期的时间， 这个时间足够的长，一般一周左右， 如果在过期之前没有数据没有上传完成，
// 那么放弃上传， 返回上传失败。
func (u *Kodo) UploadMultipart(input *UploadInput, out interface{}) error {
	if err := input.init(u); err != nil {
		return err
	}
	uploader, err := newMultipartUploader(input, u)
	if err != nil {
		return err
	}
	err = uploader.upload(context.Background(), out)
	return errUpload(err, uploader.reqID, uploader.statusCode, uploader.UploadID)
}

// UploadMultipartContext 使用v2版本分片上传上传数据到七牛存储
// 和UploadMultipart的区别是， 该方法多了一个context参数， 可以用来中断正在上传过程
// v2版本分片上传的过程：
// 1. 使用init接口在服务端创建相应的数据结构， 返回一个UploadID
// 2. 把数据切成一块一块的数据， 分别使用uploadPart接口上传每一块，默认块的大小为DefaultUploadPartSize
// 3. 调用complete接口，表示文件上传完毕
//
// 返回的UploadID有一个过期的时间， 这个时间足够的长，一般一周左右， 如果在过期之前没有数据没有上传完成，
// 那么放弃上传， 返回上传失败。
func (u *Kodo) UploadMultipartContext(ctx context.Context, input *UploadInput, out interface{}) error {
	uploader, err := newMultipartUploader(input, u)
	if err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	err = uploader.upload(ctx, out)
	return errUpload(err, uploader.reqID, uploader.statusCode, uploader.UploadID)
}

func errUpload(err error, reqID string, statusCode int, uploadID string) error {
	if err == nil {
		return nil
	}
	if statusCode <= 0 {
		return err
	}
	if aerr, ok := err.(qerr.Error); ok {
		return newMultiUploadFailure(aerr, statusCode, reqID, uploadID)
	}
	return newMultiUploadFailure(qerr.New("MultipartUploadError", "Upload failure", err), statusCode, reqID, uploadID)
}

// Upload 上传数据到七牛存储
// 在数据大小知道的前提下， 程序会根据数据的大小选择合适地上传方式进行上传
// 如果要上传的数据大于DefaultFormSize， 那么使用分片上传, 否则使用表单进行上传
//
// 如果input.Data不是ReadSeeker, 那么在上传之前获取不到要上传的数据的总的大小，
// 这个时候会使用分片上传。 如果input.Data是ReadSeeker那么可以获取其大小， 然后
// 根据上述的上传逻辑选择合适的上传方式。
//
// 在上传期间， 不要修改input的值， 程序会根据一些逻辑设置相应的字段
func (u *Kodo) Upload(input *UploadInput, out interface{}) error {
	if err := input.init(u); err != nil {
		return err
	}
	if input.totalSize > DefaultFormSize || input.totalSize == -1 {
		return u.UploadMultipart(input, out)
	}
	return u.UploadForm(input, out)
}

// UploadContext 上传数据到七牛存储
// Context可以用来中断上传请求
// 在数据大小知道的前提下， 程序会根据数据的大小选择合适地上传方式进行上传
// 如果要上传的数据大于DefaultFormSize， 那么使用分片上传, 否则使用表单进行上传
//
// 如果input.Data不是ReadSeeker, 那么在上传之前获取不到要上传的数据的总的大小，
// 这个时候会使用分片上传。 如果input.Data是ReadSeeker那么可以获取其大小， 然后
// 根据上述的上传逻辑选择合适的上传方式。
//
// 在上传期间， 不要修改input的值， 程序会根据一些逻辑设置相应的字段
func (u *Kodo) UploadContext(ctx context.Context, input *UploadInput, out interface{}) error {
	// init可能被分片上传和表单上传重复调用了，没有影响，input第二次初始化没有效果
	// 这个地方初始化是为了获取上传数据的总的大小，然后决定使用那种上传方式
	if err := input.init(u); err != nil {
		return err
	}
	if input.totalSize > DefaultFormSize || input.totalSize == -1 {
		return u.UploadMultipartContext(ctx, input, out)
	}
	return u.UploadFormContext(ctx, input, out)
}

// MultiUploadFailure 是分片上传出错的接口， 如果一个分片上传错误失败，那么
// 会返回一个符合该接口的错误类型， 可以从该错误类型获取上传错误的UploadID
//
// 示例:
//
//     kodo := session.New()
//     output := &UploadOutput{}
//     err := kodo.Upload(input, output)
//     if err != nil {
//         if multierr, ok := err.(kodo.MultiUploadFailure); ok {
//             // Process error and its associated uploadID
//             fmt.Println("Error:", multierr.Code(), multierr.Message(), multierr.UploadID(), multierr.StatusCode(), multierr.RequestID())
//         } else {
//             // Process error generically
//             fmt.Println("Error:", err.Error())
//         }
//     }
//
type MultiUploadFailure interface {
	qerr.RequestFailure

	// 返回分片上传UploadID
	UploadID() string
}

func newMultiUploadFailure(err qerr.Error, statusCode int, reqID, uploadID string) MultiUploadFailure {
	reqFailure := qerr.NewRequestFailure(err, statusCode, reqID)
	return &multiUploadError{
		RequestFailure: reqFailure,
		uploadID:       uploadID,
	}
}

// multiUploadError 表示分片上传过程出错，
// 分装了分片上传的uploadID, 本身是符合qerr.Error接口的
type multiUploadError struct {
	qerr.RequestFailure

	// 分片上传的ID
	uploadID string
}

// Error 返回表示该错误信息的字符串
func (m multiUploadError) Error() string {
	extra := fmt.Sprintf("upload id: %s", m.uploadID)
	return qerr.SprintError(m.Code(), m.Message(), extra, m.OrigErr())
}

// String 调用Error方法
func (m multiUploadError) String() string {
	return m.Error()
}

// UploadID 返回分片上传的错误的ID
func (m multiUploadError) UploadID() string {
	return m.uploadID
}

// multipartUploader 用来进行分片上传
type multipartUploader struct {
	*resumeRecorder

	base64Key string
	*Kodo

	*UploadInput

	pool sync.Pool

	wg     sync.WaitGroup
	mu     sync.Mutex
	limitC chan struct{}
	done   chan struct{}

	// 每块数据的大小
	partSize int64

	storeNumber int

	err error

	// 上传请求发生错误的reqid
	// 多个goroutine上传的时候，如果多个goroutine都发生了错误， 保存最新的发生错误的reqid
	reqID string

	// 上传发生错误的请求的状态码
	statusCode int

	// 当前数据的读取位置
	readPos int64

	// 上个块的索引值
	lastIndex int

	recorder ProgressRecorder
}

type completedPart struct {
	Index int    `json:"partNumber"`
	Etag  string `json:"etag"`
	size  int64
}

type completedParts []*completedPart

func (p completedParts) Less(i, j int) bool { return p[i].Index < p[j].Index }
func (p completedParts) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p completedParts) Len() int           { return len(p) }

// 上传完成的片的总的大小
func (p completedParts) Size() int64 {
	var size int64 = 0
	for _, sp := range p {
		size += sp.size
	}
	return size
}

func newMultipartUploader(input *UploadInput, kodo *Kodo) (*multipartUploader, error) {
	var (
		storeNumber int
		recorder    ProgressRecorder
	)
	if kodo.Config.ProgressRecorder != nil {
		if r, ok := kodo.Config.ProgressRecorder.(ProgressRecorder); ok {
			recorder = r
		}
	}
	if recorder == nil {
		recorder = defaultRecorder(kodo.Config.Logger)
	}

	storeNumber = DefaultStoreNumber
	if qiniu.IntValue(kodo.Config.StoreNumber) > 0 {
		storeNumber = qiniu.IntValue(kodo.Config.StoreNumber)
	}
	key := base64.URLEncoding.EncodeToString([]byte(input.Key))
	uploader := &multipartUploader{
		resumeRecorder: &resumeRecorder{},
		UploadInput:    input,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, DefaultUploadPartSize)
			},
		},
		partSize:    DefaultUploadPartSize,
		limitC:      make(chan struct{}, input.Concurrency),
		Kodo:        input.kodo,
		base64Key:   key,
		recorder:    recorder,
		storeNumber: storeNumber,
	}
	if !qiniu.BoolValue(kodo.Config.DisableResume) && exists(resumeFilePath(input.Filename)) && input.Filename != "" {
		if err := uploader.recovery(resumeFilePath(input.Filename)); err == nil {
			seekOk := false
			if r, ok := uploader.Data.(*os.File); ok {
				if _, err := r.Seek(uploader.readPos, io.SeekStart); err == nil {
					uploader.Data = r
					seekOk = true
				}
			}
			if !seekOk { // seek 文件失败， 忽略断点续传保存的信息，从头开始上传
				uploader.reset()
			} else {
				uploader.readPos = uploader.Parts.Size()
				uploader.lastIndex = uploader.Parts.Len()
			}
		} else {
			uploader.reset()
		}
	}
	if !qiniu.BoolValue(kodo.Config.DisableResume) && uploader.LastModification.IsZero() {
		uploader.LastModification = lastMod(resumeFilePath(input.Filename))
	}
	return uploader, nil
}

// initPart 实现七牛v2版本分片上传的初始化部分
// 获取并且设置uploader上传的uploadID和expireAt过期时间
func (uploader *multipartUploader) init(reqID *string, statusCode *int) error {
	req, out, err := uploader.initRequest(reqID, statusCode)
	if err != nil {
		return err
	}
	if err := req.Send(); err != nil {
		return err
	}
	uploader.UploadID = out.UploadID
	return nil
}

func reqIDStatusCodeOption(req *request.Request, reqID *string, statusCode *int) {
	req.ApplyOptions(request.WithGetResponseHeader("X-Reqid", reqID),
		request.WithGetResponseStatusCode(statusCode))
}

func (uploader *multipartUploader) initRequest(reqID *string, statusCode *int) (*request.Request, *partInitOutput, error) {
	host, scheme, err := uploader.getUpHost()
	if err != nil {
		return nil, nil, err
	}
	op := &request.API{
		Scheme:      scheme,
		Method:      "POST",
		Path:        fmt.Sprintf("/buckets/%s/objects/%s/uploads", uploader.BucketName, uploader.base64Key),
		Host:        host,
		APIName:     "part-init",
		ServiceName: ServiceName,
	}
	out := &partInitOutput{}
	req := uploader.Kodo.newRequest(op, nil, out)
	req.HTTPRequest.Header.Set("Authorization", "UpToken "+uploader.UpToken)
	reqIDStatusCodeOption(req, reqID, statusCode)

	return req, out, nil
}

func (uploader *multipartUploader) seterr(err error) {
	uploader.mu.Lock()
	defer uploader.mu.Unlock()

	uploader.err = err
}

func (uploader *multipartUploader) setReqIDAndStatusCode(reqID string, statusCode int) {
	uploader.mu.Lock()
	defer uploader.mu.Unlock()

	uploader.reqID = reqID
	uploader.statusCode = statusCode
}

func (uploader *multipartUploader) getReqIDAndStatusCode() (string, int) {
	uploader.mu.Lock()
	defer uploader.mu.Unlock()

	return uploader.reqID, uploader.statusCode
}

func (uploader *multipartUploader) geterr() error {
	uploader.mu.Lock()
	defer uploader.mu.Unlock()
	return uploader.err
}

func (uploader *multipartUploader) uploadPartRequest(ctx context.Context, part *part, reqID *string, statusCode *int) (*request.Request, *partUploadOutput, error) {

	host, scheme, err := uploader.getUpHost()
	if err != nil {
		return nil, nil, err
	}
	op := &request.API{
		Scheme:      scheme,
		Path:        fmt.Sprintf("/buckets/%s/objects/%s/uploads/%s/%d", uploader.BucketName, uploader.base64Key, uploader.UploadID, part.index),
		Method:      "PUT",
		Host:        host,
		ContentType: defs.CONTENT_TYPE_OCTET,
		ServiceName: ServiceName,
		APIName:     "part-upload",
	}
	out := &partUploadOutput{}

	req := uploader.Kodo.newRequest(op, &part.data, out)
	req.SetContext(ctx)

	req.HTTPRequest.Header.Add("Authorization", "UpToken "+uploader.UpToken)
	if uploader.CheckMd5 {
		md5Value, err := computeMd5(part.data)
		if err != nil {
			return nil, nil, err
		}
		req.HTTPRequest.Header.Add("Content-MD5", md5Value)
	}
	reqIDStatusCodeOption(req, reqID, statusCode)

	return req, out, nil
}

// UploadPart 上传块数据到存储空间
func (uploader *multipartUploader) uploadPart(ctx context.Context, part *part) {
	debugLogMultipartUpload(uploader.Config.LogLevel, uploader.Config.Logger, part)
	defer uploader.wg.Done()
	defer func() {
		<-uploader.limitC
	}()
	defer func() {
		// 只有partSize的块才会放入sync.pool中， 对于最后一块，如果内容大小小于partSize, 不会放入该缓存池
		if int64(len(part.data)) == uploader.partSize {
			defer uploader.pool.Put(part.data)
		}
	}()
	var (
		reqID      string
		statusCode int
	)

	req, out, err := uploader.uploadPartRequest(ctx, part, &reqID, &statusCode)
	if err != nil {
		uploader.seterr(err)
		return
	}
	if err := req.Send(); err != nil {
		// uploader.seterr(err)
		// uploader.setReqIDAndStatusCode(reqID, statusCode)
		// 可能会导致err和reqID, statusCode不是一个请求的信息
		uploader.mu.Lock()
		uploader.err = err
		uploader.reqID = reqID
		uploader.statusCode = statusCode
		uploader.mu.Unlock()

		return
	}

	uploader.mu.Lock()
	defer uploader.mu.Unlock()

	uploader.Parts = append(uploader.Parts, &completedPart{
		Etag:  out.Etag,
		Index: part.index,
	})
	if !qiniu.BoolValue(uploader.Config.DisableRecorder) {
		uploader.recorder.Progress(uploader.BucketName, uploader.Filename, uploader.Key, uploader.totalSize, uploader.readPos)
	}
	if !qiniu.BoolValue(uploader.Config.DisableResume) && uploader.Parts.Len() > 0 && uploader.Parts.Len()%uploader.storeNumber == 0 {
		uploader.store(resumeFilePath(uploader.Filename))
	}
}

type completeInput struct {
	Parts      completedParts    `json:"parts,omitempty"`
	MimeType   string            `json:"mimeType,omitempty"`
	Filename   string            `json:"fname,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CustomVars map[string]string `json:"customVars,omitempty"`
}

func (uploader *multipartUploader) completeRequest(out interface{}, reqID *string, statusCode *int) (*request.Request, error) {
	host, scheme, err := uploader.getUpHost()
	if err != nil {
		return nil, err
	}
	sort.Sort(uploader.Parts)
	op := &request.API{
		Method:      "POST",
		Host:        host,
		Scheme:      scheme,
		Path:        fmt.Sprintf("/buckets/%s/objects/%s/uploads/%s", uploader.BucketName, uploader.base64Key, uploader.UploadID),
		ContentType: defs.CONTENT_TYPE_JSON,
		ServiceName: ServiceName,
		APIName:     "part-complete",
	}
	input := &completeInput{
		Parts: uploader.Parts,
	}
	if uploader.MimeType != "" {
		input.MimeType = uploader.MimeType
	}
	if len(uploader.CustomParams) > 0 {
		input.CustomVars = uploader.CustomParams
	}
	if len(uploader.MetaKeys) > 0 {
		input.Metadata = uploader.MetaKeys
	}
	req := uploader.newRequest(op, input, out)
	req.HTTPRequest.Header.Set("Authorization", "UpToken "+uploader.UpToken)
	reqIDStatusCodeOption(req, reqID, statusCode)
	return req, nil
}

// complete 调用complete接口完成文件的上传
func (uploader *multipartUploader) complete(out interface{}) error {
	var (
		reqID      string
		statusCode int
	)
	req, err := uploader.completeRequest(out, &reqID, &statusCode)
	err = req.Send()
	return errUpload(err, reqID, statusCode, uploader.UploadID)
}

func (uploader *multipartUploader) upload(ctx context.Context, out interface{}) error {
	if uploader.hasNoUploadID() {
		var (
			reqID      string
			statusCode int
		)
		if err := uploader.init(&reqID, &statusCode); err != nil {
			uploader.reqID = reqID
			uploader.statusCode = statusCode
			return err

		}
	}

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	var (
		err error
		p   *part
	)
	for uploader.geterr() == nil && err != io.EOF {

		p, err = uploader.nextPart()
		if p != nil {
			uploader.limitC <- struct{}{}
			uploader.wg.Add(1)

			uploader.uploadPart(ctx, p)
		}
		if err != nil && err != io.EOF {
			uploader.seterr(err)
		}
	}
	if err := uploader.geterr(); err != nil {
		// 存在块上传失败，取消所有的其他的块上传
		cancelFunc()

		// 等待所有上传goroutine退出
		uploader.wg.Wait()

		return err

	}
	uploader.wg.Wait()

	defer uploader.close(resumeFilePath(uploader.Filename))

	return uploader.complete(out)
}

func (uploader *multipartUploader) nextPart() (*part, error) {
	uploader.mu.Lock()
	defer uploader.mu.Unlock()

	var buf []byte
	// 最后一块
	if leftBytes := uploader.totalSize - uploader.readPos; uploader.totalSize != -1 && leftBytes < uploader.partSize && leftBytes > 0 {
		buf = make([]byte, leftBytes)
	} else {
		buf = uploader.pool.Get().([]byte)
	}

	n, err := io.ReadFull(uploader.Data, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}
	uploader.readPos += int64(n)
	uploader.lastIndex++
	return &part{
		uploadID: uploader.UploadID,
		index:    uploader.lastIndex,
		data:     buf,
	}, err
}

func resumeFilePath(filename string) string {
	dir, name := filepath.Split(filename)
	return filepath.Join(dir, "."+name+".up")
}

// 断点续传需要保存的信息
type resumeRecorder struct {
	UploadID         string         `json:"upload_id"`
	Parts            completedParts `json:"parts"`
	LastModification time.Time      `json:"last_modification"`
}

func (r *resumeRecorder) hasNoUploadID() bool {
	return r.UploadID == ""
}

func (r *resumeRecorder) store(filename string) error {
	file, err := os.Create(filename)
	defer file.Close()

	if err != nil {
		return err
	}
	return json.NewEncoder(file).Encode(r)
}

func (r *resumeRecorder) reset() {
	r.UploadID = ""
	r.Parts = r.Parts[:]
	r.LastModification = time.Time{}
}

func (r *resumeRecorder) recovery(filename string) error {
	file, err := os.Open(filename)
	defer file.Close()

	if err != nil {
		return err
	}
	err = json.NewDecoder(file).Decode(r)
	if err != nil {
		return err
	}
	fileInfo, sErr := file.Stat()
	if sErr != nil {
		return sErr
	}
	if !fileInfo.ModTime().Equal(r.LastModification) {
		return qerr.New("ErrCodeRecovery", "failed to recovery record", nil)
	}
	return nil
}

func (r *resumeRecorder) close(filename string) error {
	return os.Remove(filename)
}

func exists(file string) bool {
	_, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func lastMod(file string) time.Time {
	fileInfo, err := os.Stat(file)
	if err != nil {
		return time.Time{}
	}
	return fileInfo.ModTime()
}
