## 目标

1.小爱控制外部应用：获取到对小爱的提问时，可调用js脚本进行后台处理（比如将问题转发到外部url）
2.外部应用控制小爱：添加/task接收执行js任务功能，让xiaobot运行指定任务的js脚本(外部调用xiaobot,可以在脚本中操作小爱)。

## 扩展脚本引擎

引擎只是一个纯 js 运行时，引擎相当于是一个阉割版的浏览器环境，并且主要支持的是es5+语法。
所以，需要Polyfill提供一些常用的模块来开发扩展，例如网络访问使用的 ajax、fetch 等等。
目前参考网上的一些开源项目实现了一些常用的API，例如：XMLHttpRequest、fetch、setTimeout、setInterval、crypto、Buffer、console等等。
有了XMLHttpRequest、fetch这两个API，意味着你可以通过这两个API或者基于它们的第三方库来实现网络请求，比如axios、superagent等等。
	TextEncoder、TextDecoder
	fetch、XMLHttpRequest
	URL
	File
其他：	
// 单条命令执行	
	command(cmdstr)
// 通过调用系统 shell 来执行命令，支持管道 (|) 和输出重定向 (>) 操作
	shell(cmdrun)
// 局域网远程开机
	wol(mac地址[11:22:33:44:55:66], 广播地址[192.168.1.255], UDP端口[9/7])
	
## 运行环境​

1.  小爱音箱收到提问时执行query.bot扩展
	在该环境中，提问内容在query变量中。若脚本中无handled字样，脚本在后台异步运行；否则要等待query.bot脚本运行完成后，再据handled的true/false状态决定xiaobot是否还需要进入后续的AI流程。
2.  三方调用xiaobot的/task/{action}接口来执行{action}.bot扩展
	在该环境中，有2个默认的js对象：req/res

#### req请求体对象：

	req.method() → 返回请求方法
	req.url() → 返回完整 URL
	req.headers() → 返回请求头 map
	req.body() → 返回请求体字符串
	req.params() → 返回路由参数 map
	req.query() → 返回查询参数 map

#### res响应体对象：

	res.status(200) → 设置状态码       
	res.send('text') → 发送文本       
	res.json({}) → 发送 JSON        
	res.set('key', 'val') → 设置响应头 
	res.redirect('/url') → 重定向    

#### 全局对象：

	global
	window
	bot.storage
	全局对象在整个xiaobot运行周期有效。可以在js脚本中对它赋值，下次再运行js脚本该值仍然有效。
	示例：bot.storage.videodate = '2025-03-11'

#### 控制小爱音箱的指令：

​	bot.askAI('text'); 					//向AI提交text提问内容,返回AI的回复文本内容
​	bot.tts('text',wait);				//让小爱朗读text文本. [wait]参数指明是(true)否(false)等待朗读结束
​	bot.action('command');				//让小爱执行指令command(不发声),失败可返回error
	bot.playurl(url)					//让小爱播放url指定的音乐文件
​	bot.stopspeaker(); 					//让小爱闭嘴
​	bot.wakeup(); 						//唤醒小爱
​	bot.sleep(seconds); 				//延时<seconds>秒
​	bot.elapsed(text)					//返回预计text文本内容播放时间,单位是s(秒)
​	bot.wait()							//等待小爱播放完毕(有些型号音箱不支持)
	bot.storage							//全局变量

## 扩展调试​

在脚本中可以通过 console 对象进行日志输出，支持log、trace、debug、info、warn、error多种级别，示例：
	console.log("error");
xiaobot启动时添加了-d info参数，日志文件在 xiaobot 安装目录里的xiaobot.log文件中。可以通过tail -f xiaobot.log命令实时查看日志。

## 脚本示例

**query.bot**

```javascript
//提问时触发此脚本,将提问外发
console.log('推送http://192.168.1.178:8080/ask?query='+query);
var xhr = new XMLHttpRequest()
xhr.open('GET', 'http://192.168.1.178:8080/ask?query='+query, false)
xhr.onreadystatechange = function() {
if (xhr.readyState == 4 && xhr.status == 200) {
    console.log("响应数据: ", xhr.responseText);
  }
};
xhr.send(query);
```

**task001.bot**

```javascript
//展示req/res使用

// 请求体数据
const method = req.method();
const url = req.url();
const headers = req.headers();
const body = req.body();
const params = req.params();
const query = req.query();

// 构造响应
res.set('X-Custom-Header', 'xiaobot-api');
res.status(200);
res.json({
	code: 0,
	message: "success",
	requestInfo: {
		method: method,
		url: url,
		routeParams: params,
		queryParams: query,
		requestBody: body || "no body"
	}
});
```