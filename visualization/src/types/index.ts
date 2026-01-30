// DAG数据类型定义
export interface DAGNode { 
  id: string;
  type: string;
  label: string;
  data: {
    original: any;
    index: number;
    taskType?: string;
    color?: string;
    inputCount?: number;
    outputCount?: number;
  };
  position: { x: number; y: number };
}

export interface DAGEdge {
  id: string;
  source: string;
  target: string;
  type: string;
  label?: string; // 边的标签（用于条件分支）
  condition?: EdgeCondition; // 边的条件信息
}

// 边的条件信息
export interface EdgeCondition {
  type: 'normal' | 'conditional'; // 边的类型：普通或条件
  label?: string; // 条件标签
  checkName?: string; // 前置检查名称
  activeAction?: string; // 激活动作（如 skip, continue 等）
  conditions?: any[]; // 条件详情
}

export interface DAGData {
  nodes: DAGNode[];
  edges: DAGEdge[];
  metadata: {
    totalNodes: number;
    totalEdges: number;
    processedAt: string;
  };
}

// 应用状态类型
export interface AppState {
  dagData: DAGData | null;
  jsonText: string;
  isLoading: boolean;
  error: string | null;
  fileHistory: HistoryEntry[];
  isExporting: boolean;
  /** 是否请求在下一次 DAG 渲染完成后自动执行一次智能布局 */
  autoLayoutRequested?: boolean;
  /** 是否请求执行智能纵向布局（用于保存节点后自动布局） */
  smartLayoutRequested?: boolean;
}

// 历史记录类型
export interface HistoryEntry {
  id: string;
  name: string;
  data: any;
  source: 'manual' | 'file' | 'clipboard' | 'example';
  timestamp: number;
}

// 节点类型定义（从外部 JSON 文件加载）
export interface NodeTypeIOParam {
  key: string;
  value?: string;
  sub_params?: NodeTypeIOParam[];
}

export interface NodeTypeDefinition {
  action_name: string;
  desc?: string;
  input: NodeTypeIOParam[];
  output: NodeTypeIOParam[];
}

// 用户偏好类型
export interface UserPreferences {
  theme: 'light' | 'dark';
  autoSave: boolean;
  layoutDirection: 'horizontal' | 'vertical';
}

// 导出选项类型
export interface ExportOptions {
  includePositions: boolean;
  includeMetadata: boolean;
  simplifyData: boolean;
  fileName?: string;
}

// 任务节点类型映射
export type TaskNodeType = 'promptBuild' | 'callLLM' | 'httpRequest' | 'codeExec' | 'default'; 

// 图片导出选项类型
export interface ImageExportOptions {
  format: 'png' | 'jpg' | 'svg';
  width?: number;
  height?: number;
  backgroundColor?: string;
  filename: string;
  quality: number; // 0.1 到 1.0，仅对 JPG 有效
  includeGrid: boolean; // 是否包含背景网格
} 