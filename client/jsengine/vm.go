package jsengine

import (
	_ "embed"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"xiaobot/jsengine/console"
	"xiaobot/jsengine/inject/file"
	"xiaobot/jsengine/inject/tools"
	"xiaobot/jsengine/inject/vm"
	"xiaobot/jsengine/inject/xhr"

	//"xiaobot/jsengine/inject/data"
	//gojaerror "xiaobot/jsengine/inject/error"
	//"ninego/log"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	gojaurl "github.com/dop251/goja_nodejs/url"
)

//go:embed polyfill/out/index.js
var polyfillScript string

var vmPool = sync.Pool{
	New: func() interface{} {
		eng := NewEngine(nil)

		registry := new(require.Registry)
		registry.Enable(eng.Runtime)

		console.Enable(eng.Runtime)

		// ... 设置其他全局值 ...
		BotfuncMap["storage"] = WrapSharedData(eng.Runtime, global) //&global
		RegisterBotMap(eng.Runtime, BotfuncMap)                     //eng.Runtime.Set("bot", &Jsbot)

		return eng
	},
}

type ProxyHandler func(r *http.Request) (*url.URL, error)

type Engine struct {
	loop *eventloop.EventLoop

	Runtime *goja.Runtime
}

// RunString executes the script and returns the go type value
// if script result is promise, it will be resolved
func (e *Engine) RunString(script string) (value any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	var result goja.Value
	e.loop.Run(func(runtime *goja.Runtime) {
		result, err = runtime.RunString(script)
		if err == nil {
			go e.await(result)
		}
	})
	if err != nil {
		return
	}
	return resolveResult(result)
}

// CallFunction calls the function and returns the go type value
// if function result is promise, it will be resolved
func (e *Engine) CallFunction(fn goja.Callable, args ...any) (value any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	var result goja.Value
	e.loop.Run(func(runtime *goja.Runtime) {
		if args == nil {
			result, err = fn(nil)
		} else {
			var jsArgs []goja.Value
			for _, arg := range args {
				jsArgs = append(jsArgs, runtime.ToValue(arg))
			}
			result, err = fn(nil, jsArgs...)
		}
		if err == nil {
			go e.await(result)
		}
	})
	if err != nil {
		return
	}
	return resolveResult(result)
}

// loop.Run will hang if the script result has a non-stop code, such as setInterval.
// This method will stop the event loop when the promise result is resolved.
func (e *Engine) await(value any) {
	if value == nil {
		return
	}

	if v, ok := value.(goja.Value); ok {
		// if result is promise, wait for it to be resolved
		if p, ok := v.Export().(*goja.Promise); ok {
			if p.State() != goja.PromiseStatePending {
				return
			}

			// check promise state every 100 milliseconds, until it is resolved
			for {
				time.Sleep(time.Millisecond * 100)
				if p.State() == goja.PromiseStatePending {
					continue
				}
				break
			}

			// stop the event loop
			e.loop.StopNoWait()
		}
	}
}

func (e *Engine) Close() {
	e.loop.StopNoWait()
}

func NewEngine(proxy ProxyHandler) *Engine {
	loop := eventloop.NewEventLoop()
	engine := &Engine{
		loop: loop,
	}
	loop.Run(func(runtime *goja.Runtime) {
		engine.Runtime = runtime
		runtime.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
		vm.Enable(runtime)
		gojaurl.Enable(runtime)
		//if err := gojaerror.Enable(runtime); err != nil {
		//	return
		//}
		if err := tools.Enable(runtime); err != nil {
			return
		}
		if err := file.Enable(runtime); err != nil {
			return
		}
		//if err := data.Enable(runtime); err != nil {
		//	return
		//}
		if err := xhr.Enable(runtime, proxy); err != nil {
			return
		}
		if _, err := runtime.RunString(polyfillScript); err != nil {
			return
		}
		// polyfill global
		if err := runtime.Set("global", runtime.GlobalObject()); err != nil {
			return
		}
		// polyfill window
		if err := runtime.Set("window", runtime.GlobalObject()); err != nil {
			return
		}
		// polyfill window.location
		if _, err := runtime.RunString("global.location = new URL('http://localhost');"); err != nil {
			return
		}
		return
	})
	return engine
}

func Run(script string) (value any, err error) {
	engine := NewEngine(nil)
	return engine.RunString(script)
}

// if the value is Promise, it will be resolved and return the result.
func resolveResult(value goja.Value) (any, error) {
	export := value.Export()
	switch export.(type) {
	case *goja.Promise:
		p := export.(*goja.Promise)
		switch p.State() {
		case goja.PromiseStatePending:
			return nil, nil
		case goja.PromiseStateFulfilled:
			return p.Result().Export(), nil
		case goja.PromiseStateRejected:
			if err, ok := p.Result().Export().(error); ok {
				return nil, err
			} else {
				stack := p.Result().String()
				result := p.Result()
				if ro, ok := result.(*goja.Object); ok {
					stackVal := ro.Get("stack")
					if stackVal != nil && stackVal.String() != "" {
						stack = stackVal.String()
					}
				}
				return nil, errors.New(stack)
			}
		}
	}
	return export, nil
}
