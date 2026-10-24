package jsengine

import (
	"encoding/json"
	"io"

	"net/http"
	"net/url"

	//"sync"

	"ninego/log"

	"github.com/dop251/goja"
	//"github.com/dop251/goja_nodejs/console"
	//"github.com/dop251/goja_nodejs/require"
	//"github.com/gorilla/mux" // 用于解析路由参数（req.params）
)

// ------------------------------
// 1. 请求对象封装（对应 JS 的 req）
// ------------------------------

// JSRequest 封装 *http.Request + 路由参数，提供 JS 风格的字段和方法
type JSRequest struct {
	req    *http.Request     // 原生 HTTP 请求对象
	params map[string]string // 路由参数（如 /user/{id} 中的 id）
	body   string            // 预解析的请求体（供 JS 直接访问）
	parsed bool              // 标记请求体是否已解析
}

// 字段访问：请求方法（GET/POST 等）
func (j *JSRequest) GetMethod() string {
	return j.req.Method
}

// 字段访问：完整请求 URL（含查询参数）
func (j *JSRequest) GetURL() string {
	return j.req.URL.String()
}

// 字段访问：请求头（返回 map[string][]string，JS 可直接遍历）
func (j *JSRequest) GetHeaders() http.Header {
	return j.req.Header
}

// 字段访问：请求体（修复ReadFrom问题，改用io.ReadAll读取）
func (j *JSRequest) GetBody() (string, error) {
	if j.parsed {
		return j.body, nil
	}
	defer j.req.Body.Close()

	// 修复：用io.ReadAll替代ReadFrom（strings.Builder不支持ReadFrom）
	bodyBytes, err := io.ReadAll(j.req.Body)
	if err != nil {
		return "", err
	}

	j.body = string(bodyBytes)
	j.parsed = true // 标记已解析，避免重复读取
	return j.body, nil
}

// 字段访问：路由参数（如 /api/{id} 中的 id，依赖 gorilla/mux 解析）
func (j *JSRequest) GetParams() map[string]string {
	return j.params
}

// 字段访问：查询字符串参数（如 ?name=foo&age=20，返回 map[string][]string）
func (j *JSRequest) GetQuery() url.Values {
	return j.req.URL.Query()
}

// ------------------------------
// 2. 响应对象封装（对应 JS 的 res）
// ------------------------------

// JSResponse 封装 http.ResponseWriter，新增req字段用于重定向（修复undefined问题）
type JSResponse struct {
	w        http.ResponseWriter // 原生 HTTP 响应对象
	req      *http.Request       // 原生请求对象（用于重定向，修复j.req undefined）
	wrote    bool                // 标记是否已发送响应（避免重复写）
	status   int                 // 响应状态码（默认 200）
	redirect bool                // 标记是否已触发重定向
}

// 方法：设置响应状态码（如 res.status(200)）
func (j *JSResponse) Status(code int) {
	if j.wrote || j.redirect {
		return // 已发送响应/重定向，忽略状态码修改
	}
	j.status = code
	j.w.Header().Set("X-Status-Code", string(rune(code))) // 临时标记，Write 时生效
}

// 方法：发送文本响应（如 res.send('Hello')）
func (j *JSResponse) Send(body string) error {
	if j.wrote || j.redirect {
		return nil // 避免重复发送
	}

	// 若未显式设置状态码，默认 200
	if j.status == 0 {
		j.status = http.StatusOK
	}

	// 设置默认响应头（若未手动设置）
	if j.w.Header().Get("Content-Type") == "" {
		j.w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}

	// 发送状态码和响应体
	j.w.WriteHeader(j.status)
	_, err := j.w.Write([]byte(body))
	j.wrote = true // 标记已发送
	return err
}

// 方法：发送 JSON 响应（如 res.json({key: 'val'})）
func (j *JSResponse) Json(data interface{}) error {
	if j.wrote || j.redirect {
		return nil
	}

	// 序列化为 JSON
	jsonBody, err := json.Marshal(data)
	if err != nil {
		return err // JSON 序列化失败，返回错误给 JS
	}

	// 若未显式设置状态码，默认 200
	if j.status == 0 {
		j.status = http.StatusOK
	}

	// 设置 JSON 响应头（覆盖可能存在的文本头）
	j.w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// 发送响应
	j.w.WriteHeader(j.status)
	_, err = j.w.Write(jsonBody)
	j.wrote = true
	return err
}

