import React, { useState } from 'react';
import { useApp } from '../context/AppContext';
import ImageExportDialog from './ImageExportDialog';
import ConfirmDialog from './ConfirmDialog';
import type { ImageExportOptions } from '../types';

const Toolbar: React.FC = () => {
  const { loadExampleData, loadLocalFile, loadNodeTypesFile, exportDAGConfig, exportImage, clearCanvas, state, isVarsEditorOpen, setIsVarsEditorOpen } = useApp();
  const [isExportDialogOpen, setIsExportDialogOpen] = useState(false);
  const [showClearCanvasConfirm, setShowClearCanvasConfirm] = useState(false);

  const handleLoadLocalFile = async () => {
    try {
      await loadLocalFile();
    } catch (error) {
      console.error('加载本地文件失败:', error);
    }
  };

  const handleLoadExampleData = async () => {
    try {
      await loadExampleData();
    } catch (error) {
      console.error('加载示例数据失败:', error);
    }
  };

  const handleExportConfig = () => {
    exportDAGConfig();
  };

  const handleClearCanvas = () => {
    if (state.dagData) {
      setShowClearCanvasConfirm(true);
    } else {
      clearCanvas(); // 如果没有数据就直接清空
    }
  };

  const confirmClearCanvas = () => {
    clearCanvas();
    setShowClearCanvasConfirm(false);
  };

  const handleExportImage = () => {
    setIsExportDialogOpen(true);
  };

  const handleImageExport = async (options: ImageExportOptions) => {
    try {
      await exportImage(options);
      setIsExportDialogOpen(false);
    } catch (error) {
      // 错误已经在 AppContext 中处理了，这里不需要额外处理
      console.error('图片导出失败:', error);
    }
  };

  return (
    <div className="toolbar">
      <div className="toolbar-left">
        <div className="app-logo">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <polyline points="3.27,6.96 12,12.01 20.73,6.96" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <line x1="12" y1="22.08" x2="12" y2="12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
          <div>
            <div className="app-logo-text">DAG Visualizer</div>
            <div className="app-logo-subtitle">专业的工作流可视化工具</div>
          </div>
        </div>
      </div>
      
      <div className="toolbar-right">
        <button 
          className="toolbar-btn"
          onClick={handleLoadLocalFile}
          disabled={state.isLoading}
          title="从本地加载JSON文件"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
          加载本地文件
        </button>

        <button 
          className="toolbar-btn"
          onClick={async () => {
            try {
              await loadNodeTypesFile();
            } catch (error) {
              console.error('加载节点类型文件失败:', error);
            }
          }}
          disabled={state.isLoading}
          title="从本地加载节点类型配置文件"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2" stroke="currentColor" strokeWidth="2"/>
            <rect x="7" y="7" width="3" height="9" stroke="currentColor" strokeWidth="2"/>
            <rect x="14" y="7" width="3" height="5" stroke="currentColor" strokeWidth="2"/>
          </svg>
          加载节点类型
        </button>
        
        <button 
          className="toolbar-btn"
          onClick={handleLoadExampleData}
          disabled={state.isLoading}
          title="加载示例DAG数据"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <polyline points="14 2 14 8 20 8" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <line x1="16" y1="13" x2="8" y2="13" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <line x1="16" y1="17" x2="8" y2="17" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
          加载示例数据
        </button>
        
        <button 
          className="toolbar-btn"
          onClick={() => setIsVarsEditorOpen(!isVarsEditorOpen)}
          disabled={state.isLoading || !state.jsonText}
          title="编辑工作流变量"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M12 2L2 7l10 5 10-5-10-5z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <path d="M2 17l10 5 10-5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <path d="M2 12l10 5 10-5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
          编辑变量
        </button>
        
        <button 
          className="toolbar-btn"
          onClick={handleExportConfig}
          disabled={state.isLoading || !state.dagData}
          title="导出当前配置"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <polyline points="7 10 12 15 17 10" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <line x1="12" y1="15" x2="12" y2="3" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
          导出配置
        </button>
        
        <button 
          className="toolbar-btn primary"
          onClick={handleExportImage}
          disabled={state.isLoading || !state.dagData}
          title="导出图片 (Ctrl+E)"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2" stroke="currentColor" strokeWidth="2"/>
            <circle cx="8.5" cy="8.5" r="1.5" stroke="currentColor" strokeWidth="2"/>
            <polyline points="21 15 16 10 5 21" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
          导出图片
        </button>
        
        <button 
          className="toolbar-btn danger"
          onClick={handleClearCanvas}
          disabled={state.isLoading}
          title="清空画布"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <polyline points="3 6 5 6 21 6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
          清空画布
        </button>
        
        {state.isLoading && (
          <div className="toolbar-loading">
            <span className="loading-text">处理中...</span>
          </div>
        )}
      </div>
      
      <ImageExportDialog
        isOpen={isExportDialogOpen}
        onClose={() => setIsExportDialogOpen(false)}
        onExport={handleImageExport}
        isExporting={state.isExporting}
      />
      
      <ConfirmDialog
        isOpen={showClearCanvasConfirm}
        title="清空画布"
        message="确定要清空画布吗？这将删除当前的所有节点、连线和可视化数据，且操作不可撤销。"
        confirmText="清空画布"
        cancelText="取消"
        type="danger"
        onConfirm={confirmClearCanvas}
        onCancel={() => setShowClearCanvasConfirm(false)}
      />
    </div>
  );
};

export default Toolbar; 