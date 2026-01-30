// 节点类型管理器 - 智能节点创建的核心工具
export interface NodeTypeDefinition {
  id: string;
  label: string;
  description: string;
  icon: string;
  defaultColor: string;
  className: string;
  template: {
    taskType: string;
    '@type': string;
    input: any[];
    output: any[];
    dependencies: string[];
  };
}

// 4种默认节点类型（基于dag-question-rewrite-rerank.json）
export const DEFAULT_NODE_TYPES: NodeTypeDefinition[] = [
  {
    id: 'PROMPT_BUILD',
    label: '提示构建',
    description: '构建和处理提示模板',
    icon: '🔧',
    defaultColor: '#4ade80', // 绿色 - 构建类
    className: 'node-prompt-build',
    template: {
      taskType: 'PROMPT_BUILD',
      '@type': 'com.xiaohongshu.data.aimi.workflow.nodes.TemplateTransformNode',
      input: [], // 简化为空数组
      output: [], // 简化为空数组
      dependencies: []
    }
  },
  {
    id: 'CALL_LLM',
    label: 'LLM调用',
    description: '调用大语言模型处理',
    icon: '🤖',
    defaultColor: '#3b82f6', // 蓝色 - AI类
    className: 'node-call-llm',
    template: {
      taskType: 'CALL_LLM',
      '@type': 'com.xiaohongshu.data.aimi.workflow.nodes.LLMNode',
      input: [], // 简化为空数组
      output: [], // 简化为空数组
      dependencies: []
    }
  },
  {
    id: 'HTTP_REQUEST',
    label: 'HTTP请求',
    description: '发送HTTP请求获取数据',
    icon: '🌐',
    defaultColor: '#f59e0b', // 橙色 - 网络类
    className: 'node-http-request',
    template: {
      taskType: 'HttpRequestNode',
      '@type': 'com.xiaohongshu.data.aimi.workflow.nodes.HttpRequestNode',
      input: [], // 简化为空数组
      output: [], // 简化为空数组
      dependencies: []
    }
  },
  {
    id: 'CODE_EXEC',
    label: '代码执行',
    description: '执行自定义代码逻辑',
    icon: '💻',
    defaultColor: '#8b5cf6', // 紫色 - 计算类
    className: 'node-code-exec',
    template: {
      taskType: 'CodeNode',
      '@type': 'com.xiaohongshu.data.aimi.workflow.nodes.CodeNode',
      input: [], // 简化为空数组
      output: [], // 简化为空数组
      dependencies: []
    }
  }
];

// 颜色管理系统
export class ColorManager {
  // 任务前缀到颜色的映射（精心搭配的色彩方案）
  private static readonly PREFIX_COLORS: Record<string, string> = {
  };

  // 随机颜色池（用于未匹配前缀的任务）
  private static readonly FALLBACK_COLORS = [
    '#3b82f6', // 蓝色
    '#8b5cf6', // 紫色
    '#06b6d4', // 青色
    '#ec4899', // 粉色
    '#f59e0b', // 琥珀色
    '#10b981', // 翠绿色
    '#ef4444', // 红色
    '#6366f1', // 靛蓝色
    '#eab308', // 黄色
    '#f97316', // 橙色
    '#14b8a6', // 青绿色
    '#a855f7', // 紫红色
  ];

  // 预定义的随机颜色池（语义化色彩）
  private static readonly RANDOM_COLORS = [
    '#ef4444', // 红色
    '#f97316', // 橙色  
    '#eab308', // 黄色
    '#22c55e', // 绿色
    '#06b6d4', // 青色
    '#3b82f6', // 蓝色
    '#6366f1', // 靛蓝色
    '#8b5cf6', // 紫色
    '#ec4899', // 粉色
    '#64748b', // 灰色
  ];

  // 自定义类型颜色存储
  private static customTypeColors: Map<string, string> = new Map();
  
  // 本地存储键名
  private static readonly STORAGE_KEY = 'dag-visualizer-colors';
  
  // 初始化时从localStorage加载颜色配置
  static {
    this.loadColorsFromStorage();
  }

