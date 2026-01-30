import React, { Component } from 'react';
import type { ReactNode } from 'react';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('DAG可视化器错误:', error, errorInfo);
    
    // 可以发送错误报告到分析服务
    this.reportError(error, errorInfo);
  }

  reportError = (error: Error, errorInfo: React.ErrorInfo) => {
    // 错误报告逻辑（可选）
    const errorReport = {
      message: error.message,
      stack: error.stack,
      timestamp: new Date().toISOString(),
      userAgent: navigator.userAgent,
      componentStack: errorInfo.componentStack
    };
    
    console.log('错误报告:', errorReport);
  };

  render() {
    if (this.state.hasError) {
      return (
        <div style={{
          padding: '40px 20px',
          textAlign: 'center',
          fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto',
          backgroundColor: '#fff',
          minHeight: '100vh',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center'
        }}>
          <h2 style={{ color: '#ff4d4f', marginBottom: '16px' }}>
            🚨 应用出现错误
          </h2>
          <p style={{ color: '#666', marginBottom: '24px' }}>
            很抱歉，DAG可视化器遇到了问题。
          </p>
          <details style={{ marginBottom: '24px', textAlign: 'left' }}>
            <summary style={{ cursor: 'pointer', marginBottom: '8px' }}>
              错误详情
            </summary>
            <pre style={{
              background: '#f5f5f5',
              padding: '12px',
              borderRadius: '4px',
              fontSize: '12px',
              color: '#333',
              overflow: 'auto',
              maxWidth: '600px'
            }}>
              {this.state.error?.stack}
            </pre>
          </details>
          <button
            onClick={() => window.location.reload()}
            style={{
              padding: '8px 16px',
              backgroundColor: '#40a9ff',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '14px'
            }}
          >
            🔄 重新加载页面
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary; 