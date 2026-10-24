package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var (
	ilist      = flag.Bool("list", false, "列示本机的网络接口")
	iface      = flag.String("i", "", "监控的本机网络接口名称 (必填)")
	coolsec    = flag.Int("c", 12, "冷却时间（秒）：避免短时间内重复触发")
	trigger    = flag.String("t", "127.0.0.1:9997", "xiaobot部署地址")
	filterFile = flag.String("f", "", "从指定文件加载filter过滤器信息")
	outputFile = flag.String("w", "", "输出日志文件，为空则打印到控制台")
	//= flag.String("mac", "", "小爱音箱的MAC地址 (必填，格式如 aa:bb:cc:dd:ee:ff)")
	filterStr  string = ""
	SpeakerMAC string = ""
	SpeakerIP  string = ""
)

func FindAlldevs() error {
	// 查找网络设备
	devices, err := pcap.FindAllDevs()
	if err != nil {
		fmt.Errorf("查找设备失败: ", err)
		return err
	}
	for _, k := range devices {
		if len(k.Addresses) > 0 && !strings.Contains(k.Description, "loop") {
			fmt.Println("设备：", k.Name, k.Description, k.Addresses[0].IP)
		}
	}
	return nil
}

func FindWdevs() (string, error) {
	// 查找网络设备
	devices, err := pcap.FindAllDevs()
	if err != nil {
		fmt.Errorf("查找设备失败: ", err)
		return "", err
	}
	iface := ""
	i := 0
	for _, k := range devices {
		if len(k.Addresses) > 0 && !strings.Contains(k.Description, "loop" /*"Virtual"*/) {
			log.Println("设备：", k.Name, k.Description)
			for _, addr := range k.Addresses {
				log.Println(addr.IP)
			}
			iface = k.Name
			// 打开设备进行捕获
			handle, err := pcap.OpenLive(iface, 1600, true, pcap.BlockForever)
			if err != nil {
				fmt.Errorf("无法打开设备 %s: %v", iface, err)
				continue
			}
			defer handle.Close()
			// 设置BPF过滤器，只捕获与小爱音箱MAC相关的流量
			filter := fmt.Sprintf("ether host %s", SpeakerMAC)
			err = handle.SetBPFFilter(filter)
			if err != nil {
				fmt.Errorf("设置过滤器失败: %v", err)
				continue
			}

			SpeakerIP = ""
			i++
			fmt.Print("小爱音箱连接测试...", i)

			// 开始捕获数据包
			packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

			// 创建退出信号通道（用于通知goroutine停止）
			quitChan := make(chan struct{})
			// 创建一个通道用于接收任务结果
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
						processPacket(packet, SpeakerMAC, nil)
						// 按原逻辑处理完一个包后退出（如果需要持续处理可去掉break）
						overChan <- true
						return
					}
				}
			}()
			// 等待结果或超时（3秒）
			select {
			case <-overChan:
				// 任务在超时前完成
				fmt.Println("ok")
			case <-time.After(3 * time.Second):
				// 任务超时
				fmt.Println("设备超时")
				close(quitChan) // 发送退出信号给goroutine。关闭通道作为退出信号（所有监听者都会收到）
			}
		}
		if SpeakerIP != "" {
			fmt.Println("连接测试成功")
			//break
			return iface, nil
		}
	}
	return "", fmt.Errorf("未查找到网口")
}

func main() {
	flag.Parse()
	if *ilist {
		FindAlldevs()
		return
	}
	filterStr = ""
	i := 1
	for i < len(os.Args) {
		if os.Args[i][0] == '-' {
			i += 1
		} else {
			filterStr += os.Args[i] + " "
		}
		i += 1
	}

	if *filterFile != "" {
		str, err := os.ReadFile(*filterFile)
		if err != nil {
			filterStr = string(str)
		}
	}

	if *iface == "" {
		if filterStr == "" {
			log.Fatal("请使用 -iface 参数指定网络接口名称")
		}
		SpeakerMAC, _ = FindMAC(filterStr)
		log.Println("自动查找网络接口...", SpeakerMAC)
		if netface, err := FindWdevs(); err == nil {
			log.Println("find device ok:", netface)
			*iface = netface
		} else {
			log.Fatal(err)
		}
	}
	log.Println("xiaoai speaker mac & ip:", SpeakerMAC, SpeakerIP)

	// 打开输出文件
	var output *os.File
	var err error
	if *outputFile != "" {
		output, err = os.OpenFile(*outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("无法打开输出文件: %v", err)
		}
		defer output.Close()
	}

	// 查找网络设备并检查指定接口是否存在
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatalf("查找设备失败: %v", err)
	}

	found := false
	for _, dev := range devices {
		if dev.Name == *iface || dev.Description == *iface || containsInterfaceAlias(dev, *iface) {
			found = true
			*iface = dev.Name
			break
		}
	}
	if !found {
		log.Fatalf("找不到网络接口: %s，请使用正确的接口名称", *iface)
	}

	// 打开设备进行捕获
	handle, err := pcap.OpenLive(*iface, 1600, true, pcap.BlockForever)
	if err != nil {
		log.Fatalf("无法打开设备 %s: %v", *iface, err)
	}
	defer handle.Close()

	// 设置BPF过滤器，只捕获与小爱音箱MAC相关的流量
	log.Printf("设置filter: %s\n", filterStr)
	filter := filterStr //fmt.Sprintf("ether host %s", *speakerMAC)
	err = handle.SetBPFFilter(filter)
	if err != nil {
		log.Fatalf("设置过滤器失败: %v", err)
	}

	log.Printf("监控接口: %s\n", *iface)
	log.Printf("开始监控小爱音箱的网络连接...\n")
	if *outputFile != "" {
		log.Printf("日志将保存到: %s\n", *outputFile)
	}

	// 开始捕获数据包
	start := time.Now().Unix()
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		if ok := processPacket(packet, SpeakerMAC, output); ok {
			end := time.Now().Unix()
			if end-start < int64(*coolsec) {
				continue
			}
			start = end
			log.Println("send monitor")
			FireAndForgetGET(fmt.Sprintf("http://%s/monitor", *trigger))
		}
	}
}

