package music

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	mp4lib "github.com/abema/go-mp4"
	"github.com/hajimehoshi/go-mp3"
	flaclib "github.com/mewkiz/flac"
	"github.com/youpy/go-wav"
)

// GetAudioDuration 快速获取音频文件时长和真实音乐名称
// filePath: 音频文件路径
// 返回值: 时长(秒)、真实音乐名称、错误信息
func GetAudioDuration(filePath string) (time.Duration, error) {
	// 检查文件是否存在
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 获取文件扩展名，统一转为小写
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return 0, errors.New("无法识别文件格式: 无扩展名")
	}
	ext = ext[1:] // 去掉点号

	// 存储音乐时长
	var duration time.Duration

	// 根据不同格式解析时长
	switch ext {
	case "mp3":
		duration, err = parseMP3Duration(file)
	case "wav":
		duration, err = parseWAVDuration(file)
	case "flac":
		duration, err = parseFLACDuration(file)
	case "m4a", "mp4":
		duration, err = parseM4ADuration(file)
	default:
		duration, err = parseAudioDuration(filePath)
		//return 0, fmt.Errorf("不支持的音频格式: %s（当前仅支持MP3/WAV/FLAC/M4A/MP4）", ext)
	}

	if err != nil {
		return 0, fmt.Errorf("获取%s时长失败: %w", ext, err)
	}

	return duration, nil
	// 转换为秒数返回
	//seconds := duration.Seconds()
	//return seconds, nil
}

/*### 前置条件
1. **安装 ffmpeg**：
   - Windows：从 [FFmpeg 官网](https://ffmpeg.org/download.html) 下载，解压后将 `ffprobe.exe` 所在路径添加到系统环境变量
   - macOS：`brew install ffmpeg`
   - Linux（Ubuntu/Debian）：`sudo apt update && sudo apt install ffmpeg`
   - Linux（CentOS）：`sudo yum install ffmpeg`

2. **验证安装**：打开终端执行 `ffprobe -version`，能输出版本信息即安装成功。
*/
func parseAudioDuration(filePath string) (time.Duration, error) {
	var currentDir string
	if v := os.Getenv("FFMPEG_PATH"); v != "" {
		currentDir = v
	} else {
		// 先获取当前程序运行目录
		v, err := os.Getwd()
		if err != nil {
			return 0, fmt.Errorf("获取当前目录失败: %v", err)
		}
		currentDir = v
	}
	// 拼接ffprobe的路径（ffprobe放在当前目录下）
	ffprobePath := filepath.Join(currentDir, "ffprobe")
	// Windows系统需拼接.exe后缀
	if runtime.GOOS == "windows" {
		ffprobePath += ".exe"
	}
	// 调用 ffprobe 获取音频元数据（JSON 格式）
	cmd := exec.Command(
		ffprobePath,   //"ffprobe",
		"-v", "quiet", // 静默输出，只返回必要信息
		"-print_format", "json", // 输出格式为 JSON
		"-show_format",  // 显示格式信息
		"-show_streams", // 显示流信息
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe 执行失败: %v", err)
	}

	// 解析 JSON 数据
	var ffprobeData struct {
		Streams []struct {
			CodecType string `json:"codec_type"` // 流类型（audio/video）
			Duration  string `json:"duration"`   // 时长（字符串格式的浮点数）
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"` // 整体时长
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &ffprobeData); err != nil {
		return 0, fmt.Errorf("解析 ffprobe 输出失败: %v", err)
	}

	// 提取时长（优先取音频流的时长，没有则取格式的时长）
	durationStr := ""
	for _, stream := range ffprobeData.Streams {
		if stream.CodecType == "audio" && stream.Duration != "" {
			durationStr = stream.Duration
			break
		}
	}
	if durationStr == "" {
		durationStr = ffprobeData.Format.Duration
	}

	if durationStr == "" {
		return 0, errors.New("无法提取音频时长")
	}

	// 转换为浮点数（秒）
	seconds, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("时长转换失败: %v", err)
	}

	// 将浮点型秒数转换为 time.Duration
	integerPart := int64(seconds)                    // 拆分整数秒（如 123.456 的 123）
	fractionalPart := seconds - float64(integerPart) // 拆分小数部分（如 123.456 的 0.456）
	// 整数部分转 Duration
	integerDuration := time.Duration(integerPart) * time.Second
	// 小数部分转纳秒（1秒 = 1e9纳秒），用Round避免浮点精度误差
	fractionalNanos := time.Duration(fractionalPart*1e9 + 0.5) // +0.5 实现四舍五入
	// 合并结果
	durationSec := integerDuration + fractionalNanos
	// 构造返回结果
	return durationSec, nil
}

