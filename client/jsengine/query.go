package jsengine

import (
	"net/http"
	"ninego/log"

	"github.com/dop251/goja"
)

func Exec_queryJS(query, customJSScript string) (handled bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Recovered from panic:", r)
		}
	}()

	handled = false
	// 创建 JS 虚拟机
	engine := vmPool.Get().(*Engine)
	// 将 VM 放回池中以供将来重用
	defer vmPool.Put(engine)

	// 将 Go 对象注入 JS 全局作用域
	engine.Runtime.Set("query", query)
	engine.Runtime.Set("handled", false)

	// 执行 JS 脚本，处理错误
	_, err = engine.RunString("{" + customJSScript + "\n}")
	if err != nil {
		if evalErr, ok := err.(*goja.Exception); ok {
			log.Error("JS 脚本错误: "+evalErr.String(), http.StatusInternalServerError)
			return
		}
		log.Error("JS 执行失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	handled = engine.Runtime.Get("handled").ToBoolean()
	log.Println("handled =", handled)
	return
}
