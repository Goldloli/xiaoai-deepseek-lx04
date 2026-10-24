// ========== 封装音频播放器对象 ==========
const audioPlayer = {
  
  // 播放方法
  async play() {
    try {
		const response = await fetch(`/player/play`);
		if (!response.ok) {
		  const errText = await response.text();
		  console.error('播放接口返回错误:', errText);
		  return false;
		}
		return true;
    } catch (error) {
        console.error('播放失败:', error);
	    return false;
    }
  },
  
  // 暂停方法
  async pause() {
    try {
		const response = await fetch(`/player/stop`);
		if (!response.ok) {
		  const errText = await response.text();
		  console.error('暂停接口返回错误:', errText);
		  return false;
		}
		return true;
    } catch (error) {
        console.error('Stop出错:', error);
    	return false;
    }
  },
  
  async skip(value) {
    try {
		const response = await fetch(`/player/skip?value=${value}`);
		if (!response.ok) {
		  const errText = await response.text();
		  console.error('skip接口返回错误:', errText);
		  return false;
		}
		return true;
    } catch (error) {
        console.error('skip出错:', error);
	    return false;
    }
  },

  // 设置音频源（模拟 src 属性赋值）
  async set_src(value) {
    
  },

  // 
  async set_playmode(value) {
    try {
		const response = await fetch(`/player/playMode?mode=${value}`);
		if (!response.ok) {
		  const errText = await response.text();
		  console.error('播放模式接口返回错误:', errText);
		  return false;
		}
		return true;
    } catch (error) {
        console.error('设置播放模式出错:', error);
	    return false;
    }
  },
  
  // 设置音量（模拟 volume 属性赋值）
  async set_volume(value) {
    try {
		const response = await fetch(`/player/set_volume?value=${value}`);
		if (!response.ok) {
		  const errText = await response.text();
		  console.error('音量接口返回错误:', errText);
		  return false;
		}
		const data = await response.json();
		return (value-data.volume===0);
    } catch (error) {
        console.error('设置音量出错:', error);
	    return false;
    }
  }

};


//const audioPlayer = document.getElementById('audioPlayer');
const playBtn = document.getElementById('playBtn');
const stopBtn = document.getElementById('stopBtn');
const currentSongName = document.getElementById('currentSongName');

const volumeSlider = document.getElementById('volumeSlider');
const volumeValue = document.getElementById('volumeValue');
const searchInput = document.getElementById('searchInput');
const breadcrumb = document.getElementById('breadcrumb');
const musicList = document.getElementById('musicList');

let currentMusicDir = 'music';
let currentMusicFiles = [];
let filteredMusicFiles = [];
let currentPlayingIndex = -1;
let isPlaying = false;
let playMode = 'single';
let isSequencePlaying = false;
let isRandomPlaying = false;
let volume = 50;

let ws = null;
let wsReconnectAttempts = 0;
const MAX_WS_RECONNECT_ATTEMPTS = 2;

function goBack() {
    window.location.href = 'index.html';
}

function connectWebSocket() {
    try {
        // 构建WebSocket URL (基于当前页面的协议和主机)
        const protocol = 'ws:';
        const host = window.location.host;
        const wsUrl = `${protocol}//${host}/ws`;
        
        console.log('connectWebSocket:', wsUrl)
        ws = new WebSocket(wsUrl);
        
        ws.onopen = function() {
            console.log('WebSocket连接成功');
            wsReconnectAttempts = 0;
        };
        
        ws.onmessage = function(event) {
            handleWebSocketMessage(event.data);
        };
        
        ws.onclose = function() {
            console.log('WebSocket连接关闭');
            attemptReconnect();
        };
        
        ws.onerror = function(error) {
            console.error('WebSocket错误:', error);
        };
        
    } catch (error) {
        console.error('WebSocket连接失败:', error);
        attemptReconnect();
    }
}

