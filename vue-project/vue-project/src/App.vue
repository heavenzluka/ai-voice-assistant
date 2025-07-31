<template>
  <div class="container">
     <!-- 录音控制 -->
    <button id="startBtn" :disabled="startBtnDisabled" @click="startRecording">开始通话</button>

    <button
    id="togglePauseBtn"
    :disabled="togglePauseBtnDisabled"
    @click="togglePause"
    :class="{ paused: isPaused }"
  >
    {{ isPaused ? '▶️ 继续说话' : '⏸️ 暂停说话' }}
  </button>

    <button id="stopBtn" :disabled="stopBtnDisabled" @click="stopRecording">挂断电话</button>
    <!-- <button id="exportBtn" @click="exportPCM">导出 PCM</button> -->
    <!-- 由于收音过于灵敏,静音检测的方式可能难以面对无耳机使用/嘈杂环境,所以可以直接点击发送按钮发送语言给llm -->
    <button @click="sendGoCommand">发送</button>

    <button @click="clearData">清除数据</button>

<!-- 地理信息相关 -->
    <div class="location-control">
      <button
        id="getLocationBtn"
        @click="getLocation"
        :disabled="isGettingLocation" 
      ><!-- 防止重复点击 -->
        {{ hasLocationPermission ? '🔄 刷新位置' : (locationInfo ? '📍 位置已获取' : '📍 获取我的位置') }}
      </button>
      <!-- 显示位置状态或错误信息 -->
      <div v-if="locationInfo" class="location-info success">
        位置: {{ locationInfo.latitude.toFixed(6) }}, {{ locationInfo.longitude.toFixed(6) }} (精度: {{ locationInfo.accuracy.toFixed(1) }}m)
      </div>
      <div v-else-if="locationError" class="location-info error">
        位置错误: {{ locationError }}
      </div>
      <div v-else-if="isGettingLocation" class="location-info">
        正在获取位置...
      </div>
      <!-- <div v-else class="location-info hint">
        点击上方按钮获取位置信息，可用于提供本地化服务。
      </div> -->
    </div>

<!-- 角色设定区 -->
    <div class="role-config">
      <h3>🤖 AI 角色设定</h3>
      <div>
        <label for="role-system">bot身份设定:</label>
        <input type="text" id="role-system" v-model="roleSystem" placeholder="例: 你是一个一个猫娘" />
      </div>
      <div>
        <label for="role-user-design">bot初始设定:(语气口癖等)</label>
        <input type="text" id="role-user-design" v-model="roleUserDesign" placeholder='例: 请在结尾加"喵~"' />
      </div>
<div class="voice-control">
  <button id="voice-up" @click="sendCommand('up')">+</button>
  <button id="voice-down" @click="sendCommand('down')">-</button>
  <button id="voice-fast" @click="sendCommand('fast')">++</button>
  <button id="voice-late" @click="sendCommand('late')">--</button>
</div>

      <p><small>💡 修改后需重新开始录音才会生效</small></p>
    </div>

     <!-- 状态信息 -->
    <div id="status">{{ status }}</div>
    <div id="pcm">PCM数据（前10个采样点）: [{{ pcmDataDisplay }}]</div>
    <div id="result"><strong>识别结果：</strong><pre>{{ result }}</pre></div>
    <div id="answer"><strong>LLM 回复：</strong><pre>{{ answer }}</pre></div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue';

const startBtnDisabled = ref(false);
const stopBtnDisabled = ref(true);
const togglePauseBtnDisabled = ref(true); // 控制“暂停/继续”按钮是否可用
const status = ref('等待开始');
const pcmDataDisplay = ref('');
const result = ref('等待语音识别结果: \n');
const answer = ref('等待大模型回复: \n');
// 内网穿透
const url = '8cb8d4b3957f.ngrok-free.app';
const wsUrl = 'wss://' + url + '/asr-stream';

// 音频相关
let socket = null;
let isRecording = ref(false);
let audioContext = null;
let mediaStream = null;
let mediaStreamSource = null;
let processor = null;
let pcmChunks = [];

