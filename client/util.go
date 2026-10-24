package xiaobot

import (
	"errors"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// 语速常量（每分钟单词数）
// 中文约150-200字/分钟，英文约120-160词/分钟
const (
	ChineseSpeedPerMinute = 256 // 中文每分钟字数
	EnglishSpeedPerMinute = 150 // 英文每分钟单词数
)

// 标点停顿时间（秒）
var punctuationPause = map[string]float64{
	".": 0.8,
	"?": 0.8,
	"!": 0.8,
	",": 0.3,
	";": 0.5,
	":": 0.5,
	"，": 0.3,
	"。": 0.8,
	"？": 0.8,
	"！": 0.8,
	"；": 0.5,
	"：": 0.5,
}

var (
	chineseRegex = regexp.MustCompile(`[\p{Han}]`)      // 匹配汉字
	englishRegex = regexp.MustCompile(`[a-zA-Z]+`)      // 匹配英文单词
	punctRegex   = regexp.MustCompile(`[.,!?;:。，！？；：]`) // 匹配标点
)

// calculateTtsElapse 更准确地计算TTS播放时间
func calculateTtsElapse(text string) time.Duration {
	if text == "" {
		return 0
	}

	// 1. 计算中文字符数
	chineseCount := len(chineseRegex.FindAllString(text, -1))

	// 2. 计算英文单词数（简单分割，实际可能需要更复杂的分词）
	englishWords := englishRegex.FindAllString(text, -1)
	englishCount := len(englishWords)

	// 3. 计算标点符号及停顿时间
	punctuations := punctRegex.FindAllString(text, -1)
	pauseTime := 0.0
	for _, p := range punctuations {
		if t, ok := punctuationPause[p]; ok {
			pauseTime += t
		}
	}

	// 4. 计算段落停顿（按换行分割）
	paragraphs := strings.Split(text, "\n")
	paragraphPause := float64(len(paragraphs)-1) * 1.0 // 段落间停顿1秒

	// 5. 计算基础播放时间（分钟转换为秒）
	chineseTime := float64(chineseCount) / ChineseSpeedPerMinute * 60
	englishTime := float64(englishCount) / EnglishSpeedPerMinute * 60

	// 总时间 = 内容时间 + 停顿时间
	totalSeconds := chineseTime + englishTime + pauseTime + paragraphPause

	// 确保最小时间为1秒
	if totalSeconds < 1.0 {
		totalSeconds = 1.0
	}

	return time.Duration(totalSeconds + 0.99 /** float64(time.Second)*/)
}

// GetExecutableDir 获取当前运行程序所在的目录
func GetExecutableDir() string {
	// 获取当前程序的可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return "."
	}
	// 从可执行文件路径中提取目录部分
	exeDir := filepath.Dir(exePath)
	return exeDir
}

// 设置为东八时区
func FixedZone() {
	cstZone := time.FixedZone("CST", 8*3600)
	time.Local = cstZone
}

// 标准化MAC地址为小写格式
func normalizeMAC(mac string) string {
	parsedMAC, err := net.ParseMAC(mac)
	if err != nil {
		return mac
	}
	return parsedMAC.String()
}

// possentence 从字符串末尾搜索合适的断句位置
func possentence(str string) (idx int) {
	keys := "\n\r。：；！，:;!,"
	runes := []rune(str)
	n := len(runes) //n := utf8.RuneCountInString(str)
	if n > maxttsWord {
		n = maxttsWord
	}
	for i := n - 1; i >= 0; i-- {
		if strings.ContainsRune(keys, runes[i]) {
			return i
		}
	}
	if n >= maxttsWord {
		return n - 1
	}
	return -1
}

// delstr 从字符串中删除指定范围的字符
func delstr(s *string, start, length int) {
	*s = string([]rune(*s)[:start]) + string([]rune(*s)[start+length:])
}

// substr 获取字符串的子串
func substr(s string, start, length int) string {
	return string([]rune(s)[start : start+length])
}

/*func findKeyByPartialString(dictionary map[string]string, partialKey string) string {
	for key, value := range dictionary {
		if strings.Contains(partialKey, key) {
			return value
		}
	}
	return ""
}*/

func validateProxy(proxyStr string) error {
	parsed, err := url.Parse(proxyStr)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("Proxy scheme must be http or https")
	}
	if parsed.Hostname() == "" || parsed.Port() == "" {
		return errors.New("Proxy hostname and port must be set")
	}
	return nil
}

/*func getHostname() string {
	if hostname, exists := os.LookupEnv("XIAOGPT_HOSTNAME"); exists {
		return hostname
	}

	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

func Normalize(message string) string {
	message = strings.TrimSpace(message)
	message = strings.ReplaceAll(message, " ", "--")
	message = strings.ReplaceAll(message, "\n", "，")
	message = strings.ReplaceAll(message, "\"", "，")
	return message
}*/

