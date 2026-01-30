import React, { useState, useEffect } from 'react';
import { useApp } from '../context/AppContext';

interface Variable {
  name: string;
  desc?: string;
  default_value?: any;
}

interface VarsEditorDialogProps {
  isOpen: boolean;
  onClose: () => void;
}

const VarsEditorDialog: React.FC<VarsEditorDialogProps> = ({ isOpen, onClose }) => {
  const { state, dispatch, loadDAGData } = useApp();
  const [vars, setVars] = useState<Variable[]>([]);
  const [hasChanges, setHasChanges] = useState(false);

  // 从当前JSON中加载vars
  useEffect(() => {
    if (isOpen && state.jsonText) {
      try {
        const jsonData = JSON.parse(state.jsonText);
        const loadedVars = jsonData.vars || [];
        setVars(loadedVars.length > 0 ? loadedVars : []);
        setHasChanges(false);
      } catch (error) {
        console.error('解析JSON失败:', error);
        setVars([]);
      }
    }
  }, [isOpen, state.jsonText]);

  const handleAddVar = () => {
    setVars([...vars, { name: '', default_value: '', desc: '' }]);
    setHasChanges(true);
  };

  const handleRemoveVar = (index: number) => {
    const newVars = vars.filter((_, i) => i !== index);
    setVars(newVars);
    setHasChanges(true);
  };

  const handleVarChange = (index: number, field: keyof Variable, value: any) => {
    const newVars = [...vars];
    newVars[index] = { ...newVars[index], [field]: value };
    setVars(newVars);
    setHasChanges(true);
  };

  const handleSave = async () => {
    if (!state.jsonText) return;

    try {
      const jsonData = JSON.parse(state.jsonText);
      
      // 更新vars
      const updatedData = {
        ...jsonData,
        vars: vars.filter(v => v.name.trim() !== '') // 过滤掉空名称的变量
      };

      const updatedJsonText = JSON.stringify(updatedData, null, 2);
      dispatch({ type: 'SET_JSON_TEXT', payload: updatedJsonText });
      
      // 重新加载数据以刷新可视化
      await loadDAGData(updatedData, false); // 不执行自动布局
      
      setHasChanges(false);
      alert('变量已保存');
      onClose();
    } catch (error) {
      console.error('保存变量失败:', error);
      alert('保存失败: ' + (error instanceof Error ? error.message : '未知错误'));
    }
  };

  if (!isOpen) return null;

  return (
    <div 
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        backgroundColor: 'rgba(0, 0, 0, 0.3)',
        zIndex: 1000,
        display: 'flex',
        justifyContent: 'flex-end',
        alignItems: 'stretch'
      }}
      onClick={onClose}
    >
      <div 
        className="node-edit-drawer" 
        onClick={(e) => e.stopPropagation()} 
        style={{ 
          width: '480px',
          maxWidth: '90vw',
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          position: 'relative'
        }}
      >
        {/* 抽屉头部 - 完全复用节点编辑的样式 */}
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          padding: '16px 20px',
          borderBottom: '1px solid #e5e7eb',
          backgroundColor: '#f9fafb'
        }}>
          <h4 style={{
            margin: 0,
            fontSize: '16px',
            fontWeight: '600',
            color: '#1f2937'
          }}>工作流变量管理</h4>
          <button
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              fontSize: '20px',
              color: '#6b7280',
              cursor: 'pointer',
              padding: '4px',
              borderRadius: '4px',
              transition: 'all 0.2s'
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLElement).style.backgroundColor = '#e5e7eb';
              (e.currentTarget as HTMLElement).style.color = '#374151';
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLElement).style.backgroundColor = 'transparent';
              (e.currentTarget as HTMLElement).style.color = '#6b7280';
            }}
          >
            ✕
          </button>
        </div>
            
        {/* 抽屉内容 - 使用相同的类名 */}
        <div className="node-edit-drawer-content">
          {vars.length === 0 ? (
            <div className="edit-form-section" style={{ 
              textAlign: 'center', 
              padding: '60px 20px', 
              backgroundColor: '#f9fafb',
              borderRadius: '8px',
              border: '2px dashed #d1d5db'
            }}>
              <div style={{ fontSize: '48px', marginBottom: '16px' }}>📝</div>
              <p style={{ fontSize: '16px', color: '#6b7280', margin: '0 0 8px 0' }}>暂无变量</p>
              <p style={{ fontSize: '13px', color: '#9ca3af', margin: 0 }}>点击下方"添加变量"按钮来创建新变量</p>
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              {vars.map((variable, index) => (
                <div key={index} className="edit-form-section" style={{
                  border: '1px solid #e5e7eb',
                  borderRadius: '8px',
                  padding: '16px',
                  backgroundColor: '#f9fafb',
                  marginBottom: '16px'
                }}>
                  <div style={{ 
                    display: 'flex', 
                    justifyContent: 'space-between', 
                    alignItems: 'center', 
                    marginBottom: '16px',
                    paddingBottom: '12px',
                    borderBottom: '1px solid #e5e7eb'
                  }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <span style={{ 
                        backgroundColor: '#3b82f6', 
                        color: 'white', 
                        padding: '2px 10px', 
                        borderRadius: '12px', 
                        fontSize: '12px',
                        fontWeight: '600'
                      }}>
                        #{index + 1}
                      </span>
                      <span style={{ fontSize: '14px', color: '#6b7280', fontWeight: '500' }}>
                        {variable.name || '未命名变量'}
                      </span>
                    </div>
                    <button
                      onClick={() => handleRemoveVar(index)}
                      style={{
                        padding: '6px 12px',
                        backgroundColor: '#fee2e2',
                        color: '#dc2626',
                        border: 'none',
                        borderRadius: '6px',
                        cursor: 'pointer',
                        fontSize: '13px',
                        fontWeight: '500',
                        transition: 'all 0.2s'
                      }}
                      onMouseOver={(e) => e.currentTarget.style.backgroundColor = '#fecaca'}
                      onMouseOut={(e) => e.currentTarget.style.backgroundColor = '#fee2e2'}
                    >
                      删除
                    </button>
                  </div>

                  <div className="edit-form-section">
                    <label>变量名 <span style={{ color: '#ef4444' }}>*</span></label>
                    <input
                      className="edit-input"
                      type="text"
                      value={variable.name}
                      onChange={(e) => handleVarChange(index, 'name', e.target.value)}
                      placeholder="例如: user_id"
                    />
                  </div>

                  <div className="edit-form-section">
                    <label>默认值</label>
                    <input
                      className="edit-input"
                      type="text"
                      value={variable.default_value || ''}
                      onChange={(e) => handleVarChange(index, 'default_value', e.target.value)}
                      placeholder="例如: 空字符串"
                    />
                  </div>

                  <div className="edit-form-section">
                    <label>描述</label>
                    <textarea
                      className="edit-input"
                      value={variable.desc || ''}
                      onChange={(e) => handleVarChange(index, 'desc', e.target.value)}
                      placeholder="描述这个变量的用途..."
                      rows={2}
                      style={{ fontFamily: 'inherit', resize: 'vertical' }}
                    />
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div style={{ 
          display: 'flex', 
          gap: '12px', 
          justifyContent: 'space-between',
          padding: '16px 20px',
          borderTop: '1px solid #e5e7eb',
          backgroundColor: '#f9fafb',
          flexShrink: 0
        }}>
          <button
            onClick={handleAddVar}
            style={{
              padding: '10px 16px',
              backgroundColor: '#3b82f6',
              color: 'white',
              border: 'none',
              borderRadius: '6px',
              cursor: 'pointer',
              fontSize: '14px',
              fontWeight: '500',
              transition: 'all 0.2s'
            }}
            onMouseOver={(e) => e.currentTarget.style.backgroundColor = '#2563eb'}
            onMouseOut={(e) => e.currentTarget.style.backgroundColor = '#3b82f6'}
          >
            ➕ 添加变量
          </button>
          
          <div style={{ display: 'flex', gap: '8px' }}>
            <button
              onClick={onClose}
              style={{
                padding: '10px 16px',
                backgroundColor: 'white',
                color: '#6b7280',
                border: '1px solid #d1d5db',
                borderRadius: '6px',
                cursor: 'pointer',
                fontSize: '14px',
                fontWeight: '500',
                transition: 'all 0.2s'
              }}
              onMouseOver={(e) => e.currentTarget.style.backgroundColor = '#f3f4f6'}
              onMouseOut={(e) => e.currentTarget.style.backgroundColor = 'white'}
            >
              取消
            </button>
            <button
              onClick={handleSave}
              disabled={!hasChanges}
              className="save-btn"
              style={{
                padding: '10px 20px',
                backgroundColor: hasChanges ? '#22c55e' : '#d1d5db',
                color: hasChanges ? 'white' : '#9ca3af',
                border: 'none',
                borderRadius: '6px',
                cursor: hasChanges ? 'pointer' : 'not-allowed',
                fontSize: '14px',
                fontWeight: '500',
                transition: 'all 0.2s',
                minWidth: '80px'
              }}
              onMouseOver={(e) => hasChanges && (e.currentTarget.style.backgroundColor = '#16a34a')}
              onMouseOut={(e) => hasChanges && (e.currentTarget.style.backgroundColor = '#22c55e')}
            >
              保存
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default VarsEditorDialog;