function attemptReconnect() {
    if (wsReconnectAttempts < MAX_WS_RECONNECT_ATTEMPTS) {
        wsReconnectAttempts++;
        const delay = Math.pow(2, wsReconnectAttempts) * 1000; // 指数退避
        console.log(`尝试重新连接WebSocket (${wsReconnectAttempts}/${MAX_WS_RECONNECT_ATTEMPTS})...`);
        setTimeout(connectWebSocket, delay);
    } else {
        console.error('WebSocket重连失败，已达到最大尝试次数');
    }
}

function handleWebSocketMessage(data) {
    try {
        /*const message = JSON.parse(data);
        if (message.type === 'current_song') {
            updateCurrentSongName(message.data);
        }*/
        if (data==='-end-'){
		    if (isSequencePlaying) {
		        toggleSequencePlay();
		    }
		    if (isRandomPlaying) {
		        toggleRandomPlay();
		    }
        } else {
        	updateCurrentSongName(data);
        }
    } catch (error) {
        console.error('处理WebSocket消息失败:', error);
    }
}

function updateCurrentSongName(songName) {
    currentSongName.textContent = songName || '未选择音乐';
}

async function playMusic() {
    if (currentPlayingIndex >= 0 && currentPlayingIndex < filteredMusicFiles.length) {
        const isSuccess = await audioPlayer.play();
        if (!isSuccess) return;
        
        isPlaying = true;
    }
}

/*async function pauseMusic() {
    const isSuccess = await audioPlayer.pause();
    if (!isSuccess) return;
    
    isPlaying = false;
}*/

async function stopMusic() {
    const isSuccess = await audioPlayer.pause();
    if (!isSuccess) return;
    
    isPlaying = false;
    
    if (isSequencePlaying) {
        toggleSequencePlay();
    }
    if (isRandomPlaying) {
        toggleRandomPlay();
    }
}

async function setPlayMode(value) {
	const isSuccess = await audioPlayer.set_playmode(value);
	if (isSuccess) {
	    playMode = value;
	    console.log('播放模式已设置为:', value);
	}
}

async function setVolume(value) {
	const isSuccess = await audioPlayer.set_volume(value);
    if (isSuccess) {
    	volumeValue.textContent = `${value}%`;
		volume = value;
    } else {
	    volumeValue.textContent = `${volume}%`;
		volumeSlider.value = volume;
    }
}

function toggleSequencePlay() {
    isSequencePlaying = !isSequencePlaying;
    const randomBtn = document.getElementById('randomBtn');
    const sequenceBtn = document.getElementById('sequenceBtn');
    
    if (isSequencePlaying) {
       // 禁用随机按钮，启用顺序按钮
        randomBtn.disabled = true;
        sequenceBtn.disabled = false;
        sequenceBtn.classList.add('active');
        randomBtn.classList.remove('active');
        disableMusicList();
        
        currentPlayingIndex = 0;
	    sendPlayStateToBackend();
    } else {
        // 启用所有按钮
        randomBtn.disabled = false;
        sequenceBtn.disabled = false;
        sequenceBtn.classList.remove('active');
        enableMusicList();
	    sendPlayStateToBackend(0);
    }
    
    if (isSequencePlaying && currentPlayingIndex >= 0) {
        playMusic();//loadMusic(currentPlayingIndex);
    }
}

function toggleRandomPlay() {
    isRandomPlaying = !isRandomPlaying;
    const randomBtn = document.getElementById('randomBtn');
    const sequenceBtn = document.getElementById('sequenceBtn');
    
    if (isRandomPlaying) {
        // 禁用顺序按钮，启用随机按钮
        sequenceBtn.disabled = true;
        randomBtn.disabled = false;
        randomBtn.classList.add('active');
        sequenceBtn.classList.remove('active');
        disableMusicList();
        
        currentPlayingIndex = generateRandomPlayIndex();
	    sendPlayStateToBackend();
    } else {
        // 启用所有按钮
        sequenceBtn.disabled = false;
        randomBtn.disabled = false;
        randomBtn.classList.remove('active');
        enableMusicList();
	    sendPlayStateToBackend(0);
    }
    
    if (isRandomPlaying && currentPlayingIndex >= 0) {
        playMusic();//loadMusic(currentPlayingIndex);
    }
}

