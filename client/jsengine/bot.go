package jsengine

import (
	"reflect"

	"github.com/dop251/goja"
)

// 供js中使用的bot对象
var BotfuncMap = map[string]interface{}{}

// RegisterBotMap 将方法 map 注册到 goja 引擎（兼容最旧版 goja）
func RegisterBotMap(rt *goja.Runtime, botMap map[string]interface{}) {
	gojaBot := make(map[string]interface{}, len(botMap))
	for name, fn := range botMap {
		if reflect.ValueOf(fn).Kind() == reflect.Func {
			gojaBot[name] = adaptToGoja(rt, fn)
		} else {
			gojaBot[name] = fn
		}
	}
	rt.Set("bot", gojaBot)
}

// adaptToGoja 适配最旧版 goja 的函数转换
func adaptToGoja(rt *goja.Runtime, fn interface{}) goja.Value {
	fnVal := reflect.ValueOf(fn)
	if fnVal.Kind() != reflect.Func {
		// 旧版直接返回错误字符串
		return rt.ToValue("bot 方法必须是函数")
	}

	return rt.ToValue(func(call goja.FunctionCall) goja.Value {
		fnType := fnVal.Type()
		numParams := fnType.NumIn()
		args := make([]reflect.Value, 0, numParams)

		// 处理参数（旧版类型转换）
		for i := 0; i < numParams; i++ {
			paramType := fnType.In(i)
			if i >= len(call.Arguments) {
				args = append(args, reflect.Zero(paramType))
				continue
			}

			argVal := call.Argument(i)
			var goVal reflect.Value

			switch paramType.Kind() {
			case reflect.String:
				goVal = reflect.ValueOf(argVal.String())
			case reflect.Bool:
				// 旧版 ToBoolean() 直接返回 bool
				goVal = reflect.ValueOf(argVal.ToBoolean())
			case reflect.Float64:
				// 旧版 ToFloat() 直接返回 float64
				goVal = reflect.ValueOf(argVal.ToFloat())
			case reflect.Int:
				// 转为 int
				goVal = reflect.ValueOf(int(argVal.ToFloat()))
			default:
				// 直接返回错误字符串
				return rt.ToValue("不支持的参数类型: " + paramType.Kind().String())
			}

			args = append(args, goVal)
		}

		// 调用函数
		returnVals := fnVal.Call(args)

		// 处理返回值
		if len(returnVals) == 1 {
			ret := returnVals[0]
			if ret.CanInterface() {
				// 处理 error 类型返回值
				if err, ok := ret.Interface().(error); ok {
					if err != nil {
						return rt.ToValue(err.Error()) // 直接返回错误字符串
					}
					return goja.Undefined()
				}
				// 其他类型直接返回
				return rt.ToValue(ret.Interface())
			}
		}

		return goja.Undefined()
	})
}
