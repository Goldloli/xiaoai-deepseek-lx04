// 格式化消息文本
function formatMessage(text) {
    if (!text) return '';
    
    // 处理标题和换行
    let lines = text.split('\n');
    let formattedLines = lines.map(line => {
        // 处理标题（**文本**）
        line = line.replace(/\*\*(.*?)\*\*/g, '<span class="bold-text">$1</span>');
        return line;
    });
    
    // 将 ### 替换为换行，并确保每个部分都是一个段落
    let processedText = formattedLines.join('\n');
    let sections = processedText
        .split('###')
        .filter(section => section.trim())
        .map(section => {
            // 移除多余的换行和空格
            let lines = section.split('\n').filter(line => line.trim());
            
            if (lines.length === 0) return '';
            
            // 处理每个部分
            let result = '';
            let currentIndex = 0;
            
            while (currentIndex < lines.length) {
                let line = lines[currentIndex].trim();
                
                // 如果是数字开头（如 "1.")
                if (/^\d+\./.test(line)) {
                    result += `<p class="section-title">${line}</p>`;
                }
                // 如果是小标题（以破折号开头）
                else if (line.startsWith('-')) {
                    result += `<p class="subsection"><span class="bold-text">${line.replace(/^-/, '').trim()}</span></p>`;
                }
                // 如果是正文（包含冒号的行）
                else if (line.includes(':')) {
                    let [subtitle, content] = line.split(':').map(part => part.trim());
                    result += `<p><span class="subtitle">${subtitle}</span>: ${content}</p>`;
                }
                // 普通文本
                else {
                    result += `<p>${line}</p>`;
                }
                currentIndex++;
            }
            return result;
        });
    
    return sections.join('');
}

// 显示消息
function displayMessage(role, message) {
    const messagesContainer = document.getElementById('messages');
    const messageElement = document.createElement('div');
    messageElement.className = `message ${role}`;
    
    const avatar = document.createElement('img');
    avatar.src = role === 'user' ? 'user-avatar.png' : 'bot-avatar.png';
    avatar.alt = role === 'user' ? 'User' : 'Bot';

    const messageContent = document.createElement('div');
    messageContent.className = 'message-content';
    
    // 用户消息直接显示，机器人消息需要格式化
    messageContent.innerHTML = role === 'user' ? message : formatMessage(message);

    messageElement.appendChild(avatar);
    messageElement.appendChild(messageContent);
    messagesContainer.appendChild(messageElement);
    
    // 平滑滚动到底部
    messageElement.scrollIntoView({ behavior: 'smooth' });
}

function sendMessage() {
    const inputElement = document.getElementById('chat-input');
    const message = inputElement.value;
    if (!message.trim()) return;

    displayMessage('user', message);
    inputElement.value = '';

    // 显示加载动画
    const loadingElement = document.getElementById('loading');
    if (loadingElement) {
        loadingElement.style.display = 'block';
    }

	// 获取复选框状态
    const playQuery = document.getElementById('playQuery').checked;
    const playAnswer = document.getElementById('playAnswer').checked;

    const endpoint = '/chat';

    const payload = {
        message: message,
        playQuery: playQuery,
        playAnswer: playAnswer,
        stream: !playQuery && !playAnswer // 当两个播放选项都关闭时启用流式
    };

	if (payload.stream) {
		// 创建FormData以支持流式传输
	    //const formData = new FormData();
	    //Object.entries(payload).forEach(([key, value]) => {
	    //    formData.append(key, value);
	    //});
	
	    // 初始化机器人消息元素，用于流式拼接内容
	    const messagesContainer = document.getElementById('messages');
	    const botMessageElement = document.createElement('div');
	    botMessageElement.className = 'message bot';
	    
	    const avatar = document.createElement('img');
	    avatar.src = 'bot-avatar.png';
	    avatar.alt = 'Bot';
	
	    const messageContent = document.createElement('div');
	    messageContent.className = 'message-content';
	    messageContent.innerHTML = '';  // 初始为空
	
	    botMessageElement.appendChild(avatar);
	    botMessageElement.appendChild(messageContent);
	    messagesContainer.appendChild(botMessageElement);
	
	    // 发起流式请求
	    fetch(endpoint, {
	        method: 'POST',
	        headers: {
	            'Content-Type': 'application/json'
	        },
	        body: JSON.stringify(payload)
	    }).then(response => {
	        if (!response.ok) {
	            throw new Error('Network response was not ok');
	        }
	        
	        // 隐藏加载动画
	        if (loadingElement) {
	            loadingElement.style.display = 'none';
	        }
	
            // 检查响应类型
            const contentType = response.headers.get('content-type');
            const isJsonResponse = contentType && contentType.includes('application/json');
            // 后端返回了JSON，按非流式处理
            if (isJsonResponse) {
                return response.json().then(data => {
                    //loadingElement.style.display = 'none';
                    messageContent.innerHTML = formatMessage(data.message || data);
                    // 平滑滚动到底部
    				botMessageElement.scrollIntoView({ behavior: 'smooth' });
                });
            }

            const reader = response.body.getReader();
            const decoder = new TextDecoder();
            let accumulatedContent = '';

            // 递归读取流数据
            function readStream() {
                reader.read().then(({ done, value }) => {
                    if (done) {
                        // 流结束，最终格式化
                        messageContent.innerHTML = formatMessage(accumulatedContent);
                        // 平滑滚动到底部
    					botMessageElement.scrollIntoView({ behavior: 'smooth' });
					    return;
                    }

                    const chunk = decoder.decode(value, { stream: true });
			        // 按行分割SSE消息，处理可能的多行数据
			        const lines = chunk.split('\n');
			        lines.forEach(line => {
			            //line = line.trim(); 删掉这行！保留原始行内容
			            // 只处理以data:开头的行
			            if (line.startsWith('data:')) {
			                // 提取data:后面的内容
			                const data = line.slice(5);
			                if (data.trim() !== '[DONE]') {
								accumulatedContent += data;
			                    messageContent.innerHTML = formatMessage(accumulatedContent);
			                    // 平滑滚动到底部
			    				botMessageElement.scrollIntoView({ behavior: 'smooth' });
			                }
			            } else if (line) {
                                // 处理非SSE格式的流式数据
                                accumulatedContent += line;
                                messageContent.innerHTML = formatMessage(accumulatedContent);
			                    // 平滑滚动到底部
			    				botMessageElement.scrollIntoView({ behavior: 'smooth' });
                        }
			        });

                    readStream(); // 继续读取下一块
                }).catch(error => {
                    console.error('Stream error:', error);
                    messageContent.innerHTML = '流式传输出错';
                });
            }

            readStream(); // 启动流读取
	    }).catch(error => {
	        // 隐藏加载动画
	        if (loadingElement) {
	            loadingElement.style.display = 'none';
	        }
	
	        // 更新错误消息
	        botMessageElement.querySelector('.message-content').innerHTML = '出错了，请稍后再试。';
	        console.error('Error:', error);
	    });		
	} else {
		fetch(endpoint, {
	        method: 'POST',
	        headers: {
	            'Content-Type': 'application/json'
	        },
	        body: JSON.stringify(payload)
	    })
	    .then(response => response.json())
	    .then(data => {
	        // 隐藏加载动画
	        if (loadingElement) {
	            loadingElement.style.display = 'none';
	        }
	
	        if (data.message ) {
	            displayMessage('bot', data.message);
	        } else {
	            displayMessage('bot', '出错了，请稍后再试！');
	        }
	    })
	    .catch(error => {
	        // 隐藏加载动画
	        if (loadingElement) {
	            loadingElement.style.display = 'none';
	        }
	
	        displayMessage('bot', '出错了，请稍后再试。');
	        console.error('Error:', error);
	    });
	}    
}