function generateRandomPlayIndex() {
    return Math.floor(Math.random() * filteredMusicFiles.length);
}

function disableMusicList() {
    searchInput.disabled = true;
    musicList.classList.add('disabled');

    // 添加禁用样式类（核心）
    breadcrumb.classList.add('disabled');
    // 额外保险：移除所有子元素的 onclick 事件（防止 pointer-events 失效时触发）
    const items = breadcrumb.querySelectorAll('.breadcrumb-item');
    items.forEach(item => {
        // 先保存原有事件，方便后续恢复
        item.dataset.originalOnclick = item.onclick;
        item.onclick = null;
    });
}

function enableMusicList() {
    searchInput.disabled = false;
    musicList.classList.remove('disabled');
    
    // 移除禁用样式类
    breadcrumb.classList.remove('disabled');
    // 恢复原有 onclick 事件
    const items = breadcrumb.querySelectorAll('.breadcrumb-item');
    items.forEach(item => {
        if (item.dataset.originalOnclick) {
            // 还原点击事件（根据原有字符串恢复）
            const originalFunc = item.dataset.originalOnclick;
            if (originalFunc.includes('loadMusicDirectory')) {
                // 解析原有参数（比如 'history' 或 'xxx/xxx'）
                const match = originalFunc.match(/loadMusicDirectory\('(.*?)'\)/);
                if (match) {
                    const path = match[1];
                    item.onclick = () => loadMusicDirectory(path);
                }
            }
            // 清除临时存储的事件
            delete item.dataset.originalOnclick;
        }
    });
}

// 0=空 1=当前(1条) null=全部 
function sendPlayStateToBackend(shouldSend = null) {
    const playState = {
        filteredMusicFiles: [],
        currentPlayingIndex: currentPlayingIndex,
        playMode: playMode,
        isSequencePlaying: isSequencePlaying,
        isRandomPlaying: isRandomPlaying,
        currentMusicDir: currentMusicDir
    };
	if (!shouldSend) {
        playState.filteredMusicFiles = filteredMusicFiles.map(f => ({
            name: f.name,
            path: f.path,
            isDir: f.isDir || false
        }));
	} else if (shouldSend===1){
	    playState.filteredMusicFiles = (
	        currentPlayingIndex >= 0 && currentPlayingIndex < filteredMusicFiles.length 
	            ? [filteredMusicFiles[currentPlayingIndex]]  // 包装成单元素数组
	            : []  // 索引无效时返回空数组
	    ).map(f => ({
	        name: f.name,
	        path: f.path,
	        isDir: f.isDir || false  // 保留默认值逻辑
	    }));
		playState.currentPlayingIndex = 0;
	}
	
    fetch('/player/src', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(playState)
    })
    .then(response => response.json())
    .then(data => {
        console.log('播放状态已发送:', data);
    })
    .catch(error => {
        console.error('发送播放状态到后端失败:', error);
    });
}

async function skipMusic(value) {
	const isSuccess = await audioPlayer.skip(value);
	if (isSuccess) {
	    console.log('skip:', value);
	}
}

function playPrevious() {
    if (filteredMusicFiles.length === 0) return;
    
    if (isSequencePlaying || isRandomPlaying) {
    	skipMusic(-1)
    } else {
        currentPlayingIndex = (currentPlayingIndex - 1 + filteredMusicFiles.length) % filteredMusicFiles.length;
      	loadMusic(currentPlayingIndex);
    }
}

function playNext() {
    if (filteredMusicFiles.length === 0) return;
    
    if (isSequencePlaying || isRandomPlaying) {
    	skipMusic(+1)
    } else {
	    if (isRandomPlaying) {
	        currentPlayingIndex = generateRandomPlayIndex();
	        if (currentPlayingIndex >= filteredMusicFiles.length) {
	            currentPlayingIndex = 0;
	        }
	    } else if (isSequencePlaying) {
	        currentPlayingIndex++;
	        if (currentPlayingIndex >= filteredMusicFiles.length) {
	            currentPlayingIndex = 0;
	        }
	    } else {
	        currentPlayingIndex = (currentPlayingIndex + 1) % filteredMusicFiles.length;
	    }
    
      	loadMusic(currentPlayingIndex);
    }
}

