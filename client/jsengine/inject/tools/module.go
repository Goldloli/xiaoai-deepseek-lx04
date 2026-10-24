package tools

import (
	"github.com/dop251/goja"
)

func Enable(runtime *goja.Runtime) error {
	if err := runtime.Set("wol", wakeOnLAN); err != nil {
		return err
	}
	if err := runtime.Set("command", command); err != nil {
		return err
	}
	return runtime.Set("shell", shell)
}
