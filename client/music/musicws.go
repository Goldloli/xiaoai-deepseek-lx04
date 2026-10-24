package music

import (
	"net/http"
	"ninego/log"
	"time"

	"github.com/gorilla/websocket"
)

// 1. 定义WS升级器：将HTTP连接升级为WS连接
// CheckOrigin关闭跨域校验（测试用，生产环境需根据实际配置跨域规则）
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024, // 读缓冲区大小
	WriteBufferSize: 1024, // 写缓冲区大小
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有跨域请求，生产需替换为真实跨域校验
	},
}

// 2. 连接管理器：管理所有活跃的WS连接（线程安全）
type ClientManager struct {
	clients    map[*Client]bool // 所有活跃客户端
	broadcast  chan []byte      // 广播消息通道
	register   chan *Client     // 客户端注册通道
	unregister chan *Client     // 客户端注销通道
}

// 3. 单个WS客户端连接
type Client struct {
	conn *websocket.Conn // 客户端WS连接
	send chan []byte     // 客户端消息发送通道
}

// 全局连接管理器实例
var manager *ClientManager

// 4. 管理器核心循环：处理注册、注销、广播
func (m *ClientManager) Run() {
	for {
		select {
		// 新客户端注册
		case client := <-m.register:
			m.clients[client] = true
			log.Debugf("客户端连线，当前在线数：%d", len(m.clients))
		// 客户端注销
		case client := <-m.unregister:
			if _, ok := m.clients[client]; ok {
				close(client.send)        // 关闭客户端发送通道
				delete(m.clients, client) // 从管理器移除
				log.Debugf("客户端下线，当前在线数：%d", len(m.clients))
			}
		// 广播消息给所有在线客户端
		case msg := <-m.broadcast:
			for client := range m.clients {
				select {
				case client.send <- msg: // 消息发送到客户端通道
				default:
					close(client.send)
					delete(m.clients, client)
				}
			}
		}
	}
}

// 5. 客户端读循环：持续读取客户端发送的消息
func (c *Client) ReadPump() {
	defer func() {
		manager.unregister <- c // 异常时注销客户端
		c.conn.Close()          // 关闭WS连接
	}()

	// 设置WS连接参数：读超时（心跳检测基础）
	c.conn.SetReadLimit(512) // 限制单次读取消息大小，防止超大消息攻击
	// 读超时：30秒内未收到客户端消息（含心跳），则触发读错误
	c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	// 设置心跳检测回调：收到任意消息（含心跳）时，重置读超时
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		return nil
	})

	// 持续读取客户端消息
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			// 客户端主动关闭或网络异常，退出读循环
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("读取WS消息错误：%v", err)
			}
			break
		}
		// 收到消息后，广播给所有在线客户端
		log.Debug("收到客户端消息：%s", string(msg))
		manager.broadcast <- msg
	}
}

// 6. 客户端写循环：持续将消息发送给客户端
func (c *Client) WritePump() {
	// 心跳定时器：每25秒给客户端发一次心跳（ping），比读超时30秒短，避免误判
	ticker := time.NewTicker(25 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	// 持续发送消息（业务消息+心跳）
	for {
		select {
		case msg, ok := <-c.send:
			// 客户端发送通道关闭，退出写循环
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 写入业务消息到客户端
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(msg)

			// 批量写入通道中剩余的消息（防止消息堆积）
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			// 发送心跳（ping），检测客户端是否在线
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// 7. WS服务处理函数：HTTP升级为WS，初始化客户端
func WsHandler(w http.ResponseWriter, r *http.Request) {
	if manager == nil {
		manager = &ClientManager{
			broadcast:  make(chan []byte),
			register:   make(chan *Client),
			unregister: make(chan *Client),
			clients:    make(map[*Client]bool),
		}
		// 启动连接管理器的核心循环（后台协程）
		go manager.Run()
	}

	// 将HTTP请求升级为WS连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("升级WS连接失败：%v", err)
		return
	}

	// 初始化新客户端
	client := &Client{
		conn: conn,
		send: make(chan []byte, 256),
	}
	// 客户端注册到管理器
	manager.register <- client

	// 启动协程处理：写消息（服务端→客户端）、读消息（客户端→服务端）
	// 两个协程分离，避免读写阻塞
	go client.WritePump() //写消息（服务端→客户端）
	//go client.ReadPump() //读消息（客户端→服务端）
}

// PushToAll 服务端向【所有在线客户端】广播消息（复用原有broadcast通道，最简洁）
// msg：要发送的消息内容（字节数组，文本/JSON均可）
func PushToAll(msg string) {
	if manager == nil {
		return
	}

	if len(manager.clients) == 0 {
		//log.Println("无在线客户端，无需广播")
		return
	}
	// 直接往manager.broadcast通道发消息，原有Run()循环会自动广播给所有客户端
	manager.broadcast <- []byte(msg)
	//log.Printf("全局广播，消息：%s，在线客户端数：%d", msg, len(manager.clients))
}