function loadMusic(index) {
    if (index < 0 || index >= filteredMusicFiles.length) return;
    
    const music = filteredMusicFiles[index];
    if (music.isDir) return;
    
    currentPlayingIndex = index;
    
    //currentSongName.textContent = music.name;
    sendPlayStateToBackend(1) //audioPlayer.src = `${currentMusicDir}/${music.name}`;

    
    document.querySelectorAll('.music-item').forEach((item, i) => {
        item.classList.toggle('active', i === index);
    });
    
    playMusic();
}

function loadMusicDirectory(dir,filter) {
	if (dir!='history' && dir!='favorite') {
	  	currentMusicDir = dir.replace(/\/$/, '');
	}
    let filterStr = '';
    if (filter) filterStr = encodeURIComponent(filter);
     
    fetch(`/music/list?dir=${encodeURIComponent(dir)}&filter=${filterStr}`)
        .then(response => response.json())
        .then(data => {
            currentMusicFiles = data || [];
            filteredMusicFiles = [...currentMusicFiles];
            updateBreadcrumb(currentMusicDir);
            renderMusicList();
        })
        .catch(error => {
            console.error('加载音乐列表失败:', error);
            renderMockMusicList();
        });
        
}

function updateBreadcrumb(dir) {
    const parts = dir.split('/').filter(p => p);
    let html = '';
    let currentPath = '';
    
    html += `<span class="breadcrumb-item" onclick="loadMusicDirectory('history')">历史</span>`;
    html += `<span class="breadcrumb-separator">/</span>`;
    html += `<span class="breadcrumb-item" onclick="loadMusicDirectory('favorite')">收藏</span>`;
    
    parts.forEach((part, index) => {
        currentPath = currentPath ? `${currentPath}/${part}` : part;
        html += `<span class="breadcrumb-separator">/</span>`;
        html += `<span class="breadcrumb-item" onclick="loadMusicDirectory('${currentPath}')">${part}</span>`;
    });
    
    breadcrumb.innerHTML = html;
}

function renderMusicList() {
    if (filteredMusicFiles.length === 0) {
        const searchInput = document.getElementById('searchInput');
        if (searchInput && searchInput.value.trim()) {
            musicList.innerHTML = '<p style="text-align: center; color: #666;">未找到匹配的音乐</p>';
        } else {
            musicList.innerHTML = '<p style="text-align: center; color: #666;">该目录下没有音乐文件</p>';
        }
        return;
    }
    
    musicList.innerHTML = filteredMusicFiles.map((music, filteredIndex) => {
        const isActive = false;
        const folderClass = music.isDir ? ' folder' : '';
        const favoriteClass = music.isFav ? ' favorited' : '';
        return `
            <div class="music-item ${folderClass} ${isActive ? 'active' : ''}" onclick="${music.isDir ? `loadMusicDirectory('${currentMusicDir}/${music.name.replace(/\/$/, '')}')` : `loadMusic(${filteredIndex})`}">
                <div class="icon">
                    <i class="fa ${music.isDir ? 'fa-folder' : 'fa-music'}"></i>
                </div>
                <div class="info">
                    <h4>${music.name}</h4>
                </div>
                ${!music.isDir ? `<i class="fa ${music.isFav ? 'fa-star' : 'fa-star-o'} favorite-btn${favoriteClass}" onclick="toggleFavorite(${filteredIndex});"></i>` : ''}
                ${isActive && isPlaying ? '<i class="fa fa-volume-up" style="color: #007bff;"></i>' : ''}
            </div>
        `;
    }).join('');
}

function filterMusicList() {
    const searchInput = document.getElementById('searchInput');
    const searchTerm = searchInput.value.toLowerCase().trim();
    
    if (searchTerm === '') {
        filteredMusicFiles = [...currentMusicFiles];
    } else {
        filteredMusicFiles = currentMusicFiles.filter(file => 
            file.name.toLowerCase().includes(searchTerm)
        );
    }
    
    renderMusicList();
}