  // 根据action_name获取名称前缀颜色，未匹配时返回稳定的随机颜色
  static getColorByActionName(actionName: string): string {
    // 遍历前缀映射，找到匹配的前缀
    for (const [prefix, color] of Object.entries(this.PREFIX_COLORS)) {
      if (actionName.toLowerCase().startsWith(prefix)) {
        return color;
      }
    }
    // 如果没有匹配的前缀，使用action_name生成一个稳定的随机颜色
    // 这样相同的action_name总是得到相同的颜色
    const hash = actionName.split('').reduce((acc, char) => {
      return ((acc << 5) - acc) + char.charCodeAt(0);
    }, 0);
    const colorIndex = Math.abs(hash) % this.FALLBACK_COLORS.length;
    return this.FALLBACK_COLORS[colorIndex];
  }

  // 根据任务ID获取前缀颜色，未匹配时返回稳定的随机颜色
  static getColorByTaskId(taskId: string): string {
    // 兼容旧代码，内部改为调用 getColorByActionName
    return this.getColorByActionName(taskId);
  }

  // 获取节点类型的默认颜色
  static getDefaultColor(nodeType: string): string {
    // 首先检查是否为预定义类型
    const nodeTypeDefinition = DEFAULT_NODE_TYPES.find(type => type.id === nodeType);
    if (nodeTypeDefinition) {
      return nodeTypeDefinition.defaultColor;
    }
    
    // 检查自定义类型颜色映射
    if (this.customTypeColors.has(nodeType)) {
      return this.customTypeColors.get(nodeType)!;
    }
    
    // 返回自定义类型的默认颜色
    return '#64748b';
  }

  // 设置自定义类型的默认颜色
  static setCustomTypeColor(nodeType: string, color: string): void {
    if (this.isValidHexColor(color)) {
      this.customTypeColors.set(nodeType, color);
      this.saveColorsToStorage();
    }
  }

  // 获取所有自定义类型颜色
  static getCustomTypeColors(): Map<string, string> {
    return new Map(this.customTypeColors);
  }

  // 批量更新同类型节点颜色
  static updateTypeDefaultColor(nodeType: string, color: string): void {
    if (!this.isValidHexColor(color)) return;
    
    // 如果是预定义类型，更新DEFAULT_NODE_TYPES中的颜色
    const nodeTypeIndex = DEFAULT_NODE_TYPES.findIndex(type => type.id === nodeType);
    if (nodeTypeIndex !== -1) {
      DEFAULT_NODE_TYPES[nodeTypeIndex].defaultColor = color;
    } else {
      // 如果是自定义类型，更新自定义类型颜色映射
      this.setCustomTypeColor(nodeType, color);
    }
    
    // 保存到localStorage
    this.saveColorsToStorage();
  }

  // 从localStorage加载颜色配置
  private static loadColorsFromStorage(): void {
    try {
      const stored = localStorage.getItem(this.STORAGE_KEY);
      if (stored) {
        const colorData = JSON.parse(stored);
        
        // 恢复预定义类型颜色
        if (colorData.presetTypes) {
          Object.entries(colorData.presetTypes as Record<string, string>).forEach(([typeId, color]) => {
            const nodeTypeIndex = DEFAULT_NODE_TYPES.findIndex(type => type.id === typeId);
            if (nodeTypeIndex !== -1) {
              DEFAULT_NODE_TYPES[nodeTypeIndex].defaultColor = color;
            }
          });
        }
        
        // 恢复自定义类型颜色
        if (colorData.customTypes) {
          this.customTypeColors = new Map(Object.entries(colorData.customTypes));
        }
        
        console.log('✅ 颜色配置从localStorage恢复成功');
      }
    } catch (error) {
      console.warn('⚠️ 加载颜色配置失败:', error);
    }
  }

  // 保存颜色配置到localStorage
  private static saveColorsToStorage(): void {
    try {
      const colorData = {
        presetTypes: Object.fromEntries(
          DEFAULT_NODE_TYPES.map(type => [type.id, type.defaultColor])
        ),
        customTypes: Object.fromEntries(this.customTypeColors)
      };
      
      localStorage.setItem(this.STORAGE_KEY, JSON.stringify(colorData));
      console.log('✅ 颜色配置保存到localStorage成功');
    } catch (error) {
      console.warn('⚠️ 保存颜色配置失败:', error);
    }
  }