// 地理位置相关响应式变量
const locationInfo = ref(null); // 存储获取到的位置信息 { latitude, longitude, accuracy }
const locationName = ref(''); // 对经纬度进行反编码获得具体地理位置名
const locationError = ref(null); // 存储获取位置时的错误信息
const hasLocationPermission = ref(false); // 记录用户是否授予了位置权限
const isGettingLocation = ref(false)// 防止重复点击

// 角色设定
let roleSystem = ref('你是一只名为米雪儿的猫娘');
let roleUserDesign = ref('请在每句话结尾加上"喵",称呼我为"主人",自称为"唐猫"');

// 用于累积 ASR 识别结果
let currentAsrText = '';

// 暂停状态
let isPaused = false;


// 获取角色设定
function getRoleDesign() {
  var baseData = {
    Type: "init",
    System: roleSystem.value.trim(),
    User: roleUserDesign.value.trim()
  };
}


// 清除数据（刷新页面）
function clearData() {
   locationInfo.value = null; 
  locationError.value = null; 
  hasLocationPermission.value = false;
  isGettingLocation.value = false;
  location.reload();
}

// 连接 WebSocket
function connectWebSocket() {
  if (socket) {
    socket.close();
  }

  socket = new WebSocket(wsUrl);

  socket.onopen = () => {
    status.value = "已连接到语音识别服务";
    console.log('WebSocket已连接');
    const initRoleData = getRoleDesign();
    socket.send(JSON.stringify(initRoleData));
    console.log('已发送角色初始化信息:', initRoleData);
  };

  socket.onmessage = (event) => {
    if (typeof event.data === "string") {
      try {
        const data = JSON.parse(event.data);
        if (data.asrReturn !== undefined) {
          // 累积识别结果
          result.value += data.asrReturn + '\n';
          currentAsrText += data.asrReturn; // 累加，用于发送
        }
        if (data.answer !== undefined) {
          answer.value += data.answer + '\n';
        }
      } catch (e) {
        console.error('JSON 解析失败:', e, '原始数据:', event.data);
      }
    } else if (event.data instanceof Blob || event.data instanceof ArrayBuffer) {
      const url = URL.createObjectURL(event.data);
      const audio = new Audio(url);
      audio.play().catch(e => {
        console.error('播放音频失败:', e);
        URL.revokeObjectURL(url);
      });
    } else {
      console.warn('未知类型消息:', typeof event.data, event.data);
    }
  };

  socket.onclose = () => {
    if (isRecording.value) {
      status.value = "连接断开，正在尝试重新连接...";
      setTimeout(connectWebSocket, 1000);
    }
  };

  socket.onerror = (error) => {
    console.error('WebSocket error:', error);
    status.value = "连接错误: " + error.message;
  };
}

// 发送tts控制命令
function sendCommand(command) {
  if (socket && socket.readyState === WebSocket.OPEN) {
    const message = { type: command };
    socket.send(JSON.stringify(message));
    console.log(`已发送命令: ${command}`);
  } else {
    console.error('WebSocket未连接');
  }
}

// 开始录音
async function startRecording() {
  try {
    status.value = "正在获取麦克风权限...";
    startBtnDisabled.value = true;

    mediaStream = await navigator.mediaDevices.getUserMedia({ audio: true });

    audioContext = new (window.AudioContext || window.webkitAudioContext)({
      sampleRate: 16000,
    });

    console.log('当前采样率:', audioContext.sampleRate);

    mediaStreamSource = audioContext.createMediaStreamSource(mediaStream);
    processor = audioContext.createScriptProcessor(1024, 1, 1);

    // 绑定音频处理
    processor.onaudioprocess = (event) => {
      if (isPaused || !isRecording.value || !socket || socket.readyState !== WebSocket.OPEN) {
        return;
      }

      const inputData = event.inputBuffer.getChannelData(0);
      const pcmBuffer = convertFloat32ToInt16(inputData);
      pcmChunks.push(pcmBuffer);

      const displayData = Array.from(pcmBuffer.slice(0, 10)).join(', ');
      pcmDataDisplay.value = displayData + '...';

      socket.send(pcmBuffer.buffer);
    };

    mediaStreamSource.connect(processor);
    processor.connect(audioContext.destination);

    isRecording.value = true;
    stopBtnDisabled.value = false;
    togglePauseBtnDisabled.value = false; // 启用暂停/继续按钮
    isPaused = false;
    status.value = "正在录音...";

    connectWebSocket();

  } catch (error) {
    console.error('录音失败:', error);
    status.value = "录音失败: " + error.message;
    startBtnDisabled.value = false;
  }
}

