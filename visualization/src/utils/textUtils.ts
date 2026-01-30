/**
 * 文本测量和自适应布局工具
 */

/**
 * 文本样式配置
 */
export interface TextStyle {
  fontSize: number;
  fontFamily: string;
  fontWeight: string;
  fontStyle?: string;
}

/**
 * 节点文本配置
 */
export interface NodeTextConfig {
  text: string;
  maxWidth: number;
  maxHeight: number;
  minFontSize: number;
  maxFontSize: number;
  lineHeight: number;
  padding: { x: number; y: number };
}

/**
 * 文本测量结果
 */
export interface TextMeasurement {
  width: number;
  height: number;
  lines: string[];
  fontSize: number;
  lineCount: number;
}

/**
 * 默认文本样式
 */
export const DEFAULT_TEXT_STYLE: TextStyle = {
  fontSize: 12,
  fontFamily: 'Arial, sans-serif',
  fontWeight: 'bold'
};

/**
 * 默认节点文本配置
 */
export const DEFAULT_NODE_TEXT_CONFIG: NodeTextConfig = {
  text: '',
  maxWidth: 200,
  maxHeight: 80,
  minFontSize: 8,
  maxFontSize: 18,
  lineHeight: 1.2,
  padding: { x: 12, y: 8 }
};

/**
 * 创建Canvas上下文用于文本测量
 */
let measureCanvas: HTMLCanvasElement | null = null;
let measureContext: CanvasRenderingContext2D | null = null;

function getMeasureContext(): CanvasRenderingContext2D {
  if (!measureCanvas) {
    measureCanvas = document.createElement('canvas');
    measureCanvas.width = 1000;
    measureCanvas.height = 1000;
    measureContext = measureCanvas.getContext('2d');
  }
  
  if (!measureContext) {
    throw new Error('无法创建Canvas上下文');
  }
  
  return measureContext;
}

/**
 * 设置Canvas字体样式
 */
function setCanvasFont(ctx: CanvasRenderingContext2D, style: TextStyle): void {
  const fontString = `${style.fontStyle || 'normal'} ${style.fontWeight} ${style.fontSize}px ${style.fontFamily}`;
  ctx.font = fontString;
}

/**
 * 测量单行文本宽度
 */
export function measureTextWidth(text: string, style: TextStyle = DEFAULT_TEXT_STYLE): number {
  const ctx = getMeasureContext();
  setCanvasFont(ctx, style);
  return ctx.measureText(text).width;
}

/**
 * 测量文本高度（基于字体大小和行高）
 */
export function measureTextHeight(style: TextStyle, lineCount: number = 1, lineHeight: number = 1.2): number {
  return style.fontSize * lineHeight * lineCount;
}

/**
 * 将长文本分割成适合指定宽度的行
 */
export function wrapText(text: string, maxWidth: number, style: TextStyle = DEFAULT_TEXT_STYLE): string[] {
  if (!text || text.trim() === '' || maxWidth <= 10) return [text || ''];
  
  const ctx = getMeasureContext();
  setCanvasFont(ctx, style);
  
  // 先检查单行是否能放下
  const singleLineWidth = ctx.measureText(text).width;
  if (singleLineWidth <= maxWidth) {
    return [text];
  }
  
  console.log(`文本"${text}"需要换行: 单行宽度${singleLineWidth}px > 最大宽度${maxWidth}px`);
  
  // 智能分词：优先保持单词边界
  const segments = smartTokenize(text);
  console.log(`分词结果:`, segments);
  
  // 如果分词已经得到合理的段落，优先使用分词结果
  if (segments.length > 1) {
    // 检查每个段落是否能独立放下
    let allSegmentsFit = true;
    for (const segment of segments) {
      const segmentWidth = ctx.measureText(segment).width;
      if (segmentWidth > maxWidth) {
        allSegmentsFit = false;
        break;
      }
    }
    
    // 如果所有段落都能独立放下，直接使用分词结果
    if (allSegmentsFit) {
      console.log(`使用智能分词结果，避免字符分割`);
      return segments;
    }
  }
  
  // 否则按宽度逐段组合
  const lines: string[] = [];
  let currentLine = '';
  
  for (const segment of segments) {
    const testLine = currentLine ? `${currentLine}${segment}` : segment;
    const testWidth = ctx.measureText(testLine).width;
    
    if (testWidth <= maxWidth) {
      currentLine = testLine;
    } else {
      // 如果当前行有内容，保存并开始新行
      if (currentLine) {
        lines.push(currentLine);
        currentLine = segment;
      } else {
        // 单个片段太长，才进行字符换行
        console.log(`片段"${segment}"过长，进行字符分割`);
        const charLines = breakLongWord(segment, maxWidth, ctx);
        lines.push(...charLines.slice(0, -1));
        currentLine = charLines[charLines.length - 1] || '';
      }
    }
  }
  
  if (currentLine) {
    lines.push(currentLine);
  }
  
  console.log(`换行结果: ${lines.length}行`, lines);
  return lines.length > 0 ? lines : [text];
}

