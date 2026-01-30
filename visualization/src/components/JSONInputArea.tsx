import React, { useState, useCallback, useRef } from 'react';
// @ts-ignore - Monaco Editor type compatibility   
import Editor, { loader } from '@monaco-editor/react';
import { useApp } from '../context/AppContext';
import { validateWorkflowData } from '../utils/dagDataProcessor';
import ConfirmDialog from './ConfirmDialog';

// 声明 chrome 类型
declare const chrome: any;

// 检测是否在 Chrome 扩展环境中
const isExtension = typeof chrome !== 'undefined' && chrome.runtime && chrome.runtime.getURL;

// 配置Monaco Editor使用本地资源
if (isExtension) {
  // Chrome 扩展环境：使用 chrome.runtime.getURL
  loader.config({
    paths: {
      vs: chrome.runtime.getURL('monaco-editor/min/vs')
    }
  });
} else {
  // 普通 Web 环境
  loader.config({
    paths: {
      vs: '/monaco-editor/min/vs'
    }
  });
}

// 配置 Monaco Editor 的 Web Worker
// @ts-ignore
window.MonacoEnvironment = {
  getWorkerUrl: function (_moduleId: string, _label: string) {
    const workerPath = 'monaco-editor/min/vs/base/worker/workerMain.js';
    if (isExtension) {
      return chrome.runtime.getURL(workerPath);
    }
    return `/${workerPath}`;
  }
};

