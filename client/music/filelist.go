package music

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileItem 定义返回的文件/目录项结构
type FileItem struct {
	Name        string `json:"name"`  // 文件名/目录名
	Path        string `json:"path"`  // 文件绝对路径（目录为空）
	IsDir       bool   `json:"isDir"` // 是否为目录
	IsFavorited bool   `json:"isFav"`
}

// 定义支持的音乐文件扩展名（小写）
var supportedAudioExts = map[string]bool{
	".mp3":  true,
	".wav":  true,
	".flac": true,
	".ogg":  true,
	".oga":  true, // OGG的另一种扩展名
	".aac":  true,
	".m4a":  true,
	".ape":  true,
}

// FindAudioFiles 查找指定目录下的音乐文件/目录，返回JSON格式字符串
// dir: 要查找的根目录
// filter: 文件名过滤关键词（为空时返回当前目录下的音乐文件+子目录；不为空时递归查找含关键词的音乐文件）
// 返回值: JSON格式字符串、错误信息
func FindAudioFiles(baseDir, currentDir string, filter string) ([]FileItem, error) {
	// 存储结果的切片
	//var items []FileItem //nil切片,转为JSON是null
	items := []FileItem{} //显式初始化为空切片[]

	dir := filepath.Join(baseDir, currentDir)
	// 先校验目录是否存在
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return items, fmt.Errorf("目录不存在或无法访问: %w", err)
	}
	if !dirInfo.IsDir() {
		return items, fmt.Errorf("指定路径不是目录: %s", dir)
	}

	if filter == "" {
		// 逻辑1: filter为空 → 仅返回当前目录下的子目录（不递归） + 当前目录下的音乐文件
		files, err := os.ReadDir(dir)
		if err != nil {
			return items, fmt.Errorf("读取目录失败: %w", err)
		}

		// 遍历当前目录的直接内容（不递归）
		for _, file := range files {
			// 获取文件绝对路径
			absPath, err := filepath.Abs(filepath.Join(dir, file.Name()))
			if err != nil {
				fmt.Printf("警告：无法获取文件绝对路径 %s: %v\n", file.Name(), err)
				continue
			}
			absPath = filepath.Dir(absPath)
			absPath, _ = filepath.Rel(baseDir, absPath)
			if file.IsDir() {
				// 子目录项插入到items最前面
				items = append([]FileItem{{
					Name:  file.Name(),
					Path:  absPath,
					IsDir: true,
				}}, items...)
			} else {
				// 仅筛选当前目录下的音乐文件，追加到items末尾
				ext := strings.ToLower(filepath.Ext(file.Name()))
				if supportedAudioExts[ext] {
					items = append(items, FileItem{
						Name:  file.Name(),
						Path:  absPath,
						IsDir: false,
					})
				}
			}
		}
	} else {
		// 逻辑2: filter不为空 → 递归查找所有子目录中文件名包含filter的音乐文件（不返回目录）
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			// 跳过遍历过程中出现的错误（如权限不足的目录）
			if err != nil {
				fmt.Printf("警告：遍历路径 %s 失败: %v\n", path, err)
				return nil // 继续遍历其他路径
			}

			// 跳过目录，只处理文件
			if info.IsDir() {
				return nil
			}

			// 1. 判断文件名是否包含filter（不区分大小写）
			fileName := strings.ToLower(info.Name())
			filterLower := strings.ToLower(filter)
			if !strings.Contains(fileName, filterLower) {
				return nil
			}

			// 2. 判断是否为支持的音乐格式
			ext := strings.ToLower(filepath.Ext(info.Name()))
			if supportedAudioExts[ext] {
				// 获取文件绝对路径
				absPath, err := filepath.Abs(path)
				if err != nil {
					fmt.Printf("警告：无法获取文件绝对路径 %s: %v\n", path, err)
					return nil
				}
				absPath = filepath.Dir(absPath)
				absPath, _ = filepath.Rel(baseDir, absPath)
				items = append(items, FileItem{
					Name:  info.Name(),
					Path:  absPath, //string([]byte(path)[len(dir) : len(path)-len(info.Name())]),
					IsDir: false,
				})
			}

			return nil
		})

		if err != nil {
			return items, fmt.Errorf("递归遍历目录失败: %w", err)
		}
	}

	// 直接返回FileItem切片，不做JSON序列化
	return items, nil
}

