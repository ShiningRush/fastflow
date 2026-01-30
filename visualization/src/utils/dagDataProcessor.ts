import type { DAGData, DAGNode, DAGEdge, TaskNodeType } from '../types';
  
/**
 * DAG 定义接口 - 完整的工作流定义
 */
export interface DAGDefinition {
  dag_id?: string;
  name?: string;
  desc?: string;
  vars?: Array<{
    name: string;
    desc?: string;
    default_value?: any;
  }>;
  tasks: Task[];
}

/**
 * 任务定义接口
 */
export interface Task {
  id: string;
  name?: string;
  action_name: string;
  depend_on?: string[];
  timeout_secs?: number;
  params?: Array<{ key: string; value?: any; sub_params?: any[] }>;
  outputs?: Array<{ key: string; value: string }>;
  pre_checks?: Array<{
    name: string;
    check?: {
      active_action?: string;
      conditions?: any[];
    };
  }>;
  position?: { x: number; y: number }; // 节点位置，用于保存和恢复布局
}

/**
 * 获取节点类型颜色
 */
function getNodeTypeColor(actionName: string): string {
  const colorMap: Record<string, string> = {
    'start': '#4CAF50',           // 绿色 - 开始节点
    'end': '#FF5252',             // 深红色 - 结束节点
    'default': '#757575'          // 灰色 - 默认
  };
  return colorMap[actionName] || colorMap.default;
}

/**
 * 获取简化的节点类型
 */
function getSimplifiedNodeType(actionName: string): TaskNodeType {
  if (actionName === 'start' || actionName === 'end') {
    return 'default';
  }
  return 'default';
}

/**
 * 计算节点位置 - 分层布局
 */
function calculateNodePositions(tasks: Task[]): Record<string, { x: number; y: number }> {
  const positions: Record<string, { x: number; y: number }> = {};
  const taskMap = new Map<string, Task>();
  const inDegree = new Map<string, number>();
  const layers: string[][] = [];

  // 建立任务映射和入度统计
  tasks.forEach(task => {
    taskMap.set(task.id, task);
    inDegree.set(task.id, 0);
  });

  // 计算入度
  tasks.forEach(task => {
        if (task.depend_on && task.depend_on.length > 0) {
      task.depend_on.forEach((_dep: string) => {
        const current = inDegree.get(task.id) || 0;
        inDegree.set(task.id, current + 1);
      });
    }
  });

  // 拓扑排序分层
  const queue: string[] = [];
  tasks.forEach(task => {
    if ((inDegree.get(task.id) || 0) === 0) {
      queue.push(task.id);
    }
  });

  while (queue.length > 0) {
    const currentLayer: string[] = [...queue];
    layers.push(currentLayer);
    queue.length = 0;

    currentLayer.forEach(taskId => {
      const task = taskMap.get(taskId);
      if (task) {
        // 找到依赖此任务的其他任务
        tasks.forEach(otherTask => {
          if (otherTask.depend_on?.includes(taskId)) {
            const degree = inDegree.get(otherTask.id) || 0;
            inDegree.set(otherTask.id, degree - 1);
            if (degree - 1 === 0) {
              queue.push(otherTask.id);
            }
          }
        });
      }
    });
  }

  // 计算位置
  const layerHeight = 120;
  const horizontalSpacing = 250;

  layers.forEach((layer, layerIndex) => {
    const layerY = layerIndex * layerHeight;
    layer.forEach((taskId, nodeIndex) => {
      const totalNodesInLayer = layer.length;
      const startX = -(totalNodesInLayer - 1) * horizontalSpacing / 2;
      const nodeX = startX + nodeIndex * horizontalSpacing;
      
      positions[taskId] = {
        x: nodeX,
        y: layerY
      };
    });
  });

  return positions;
}

/**
 * 将嵌套的 params (包含 sub_params) 扁平化为简单的 { key, value } 数组
 * 支持任意深度的嵌套
 * 示例：
 *   { key: "task_json", sub_params: [{ key: "bgm", sub_params: [{ key: "url", value: "xxx" }] }] }
 * =>
 *   [{ key: "task_json.bgm.url", value: "xxx" }]
 */
export function flattenParams(params: any[] | undefined | null): Array<{ key: string; value?: any; sub_params?: any[] }> {
  if (!Array.isArray(params)) return [];

  const result: Array<{ key: string; value?: any }> = [];

  // 递归函数：处理嵌套的参数
  const flattenRecursive = (items: any[], prefix: string = '') => {
    items.forEach((item) => {
      if (!item || typeof item.key !== 'string') return;

      const currentKey = prefix ? `${prefix}.${item.key}` : item.key;
      const hasSubParams = Array.isArray(item.sub_params) && item.sub_params.length > 0;

      if (hasSubParams) {
        // 递归处理子参数
        flattenRecursive(item.sub_params, currentKey);
      } else {
        // 没有子参数时，添加到结果
        result.push({
          key: currentKey,
          value: item.value,
        });
      }
    });
  };

  flattenRecursive(params);
  return result;
}

