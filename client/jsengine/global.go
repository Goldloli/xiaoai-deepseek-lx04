package jsengine

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"

	"github.com/dop251/goja"
)

// 供js中使用的全局对象
var global = NewSharedData() //map[string]interface{}{}

func SaveStorageToFile() error {
	if global != nil {
		// 将map转换为JSON字节切片
		marshal, err := json.Marshal(global.data)
		if err != nil {
			return err
		}
		// 保存到文件
		err = ioutil.WriteFile("botdata.storage", marshal, 0666)
		if err != nil {
			return err
		}
		//instance = nil
	}
	return nil
}

func LoadStorageFromFile() error {
	if global != nil {
		bytes, err := os.ReadFile("botdata.storage")
		if err != nil {
			return err
		}
		// 将JSON字节切片转换为map
		err = json.Unmarshal(bytes, &global.data)
		if err != nil {
			return err
		}
		// 初始化值
		global.Set("runcount", 0)
	}
	return nil
}

// SharedData 带读写锁的共享数据
type SharedData struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

func NewSharedData() *SharedData {
	return &SharedData{
		data: make(map[string]interface{}),
	}
}

func (s *SharedData) Get(key string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[key]
}

func (s *SharedData) Set(key string, val interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = val
}

func (s *SharedData) Update(key string, val int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i, ok := s.data[key]; ok {
		switch i.(type) {
		case int:
			s.data[key] = i.(int) + val
		case float64:
			s.data[key] = i.(float64) + float64(val)
		}
		//s.data[key] += 1
	} else {
		s.data[key] = val
	}
}

func (s *SharedData) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

func (s *SharedData) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

// 将 SharedData 封装成 JS Proxy 对象
func WrapSharedData(vm *goja.Runtime, sd *SharedData) goja.Value {
	// 注入 getter/setter/delete/keys 到 JS
	vm.Set("__getter", func(key string) goja.Value {
		return vm.ToValue(sd.Get(key))
	})
	vm.Set("__setter", func(key string, val interface{}) {
		sd.Set(key, val)
	})
	vm.Set("__deleter", func(key string) {
		sd.Delete(key)
	})
	vm.Set("__keys", func() []string {
		return sd.Keys()
	})

	// 创建 Proxy 对象覆盖所有属性访问
	script := `
	(function() {
		return new Proxy({}, {
			get: function(target, prop) {
				if (prop === Symbol.iterator) {
					const keys = __keys();
					let index = 0;
					return function() {
						return {
							next: function() {
								if (index < keys.length) {
									const key = keys[index++];
									return { value: [key, __getter(key)], done: false };
								}
								return { done: true };
							}
						};
					};
				}
				return __getter(prop);
			},
			set: function(target, prop, value) {
				__setter(prop, value);
				return true;
			},
			deleteProperty: function(target, prop) {
				__deleter(prop);
				return true;
			},
			ownKeys: function(target) {
				return __keys();
			},
			getOwnPropertyDescriptor: function(target, prop) {
				return {
					value: __getter(prop),
					writable: true,
					configurable: true,
					enumerable: true
				};
			}
		});
	})()
	`

	v, err := vm.RunString(script)
	if err != nil {
		panic(err)
	}
	return v
}
