package webui

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"time"
)

// 全局配置
const (
	CheckTimeout = 2 * time.Second // 检测超时
	PingCount    = 2               // Ping包数量（2个足够，避免检测过久）
	PingTimeout  = 1               // 单个Ping包超时（秒）
)

// getLocalValidIPs 过滤本机有效IPv4（非回环、网卡启用）
func getLocalValidIPs() ([]string, error) {
	var validIPs []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("获取网卡失败：%v", err)
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}
			validIPs = append(validIPs, ipNet.IP.String())
		}
	}

	if len(validIPs) == 0 {
		return nil, fmt.Errorf("未找到本机有效IPv4地址")
	}
	return validIPs, nil
}

// isSameSubnet 修复版：判断两个IPv4是否同网段（无函数调用错误）
func isSameSubnet(serverIP, clientIP string) (bool, error) {
	sIP := net.ParseIP(serverIP).To4()
	cIP := net.ParseIP(clientIP).To4()
	if sIP == nil || cIP == nil {
		return false, fmt.Errorf("无效IPv4：%s/%s", serverIP, clientIP)
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return false, fmt.Errorf("获取网卡失败：%v", err)
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}
			if ipNet.Contains(sIP) || ipNet.Contains(cIP) {
				sNet := sIP.Mask(ipNet.Mask)
				cNet := cIP.Mask(ipNet.Mask)
				return sNet.Equal(cNet), nil
			}
		}
	}
	return false, nil
}

// ping 指定从「serverIP」向「clientIP」发起Ping，返回是否Ping通（跨平台）
// 核心：指定出口IP，确保检测的是当前服务端IP与客户端的连通性
func ping(serverIP, clientIP string) bool {
	var cmd *exec.Cmd
	// 拼接Ping命令，跨平台适配+指定出口IP+指定包数+超时
	switch runtime.GOOS {
	case "linux":
		// Linux: ping -I 出口IP -c 包数 -W 超时(秒) 目标IP
		cmd = exec.Command("ping", "-I", serverIP, "-c", fmt.Sprintf("%d", PingCount), "-W", fmt.Sprintf("%d", PingTimeout), clientIP)
	case "darwin": // macOS
		// macOS: ping -S 出口IP -c 包数 -t 超时(秒) 目标IP
		cmd = exec.Command("ping", "-S", serverIP, "-c", fmt.Sprintf("%d", PingCount), "-t", fmt.Sprintf("%d", PingTimeout), clientIP)
	case "windows":
		// Windows: ping -S 出口IP -n 包数 -w 超时(毫秒) 目标IP
		cmd = exec.Command("ping", "-S", serverIP, "-n", fmt.Sprintf("%d", PingCount), "-w", fmt.Sprintf("%d", PingTimeout*1000), clientIP)
	default:
		return false
	}

	// 执行Ping，忽略输出，只判断执行结果（0为成功）
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return true
	}
	return false
}

// CheckAllServerIP 主函数：遍历本机所有有效IP，检测对客户端IP的连通性（无客户端端口）
func CheckServerIP(clientIP string) (string, error) {
	// 参数校验
	if clientIP == "" || net.ParseIP(clientIP).To4() == nil {
		return "", fmt.Errorf("客户端IP无效，仅支持IPv4")
	}

	// 步骤1：获取本机所有有效IP
	serverIPs, err := getLocalValidIPs()
	if err != nil {
		return "", err
	}

	// 步骤2：遍历每个服务端IP检测
	ResultIP := ""
	for _, sip := range serverIPs {
		// 子步骤1：判断是否同网段
		sameSubnet, err := isSameSubnet(sip, clientIP)
		if err != nil {
			continue
		}
		if sameSubnet {
			ResultIP = sip
			break
		}

		// 子步骤2：核心-Ping检测（指定从当前服务端IP发起）
		pingOk := ping(sip, clientIP)
		if pingOk {
			ResultIP = sip
			break
		}
	}

	if ResultIP == "" {
		return "", fmt.Errorf("本地无IP连通%s", clientIP)
	}
	return ResultIP, nil
}