/**
 * 智能分词：按单词边界分割，基于实际宽度计算
 * 例如: "EVALUATE_SCORE_HTTP_REQUEST" -> ["EVALUATE_SCORE_HTTP", "_REQUEST"]
 */
function smartTokenize(text: string): string[] {
  console.log(`开始分词: "${text}"`);
  
  // 如果文本较短，不需要分割
  if (text.length <= 12) {
    console.log(`文本较短，无需分割:`, [text]);
    return [text];
  }
  
  // 方案1: 按下划线分割，基于宽度智能合并
  if (text.includes('_')) {
    const parts = text.split('_');
    console.log(`下划线分割结果:`, parts);
    
    // 尝试两行布局：找到最佳分割点
    const bestSplit = findBestSplit(parts);
    console.log(`最佳分割结果:`, bestSplit);
    return bestSplit;
  }
  
  // 方案2: 驼峰命名分割
  if (/[a-z][A-Z]/.test(text)) {
    const segments = splitCamelCase(text);
    console.log(`驼峰分割结果:`, segments);
    
    if (segments.length > 1) {
      // 尝试合并为两行
      const mid = Math.ceil(segments.length / 2);
      const line1 = segments.slice(0, mid).join('');
      const line2 = segments.slice(mid).join('');
      return [line1, line2];
    }
    return segments;
  }
  
  // 方案3: 简单二分法
  const mid = Math.ceil(text.length / 2);
  const result = [text.slice(0, mid), text.slice(mid)];
  console.log(`二分法分割结果:`, result);
  return result;
}

/**
 * 找到最佳的两行分割点
 */
function findBestSplit(parts: string[]): string[] {
  if (parts.length <= 2) {
    return parts.map((part, i) => i === 0 ? part : '_' + part);
  }
  
  let bestSplit: string[] = [];
  let bestBalance = Infinity;
  
  // 尝试每个可能的分割点
  for (let i = 1; i < parts.length; i++) {
    const line1Parts = parts.slice(0, i);
    const line2Parts = parts.slice(i);
    
    const line1 = line1Parts.join('_');
    const line2 = '_' + line2Parts.join('_');
    
    // 计算两行长度的平衡度
    const balance = Math.abs(line1.length - line2.length);
    
    if (balance < bestBalance) {
      bestBalance = balance;
      bestSplit = [line1, line2];
    }
  }
  
  return bestSplit.length > 0 ? bestSplit : [parts.join('_')];
}

/**
 * 驼峰命名分割
 */
function splitCamelCase(text: string): string[] {
  const segments: string[] = [];
  let currentSegment = '';
  
  for (let i = 0; i < text.length; i++) {
    const char = text[i];
    const nextChar = text[i + 1];
    
    currentSegment += char;
    
    // 小写后跟大写，在大写前分割
    if (char && nextChar && 
        char.toLowerCase() === char && 
        nextChar.toUpperCase() === nextChar && 
        char !== nextChar) {
      segments.push(currentSegment);
      currentSegment = '';
    }
  }
  
  if (currentSegment) {
    segments.push(currentSegment);
  }
  
  return segments.filter(seg => seg.length > 0);
}



/**
 * 强制分割过长的单词
 */
function breakLongWord(word: string, maxWidth: number, ctx: CanvasRenderingContext2D): string[] {
  const lines: string[] = [];
  let currentLine = '';
  
  for (const char of word) {
    const testLine = currentLine + char;
    const testWidth = ctx.measureText(testLine).width;
    
    if (testWidth <= maxWidth) {
      currentLine = testLine;
    } else {
      if (currentLine) {
        lines.push(currentLine);
        currentLine = char;
      } else {
        // 即使单个字符也超宽，强制添加
        lines.push(char);
        currentLine = '';
      }
    }
  }
  
  if (currentLine) {
    lines.push(currentLine);
  }
  
  return lines.length > 0 ? lines : [word];
}

/**
 * 寻找适合指定约束的最佳字体大小
 */