// 暂停录音
function pauseRecording() {


  if (processor) {
    processor.onaudioprocess = null; // 停止发送音频
  }
  isPaused = true;

  status.value = "🎙️ 已暂停录音，连接保持中...";
}

// 继续录音
function resumeRecording() {
  if (!isRecording.value || !processor || !audioContext) return;

  // 重新绑定音频处理
  processor.onaudioprocess = (event) => {
    if (isPaused || !isRecording.value || !socket || socket.readyState !== WebSocket.OPEN) {
      return;
    }

    const inputData = event.inputBuffer.getChannelData(0);
    const pcmBuffer = convertFloat32ToInt16(inputData);
    pcmChunks.push(pcmBuffer);

    const displayData = Array.from(pcmBuffer.slice(0, 10)).join(', ');
    pcmDataDisplay.value = displayData + '...';

    socket.send(pcmBuffer.buffer);
  };

  isPaused = false;
  status.value = "🎤 正在录音...";
}

// 切换暂停/继续
function togglePause() {
  if (isPaused) {
    resumeRecording();
  } else {
    pauseRecording();
  }
}

// 手动发送当前识别内容
function sendGoCommand() {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    alert("WebSocket 连接未建立");
    return;
  }

  const userText = currentAsrText.trim() || "（无识别内容）";
  const goMsg = {
    type: "go",
    User: userText
  };

  socket.send(JSON.stringify(goMsg));
  console.log("已发送 go 指令:", goMsg);
  status.value = "已发送: " + userText;

  // 可选：发送后清空累积文本（防止重复发送）
  // currentAsrText = '';
}

// 停止录音（挂断）
function stopRecording() {
  const confirmed = window.confirm(
    "⚠️ 警告：您即将断开服务器连接！\n\n" +
    "此操作将立即终止与语音助手的会话连接。\n\n" +
    "断开后，当前对话上下文将丢失，\n" +
    "是否确认要挂断电话？"
  );

  if (!confirmed) {
    console.log("用户取消了断开操作");
    return;
  }

  isRecording.value = false;
  startBtnDisabled.value = false;
  stopBtnDisabled.value = true;
  togglePauseBtnDisabled.value = true; // 禁用暂停/继续按钮
  isPaused = false;
  status.value = "录音已停止";

  // 发送 hangup 消息
  if (socket && socket.readyState === WebSocket.OPEN) {
    try {
      socket.send(JSON.stringify({ type: "hangup" }));
      console.log('已发送 hangup 消息');
    } catch (error) {
      console.warn('发送 hangup 失败:', error);
    }
    socket.close();
    socket = null;
  } else if (socket) {
    socket.close();
    socket = null;
  }

  // 释放资源
  if (processor) {
    processor.onaudioprocess = null;
    processor.disconnect();
    processor = null;
  }

  if (mediaStreamSource) {
    mediaStreamSource.disconnect();
    mediaStreamSource = null;
  }

  if (audioContext && audioContext.state !== 'closed') {
    audioContext.close().then(() => {
      audioContext = null;
    });
  }

  if (mediaStream) {
    mediaStream.getTracks().forEach(track => track.stop());
    mediaStream = null;
  }

  clearData();
}

// PCM 转换
function convertFloat32ToInt16(float32Array) {
  const length = float32Array.length;
  const int16Array = new Int16Array(length);

  for (let i = 0; i < length; i++) {
    let s = float32Array[i];
    s = Math.max(-1.0, Math.min(1.0, s));
    int16Array[i] = s < 0 ? s * 0x8000 : s * 0x7FFF;
  }

  return int16Array;
}

