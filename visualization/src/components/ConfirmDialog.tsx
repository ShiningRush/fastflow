import React from 'react';

interface ConfirmDialogProps {
  isOpen: boolean;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  onConfirm: () => void;
  onCancel: () => void;
  type?: 'warning' | 'danger' | 'info';
}

const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
  isOpen,
  title,
  message,
  confirmText = '确定',
  cancelText = '取消',
  onConfirm,
  onCancel,
  type = 'warning'
}) => {
  if (!isOpen) return null;

  const getTypeStyles = () => {
    switch (type) {
      case 'danger':
        return {
          iconColor: '#f44336',
          confirmBg: 'linear-gradient(135deg, #f44336 0%, #d32f2f 100%)',
          confirmHover: 'linear-gradient(135deg, #e53935 0%, #c62828 100%)'
        };
      case 'info':
        return {
          iconColor: '#2196f3',
          confirmBg: 'linear-gradient(135deg, #2196f3 0%, #1976d2 100%)',
          confirmHover: 'linear-gradient(135deg, #1e88e5 0%, #1565c0 100%)'
        };
      default: // warning
        return {
          iconColor: '#ff9800',
          confirmBg: 'linear-gradient(135deg, #ff9800 0%, #f57c00 100%)',
          confirmHover: 'linear-gradient(135deg, #fb8c00 0%, #ef6c00 100%)'
        };
    }
  };

  const typeStyles = getTypeStyles();

  return (
    <div className="confirm-dialog-overlay">
      <div className="confirm-dialog">
        <div className="confirm-dialog-header">
          <div className="confirm-dialog-icon" style={{ color: typeStyles.iconColor }}>
            {type === 'danger' ? '🗑️' : type === 'info' ? 'ℹ️' : '⚠️'}
          </div>
          <h3 className="confirm-dialog-title">{title}</h3>
        </div>
        
        <div className="confirm-dialog-content">
          <p className="confirm-dialog-message">{message}</p>
        </div>
        
        <div className="confirm-dialog-actions">
          <button 
            className="confirm-dialog-cancel"
            onClick={onCancel}
          >
            {cancelText}
          </button>
          <button 
            className="confirm-dialog-confirm"
            onClick={onConfirm}
            style={{ 
              background: typeStyles.confirmBg,
              '--hover-bg': typeStyles.confirmHover
            } as React.CSSProperties}
          >
            {confirmText}
          </button>
        </div>
      </div>
    </div>
  );
};

export default ConfirmDialog;