// 全局变量
var HistoryMusicFiles []FileItem   //历史音乐文件切片
var FavoritedMusicFiles []FileItem //收藏音乐文件切片
// 读写锁：保护HistoryMusicFiles的并发读写（新增，防止切片修改与文件写入冲突）
var historyMu sync.RWMutex
var favoriteMu sync.RWMutex

// 常量定义：历史记录最大数量、历史文件路径
const (
	maxHistoryCount   = 50             // 最大历史记录数
	historyFileName   = "history.txt"  // 持久化文件
	favoritedFileName = "favorite.txt" // 持久化文件
)

// InitHistory 初始化历史记录：从history.txt加载数据
// 程序启动时调用一次即可，文件不存在/格式错误则初始化空切片
func InitHistory() error {
	// 检查文件是否存在
	if _, err := os.Stat(historyFileName); os.IsNotExist(err) {
		HistoryMusicFiles = []FileItem{} // 文件不存在，初始化空切片
		return nil
	}
	// 读取文件内容
	data, err := os.ReadFile(historyFileName)
	if err != nil {
		return fmt.Errorf("读取历史文件失败：%w", err)
	}
	// 反序列化为FileItem切片（若文件为空，直接初始化空切片）
	if len(data) == 0 {
		HistoryMusicFiles = []FileItem{}
		return nil
	}
	if err := json.Unmarshal(data, &HistoryMusicFiles); err != nil {
		return fmt.Errorf("解析历史文件失败：%w", err)
	}
	// 加载后若数量超过50，截断为最新50个（防止手动修改文件导致数量超标）
	trimHistory()

	////////////////////////////////////////////////////////////////////////////

	// 检查文件是否存在
	if _, err := os.Stat(favoritedFileName); os.IsNotExist(err) {
		FavoritedMusicFiles = []FileItem{} // 文件不存在，初始化空切片
		return nil
	}
	// 读取文件内容
	data, err = os.ReadFile(favoritedFileName)
	if err != nil {
		return fmt.Errorf("读取收藏文件失败：%w", err)
	}
	// 反序列化为FileItem切片（若文件为空，直接初始化空切片）
	if len(data) == 0 {
		FavoritedMusicFiles = []FileItem{}
		return nil
	}
	if err := json.Unmarshal(data, &FavoritedMusicFiles); err != nil {
		return fmt.Errorf("解析收藏文件失败：%w", err)
	}

	return nil
}

// AddToHistory 添加文件到历史记录（自动去重+长度控制）
// 参数：item 要添加的FileItem（仅非目录、Path非空的文件会被添加）
func AddToHistory(item FileItem) {
	// 加写锁：保护切片修改，防止与异步写入时的读操作冲突
	historyMu.Lock()
	defer historyMu.Unlock()

	// 过滤：仅添加非目录、且文件路径非空的项（符合音乐文件场景）
	if item.IsDir /*|| item.Path == ""*/ {
		return
	}

	// 步骤1：去重（根据Path判断，同一文件不重复添加）
	// 先删除原有重复项，再将新项添加到末尾（保证最新）
	for i, oldItem := range HistoryMusicFiles {
		if oldItem.Path == item.Path && oldItem.Name == item.Name {
			// 删除重复项，直接调用之前的切片原地删除逻辑
			HistoryMusicFiles = append(HistoryMusicFiles[:i], HistoryMusicFiles[i+1:]...)
			break
		}
	}

	// 步骤2：添加新项到切片首（最新的记录在最前）
	HistoryMusicFiles = append([]FileItem{item}, HistoryMusicFiles...)

	// 步骤3：长度控制，超过50则删除最旧的（头部）
	trimHistory()

	// 步骤4：添加后立即持久化到文件（保证实时保存）
	if err := SaveHistory(); err != nil {
		fmt.Printf("添加历史记录后保存失败：%v\n", err)
	}
}

