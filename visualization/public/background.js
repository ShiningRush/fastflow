// Chrome Extension Background Service Worker
// 最小化实现 - 仅负责打开独立页面

// 点击插件图标时打开DAG可视化器页面
chrome.action.onClicked.addListener(() => {
  chrome.tabs.create({
    url: chrome.runtime.getURL('index.html')
  });
});

// 处理插件安装
chrome.runtime.onInstalled.addListener((details) => {
  if (details.reason === 'install') {
    // 首次安装时的初始化
    chrome.storage.local.set({
      'first_install': true,
      'install_time': Date.now(),
      'user_preferences': {
        theme: 'light',
        autoSave: true,
        layoutDirection: 'horizontal'
      }
    });
    
    console.log('DAG可视化器插件已安装');
  }
});

// 保留基础消息监听能力（当前无需特殊处理）
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  console.log('Background received message:', message);
  sendResponse({ success: true });
  return true;
}); 