export function findOptimalFontSize(config: NodeTextConfig): TextMeasurement {
  const { text, maxWidth, maxHeight, minFontSize, maxFontSize, lineHeight, padding } = config;
  
  if (!text || text.trim() === '') {
    return {
      width: Math.max(100, maxWidth * 0.6),
      height: Math.max(32, maxHeight * 0.4),
      lines: [''],
      fontSize: minFontSize,
      lineCount: 0
    };
  }
  
  const availableWidth = maxWidth - padding.x * 2; // 左右padding
  const availableHeight = maxHeight - padding.y * 2; // 上下padding
  
  console.log(`文本"${text}": 可用空间 ${availableWidth}x${availableHeight}px`);
  
  // 从最大字体开始尝试
  for (let fontSize = maxFontSize; fontSize >= minFontSize; fontSize -= 0.5) {
    const style: TextStyle = {
      ...DEFAULT_TEXT_STYLE,
      fontSize
    };
    
    const lines = wrapText(text, availableWidth, style);
    const textHeight = measureTextHeight(style, lines.length, lineHeight);
    
    console.log(`字体${fontSize}px: ${lines.length}行, 高度${textHeight}px, 可用${availableHeight}px`);
    
    // 检查是否同时满足宽度和高度约束
    if (textHeight <= availableHeight && lines.length > 0) {
      const textWidth = Math.max(...lines.map(line => measureTextWidth(line, style)));
      
      console.log(`选择字体${fontSize}px: 文本宽度${textWidth}px`);
      
      return {
        width: Math.min(textWidth + padding.x * 2, maxWidth),
        height: Math.min(textHeight + padding.y * 2, maxHeight),
        lines,
        fontSize,
        lineCount: lines.length
      };
    }
  }
  
  // 如果最小字体也不合适，使用最小字体并截断文本
  const style: TextStyle = {
    ...DEFAULT_TEXT_STYLE,
    fontSize: minFontSize
  };
  
  const lines = wrapText(text, availableWidth, style);
  const maxLines = Math.floor(availableHeight / (minFontSize * lineHeight));
  const truncatedLines = lines.slice(0, Math.max(1, maxLines));
  
  const textWidth = Math.max(...truncatedLines.map(line => measureTextWidth(line, style)));
  const textHeight = measureTextHeight(style, truncatedLines.length, lineHeight);
  
  console.log(`强制使用最小字体${minFontSize}px: ${truncatedLines.length}行`);
  
  return {
    width: Math.min(textWidth + padding.x * 2, maxWidth),
    height: Math.min(textHeight + padding.y * 2, maxHeight),
    lines: truncatedLines,
    fontSize: minFontSize,
    lineCount: truncatedLines.length
  };
}

/**
 * 计算节点的最佳尺寸
 */
export function calculateOptimalNodeSize(
  text: string, 
  constraints: Partial<NodeTextConfig> = {}
): { width: number; height: number; fontSize: number; lines: string[] } {
  const config: NodeTextConfig = {
    ...DEFAULT_NODE_TEXT_CONFIG,
    ...constraints,
    text
  };
  
  const measurement = findOptimalFontSize(config);
  
  return {
    width: Math.max(measurement.width, 100), // 最小宽度100px
    height: Math.max(measurement.height, 32), // 最小高度32px
    fontSize: measurement.fontSize,
    lines: measurement.lines
  };
}

/**
 * 批量计算多个节点的统一尺寸
 */
export function calculateUniformNodeSize(
  texts: string[],
  constraints: Partial<NodeTextConfig> = {}
): { width: number; height: number; fontSize: number } {
  if (texts.length === 0) {
    return { width: 180, height: 40, fontSize: 11 };
  }
  
  let maxWidth = 0;
  let maxHeight = 0;
  let minFontSize = constraints.maxFontSize || DEFAULT_NODE_TEXT_CONFIG.maxFontSize;
  
  // 为每个文本计算最佳尺寸
  texts.forEach(text => {
    if (text && text.trim()) { // 只处理非空文本
      const result = calculateOptimalNodeSize(text.trim(), constraints);
      maxWidth = Math.max(maxWidth, result.width);
      maxHeight = Math.max(maxHeight, result.height);
      minFontSize = Math.min(minFontSize, result.fontSize);
    }
  });
  
  // 确保有合理的最小值
  return {
    width: Math.max(maxWidth, 120),
    height: Math.max(maxHeight, 36),
    fontSize: Math.max(minFontSize, 10)
  };
}

/**
 * 判断文本是否需要换行
 */
export function needsTextWrapping(text: string, maxWidth: number, style: TextStyle = DEFAULT_TEXT_STYLE): boolean {
  const textWidth = measureTextWidth(text, style);
  return textWidth > maxWidth;
}

/**
 * 截断文本并添加省略号
 */
export function truncateTextWithEllipsis(
  text: string, 
  maxWidth: number, 
  style: TextStyle = DEFAULT_TEXT_STYLE
): string {
  if (!needsTextWrapping(text, maxWidth, style)) {
    return text;
  }
  
  const ellipsis = '...';
  const ellipsisWidth = measureTextWidth(ellipsis, style);
  const availableWidth = maxWidth - ellipsisWidth;
  
  let truncated = '';
  let currentWidth = 0;
  
  for (const char of text) {
    const charWidth = measureTextWidth(char, style);
    if (currentWidth + charWidth <= availableWidth) {
      truncated += char;
      currentWidth += charWidth;
    } else {
      break;
    }
  }
  
  return truncated + ellipsis;
} 