package console

import (
	"fmt"
	"ninego/log"
	"sync"
	"time"

	"github.com/dop251/goja"
)

type Console struct {
	timeMtx sync.Mutex
	timeMap map[string]time.Time
}

func (c *Console) log(args ...any) {
	//fmt.Println(args...)
	log.Println(args...)
}

func (c *Console) time(key string) {
	c.timeMtx.Lock()
	defer c.timeMtx.Unlock()
	if _, ok := c.timeMap[key]; ok {
		c.log(fmt.Sprintf("Timer '%s' already exists", key))
		return
	}
	c.timeMap[key] = time.Now()
}

func (c *Console) timeEnd(key string) {
	c.timeMtx.Lock()
	defer c.timeMtx.Unlock()
	start, ok := c.timeMap[key]
	if !ok {
		c.log(fmt.Sprintf("Timer '%s' does not exist", key))
		return
	}
	delete(c.timeMap, key)
	elapsed := time.Since(start)
	fmt.Printf("%s: %v\n", key, elapsed)
}

func (c *Console) Inject(vm *goja.Runtime) error {
	consoleObj := vm.NewObject()
	if err := consoleObj.Set("alert", c.log); err != nil {
		return err
	}
	if err := consoleObj.Set("log", c.log); err != nil {
		return err
	}
	if err := consoleObj.Set("trace", c.log); err != nil {
		return err
	}
	if err := consoleObj.Set("debug", c.log); err != nil {
		return err
	}
	if err := consoleObj.Set("info", c.log); err != nil {
		return err
	}
	if err := consoleObj.Set("warn", c.log); err != nil {
		return err
	}
	if err := consoleObj.Set("error", c.log); err != nil {
		return err
	}
	if err := consoleObj.Set("time", c.time); err != nil {
		return err
	}
	if err := consoleObj.Set("timeEnd", c.timeEnd); err != nil {
		return err
	}
	if err := vm.Set("console", consoleObj); err != nil {
		return err
	}
	return nil
}

func Enable(vm *goja.Runtime) error {
	console := &Console{
		timeMap: make(map[string]time.Time),
	}
	return console.Inject(vm)
}