  // 清除所有颜色配置
  static clearStoredColors(): void {
    localStorage.removeItem(this.STORAGE_KEY);
    this.customTypeColors.clear();
    // 恢复默认颜色
    DEFAULT_NODE_TYPES[0].defaultColor = '#4ade80'; // PROMPT_BUILD
    DEFAULT_NODE_TYPES[1].defaultColor = '#3b82f6'; // CALL_LLM  
    DEFAULT_NODE_TYPES[2].defaultColor = '#f59e0b'; // HTTP_REQUEST
    DEFAULT_NODE_TYPES[3].defaultColor = '#8b5cf6'; // CODE_EXEC
  }

  // 生成随机颜色
  static generateRandomColor(): string {
    const randomIndex = Math.floor(Math.random() * this.RANDOM_COLORS.length);
    return this.RANDOM_COLORS[randomIndex];
  }

  // 验证颜色格式（十六进制）
  static isValidHexColor(color: string): boolean {
    return /^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$/.test(color);
  }

  // 获取颜色的对比文字颜色
  static getContrastTextColor(backgroundColor: string): string {
    // 简化的对比度计算
    const hex = backgroundColor.replace('#', '');
    const r = parseInt(hex.substr(0, 2), 16);
    const g = parseInt(hex.substr(2, 2), 16);
    const b = parseInt(hex.substr(4, 2), 16);
    const brightness = (r * 299 + g * 587 + b * 114) / 1000;
    return brightness > 128 ? '#000000' : '#ffffff';
  }
}

// 节点创建配置接口
export interface NodeCreationConfig {
  id: string;
  label: string;
  nodeType: string;
  position: { x: number; y: number };
  color?: string;
  customProperties?: {
    isCustomType?: boolean;
    customNodeType?: string;
    [key: string]: any;
  };
}

// 节点创建工厂
export class NodeFactory {
  // 创建新节点
  static createNode(config: NodeCreationConfig): any {
    let nodeTypeDef = DEFAULT_NODE_TYPES.find(type => type.id === config.nodeType);
    
    // 如果没有找到预定义类型，创建自定义类型定义
    if (!nodeTypeDef && config.customProperties?.isCustomType && config.customProperties.customNodeType) {
      const customType = config.customProperties.customNodeType;
      nodeTypeDef = {
        id: customType,
        label: customType,
        description: '自定义节点',
        icon: '⚙️', // 自定义类型使用固定图标
        defaultColor: '#64748b',
        className: 'node-custom',
        template: {
          taskType: customType,
          '@type': `custom.node.${customType}`,
          input: [],
          output: [],
          dependencies: []
        }
      };
    }
    
    if (!nodeTypeDef) {
      throw new Error(`未知的节点类型: ${config.nodeType}`);
    }

    // 确定节点颜色 - 修复自定义类型颜色bug
    let finalColor = config.color;
    
    if (!finalColor) {
      // 如果是自定义类型且有自定义类型定义，使用其默认颜色
      if (config.customProperties?.isCustomType && nodeTypeDef) {
        finalColor = nodeTypeDef.defaultColor;
      } else {
        // 否则使用ColorManager的默认颜色
        finalColor = ColorManager.getDefaultColor(config.nodeType);
      }
    }

    // 生成唯一的taskId（使用用户输入的label作为taskId）
    const taskId = config.label.trim();

    // 创建ReactFlow节点
    const reactFlowNode = {
      id: taskId, // 使用taskId作为节点ID
      type: 'custom', // 使用自定义节点类型
      position: config.position,
      data: {
        label: taskId, // label就是taskId
        taskId: taskId,
        taskType: nodeTypeDef.template.taskType,
        '@type': nodeTypeDef.template['@type'],
        input: [...nodeTypeDef.template.input],
        output: [...nodeTypeDef.template.output],
        dependencies: [...nodeTypeDef.template.dependencies],
        nodeTypeId: config.nodeType,
        icon: nodeTypeDef.icon,
        color: finalColor,
        textColor: ColorManager.getContrastTextColor(finalColor),
        className: nodeTypeDef.className,
        isCustomType: config.customProperties?.isCustomType || false,
        ...config.customProperties
      }
    };

    return reactFlowNode;
  }

  // 获取所有可用的节点类型
  static getAvailableNodeTypes(): NodeTypeDefinition[] {
    return [...DEFAULT_NODE_TYPES];
  }

  // 根据节点类型ID获取定义
  static getNodeTypeDefinition(nodeTypeId: string): NodeTypeDefinition | undefined {
    return DEFAULT_NODE_TYPES.find(type => type.id === nodeTypeId);
  }
}

// 导出类型定义已在上面interface前直接使用export完成 