// 添加主题切换功能
function toggleTheme() {
    document.body.classList.toggle('dark-mode');
    const chatContainer = document.querySelector('.chat-container');
    const messages = document.querySelector('.messages');
    
    // 同时切换容器的深色模式
    chatContainer.classList.toggle('dark-mode');
    messages.classList.toggle('dark-mode');
    
    // 保存主题设置
    const isDarkMode = document.body.classList.contains('dark-mode');
    localStorage.setItem('darkMode', isDarkMode);
}

// 添加下拉菜单功能
function toggleDropdown(event) {
    event.preventDefault();
    document.getElementById('dropdownMore').classList.toggle('show');
}

// 点击其他地方关闭下拉菜单
window.onclick = function(event) {
    if (!event.target.matches('.dropdown button')) {
        const dropdowns = document.getElementsByClassName('dropdown-content');
        for (const dropdown of dropdowns) {
            if (dropdown.classList.contains('show')) {
                dropdown.classList.remove('show');
            }
        }
    }
}

// 添加回车发送功能
document.getElementById('chat-input').addEventListener('keypress', function(event) {
    if (event.key === 'Enter' && !event.shiftKey) {
        event.preventDefault();
        sendMessage();
    }
});

// 页面加载时
document.addEventListener('DOMContentLoaded', () => {
	// 页面加载时检查主题设置
    const isDarkMode = localStorage.getItem('darkMode') === 'true';
    if (isDarkMode) {
        document.body.classList.add('dark-mode');
        document.querySelector('.chat-container').classList.add('dark-mode');
        document.querySelector('.messages').classList.add('dark-mode');
    }
    
    // 状态保存功能,为了保持页面刷新后复选框状态不变
    const playQuery = localStorage.getItem('playQuery') == 'true'; // 默认false
    const playAnswer = localStorage.getItem('playAnswer') == 'true'; // 默认false
    
    document.getElementById('playQuery').checked = playQuery;
    document.getElementById('playAnswer').checked = playAnswer;

    // 为复选框添加change事件监听器，保存状态
    document.getElementById('playQuery').addEventListener('change', function() {
        localStorage.setItem('playQuery', this.checked);
    });
    
    document.getElementById('playAnswer').addEventListener('change', function() {
        localStorage.setItem('playAnswer', this.checked);
    });
    
    
    // 菜单元素
    const menuBtn = document.getElementById('menuBtn');
    const dropdownMenu = document.getElementById('dropdownMenu');
    
    // 菜单状态
    let menuOpen = false;
    
    // 点击菜单按钮切换菜单显示状态
    menuBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        menuOpen = !menuOpen;
        
        if (menuOpen) {
            dropdownMenu.classList.add('visible');
        } else {
            dropdownMenu.classList.remove('visible');
        }
    });
    
    // 点击页面其他地方关闭菜单
    document.addEventListener('click', () => {
        if (menuOpen) {
            menuOpen = false;
            dropdownMenu.classList.remove('visible');
        }
    });
    
    // 阻止菜单内部点击事件冒泡
    dropdownMenu.addEventListener('click', (e) => {
        e.stopPropagation();
        // 点击菜单项后关闭菜单
        menuOpen = false;
        dropdownMenu.classList.remove('visible');
    });
    
});