// 方法：设置响应头（如 res.set('Content-Type', 'text/html')）
func (j *JSResponse) Set(key, value string) {
	if j.wrote || j.redirect {
		return // 已发送响应，忽略头修改
	}
	j.w.Header().Set(key, value)
}

// 方法：重定向（如 res.redirect('/new-url')）
func (j *JSResponse) Redirect(url string) {
	if j.wrote || j.redirect {
		return
	}

	// 修复：使用结构体中定义的req字段（不再报j.req undefined）
	http.Redirect(j.w, j.req, url, http.StatusFound)
	j.redirect = true // 标记已重定向
}

// ------------------------------
// 3. 核心：将 req/res 注入 JS 环境
// ------------------------------

// injectJSContext 把 Go 封装的 JSRequest/JSResponse 注入 JS 虚拟机
func injectJSContext(vm *goja.Runtime, w http.ResponseWriter, r *http.Request) error {
	// 1. 解析路由参数
	//params := mux.Vars(r)
	params := map[string]string{"action": r.PathValue("action")}

	// 2. 创建 Go 封装的请求对象
	jsReq := &JSRequest{
		req:    r,
		params: params,
	}

	// 3. 创建 Go 封装的响应对象（修复：显式传入req字段，不再报unknown field）
	jsRes := &JSResponse{
		w:   w,
		req: r, // 显式赋值req，解决struct literal unknown field问题
	}

	// 4. 将 Go 对象注入 JS 全局作用域，映射为 req/res
	if err := vm.Set("req", map[string]interface{}{
		"method":  jsReq.GetMethod,  // JS: req.method() → 返回请求方法
		"url":     jsReq.GetURL,     // JS: req.url() → 返回完整 URL
		"headers": jsReq.GetHeaders, // JS: req.headers() → 返回请求头 map
		"body":    jsReq.GetBody,    // JS: req.body() → 返回请求体字符串
		"params":  jsReq.GetParams,  // JS: req.params() → 返回路由参数 map
		"query":   jsReq.GetQuery,   // JS: req.query() → 返回查询参数 map
	}); err != nil {
		return err
	}

	if err := vm.Set("res", map[string]interface{}{
		"status":   jsRes.Status,   // JS: res.status(200) → 设置状态码
		"send":     jsRes.Send,     // JS: res.send('text') → 发送文本
		"json":     jsRes.Json,     // JS: res.json({}) → 发送 JSON
		"set":      jsRes.Set,      // JS: res.set('key', 'val') → 设置响应头
		"redirect": jsRes.Redirect, // JS: res.redirect('/url') → 重定向
	}); err != nil {
		return err
	}

	return nil
}

// ------------------------------
// 4. API 服务实现（配合 gorilla/mux 路由）
// ------------------------------

func Do_Action(w http.ResponseWriter, r *http.Request, customJSScript *string) (err error) {
	defer func() {
		global.Update("runcount", -1)
		if r := recover(); r != nil {
			log.Error("Recovered from panic:", r)
		}
	}()
	global.Update("runcount", +1)

	// 1. 创建 JS 虚拟机
	engine := vmPool.Get().(*Engine)
	// 将 VM 放回池中以供将来重用
	defer vmPool.Put(engine)
	// 2. 注入 req/res 到 JS 环境
	if err = injectJSContext(engine.Runtime, w, r); err != nil {
		http.Error(w, "JS 环境初始化失败: "+err.Error(), http.StatusInternalServerError)
		log.Error("JS 环境初始化失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = engine.RunString("{" + *customJSScript + "\n}")
	if err != nil {
		if evalErr, ok := err.(*goja.Exception); ok {
			http.Error(w, "JS 脚本错误: "+evalErr.String(), http.StatusInternalServerError)
			log.Error("JS 脚本错误: "+evalErr.String(), http.StatusInternalServerError)
			return
		}
		http.Error(w, "JS 执行失败: "+err.Error(), http.StatusInternalServerError)
		log.Error("JS 执行失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	return
}