const JSONInputArea: React.FC = () => {
  const { state, dispatch, loadDAGData, clearCanvas } = useApp();
  const [isValid, setIsValid] = useState(true);
  const [showClearConfirm, setShowClearConfirm] = useState(false);
  const editorRef = useRef<any>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [panelWidth, setPanelWidth] = useState<number>(() => {
    const saved = localStorage.getItem('json_panel_width');
    const parsed = saved ? parseInt(saved, 10) : 300;
    return isNaN(parsed) ? 300 : Math.min(Math.max(parsed, 220), 720);
  });
  const [isResizing, setIsResizing] = useState(false);

  // 同步宽度到 CSS 变量与本地存储
  React.useEffect(() => {
    document.documentElement.style.setProperty('--json-panel-width', `${panelWidth}px`);
    localStorage.setItem('json_panel_width', String(panelWidth));
  }, [panelWidth]);

  // 监听拖拽调整
  React.useEffect(() => {
    const handleMouseMove = (event: MouseEvent) => {
      if (!isResizing || !containerRef.current) return;
      const rect = containerRef.current.getBoundingClientRect();
      const newWidth = Math.min(Math.max(event.clientX - rect.left, 220), 720);
      setPanelWidth(newWidth);
    };

    const handleMouseUp = () => {
      if (isResizing) {
        setIsResizing(false);
      }
    };

    if (isResizing) {
      window.addEventListener('mousemove', handleMouseMove);
      window.addEventListener('mouseup', handleMouseUp);
    }

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isResizing]);

  const startResize = useCallback(() => {
    setIsResizing(true);
  }, []);

  // Monaco Editor配置
  const editorOptions = {
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    wordWrap: 'on' as const,
    lineNumbers: 'on' as const,
    lineNumbersMinChars: 3, // 减少行号列宽度
    formatOnPaste: true,
    formatOnType: true,
    automaticLayout: true,
    fontSize: 12, // 减小字体大小
    fontFamily: '"Monaco", "Menlo", "Ubuntu Mono", monospace',
    tabSize: 2,
    insertSpaces: true,
    // 启用代码折叠，方便查看大型 JSON 结构
    folding: true,
    bracketPairColorization: {
      enabled: true,
    },
    suggest: {
      showKeywords: true,
    },
    scrollbar: {
      vertical: 'auto' as const,
      horizontal: 'auto' as const,
      verticalScrollbarSize: 10,
      horizontalScrollbarSize: 10,
    },
    overviewRulerBorder: false,
    hideCursorInOverviewRuler: true,
    glyphMargin: false, // 移除左侧glyph边距
    lineDecorationsWidth: 8, // 增加行号和代码之间的间距
    renderLineHighlight: 'line' as const,
        // 简洁的提示文本，避免格式化问题
    placeholder: `// 📝 请输入JSON数据或点击上方「加载示例数据」`,
  };



  // 处理文本输入变化
  const handleTextChange = useCallback(async (value: string | undefined) => {
    const newValue = value || '';
    dispatch({ type: 'SET_JSON_TEXT', payload: newValue });
    
    if (newValue.trim() === '') {
      setIsValid(true);
      dispatch({ type: 'SET_ERROR', payload: null });
      // 清空画布数据
      dispatch({ type: 'SET_DAG_DATA', payload: null });
      return;
    }
    
    // 实时验证JSON格式和工作流数据
    try {
      const parsedData = JSON.parse(newValue);
      
      // 验证工作流数据格式
      const validation = validateWorkflowData(parsedData);
      if (!validation.isValid) {
        setIsValid(false);
        // 清空画布数据但保留JSON文本
        dispatch({ type: 'SET_DAG_DATA', payload: null });
        // 将错误信息传递给右侧可视化区域
        dispatch({ type: 'SET_ERROR', payload: validation.error || 'JSON数据验证失败' });
        return;
      }
      
      setIsValid(true);
      dispatch({ type: 'SET_ERROR', payload: null });
      
      // 检查是否有位置信息
      let hasPositionInfo = false;
      let tasks: any[] = [];
      
      if (parsedData && typeof parsedData === 'object' && parsedData.tasks && Array.isArray(parsedData.tasks)) {
        tasks = parsedData.tasks;
      } else if (Array.isArray(parsedData)) {
        tasks = parsedData;
      }
      
      // 如果至少有一个任务有位置信息，就不自动布局
      if (tasks.length > 0) {
        hasPositionInfo = tasks.some((task: any) => 
          task.position && 
          typeof task.position.x === 'number' && 
          typeof task.position.y === 'number'
        );
      }
      
      console.log(`JSON输入区域检测: 任务数=${tasks.length}, 有位置信息=${hasPositionInfo}`);
      if (hasPositionInfo) {
        console.log('检测到位置信息，将不执行自动布局');
      } else {
        console.log('未检测到位置信息，将执行自动布局');
      }
      
      // 如果有位置信息就不自动布局，否则自动布局
      const shouldAutoLayout = !hasPositionInfo;
      
      // 自动解析和可视化
      await loadDAGData(parsedData, shouldAutoLayout);
    } catch (error) {
      setIsValid(false);
      // 清空画布数据但保留JSON文本
      dispatch({ type: 'SET_DAG_DATA', payload: null });
      const errorMsg = `JSON格式错误: ${error instanceof Error ? error.message : 'Unknown error'}`;
      // 将错误信息传递给右侧可视化区域
      dispatch({ type: 'SET_ERROR', payload: errorMsg });
    }
  }, [dispatch, loadDAGData]);

  // Monaco Editor挂载完成
  const handleEditorDidMount = (editor: any, monacoInstance: any) => {
    editorRef.current = editor;

    // 设置JSON验证
    monacoInstance.languages.json.jsonDefaults.setDiagnosticsOptions({
      validate: true,
      allowComments: false,
      schemas: [],
      enableSchemaRequest: false,
    });
  };

  // 剪贴板粘贴处理
  const handlePasteFromClipboard = async () => {
    try {
      if (navigator.clipboard && navigator.clipboard.readText) {
        const clipboardText = await navigator.clipboard.readText();
        if (clipboardText.trim()) {
          // 设置编辑器内容
          if (editorRef.current) {
            editorRef.current.setValue(clipboardText);
          }
          await handleTextChange(clipboardText);
        } else {
          alert('剪贴板内容为空');
        }
      } else {
        alert('浏览器不支持剪贴板API，请使用 Ctrl+V 或右键粘贴JSON内容到编辑器中');
      }
    } catch (error) {
      alert('剪贴板访问失败，请使用 Ctrl+V 或右键粘贴JSON内容到编辑器中');
    }
  };

  // 文件选择处理
  const handleFileSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      try {
        const content = await file.text();
        // 设置编辑器内容
        if (editorRef.current) {
          editorRef.current.setValue(content);
        }
        await handleTextChange(content);
      } catch (error) {
        setIsValid(false);
        // 清空画布数据但保留JSON文本
        dispatch({ type: 'SET_DAG_DATA', payload: null });
        const errorMsg = `文件读取失败: ${error instanceof Error ? error.message : 'Unknown error'}`;
        dispatch({ type: 'SET_ERROR', payload: errorMsg });
      }
    }
  };

  // 格式化JSON
  const handleFormatJSON = () => {
    if (editorRef.current && state.jsonText.trim()) {
      try {
        const parsed = JSON.parse(state.jsonText);
        const formatted = JSON.stringify(parsed, null, 2);
        editorRef.current.setValue(formatted);
        handleTextChange(formatted);
      } catch (error) {
        alert('JSON格式错误，无法格式化');
      }
    }
  };

  // 清空编辑器
  const handleClearEditor = () => {
    setShowClearConfirm(true);
  };

  const confirmClearEditor = () => {
    if (editorRef.current) {
      editorRef.current.setValue('');
    }
    // 清空JSON输入和画布数据
    clearCanvas();
    setIsValid(true);
    setShowClearConfirm(false);
  };

  return (
    <div
      className="json-input-container"
      ref={containerRef}
      style={{ width: `${panelWidth}px` }}
    >
      <div className="input-header">
        <div className="input-actions">
          <button 
            onClick={handlePasteFromClipboard}
            className="paste-btn"
            title="从剪贴板粘贴"
          >
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              <rect x="8" y="2" width="8" height="4" rx="1" ry="1" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
            粘贴
          </button>
          <button 
            onClick={handleFormatJSON}
            className="format-btn"
            title="格式化JSON"
            disabled={!state.jsonText.trim()}
          >
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <line x1="21" y1="10" x2="3" y2="10" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              <line x1="21" y1="6" x2="3" y2="6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              <line x1="21" y1="14" x2="3" y2="14" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              <line x1="21" y1="18" x2="3" y2="18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
            格式化
          </button>
          <button 
            onClick={handleClearEditor}
            className="clear-btn"
            title="清空编辑器"
            disabled={!state.jsonText.trim()}
          >
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <polyline points="3 6 5 6 21 6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
            清空
          </button>
          <label className="file-input-label" title="选择本地文件">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
            文件
            <input
              type="file"
              accept=".json"
              onChange={handleFileSelect}
              style={{ display: 'none' }}
            />
          </label>
        </div>
      </div>
      
      <div className={`editor-container ${!isValid ? 'error' : ''}`}>
        <Editor
          height="100%"
          defaultLanguage="json"
          value={state.jsonText}
          onChange={handleTextChange}
          onMount={handleEditorDidMount}
          options={editorOptions}
          theme="vs"
          loading={<div className="editor-loading">🚀 正在加载Monaco编辑器...</div>}
          // @ts-ignore - Monaco Editor type compatibility
        />
      </div>
      
      {/* 移除左侧错误显示，错误将在右侧可视化区域展示 */}
      
      <div className="input-footer">
        <div className={`status ${state.jsonText.trim() === '' ? 'empty' : (isValid ? 'valid' : 'invalid')}`}>
          <span className="status-dot"></span>
          {state.jsonText.trim() === '' 
            ? '等待输入...' 
            : (isValid ? 'JSON格式正确' : 'JSON格式错误')
          }
        </div>
        <div className="text-stats">
          {state.jsonText.length} 字符 | {state.jsonText.split('\n').length} 行
        </div>
      </div>

      {/* 拖拽调节宽度 */}
      <div
        className={`json-resizer${isResizing ? ' resizing' : ''}`}
        onMouseDown={startResize}
      />
      
      <ConfirmDialog
        isOpen={showClearConfirm}
        title="清空JSON输入"
        message="确定要清空当前的JSON输入内容吗？此操作不可撤销。"
        confirmText="清空"
        cancelText="取消"
        type="danger"
        onConfirm={confirmClearEditor}
        onCancel={() => setShowClearConfirm(false)}
      />
    </div>
  );
};

export default JSONInputArea; 