// 解析MP3时长（极简版，仅计算时长，避开标签解析的版本问题）
func parseMP3Duration(file *os.File) (time.Duration, error) {
	// 重置文件指针到开头
	file.Seek(0, 0)

	dec, err := mp3.NewDecoder(file)
	if err != nil {
		return 0, fmt.Errorf("创建mp3解码器失败: %w", err)
	}

	// 计算时长: 总样本数 / 采样率
	sampleRate := dec.SampleRate()
	totalSamples := dec.Length()
	if sampleRate == 0 {
		return 0, errors.New("无效的MP3采样率")
	}
	duration := time.Duration(float64(totalSamples)/float64(sampleRate)*1000) * time.Millisecond

	return duration, nil
}

// 解析WAV时长（极简版）
func parseWAVDuration(file *os.File) (time.Duration, error) {
	// 重置文件指针
	file.Seek(0, 0)

	reader := wav.NewReader(file)
	format, err := reader.Format()
	if err != nil {
		return 0, fmt.Errorf("读取WAV格式失败: %w", err)
	}

	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("获取文件信息失败: %w", err)
	}
	fileSize := fileInfo.Size()

	// 计算WAV数据区大小（简化版）
	headerSize := int64(44) // 标准WAV头44字节
	dataSize := fileSize - headerSize
	if dataSize <= 0 {
		return 0, errors.New("无效的WAV文件大小")
	}

	// 统一转为float64计算时长
	bytesPerSample := float64(format.BitsPerSample / 8)
	bytesPerSecond := float64(format.SampleRate) * float64(format.NumChannels) * bytesPerSample
	if bytesPerSecond == 0 {
		return 0, errors.New("无效的WAV采样参数")
	}
	seconds := float64(dataSize) / bytesPerSecond
	duration := time.Duration(seconds*1000) * time.Millisecond

	return duration, nil
}

// 解析FLAC时长（极简版，避开MetaBlocks问题）
func parseFLACDuration(file *os.File) (time.Duration, error) {
	// 重置文件指针
	file.Seek(0, 0)

	// 解析FLAC文件
	stream, err := flaclib.Parse(file)
	if err != nil {
		return 0, fmt.Errorf("解析flac失败: %w", err)
	}
	defer stream.Close()

	// 直接从Info获取时长相关信息
	sampleRate := stream.Info.SampleRate
	totalSamples := stream.Info.NSamples
	if sampleRate == 0 {
		return 0, errors.New("无效的FLAC采样率")
	}
	duration := time.Duration(float64(totalSamples)/float64(sampleRate)*1000) * time.Millisecond

	return duration, nil
}

// 解析M4A/MP4时长（极简版）
func parseM4ADuration(file *os.File) (time.Duration, error) {
	// 重置文件指针
	file.Seek(0, 0)

	// 仅调用Probe获取时长信息
	info, err := mp4lib.Probe(file)
	if err != nil {
		return 0, fmt.Errorf("解析M4A/MP4失败: %w", err)
	}

	// 修复字段名拼写（Timescale）
	if info.Duration == 0 || info.Timescale == 0 {
		return 0, errors.New("无法获取M4A/MP4时长信息")
	}
	duration := time.Duration(float64(info.Duration)/float64(info.Timescale)*1000) * time.Millisecond

	return duration, nil
}
