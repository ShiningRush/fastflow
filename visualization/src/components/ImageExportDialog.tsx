import React, { useState, useEffect } from 'react';
import { generateTimestamp } from '../utils/timeUtils';
import type { ImageExportOptions } from '../types';

interface ImageExportDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onExport: (options: ImageExportOptions) => void;
  isExporting: boolean;
}

const ImageExportDialog: React.FC<ImageExportDialogProps> = ({
  isOpen,
  onClose,
  onExport,
  isExporting
}) => {
  const [options, setOptions] = useState<ImageExportOptions>({
    format: 'png',
    width: 1920,
    height: 1080,
    backgroundColor: '#ffffff',
    filename: 'dag-diagram',
    quality: 0.9,
    includeGrid: false
  });
  


  // 预设尺寸选项
  const presetSizes = [
    { name: '自定义', width: undefined, height: undefined },
    { name: 'Full HD (1920×1080)', width: 1920, height: 1080 },
    { name: 'HD (1280×720)', width: 1280, height: 720 },
    { name: '4K (3840×2160)', width: 3840, height: 2160 },
    { name: 'A4 横向 (297×210mm)', width: 1169, height: 827 },
    { name: 'A4 纵向 (210×297mm)', width: 827, height: 1169 }
  ];

  // 生成当前时间戳的函数（使用共享工具函数）
  const generateCurrentTimestamp = () => {
    const timestamp = generateTimestamp();
    console.log('生成的时间戳:', timestamp);
    return timestamp;
  };

  // 根据当前时间生成默认文件名
  useEffect(() => {
    if (isOpen) {
      // 每次打开对话框时使用当前时间填充文件名输入框
      const timestamp = generateCurrentTimestamp();
      setOptions(prev => ({
        ...prev,
        filename: `dag-diagram-${timestamp}`
      }));
    }
  }, [isOpen]);

  const handlePresetSizeChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const selectedIndex = parseInt(e.target.value);
    const preset = presetSizes[selectedIndex];
    
    if (preset.width && preset.height) {
      setOptions(prev => ({
        ...prev,
        width: preset.width,
        height: preset.height
      }));
    }
  };

  const handleExport = () => {
    // 直接使用输入框中显示的文件名，不需要额外处理时间戳
    onExport(options);
  };

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  if (!isOpen) return null;

  return (
    <div className="dialog-backdrop" onClick={handleBackdropClick}>
      <div className="dialog export-dialog">
        <div className="dialog-header">
          <h3>📸 导出图片</h3>
          <button className="dialog-close" onClick={onClose}>×</button>
        </div>
        
        <div className="dialog-content">
          {/* 格式选择 */}
          <div className="form-group">
            <label>📄 导出格式</label>
            <div className="radio-group">
              <label className="radio-item">
                <input
                  type="radio"
                  value="png"
                  checked={options.format === 'png'}
                  onChange={(e) => setOptions(prev => ({ ...prev, format: e.target.value as 'png' }))}
                />
                <span>PNG (无损，透明背景支持)</span>
              </label>
              <label className="radio-item">
                <input
                  type="radio"
                  value="jpg"
                  checked={options.format === 'jpg'}
                  onChange={(e) => setOptions(prev => ({ ...prev, format: e.target.value as 'jpg' }))}
                />
                <span>JPG (有损，较小文件)</span>
              </label>
              <label className="radio-item">
                <input
                  type="radio"
                  value="svg"
                  checked={options.format === 'svg'}
                  onChange={(e) => setOptions(prev => ({ ...prev, format: e.target.value as 'svg' }))}
                />
                <span>SVG (矢量，可缩放)</span>
              </label>
            </div>
          </div>

          {/* 尺寸设置 */}
          <div className="form-group">
            <label>📐 图片尺寸</label>
            <select onChange={handlePresetSizeChange} className="select-input">
              {presetSizes.map((preset, index) => (
                <option key={index} value={index}>
                  {preset.name}
                </option>
              ))}
            </select>
            <div className="size-inputs">
              <div className="input-group">
                <label>宽度 (px)</label>
                <input
                  type="number"
                  value={options.width || ''}
                  onChange={(e) => setOptions(prev => ({ 
                    ...prev, 
                    width: e.target.value ? parseInt(e.target.value) : undefined 
                  }))}
                  min="100"
                  max="7680"
                  placeholder="自动"
                />
              </div>
              <div className="input-group">
                <label>高度 (px)</label>
                <input
                  type="number"
                  value={options.height || ''}
                  onChange={(e) => setOptions(prev => ({ 
                    ...prev, 
                    height: e.target.value ? parseInt(e.target.value) : undefined 
                  }))}
                  min="100"
                  max="4320"
                  placeholder="自动"
                />
              </div>
            </div>
          </div>

          {/* 背景颜色 */}
          {options.format !== 'svg' && (
            <div className="form-group">
              <label>🎨 背景颜色</label>
              <div className="color-input-group">
                <input
                  type="color"
                  value={options.backgroundColor}
                  onChange={(e) => setOptions(prev => ({ ...prev, backgroundColor: e.target.value }))}
                  className="color-picker"
                />
                <input
                  type="text"
                  value={options.backgroundColor}
                  onChange={(e) => setOptions(prev => ({ ...prev, backgroundColor: e.target.value }))}
                  className="color-text"
                  placeholder="#ffffff"
                />
                <button
                  type="button"
                  onClick={() => setOptions(prev => ({ ...prev, backgroundColor: 'transparent' }))}
                  className="transparent-btn"
                  disabled={options.format === 'jpg'}
                >
                  透明
                </button>
              </div>
              {options.format === 'jpg' && (
                <div className="form-hint">JPG格式不支持透明背景</div>
              )}
            </div>
          )}

          {/* JPG质量设置 */}
          {options.format === 'jpg' && (
            <div className="form-group">
              <label>🎯 图片质量 ({Math.round(options.quality * 100)}%)</label>
              <input
                type="range"
                min="0.1"
                max="1"
                step="0.1"
                value={options.quality}
                onChange={(e) => setOptions(prev => ({ ...prev, quality: parseFloat(e.target.value) }))}
                className="quality-slider"
              />
            </div>
          )}

          {/* 文件名设置 */}
          <div className="form-group">
            <label>📁 文件名</label>
            <div className="filename-input-group">
              <input
                type="text"
                value={options.filename}
                onChange={(e) => setOptions(prev => ({ ...prev, filename: e.target.value }))}
                className="filename-input"
                placeholder="dag-diagram"
              />
              <span className="filename-ext">.{options.format}</span>
            </div>
          </div>

          {/* 网格导出选项 */}
          <div className="form-group">
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={options.includeGrid}
                onChange={(e) => setOptions(prev => ({ ...prev, includeGrid: e.target.checked }))}
              />
              🔲 包含背景网格
            </label>
            <div className="form-hint">勾选后导出的图片将包含背景网格线</div>
          </div>
        </div>
        
        <div className="dialog-footer">
          <button 
            className="btn btn-secondary" 
            onClick={onClose}
            disabled={isExporting}
          >
            取消
          </button>
          <button 
            className="btn btn-primary" 
            onClick={handleExport}
            disabled={isExporting || !options.filename.trim()}
          >
            {isExporting ? (
              <>
                <span className="btn-icon">⏳</span>
                导出中...
              </>
            ) : (
              <>
                <span className="btn-icon">📸</span>
                导出图片
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};

export default ImageExportDialog; 