// FireAndForgetGET 发送 GET 请求，不阻塞，不关心响应
func FireAndForgetGET(url string) {
	// 使用 goroutine 异步发送
	go func() {
		// 设置超时
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		resp, err := client.Get(url)
		if err != nil {
			log.Printf("FireAndForgetGET error: %v", err)
			return
		}
		defer resp.Body.Close()

		// 必须读取响应体，也建议至少 Read 一下, 否则会阻塞连接复用
		io.Copy(io.Discard, resp.Body) // 丢弃响应体
	}()
}

// 处理捕获到的数据包
func processPacket(packet gopacket.Packet, speakerMAC string, output *os.File) (isXiaoai bool) {
	isXiaoai = false
	// 获取以太网层
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		return
	}
	eth, _ := ethLayer.(*layers.Ethernet)

	// 确定流量方向
	var isOutgoing bool // 音箱发出的请求
	srcMAC := eth.SrcMAC.String()
	dstMAC := eth.DstMAC.String()

	action := ""
	if srcMAC == speakerMAC {
		isOutgoing = true
		action = "小爱音箱发送"
	} else if dstMAC == speakerMAC {
		isOutgoing = false
		action = "小爱音箱接收"
	} else {
		//return // 不是与音箱相关的流量
	}

	// 获取IP层信息
	var srcIP, dstIP, dstHost string
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		srcIP = ip.SrcIP.String()
		dstIP = ip.DstIP.String()

		// 尝试解析目标IP的域名
		if isOutgoing {
			SpeakerIP = ip.SrcIP.String()
			names, err := net.LookupAddr(dstIP)
			if err == nil && len(names) > 0 {
				dstHost = names[0]
			} else {
				dstHost = dstIP
			}
		} else {
			SpeakerIP = ip.DstIP.String()
			names, err := net.LookupAddr(srcIP)
			if err == nil && len(names) > 0 {
				dstHost = names[0]
			} else {
				dstHost = srcIP
			}
		}
	}

	// 获取传输层协议信息
	var protocol string
	var srcPort, dstPort string

	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		protocol = "TCP"
		srcPort = tcp.SrcPort.String()
		dstPort = tcp.DstPort.String()
	}

	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		protocol = "UDP"
		srcPort = udp.SrcPort.String()
		dstPort = udp.DstPort.String()
	}

	// 获取应用层数据
	var appData string
	appLayer := packet.ApplicationLayer()
	if appLayer != nil {
		data := appLayer.Payload()
		if len(data) > 100 {
			appData = string(data[:100]) + "..."
		} else {
			appData = string(data)
		}
	}

	// 格式化日志信息
	/*action := "接收"
	if isOutgoing {
		action = "发送"
	}*/

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMsg := fmt.Sprintf("[%s] %s数据: %s %s:%s -> %s:%s 数据: %s\n",
		timestamp, action, protocol, srcIP, srcPort, dstHost, dstPort, appData)

	// 输出日志
	if output != nil {
		output.WriteString(logMsg)
	} else {
		fmt.Print(logMsg)
	}

	return isOutgoing
}

// 验证MAC地址格式
func isValidMAC(mac string) bool {
	_, err := net.ParseMAC(mac)
	return err == nil
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
		//fmt.Println(string(a))
		if a == alias {
			return true
		}
	}
	return false
}

// FindMAC 从字符串中查找第一个 MAC 地址
func FindMAC(s string) (string, bool) {
	// 正则表达式：匹配 MAC 地址格式
	macRegex := regexp.MustCompile(`([0-9a-fA-F]{2}:){5}([0-9a-fA-F]{2})`)

	// 查找第一个匹配项
	match := macRegex.FindString(s)
	if match == "" || !isValidMAC(match) {
		return "", false
	}

	return normalizeMAC(match), true
}