////////////////////////////////////////////////////////////////////////////////
// 自定义能判断是否Close的chan
type Channel struct {
	C      chan interface{}
	closed bool
	mut    sync.RWMutex
}

func NewChannel() *Channel {
	return NewChannelSize(0)
}

func NewChannelSize(size int) *Channel {
	return &Channel{
		C:      make(chan interface{}, size),
		closed: false,
		mut:    sync.RWMutex{},
	}
}

func (c *Channel) Close() {
	c.mut.Lock()
	defer c.mut.Unlock()
	if !c.closed {
		close(c.C)
		c.closed = true
	}
}

func (c *Channel) IsClosed() bool {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.closed
}

////////////////////////////////////////////////////////////////////////////////
// 自定义带计数功能的WaitGroup
type CountWaitGroup struct {
	wg    sync.WaitGroup
	count int
	mu    sync.Mutex
}

// Add 添加计数，与原生WaitGroup的Add用法相同
func (c *CountWaitGroup) Add(delta int) {
	c.mu.Lock()
	c.count += delta
	c.mu.Unlock()
	c.wg.Add(delta)
}

// Done 减少计数，与原生WaitGroup的Done用法相同
func (c *CountWaitGroup) Done() {
	if c.GetCount() > 0 {
		c.mu.Lock()
		c.count--
		c.mu.Unlock()
	} else {
		return
	}
	c.wg.Done()
}

// Wait 等待所有操作完成，与原生WaitGroup的Wait用法相同
func (c *CountWaitGroup) Wait() {
	c.wg.Wait()
}

// GetCount 获取当前计数
func (c *CountWaitGroup) GetCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

////////////////////////////////////////////////////////////////////////////////
// StateStore 存储整型状态数据并提供读写访问
type StateStore struct {
	mu   sync.RWMutex
	Data int
}

func NewState(value int) *StateStore {
	return &StateStore{Data: value}
}

func (st *StateStore) Update(value int) int {
	st.mu.RLock()         // 获取读锁
	defer st.mu.RUnlock() // 释放读锁
	st.Data += value
	return st.Data
}
func (st *StateStore) UpdateFor(lockedfunc func()) int {
	st.mu.RLock()         // 获取读锁
	defer st.mu.RUnlock() // 释放读锁
	lockedfunc()
	return st.Data
}

func (st *StateStore) Set(value int) {
	st.mu.RLock()         // 获取读锁
	defer st.mu.RUnlock() // 释放读锁
	st.Data = value
}

func (st *StateStore) Status() int {
	st.mu.RLock()         // 获取读锁
	defer st.mu.RUnlock() // 释放读锁
	return st.Data
}

/*/ TimeoutExecutor 带超时执行器：
// - timeoutSec: 超时时间（秒）
// - fn: 待执行的函数，返回值通过Result传递，错误通过Error传递
// - 返回值：执行结果/超时错误/函数内部错误
func TimeoutExec(timeoutSec int, fn func() (interface{}, error)) (Result interface{}, Error error) {
	// 1. 校验超时时间合法性
	if timeoutSec <= 0 {
		return nil, errors.New("超时时间必须大于0秒")
	}
	timeout := time.Duration(timeoutSec) * time.Second

	// 2. 定义结果通道（缓冲1个，避免goroutine泄漏）
	resultChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)
	var wg sync.WaitGroup

	defer func() {
		// 3. 清理资源：关闭通道+等待子协程退出
		close(resultChan)
		close(errChan)
		wg.Wait()

		// 4. 捕获待执行函数的panic（避免崩溃）
		if r := recover(); r != nil {
			log.Error("超时执行函数发生panic:", r)
			Error = errors.New("函数执行发生未捕获异常")
		}
	}()

	// 5. 启动子协程执行目标函数
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 执行目标函数并传递结果
		res, err := fn()
		resultChan <- res
		errChan <- err
	}()

	// 6. 超时控制：等待结果或超时
	select {
	case Result = <-resultChan:
		Error = <-errChan // 读取对应错误（即使无错也需读取，避免通道阻塞）
	case <-time.After(timeout):
		Result = nil
		Error = errors.New("函数执行超时：超过" + strconv.Itoa(timeoutSec) + "秒")
	}

	return Result, Error
}

// 示例：超时执行"AI问答"（10秒超时）
	aiTimeoutSec := 10
	query := "今天天气怎么样？"
	aiResult, aiErr := TimeoutExec(aiTimeoutSec, func() (interface{}, error) {
		// 执行AI调用（复用项目已有方法）
		return mt.askJarvis(query, "", true)
	})

	if aiErr != nil {
		log.Error("AI问答执行失败:", aiErr)
		return
	}

	// 处理AI结果（根据askJarvis返回值调整断言）
	aiMsg, ok := aiResult.(string)
	if ok && aiMsg != "" {
		log.Printf("AI回答：%s", aiMsg)
	}
*/