/**
 * 将扁平的 params (key 可能是 a 或 a.b 或 a.b.c...) 还原为嵌套结构，兼容后端期待的 params/sub_params 格式
 * 支持任意深度的嵌套
 * 示例：
 *   [{ key: "task_json.bgm.url", value: "xxx" }]
 * =>
 *   { key: "task_json", sub_params: [{ key: "bgm", sub_params: [{ key: "url", value: "xxx" }] }] }
 */
export function restoreParams(flatParams: Array<{ key: string; value?: any }> | undefined | null): any[] {
  if (!Array.isArray(flatParams)) return [];

  const rootMap = new Map<string, any>();

  flatParams.forEach((p) => {
    const rawKey = (p.key || '').trim();
    if (!rawKey) return;

    const parts = rawKey.split('.');

    if (parts.length === 1) {
      // 顶层 key
      rootMap.set(parts[0], { key: parts[0], value: p.value });
    } else {
      // 支持多层嵌套：a.b.c...
      const rootKey = parts[0];
      let root = rootMap.get(rootKey);
      
      if (!root) {
        root = { key: rootKey, sub_params: [] as any[] };
        rootMap.set(rootKey, root);
      }

      // 递归创建嵌套结构
      let current = root;
      for (let i = 1; i < parts.length; i++) {
        const partKey = parts[i];
        const isLastPart = i === parts.length - 1;

        if (isLastPart) {
          // 最后一层：直接设置值
          const subParams = (current.sub_params || []) as any[];
          const existing = subParams.find((sp) => sp.key === partKey);
          if (existing) {
            existing.value = p.value;
          } else {
            subParams.push({ key: partKey, value: p.value });
          }
          current.sub_params = subParams;
        } else {
          // 中间层：创建或获取嵌套对象
          const subParams = (current.sub_params || []) as any[];
          let next = subParams.find((sp) => sp.key === partKey);
          
          if (!next) {
            next = { key: partKey, sub_params: [] as any[] };
            subParams.push(next);
          }
          
          current.sub_params = subParams;
          current = next;
        }
      }
    }
  });

  return Array.from(rootMap.values());
}

/**
 * 解析 DAG 定义为可视化数据
 */
export function parseWorkflowToDAG(jsonData: any): DAGData {
  try {
    let tasks: Task[] = [];

    // 检查是否是完整的 DAG 定义
    if (jsonData && typeof jsonData === 'object' && jsonData.tasks && Array.isArray(jsonData.tasks)) {
      tasks = jsonData.tasks;
    } else if (Array.isArray(jsonData)) {
      tasks = jsonData;
    } else if (jsonData && typeof jsonData === 'object') {
      tasks = [jsonData];
    } else {
      throw new Error('无效的工作流数据格式');
    }

    if (tasks.length === 0) {
      throw new Error('工作流数据为空');
    }

    // 不再自动插入开始(__START__)和结束(__END__)节点，直接使用原始任务列表
    const allTasks = [...tasks];

    // 验证任务结构
    allTasks.forEach((task, index) => {
      if (!task.id) {
        throw new Error(`任务 ${index + 1} 缺少 id 字段`);
      }
      if (!task.action_name) {
        throw new Error(`任务 ${index + 1} 缺少 action_name 字段`);
      }
    });

    // 计算任务位置 - 优先使用保存的位置，否则自动计算
    const positions = calculateNodePositions(allTasks);
    
    // 使用保存的位置覆盖计算的位置
    let hasCustomPositions = false;
    allTasks.forEach(task => {
      if (task.position && typeof task.position.x === 'number' && typeof task.position.y === 'number') {
        positions[task.id] = task.position;
        hasCustomPositions = true;
        console.log(`任务 ${task.id} 使用自定义位置:`, task.position);
      }
    });
    
    if (hasCustomPositions) {
      console.log('检测到自定义位置信息，将使用保存的位置');
    } else {
      console.log('未检测到位置信息，使用自动计算的位置');
    }

    // 转换为 DAG 节点
    const dagNodes: DAGNode[] = allTasks.map((task, index) => {
      const nodeType = getSimplifiedNodeType(task.action_name);
      const position = positions[task.id] || { x: 0, y: index * 100 };
      // 节点颜色交给前端随机生成，这里只传递类型
      const nodeColor = getNodeTypeColor(task.action_name);
      const label = task.name || task.id;

      return {
        id: task.id,
        type: nodeType,
        label: label,
        data: {
          original: task,
          index: index,
          actionName: task.action_name,
          color: nodeColor,
          inputCount: task.params?.length || 0,
          outputCount: task.outputs?.length || 0,
          timeoutSecs: task.timeout_secs,
          preChecks: task.pre_checks
        },
        position,
        style: {
          backgroundColor: nodeColor,
          border: '2px solid #ffffff',
          borderRadius: '8px',
          width: '220px',
          height: '60px',
          fontSize: '13px',
          fontWeight: 'bold',
          padding: '12px 16px'
        }
      };
    });

    // 生成边（连接线）
    const dagEdges: DAGEdge[] = [];
    allTasks.forEach(task => {
      if (task.depend_on && task.depend_on.length > 0) {
        task.depend_on.forEach(depId => {
          // 验证依赖任务是否存在
          const sourceTask = allTasks.find(t => t.id === depId);
          if (sourceTask) {
            // 检查是否有前置检查条件
            const preCheck = task.pre_checks && task.pre_checks.length > 0 ? task.pre_checks[0] : null;

            const edge: DAGEdge = {
              id: `${depId}->${task.id}`,
              source: depId,
              target: task.id,
              type: 'smoothstep'
            };
            
            // 如果有前置检查，添加条件信息
            if (preCheck) {
              edge.label = preCheck.name || '条件分支';
              edge.condition = {
                type: 'conditional',
                label: preCheck.name,
                checkName: preCheck.name,
                activeAction: preCheck.check?.active_action,
                conditions: preCheck.check?.conditions
              };
            }
            
            dagEdges.push(edge);
          }
        });
      }
    });

    return {
      nodes: dagNodes,
      edges: dagEdges,
      metadata: {
        totalNodes: dagNodes.length,
        totalEdges: dagEdges.length,
        processedAt: new Date().toISOString()
      }
    };
  } catch (error) {
    console.error('DAG 数据解析错误:', error);
    throw new Error(`DAG 数据解析失败: ${error instanceof Error ? error.message : '未知错误'}`);
  }
}