async function getLocation() {
  //检查浏览器是否支持 Geolocation API
  if (!("geolocation" in navigator)) {
    status.value = "您的浏览器不支持地理位置功能。";
    locationError.value = "Geolocation not supported.";
    return;
  }

  status.value = "正在请求获取您的位置...";

  try {
    // 请求用户位置 
    const position = await new Promise((resolve, reject) => {
      navigator.geolocation.getCurrentPosition(resolve, reject, {
        enableHighAccuracy: true, // 尝试获取高精度位置
        timeout: 10000,           // 10秒超时
        maximumAge: 60000         // 缓存位置最长1分钟
      });
    });

    //  成功获取位置 
    locationInfo.value = {
      latitude: position.coords.latitude,
      longitude: position.coords.longitude,
      accuracy: position.coords.accuracy // 位置精度（米）
    };
    hasLocationPermission.value = true;
    status.value = `位置获取成功 (精度: ${position.coords.accuracy.toFixed(1)}m)`;
    console.log('获取到的地理位置:', locationInfo.value);
    // 等待发送给后端
    await sendLocationToBackend()

  } catch (error) {
    // 处理各种错误
    let errorMsg = "获取位置失败";
    switch(error.code) {
      case error.PERMISSION_DENIED:
        errorMsg = "用户拒绝了位置请求。";
        hasLocationPermission.value = false;
        break;
      case error.POSITION_UNAVAILABLE:
        errorMsg = "位置信息不可用。";
        break;
      case error.TIMEOUT:
        errorMsg = "获取位置超时。";
        break;
      default:
        errorMsg = "未知错误: " + error.message;
        break;
    }
    console.error('获取地理位置失败:', errorMsg);
    locationError.value = errorMsg;
    status.value = errorMsg;
  }



// 专门用于发送地理位置信息到后端的函数
async function sendLocationToBackend() {
  // 检查是否有有效的地理位置信息
  if (!locationInfo.value || locationInfo.value.latitude === undefined || locationInfo.value.longitude === undefined) {
    console.warn('无法发送位置信息：没有有效的 locationInfo');
    status.value = "没有可发送的位置信息。";
    return;
  }

  // 检查 WebSocket 连接是否就绪
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    console.warn('无法发送位置信息：WebSocket 连接未打开或不存在');
    status.value = "连接未建立，无法发送位置信息。";
    return;
  }

  const locationMessage = {
    Type: "updateLocation",
    Location: {
      latitude: locationInfo.value.latitude,
      longitude: locationInfo.value.longitude,
      accuracy: locationInfo.value.accuracy || 0 // 提供默认值
    }
  };

  try {
    // 发送消息
    await ws.send(JSON.stringify(locationMessage));
    console.log('位置信息已发送给后端:', locationMessage);
  } catch (error) {
    console.error('发送位置信息到后端失败:', error);
    status.value = "发送位置信息失败。";
  }
}
}

// 组件卸载前停止录音
onBeforeUnmount(() => {
  stopRecording();
});
</script>




<style scoped>
.container {
  max-width: 900px;
  margin: 0 auto;
  padding: 30px;
  font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
  background: #f7f9fc;
  min-height: 100vh;
  color: #333;
}

/* 按钮样式 */
button {
  padding: 12px 24px;
  font-size: 16px;
  border-radius: 12px;
  border: none;
  background: #6a5acd;
  color: white;
  cursor: pointer;
  margin-right: 10px;
  margin-bottom: 10px;
  transition: background 0.3s, transform 0.1s;
  box-shadow: 0 2px 6px rgba(106, 90, 205, 0.2);
}

button:hover:not(:disabled) {
  background: #5a4cbf;
}

button:active {
  transform: scale(0.98);
}

button:disabled {
  background: #cccccc;
  cursor: not-allowed;
  opacity: 0.6;
}

button:last-child {
  margin-right: 0;
}

/* 角色设定区域 */
.role-config {
  background: white;
  padding: 20px;
  border-radius: 16px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
  margin: 20px 0;
  border: 1px solid #e0e0e0;
}

.role-config h3 {
  margin: 0 0 15px 0;
  font-size: 18px;
  color: #555;
  display: flex;
  align-items: center;
  gap: 8px;
}

.role-config label {
  font-weight: 600;
  color: #444;
  margin-bottom: 6px;
  display: block;
  font-size: 14px;
}

