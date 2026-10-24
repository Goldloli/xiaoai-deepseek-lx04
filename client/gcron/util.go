package gcron

import (
	"fmt"
	"os"
	//"path/filepath"
)

// 判断文件或文件夹是否存在
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func findNewClockName() string {
	fileName := "clock0001.json"
	i := 1
	for IsExist(fileName) {
		i += 1
		fileName = fmt.Sprintf("clock%04d.json", i)
	}
	return fileName
}