function toggleFavorite(index) {
    if (index < 0 || index >= filteredMusicFiles.length) return;
    
    const music = filteredMusicFiles[index];
    if (music.isDir) return;
    
    music.isFav = (!music.isFav);
    //save
    try {
	    fetch('/player/favorited', {
	        method: 'POST',
	        headers: {
	            'Content-Type': 'application/json'
	        },
	        body: JSON.stringify(music)
	    })
	    .then(response => response.json())
	    .then(data => {
	        console.log('收藏状态已发送:', data);
	    })
	    .catch(error => {
	        console.error('发送收藏状态到后端失败:', error);
	    });
    } catch (error) {
        console.error('保存收藏音乐失败:', error);
    }

    renderMusicList();
}

//function getRealIndex(filteredIndex) {
//    if (filteredIndex < 0 || filteredIndex >= filteredMusicFiles.length) {
//        return -1;
//    }
//    const filteredFile = filteredMusicFiles[filteredIndex];
//    return currentMusicFiles.findIndex(file => file.name === filteredFile.name && file.path === filteredFile.path);
//}

// 添加搜索框回车事件
if (searchInput) {
    searchInput.addEventListener('keypress', function(event) {
        if (event.key === 'Enter') {
		    const searchInput = document.getElementById('searchInput');
		    const searchTerm = searchInput.value.toLowerCase().trim();
            loadMusicDirectory(currentMusicDir, searchTerm);
        }
    });
}

fetch(`/player/last`)
    .then(response => response.json())
    .then(data => {
        filteredMusicFiles = data.filteredMusicFiles || [];
        currentMusicFiles = [...filteredMusicFiles];
		currentPlayingIndex = data.currentPlayingIndex;
		playMode = data.playMode;
		isSequencePlaying = data.isSequencePlaying;
		isRandomPlaying = data.isRandomPlaying;
		currentMusicDir = data.currentMusicDir;

        updateBreadcrumb(currentMusicDir);
        renderMusicList();
        /*if (currentPlayingIndex >-1 && currentPlayingIndex < filteredMusicFiles.length){
        	updateCurrentSongName(filteredMusicFiles[currentPlayingIndex].name)
        }*/
        updateCurrentSongName(filteredMusicFiles[currentPlayingIndex]?.name)
        console.log('加载last音乐列表ok.');
    })
    .catch(error => {
        console.error('加载音乐列表失败:', error);
    })
	.finally(() => {
        // 无论成功/失败都会执行
		if (currentMusicFiles.length === 0) {
			loadMusicDirectory('music');
		} else if (currentMusicFiles.length === 1) {
			loadMusicDirectory(currentMusicDir);
		}

		// 获取所有播放模式选项
	    const modeOptions = document.querySelectorAll('input[name="playMode"]');
	    // 遍历选项设置选中状态
	    modeOptions.forEach(option => {
	        if (option.value === playMode) option.checked = true;
	    });
    	
		if (isSequencePlaying) {
			const sequenceBtn = document.getElementById("sequenceBtn");
  			sequenceBtn.classList.add("active"); // 切换
			const randomBtn = document.getElementById('randomBtn');
			randomBtn.disabled = true;
			disableMusicList();
		}
		if (isRandomPlaying){
			const randomBtn = document.getElementById('randomBtn');
  			randomBtn.classList.add("active"); // 切换
			const sequenceBtn = document.getElementById("sequenceBtn");
			sequenceBtn.disabled = true;
  			disableMusicList();
		}
	
    });    
    
//setVolume(volumeSlider.value);
fetch(`/player/get_volume`)
    .then(response => response.json())
    .then(data => {
        value = data.volume;
        if (value > -1) {
        	volumeValue.textContent = `${value}%`;
			volumeSlider.value = value;
			volume = value;
	        console.log('加载音箱音量ok.', value);
        } 
    })
    .catch(error => {
        console.error('加载音箱音量失败:', error);
    })

// 初始化WebSocket连接
connectWebSocket();