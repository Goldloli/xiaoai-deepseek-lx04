package monitor

import (
	"fmt"
	"net"
	"strings"
	"time"

	"xiaobot"
	"ninego/log"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var (
	SpeakerIP string
)

func FindWlandevs(mt *xiaobot.MiBot, speakerMAC string) (string, error) {
	// 查找网络设备
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Error("查找设备失败: ", err)
		return "", err
	}
	iface := ""
	for _, k := range devices {
		if len(k.Addresses) > 0 && !strings.Contains(k.Description, "loop") { //"Virtual"
			log.Println("net:", k.Name, k.Description, k.Addresses[0].IP)
			// 打开设备进行捕获
			handle, err := pcap.OpenLive(k.Name, 1600, true, pcap.BlockForever)
			if err != nil {
				log.Error("无法打开设备 %s: %v", k.Name, err)
				continue
			}
			defer handle.Close()
			// 设置BPF过滤器，只捕获与小爱音箱MAC相关的流量
			filter := fmt.Sprintf("ether host %s", speakerMAC)
			err = handle.SetBPFFilter(filter)
			if err != nil {
				//log.Error("设置过滤器失败: %v", err)
				continue
			}

			SpeakerIP = ""
			//fmt.Println("小爱音箱连接测试...")

			// 开始捕获数据包
			packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

			// 创建退出信号通道（用于通知goroutine停止）
			quitChan := make(chan struct{})
			// 创建任务完成通道
			overChan := make(chan bool)
			// 启动goroutine执行任务
			go func() {
				// 使用for循环+select监听数据包和退出信号
				for {
					select {
					// 监听退出信号，收到后立即退出goroutine
					case <-quitChan:
						//fmt.Println("goroutine收到退出信号，停止工作")
						return
					// 正常处理数据包
					case packet, ok := <-packetSource.Packets():
						// 如果通道已关闭（如pcap停止），则退出
						if !ok {
							overChan <- false
							return
						}
						// 处理数据包
						processPacket(packet, speakerMAC)
						// 按原逻辑处理完一个包后退出（如果需要持续处理可去掉break）
						overChan <- true
						iface = k.Name
						return
					}
				}
			}()
			// 等待结果或超时（3秒）
			select {
			case <-overChan:
				// 任务在超时前完成
			case <-time.After(5 * time.Second):
				// 任务超时
				close(quitChan) // 发送退出信号给goroutine。关闭通道作为退出信号（所有监听者都会收到）
			}
		}
		if SpeakerIP != "" {
			log.Println("连接测试成功")
			break
		}
	}
	return iface, nil
}
func Run(mt *xiaobot.MiBot, speakerMAC, iface string) error {
	//var iface string      // "网络接口，通常是br-lan"
	//var speakerMAC string // "小爱音箱的mac地址 (必填)"
	if speakerMAC == "" {
		speakerMAC = mt.MacAddr
	}
	if iface == "" || iface == "auto" {
		dev, err := FindWlandevs(mt, speakerMAC)
		if err != nil {
			return err
		}
		iface = dev
		if iface == "" {
			err := fmt.Errorf("找不到可监控音箱的网络接口")
			return err
		}
	} else {
		// 查找网络设备并检查指定接口是否存在
		devices, err := pcap.FindAllDevs()
		if err != nil {
			//log.Error("查找设备失败: ", err)
			return err
		}
		// 检查指定的接口是否存在
		found := false
		for _, dev := range devices {
			if dev.Name == iface || dev.Description == iface || containsInterfaceAlias(dev, iface) {
				found = true
				break
			}
		}
		if !found {
			err := fmt.Errorf("找不到网络接口: %s，请使用正确的接口名称", iface)
			return err
		}
	}

	// 标准化MAC地址（转为小写）
	speakerMAC = normalizeMAC(speakerMAC)

	// 打开设备进行捕获
	handle, err := pcap.OpenLive(iface, 1600, true, pcap.BlockForever)
	if err != nil {
		//log.Error(fmt.Sprintf("无法打开设备 %s: %v", iface, err))
		return err
	}
	defer handle.Close()

	// 设置BPF过滤器，只捕获与小爱音箱MAC相关的流量
	filter := fmt.Sprintf("ether src %s and dst port 443 and (tcp[tcpflags] & tcp-syn) != 0 and tcp[13]=2 and (dst 220.181.106.172 or dst 106.120.178.12)", speakerMAC)
	err = handle.SetBPFFilter(filter)
	if err != nil {
		//log.Error("设置过滤器失败: %v", err)
		return err
	}

	log.Printf("开始监控小爱音箱 %s 的网络连接...\n", speakerMAC)
	log.Printf("监控接口: %s\n", iface)

	//给5秒延时
	go func() {
		for {
			select {
			case <-time.After(5 * time.Second):
				mt.Updatemonitor(-1)
			}
		}
	}()
	// 开始捕获数据包
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	go func() {
		for packet := range packetSource.Packets() {
			if ok, err := processPacket(packet, speakerMAC); ok {
				//调用小爱
				mt.Updatemonitor(+1)
			} else if err != nil {
				return //err
			}
		}
	}()

	//运行小爱
	return mt.Run(0)
}

// 处理捕获到的数据包
func processPacket(packet gopacket.Packet, speakerMAC string) (bool, error) {
	// 获取以太网层
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		return false, fmt.Errorf("获取以太网层 error.")
	}
	eth, _ := ethLayer.(*layers.Ethernet)

	// 确定流量方向
	var isOutgoing bool // 音箱发出的请求
	srcMAC := eth.SrcMAC.String()
	dstMAC := eth.DstMAC.String()

	if srcMAC == speakerMAC {
		isOutgoing = true
	} else if dstMAC == speakerMAC {
		isOutgoing = false
	} else {
		return false, fmt.Errorf("不是小爱音箱") // 不是与音箱相关的流量
	}

	// 获取IP层信息
	if SpeakerIP == "" {
		ipLayer := packet.Layer(layers.LayerTypeIPv4)
		if ipLayer != nil {
			ip, _ := ipLayer.(*layers.IPv4)
			if isOutgoing {
				SpeakerIP = ip.SrcIP.String()
			} else {
				SpeakerIP = ip.DstIP.String()
			}
		}
	}

	// 启动
	return isOutgoing, nil

}

// 标准化MAC地址为小写格式
func normalizeMAC(mac string) string {
	parsedMAC, err := net.ParseMAC(mac)
	if err != nil {
		return mac
	}
	return parsedMAC.String()
}

// 检查设备是否包含指定的接口别名
func containsInterfaceAlias(dev pcap.Interface, alias string) bool {
	for _, a := range strings.Split(dev.Description, " ") {
		if a == alias {
			return true
		}
	}
	return false
}
