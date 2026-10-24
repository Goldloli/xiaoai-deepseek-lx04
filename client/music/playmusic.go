package music

import (
	"math/rand"
	"ninego/log"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
	. "xiaobot"
)

type playState struct {
	FilteredMusicFiles  []FileItem `json:"filteredMusicFiles"`
	CurrentPlayingIndex int        `json:"currentPlayingIndex"`
	CurrentMusicDir     string     `json:"currentMusicDir"`
	PlayMode            string     `json:"playMode"`
	IsSequencePlaying   bool       `json:"isSequencePlaying"`
	IsRandomPlaying     bool       `json:"isRandomPlaying"`
}

var PlayerState playState
var PlayerMutex sync.Mutex

var PlayerSvrIP string

func SetPlayMode(mode string) {
	PlayerMutex.Lock()
	PlayerState.PlayMode = mode
	PlayerMutex.Unlock()
}

func GetVolume(bot *MiBot) int {
	return bot.Box.MiGetVolume()
}

func SetVolume(bot *MiBot, vol int) error {
	err := bot.Box.MiSetVolume(vol)
	return err
}

func Stop(bot *MiBot) error {
	if PlayChannel != nil {
		PlayChannel.Close()
	}
	err := bot.Box.StopSpeaker() //StopPlayer()无效？
	return err
}

var PlayChannel *Channel //用于管理播放
var SkipChannel *Channel //用于跳过当前音乐

