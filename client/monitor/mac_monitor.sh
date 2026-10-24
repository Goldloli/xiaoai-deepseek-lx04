#!/bin/sh

# 配置参数 - 根据实际情况修改
TARGET_MAC="aa:bb:cc:dd:ee:ff"  # 目标设备的MAC地址(注意字母要小写)
INTERFACE="eth0"                # 监控的网络接口，通常是br-lan
COOLDOWN=12                     # 冷却时间(秒)，避免频繁触发

echo "MAC监控脚本启动，目标MAC: $TARGET_MAC，接口: $INTERFACE"

# 初始化最后触发时间
last_trigger=0

# 使用tcpdump监控外发流量
tcpdump -i $INTERFACE -n ether src $TARGET_MAC 'and dst port 443 and (tcp[tcpflags] & tcp-syn) != 0 and tcp[13]=0x02 and (dst 220.181.106.172 or dst 106.120.178.12)' -v -l 2>/dev/null | while read -r line; do
    # 获取当前时间戳
    current_time=$(date +%s)
    
    # 检查是否在冷却期内
    if [ $((current_time - last_trigger)) -ge $COOLDOWN ]; then
        echo "$(date)检测到目标MAC($TARGET_MAC)的外发流量"
        
        # 调用HTTP接口
        curl -s -w " %{http_code}" "http://192.168.3.111:9997/monitor"
        
        # 更新最后触发时间
        last_trigger=$current_time
    fi
done
    