.role-config input {
  width: 100%;
  padding: 10px 14px;
  border: 1px solid #ddd;
  border-radius: 8px;
  font-size: 15px;
  transition: border 0.3s;
  box-sizing: border-box;
}

.role-config input:focus {
  outline: none;
  border-color: #6a5acd;
  box-shadow: 0 0 0 2px rgba(106, 90, 205, 0.2);
}

.role-config p {
  margin: 10px 0 0 0;
  color: #777;
  font-size: 13px;
}

/* 状态显示区域 */
.status,
.pcm,
.result,
.answer {
  padding: 16px;
  border-radius: 12px;
  background: white;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  border: 1px solid #eaeaea;
  margin-bottom: 16px;
}

.status {
  background: #eef4ff;
  border-left: 5px solid #6a5acd;
}

.pcm {
  font-family: 'Courier New', monospace;
  background: #f0f0f0;
  font-size: 14px;
}

.result pre,
.answer pre {
  margin: 8px 0 0;
  padding: 10px;
  background: #f8f9ff;
  border-radius: 6px;
  border: 1px dashed #c5c5f1;
  font-size: 15px;
  line-height: 1.6;
  color: #2d2d2d;
  white-space: pre-wrap;
}

/* 响应式适配 */
@media (max-width: 768px) {
  .container {
    padding: 20px;
  }

  button {
    padding: 10px 18px;
    font-size: 15px;
  }

  .role-config input {
    font-size: 14px;
  }
}

/* 基本按钮样式 */
.voice-control {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 10px; /* 按钮之间的间距 */
  margin: 20px 0; /* 上下边距 */
}

.voice-control button {
  padding: 10px 20px; /* 内边距 */
  font-size: 16px; /* 字体大小 */
  border: none; /* 无边框 */
  border-radius: 5px; /* 圆角 */
  cursor: pointer; /* 鼠标悬停时显示手型光标 */
  transition: background-color 0.3s, transform 0.2s; /* 平滑过渡效果 */
}

/* 按钮颜色和悬停效果 */
#voice-up {
  background-color: #4CAF50; /* 绿色 */
  color: white;
}

#voice-up:hover {
  background-color: #45a049;
  transform: scale(1.05); /* 悬停时稍微放大 */
}

#voice-down {
  background-color: #f44336; /* 红色 */
  color: white;
}

#voice-down:hover {
  background-color: #da190b;
  transform: scale(1.05);
}

#voice-fast {
  background-color: #2196F3; /* 蓝色 */
  color: white;
}

#voice-fast:hover {
  background-color: #0b7dda;
  transform: scale(1.05);
}

#voice-late {
  background-color: #ff9800; /* 橙色 */
  color: white;
}

#voice-late:hover {
  background-color: #e68a00;
  transform: scale(1.05);
}

/* 按钮按下效果 */
.voice-control button:active {
  transform: scale(0.95); /* 按下时稍微缩小 */
}


.location-control {
  margin: 1rem 0;
  padding: 0.5rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  background-color: #f9f9f9;
}

.location-info {
  margin-top: 0.5rem;
  font-size: 0.9em;
  color: #555;
}

.location-info.success {
  color: #2e8b57;
}

.location-info.error {
  color: #d32f2f;
}

#getLocationBtn {
  padding: 0.5rem 1rem;
  font-size: 1rem;
  cursor: pointer;
  border: 1px solid #007bff;
  background-color: #007bff;
  color: white;
  border-radius: 4px;
}

#getLocationBtn:hover:not(:disabled) {
  background-color: #0056b3;
}

#getLocationBtn:disabled {
  background-color: #ccc;
  cursor: not-allowed;
  border-color: #ccc;
}

/* 保持原有的 .voice-control 样式 */
.voice-control {
  margin-top: 0.5rem;
}

.voice-control button {
  margin: 0 0.2rem;
  padding: 0.3rem 0.6rem;
  font-size: 1rem;
  cursor: pointer;
  border: 1px solid #007bff;
  background-color: #007bff;
  color: white;
  border-radius: 4px;
}

.voice-control button:hover:not(:disabled) {
  background-color: #0056b3;
}

.voice-control button:disabled {
  background-color: #ccc;
  cursor: not-allowed;
  border-color: #ccc;
}
</style>