func Play(bot *MiBot, cfg *Config) error {
	// 第一步：先加锁处理旧的播放协程，保证并发安全
	PlayerMutex.Lock()
	oldPlayChan := PlayChannel
	oldSkipChan := SkipChannel
	// 重置全局Channel为nil（加锁内操作，避免数据竞争）
	PlayChannel = nil
	SkipChannel = nil
	PlayerMutex.Unlock()

	// 关闭旧的Channel，优雅释放资源（无需加锁，已置为nil）
	if oldPlayChan != nil && !oldPlayChan.IsClosed() {
		oldPlayChan.Close()
	}
	if oldSkipChan != nil && !oldSkipChan.IsClosed() {
		oldSkipChan.Close()
	}

	// 停止旧的播放，同步执行（避免goroutine导致的操作未完成）
	bot.Box.StopSpeaker()
	bot.Box.MiPlayLoop(false)
	// 显式毫秒休眠，增加可读性
	time.Sleep(100 * time.Millisecond)

	// 第二步：初始化新的Channel，带1个缓冲（避免跳过操作阻塞）
	PlayChannel = NewChannel()
	SkipChannel = NewChannelSize(1)

	playChan := PlayChannel
	skipChan := SkipChannel
	// 第三步：启动播放协程
	go func(pChan *Channel, sChan *Channel) {
		// 根defer：优先释放Channel资源，执行时机为协程退出时
		defer func() {
			pChan.Close()
			sChan.Close()
			log.Println("Playback has finished.")
		}()
		// 捕获协程内的所有panic，避免程序崩溃
		defer func() {
			if r := recover(); r != nil {
				log.Error("Recovered from playback panic:", r)
			}
		}()

		// 主播放循环：判断PlayChannel是否关闭（核心退出条件）
		for !pChan.IsClosed() {
			// 所有读写PlayerState的操作都必须加锁，保证并发安全
			PlayerMutex.Lock()
			// 边界判断：无音乐文件则直接退出循环
			hasNoMusic := PlayerState.CurrentPlayingIndex >= len(PlayerState.FilteredMusicFiles) || len(PlayerState.FilteredMusicFiles) == 0
			PlayerMutex.Unlock()
			if hasNoMusic {
				break
			}

			// 加锁读取当前播放的音乐文件
			PlayerMutex.Lock()
			currentMusic := PlayerState.FilteredMusicFiles[PlayerState.CurrentPlayingIndex]
			PlayerMutex.Unlock()

			// 非文件夹则执行播放
			if !currentMusic.IsDir {
				// 拼接完整文件路径
				filePath := filepath.Join(
					cfg.MusicPath,
					currentMusic.Path,
					currentMusic.Name,
				)
				// 记录播放历史
				AddToHistory(currentMusic)
				// 获取播放时长
				duration, err := GetAudioDuration(filePath)
				if err != nil {
					log.Error("Get audio duration failed:", err)
					// 时长获取失败，直接跳转到下一首
					goto nextMusic
				}

				// 时长有效才执行播放
				if duration > 0 {
					// 播放前再次判断是否已停止，避免无效操作
					if pChan.IsClosed() {
						break
					}
					// 打印带小数的播放时长，增加可读性
					log.Printf("Playing -> %.2fs | %s\n", duration.Seconds(), filePath)

					// 拼接前端播放的URL（统一路径分隔符为/）
					subPath := strings.ReplaceAll(currentMusic.Path, string(filepath.Separator), "/")
					fullUrl := path.Join("http://", PlayerSvrIP, "music", subPath, currentMusic.Name)
					log.Println("Play URL:", fullUrl)

					// 通知前端当前播放的音乐
					PushToAll(currentMusic.Name)

					// 执行播放操作
					if err := bot.Box.MiPlay(fullUrl); err != nil {
						log.Error("Execute play failed:", err)
						goto nextMusic
					}

					var duration2 time.Duration = 0
					if duration > time.Second*10 {
						duration2 = time.Second * 3 //提前10秒判断是否已停止
						duration -= duration2
					}
				nexttail:
					// 核心：播放等待（停止/跳过/时长结束）
					select {
					case <-pChan.C:
						// 收到停止指令，直接退出协程
						return
					case val, ok := <-sChan.C:
						// 跳过当前曲目，进入下一次循环
						if ok && !PlayerState.IsRandomPlaying { //ok=非关闭(ok为false表示Channel已关闭)
							PlayerMutex.Lock()
							skipStep, _ := val.(int)
							total := len(PlayerState.FilteredMusicFiles)
							if skipStep < 0 {
								PlayerState.CurrentPlayingIndex = (PlayerState.CurrentPlayingIndex - 1 + total) % total
							} else {
								PlayerState.CurrentPlayingIndex = (PlayerState.CurrentPlayingIndex + 1) % total
							}
							PlayerMutex.Unlock()
							continue
						}
					case <-time.After(duration):
						if duration2 != 0 {
							//检查是否中止
							if status, _ := bot.Box.MusicIsPlaying(); status != 1 {
								return
							} else {
								duration = duration2
								duration2 = 0
								goto nexttail
							}
						}
						// 播放时长正常结束，继续下一首
						log.Printf("Play finished: %s\n", currentMusic.Name)
					}
				}
			}

			// 跳转到下一首的统一入口（时长失败/播放失败/正常结束）
		nextMusic:
			// 加锁更新播放索引，根据播放模式处理
			PlayerMutex.Lock()
			totalMusic := len(PlayerState.FilteredMusicFiles)
			// 无音乐则直接解锁退出
			if totalMusic == 0 {
				PlayerMutex.Unlock()
				break
			}

			if PlayerState.IsSequencePlaying {
				// 顺序播放：下一首，循环则重置为0
				PlayerState.CurrentPlayingIndex++
				if PlayerState.CurrentPlayingIndex >= totalMusic {
					if PlayerState.PlayMode == "loop" {
						PlayerState.CurrentPlayingIndex = 0
					} else {
						// 非循环则解锁退出
						PlayerMutex.Unlock()
						break
					}
				}
			} else if PlayerState.IsRandomPlaying {
				// 随机播放逻辑
				if PlayerState.PlayMode != "loop" {
					// 非循环：删除当前播放的音乐，避免重复播放
					DeleteByIndexUnordered(&PlayerState.FilteredMusicFiles, PlayerState.CurrentPlayingIndex)
					// 删除后更新总数量
					totalMusic = len(PlayerState.FilteredMusicFiles)
					// 无音乐则解锁退出
					if totalMusic == 0 {
						PlayerMutex.Unlock()
						break
					}
				}
				// 生成0~totalMusic-1的随机索引，无越界
				PlayerState.CurrentPlayingIndex = rand.Intn(totalMusic)
			} else if PlayerState.PlayMode != "loop" {
				// 既非顺序也非随机，且非循环模式，直接退出
				PlayerMutex.Unlock()
				break
			}
			// 解锁：必须放在最后，避免锁泄漏
			PlayerMutex.Unlock()
		} // 主播放循环结束

		PlayerMutex.Lock()
		PlayerState.IsSequencePlaying = false
		PlayerState.IsRandomPlaying = false
		PlayerMutex.Unlock()
		// 播放全部结束，通知前端
		PushToAll("-end-")
	}(playChan, skipChan)

	return nil
}

// DeleteByIndex 按索引原地删除切片元素，保持顺序
// 入参：s *[]T 切片指针（核心，实现原地修改），idx 要删除的索引
// 无返回值，直接修改原切片
func DeleteByIndex[T any](s *[]T, idx int) {
	// 校验索引合法性：空切片/索引越界，直接返回
	if s == nil || len(*s) == 0 || idx < 0 || idx >= len(*s) {
		return
	}
	// 核心：通过切片指针，拼接前后切片，直接覆盖原切片
	*s = append((*s)[:idx], (*s)[idx+1:]...)
}

// DeleteByIndexUnordered 按索引原地删除，不保持顺序，性能O(1)
// 入参：s *[]T 切片指针，idx 要删除的索引
func DeleteByIndexUnordered[T any](s *[]T, idx int) {
	if s == nil || len(*s) == 0 || idx < 0 || idx >= len(*s) {
		return
	}
	// 核心：1. 最后一个元素覆盖目标索引 2. 截断切片（去掉最后一个重复元素）
	lastIdx := len(*s) - 1
	(*s)[idx] = (*s)[lastIdx] // 覆盖
	*s = (*s)[:lastIdx]       // 截断，原地修改len
}

// 初始化随机数种子（全局只执行一次，避免重复seed）
func init() {
	rand.Seed(time.Now().UnixNano())
}