/**
 * 验证 JSON 数据是否为有效的工作流格式
 */
export function validateWorkflowData(jsonData: any): { isValid: boolean; error?: string } {
  try {
    if (!jsonData) {
      return { isValid: false, error: 'JSON 数据为空' };
    }

    let tasks: any[] = [];

    // 检查是否是 DAGDefinition 格式
    if (jsonData && typeof jsonData === 'object' && jsonData.tasks && Array.isArray(jsonData.tasks)) {
      tasks = jsonData.tasks;
    } else if (Array.isArray(jsonData)) {
      tasks = jsonData;
    } else if (typeof jsonData === 'object') {
      tasks = [jsonData];
    } else {
      return { isValid: false, error: '无效的数据格式' };
    }

    if (tasks.length === 0) {
      return { isValid: false, error: '工作流任务数量为 0' };
    }

    // 检查必需字段
    for (let i = 0; i < tasks.length; i++) {
      const task = tasks[i];

      if (!task.id) {
        return {
          isValid: false,
          error: `任务 ${i + 1} 缺少 id 字段`
        };
      }

      if (!task.action_name) {
        return {
          isValid: false,
          error: `任务 ${i + 1} 缺少 action_name 字段`
        };
      }
    }

    return { isValid: true };
  } catch (error) {
    return {
      isValid: false,
      error: `验证失败: ${error instanceof Error ? error.message : '未知错误'}`
    };
  }
}

/**
 * 生成示例 DAG 数据
 */
export function generateExampleData(): DAGDefinition {
  return {
    dag_id: 'example_dag',
    name: '示例工作流',
    desc: '这是一个示例 DAG',
    vars: [
      {"name": "request_id","default_value": "","desc": "请求id，推送消息时会带上"},
      {"name": "userid","default_value": "","desc": "用户id"}
    ],
    tasks: [
      {
        id: 'task_1',
        name: '任务 1',
        action_name: 'action_1',
        depend_on: [],
        timeout_secs: 3600,
        params: [
          { key: 'param1', value: 'value1' }
        ],
        outputs: [
          { key: 'output1', value: 'data.result' }
        ]
      },
      {
        id: 'task_2',
        name: '任务 2',
        action_name: 'action_2',
        depend_on: ['task_1'],
        timeout_secs: 1800,
        params: [
          { key: 'param2', value: 'value2' }
        ],
        outputs: [
          { key: 'output2', value: 'data.result' }
        ],
        pre_checks: [{
          name: "有用户id，执行",
          check: {
          active_action: "skip",
            conditions: [
              {
                source: "vars",
                key: "lyric_hash",
                value: [
                  ""
                ],
                operator: "in"
              }
            ]
          }
        }
        ]
      }
    ]
  };
} 