// SaveHistory 将当前HistoryMusicFiles写入history.txt（覆盖写入）
// 可手动调用，AddToHistory内部已自动调用，无需重复执行
func SaveHistory() error {
	// 序列化为带缩进的JSON，方便手动查看history.txt
	data, err := json.MarshalIndent(HistoryMusicFiles, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化历史记录失败：%w", err)
	}

	// 写入文件（0644权限：所有者读写，其他只读，跨平台兼容）
	err = os.WriteFile(historyFileName, data, 0644)
	if err != nil {
		return fmt.Errorf("写入历史文件失败：%w", err)
	}
	return nil
}

// trimHistory 截断历史记录，保证长度不超过maxHistoryCount
// 私有函数，内部调用，无需外部暴露
func trimHistory() {
	if len(HistoryMusicFiles) > maxHistoryCount {
		// 保留最后maxHistoryCount个元素（最新的50个），删除头部旧元素
		HistoryMusicFiles = HistoryMusicFiles[len(HistoryMusicFiles)-maxHistoryCount:]
	}
}

////////////////////////////////////////////////////////////////////////////////

func FindFavorited(item FileItem) int {
	for i, fav := range FavoritedMusicFiles {
		if fav.Name == item.Name && fav.Path == item.Path {
			return i
		}
	}
	return -1
}

func ToFavorited(item FileItem) {
	favoriteMu.Lock()
	defer favoriteMu.Unlock()

	if item.IsFavorited {
		if FindFavorited(item) >= 0 {
			return
		}
		FavoritedMusicFiles = append(FavoritedMusicFiles, item)
	} else {
		index := FindFavorited(item)
		if index < 0 {
			return
		}
		DeleteByIndex(&FavoritedMusicFiles, index)
	}

	if err := SaveFavorited(); err != nil {
		fmt.Printf("添加Favorited记录后保存失败：%v\n", err)
	}
}

func SaveFavorited() error {
	// 序列化为带缩进的JSON，方便手动查看history.txt
	data, err := json.MarshalIndent(FavoritedMusicFiles, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化历史记录失败：%w", err)
	}

	// 写入文件（0644权限：所有者读写，其他只读，跨平台兼容）
	err = os.WriteFile(favoritedFileName, data, 0644)
	if err != nil {
		return fmt.Errorf("写入历史文件失败：%w", err)
	}
	return nil
}

func CheckFavorited(files *[]FileItem) {
	// 边界校验：入参指针为nil 或 切片为空，直接返回
	if files == nil || len(*files) == 0 {
		return
	}

	// 步骤1：加读锁读取收藏列表，保证并发安全（防止读取时收藏列表被修改）
	favoriteMu.RLock()
	// 提前将收藏列表转为「Name+Path唯一键」的map，将嵌套循环O(n²)优化为O(n)，提升遍历效率
	favMap := make(map[string]struct{}, len(FavoritedMusicFiles))
	for _, favItem := range FavoritedMusicFiles {
		// 拼接唯一键：Name+Path（双字段匹配，避免单字段重复导致误判）
		// 分隔符用特殊字符（如|），防止Name/Path本身包含的字符拼接后冲突
		key := favItem.Name + "|" + favItem.Path
		favMap[key] = struct{}{} // 空结构体不占内存，仅做存在性判断
	}
	favoriteMu.RUnlock() // 立即释放读锁（无需持有到函数结束，减少锁竞争）

	// 步骤2：遍历待标记的files切片，逐行匹配并标记收藏状态
	for i := range *files {
		// 跳过目录项（你的Path字段对目录为空，无需判断收藏，可根据业务调整）
		if (*files)[i].IsDir {
			continue
		}
		// 拼接当前项的唯一键，和收藏map对比
		curKey := (*files)[i].Name + "|" + (*files)[i].Path
		// 存在则标记IsFavorited=true，不存在则保持默认false
		if _, exists := favMap[curKey]; exists {
			(*files)[i].IsFavorited = true
		}
	}
}

func init() {
	InitHistory()
}
