import React, { useEffect } from 'react';
import { ReactFlowProvider } from 'reactflow';
import { AppProvider, useApp } from './context/AppContext';
import Toolbar from './components/Toolbar';
import JSONInputArea from './components/JSONInputArea';
import DAGVisualizer from './components/DAGVisualizer';
import StatusBar from './components/StatusBar';
import ErrorBoundary from './components/ErrorBoundary';
import './styles/App.css';

// 主应用内容组件（在 AppProvider 内部）
const AppContent: React.FC = () => {
  const { state } = useApp();

  // 快捷键监听
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      // Ctrl+E 导出图片
      if (event.ctrlKey && event.key === 'e') {
        event.preventDefault();
        
        // 只有在有数据且不在加载中时才触发导出
        if (state.dagData && !state.isLoading) {
          // 触发工具栏中导出图片按钮的点击
          const exportButton = document.querySelector('[title*="导出图片"]') as HTMLButtonElement;
          if (exportButton && !exportButton.disabled) {
            exportButton.click();
          }
        }
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [state.dagData, state.isLoading]);

  return (
        <div className="app-container">
          <Toolbar />
          <JSONInputArea />
          <ReactFlowProvider>
            <DAGVisualizer />
          </ReactFlowProvider>
          <StatusBar />
        </div>
  );
};

const App: React.FC = () => {
  return (
    <ErrorBoundary>
      <AppProvider>
        <AppContent />
      </AppProvider>
    </ErrorBoundary>
  );
};

export default App;
