package tools

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"
)

// 解析MAC地址，将格式化的MAC字符串转为6字节的二进制数据
func parseMAC(macStr string) ([]byte, error) {
	// 替换MAC地址中的分隔符（:或-）为空
	macStr = strings.ReplaceAll(macStr, ":", "")
	macStr = strings.ReplaceAll(macStr, "-", "")

	// 检查MAC地址长度是否为12位（6字节）
	if len(macStr) != 12 {
		return nil, fmt.Errorf("MAC地址格式错误，正确格式如：00:11:22:33:44:55 或 00-11-22-33-44-55")
	}

	// 将16进制字符串转为二进制字节
	mac, err := hex.DecodeString(macStr)
	if err != nil {
		return nil, fmt.Errorf("解析MAC地址失败：%v", err)
	}
	return mac, nil
}

// 构建WOL魔术包
func buildMagicPacket(mac []byte) ([]byte, error) {
	if len(mac) != 6 {
		return nil, fmt.Errorf("MAC地址必须是6字节")
	}

	// 魔术包：6个0xFF + 16次重复的MAC地址
	magicPacket := bytes.Repeat([]byte{0xFF}, 6)
	magicPacket = append(magicPacket, bytes.Repeat(mac, 16)...)

	return magicPacket, nil
}

// 发送魔术包实现远程开机（支持多次发送）
func wakeOnLAN(macStr, broadcastIP string, port int) error {
	var (
		count    int           = 5                                     //发送5次
		interval time.Duration = time.Duration(200) * time.Millisecond //间隔200ms
	)
	// 1. 解析MAC地址
	mac, err := parseMAC(macStr)
	if err != nil {
		return err
	}

	// 2. 构建魔术包（只需构建一次，重复发送）
	magicPacket, err := buildMagicPacket(mac)
	if err != nil {
		return err
	}

	// 3. 配置UDP连接（广播模式）
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", broadcastIP, port))
	if err != nil {
		return fmt.Errorf("解析UDP地址失败：%v", err)
	}

	// 创建UDP连接（复用连接，避免多次创建）
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("创建UDP连接失败：%v", err)
	}
	defer conn.Close() // 延迟关闭连接

	// 4. 多次发送魔术包
	//fmt.Printf("开始发送唤醒包（共%d次，间隔%dms）...\n", count, interval.Milliseconds())
	for i := 0; i < count; i++ {
		_, err = conn.Write(magicPacket)
		if err != nil {
			return fmt.Errorf("第%d次发送失败：%v", i+1, err)
		}
		//fmt.Printf("第%d次唤醒包已发送到 MAC: %s (广播地址: %s:%d)\n", i+1, macStr, broadcastIP, port)

		// 最后一次不等待
		if i < count-1 {
			time.Sleep(interval)
		}
	}

	return nil
}
