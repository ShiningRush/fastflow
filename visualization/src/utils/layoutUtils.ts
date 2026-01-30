import type { Node, Edge } from 'reactflow';

/**
 * 智能布局算法选项
 */
export interface LayoutOptions {
  direction: 'TB' | 'LR'; // 布局方向：TB=从上到下，LR=从左到右
  nodeSpacing: { x: number; y: number }; // 节点间距
  levelSpacing: number; // 层级间距
  centerNodes: boolean; // 是否居中对齐节点
}

/**
 * 默认布局选项
 */
export const DEFAULT_LAYOUT_OPTIONS: LayoutOptions = {
  direction: 'TB', // Top to Bottom
  nodeSpacing: { x: 300, y: 180 }, // 适中的节点间距
  levelSpacing: 120, // 适中的层级间距
  centerNodes: true
};

/**
 * 计算节点的层级关系
 */
export function calculateNodeLevels(nodes: Node[], edges: Edge[]): Map<string, number> {
  const levels = new Map<string, number>();
  const inDegree = new Map<string, number>();
  const adjacencyList = new Map<string, string[]>();

  // 初始化
  nodes.forEach(node => {
    inDegree.set(node.id, 0);
    adjacencyList.set(node.id, []);
  });

  // 构建图的邻接表和入度统计
  edges.forEach(edge => {
    const from = edge.source;
    const to = edge.target;
    
    adjacencyList.get(from)?.push(to);
    inDegree.set(to, (inDegree.get(to) || 0) + 1);
  });

  // 使用拓扑排序计算层级
  const queue: string[] = [];
  
  // 找到所有入度为0的节点（根节点）
  inDegree.forEach((degree, nodeId) => {
    if (degree === 0) {
      levels.set(nodeId, 0);
      queue.push(nodeId);
    }
  });

  // 拓扑排序
  while (queue.length > 0) {
    const current = queue.shift()!;
    const currentLevel = levels.get(current)!;

    adjacencyList.get(current)?.forEach(neighbor => {
      const newDegree = (inDegree.get(neighbor) || 0) - 1;
      inDegree.set(neighbor, newDegree);

      if (newDegree === 0) {
        levels.set(neighbor, currentLevel + 1);
        queue.push(neighbor);
      }
    });
  }

  return levels;
}

/**
 * 根据层级分组节点
 */
export function groupNodesByLevel(nodes: Node[], levels: Map<string, number>): Map<number, Node[]> {
  const levelGroups = new Map<number, Node[]>();

  nodes.forEach(node => {
    const level = levels.get(node.id) ?? 0;
    if (!levelGroups.has(level)) {
      levelGroups.set(level, []);
    }
    levelGroups.get(level)!.push(node);
  });

  return levelGroups;
}

/**

 * 计算智能布局后的节点位置
 * - TB（纵向）：基于父子关系对齐，同层节点水平排布
 * - LR（横向）：按层级列排布，同层节点垂直排布（保持简单稳定，不做复杂偏移）
 */
export function calculateSmartLayout(
  nodes: Node[], 
  edges: Edge[], 
  options: LayoutOptions = DEFAULT_LAYOUT_OPTIONS
): Node[] {
  if (nodes.length === 0) return nodes;

  // 计算节点层级
  const levels = calculateNodeLevels(nodes, edges);
  const levelGroups = groupNodesByLevel(nodes, levels);
  
  // 横向布局：使用简单、稳定的按列排布逻辑，避免“混乱”
  if (options.direction === 'LR') {
    const updatedNodes: Node[] = [];
    const sortedLevels = Array.from(levelGroups.keys()).sort((a, b) => a - b);

    sortedLevels.forEach(level => {
      const levelNodes = levelGroups.get(level)!;
      const x = level * options.nodeSpacing.x; // 每一层是一个纵向“列”

      const totalHeight = (levelNodes.length - 1) * options.nodeSpacing.y;
      const startY = options.centerNodes ? -totalHeight / 2 : 0;

      levelNodes.forEach((node, index) => {
        const y = startY + index * options.nodeSpacing.y;
        updatedNodes.push({
          ...node,
          position: { x, y }
        });
      });
    });

    return updatedNodes;
  }

  // ===== TB 纵向布局：基于父子关系对齐 =====

  // 构建父子关系映射

  const parentsMap = new Map<string, string[]>();

  edges.forEach(edge => {





    if (!parentsMap.has(edge.target)) {
      parentsMap.set(edge.target, []);
    }
    parentsMap.get(edge.target)!.push(edge.source);
  });

  const nodePositions = new Map<string, { x: number; y: number }>();
  const updatedNodes: Node[] = [];

  // 按层级顺序处理节点
  const sortedLevels = Array.from(levelGroups.keys()).sort((a, b) => a - b);
  
  sortedLevels.forEach(level => {
    const levelNodes = levelGroups.get(level)!;
    const y = level * options.levelSpacing;
    
    if (level === 0) {
      // 第一层：居中排列
      const totalWidth = (levelNodes.length - 1) * options.nodeSpacing.x;
      const startX = options.centerNodes ? -totalWidth / 2 : 0;
      
      levelNodes.forEach((node, index) => {
        const x = startX + index * options.nodeSpacing.x;
        nodePositions.set(node.id, { x, y });
      });
    } else {
      // 其他层：基于父节点位置计算
      levelNodes.forEach(node => {
        const parents = parentsMap.get(node.id) || [];
        
        if (parents.length > 0) {
          // 计算所有父节点的平均X位置
          const parentPositions = parents
            .map(parentId => nodePositions.get(parentId))
            .filter(pos => pos !== undefined) as { x: number; y: number }[];
          
          if (parentPositions.length > 0) {
            const avgParentX = parentPositions.reduce((sum, pos) => sum + pos.x, 0) / parentPositions.length;
            nodePositions.set(node.id, { x: avgParentX, y });
          } else {
            // 如果父节点位置未知，使用默认位置
            nodePositions.set(node.id, { x: 0, y });
          }
        } else {
          // 没有父节点，使用默认位置
          nodePositions.set(node.id, { x: 0, y });
        }
      });
      

      // 调整同层节点避免重叠（从左到右扩开）
      const sortedLevelNodes = levelNodes
        .map(node => ({ node, pos: nodePositions.get(node.id)! }))
        .sort((a, b) => a.pos.x - b.pos.x);
      

      for (let i = 1; i < sortedLevelNodes.length; i++) {
        const prev = sortedLevelNodes[i - 1];
        const curr = sortedLevelNodes[i];
        const minDistance = options.nodeSpacing.x;
        
        if (curr.pos.x - prev.pos.x < minDistance) {
          curr.pos.x = prev.pos.x + minDistance;
          nodePositions.set(curr.node.id, curr.pos);
        }
      }
    }
  });

  // 生成最终节点列表
  nodes.forEach(node => {
    const pos = nodePositions.get(node.id);
    if (pos) {
      updatedNodes.push({
        ...node,
        position: pos
      });
    }
  });

  return updatedNodes;
}

/**
 * 节点网格对齐
 */
export function alignNodesToGrid(nodes: Node[], gridSize: number = 20): Node[] {
  return nodes.map(node => ({
    ...node,
    position: {
      x: Math.round(node.position.x / gridSize) * gridSize,
      y: Math.round(node.position.y / gridSize) * gridSize
    }
  }));
}

/**
 * 计算节点边界框
 */
export function getNodesBounds(nodes: Node[]): { 
  minX: number; 
  minY: number; 
  maxX: number; 
  maxY: number; 
  width: number; 
  height: number; 
} {
  if (nodes.length === 0) {
    return { minX: 0, minY: 0, maxX: 0, maxY: 0, width: 0, height: 0 };
  }

  const nodeWidth = 180; // 默认节点宽度
  const nodeHeight = 40;  // 默认节点高度

  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;

  nodes.forEach(node => {
    minX = Math.min(minX, node.position.x);
    minY = Math.min(minY, node.position.y);
    maxX = Math.max(maxX, node.position.x + nodeWidth);
    maxY = Math.max(maxY, node.position.y + nodeHeight);
  });

  return {
    minX,
    minY,
    maxX,
    maxY,
    width: maxX - minX,
    height: maxY - minY
  };
}

/**
 * 布局方向配置
 */
export const LAYOUT_DIRECTIONS = {
  'TB': { name: '纵向布局', icon: 'V' },
  'LR': { name: '横向布局', icon: 'H' }
} as const;

/**
 * 对齐选项配置
 */
export interface AlignmentOptions {
  snapToGrid: boolean;
  gridSize: number;
  snapToNodes: boolean;
  snapDistance: number;
  alignToEdges: boolean;
  alignToCenter: boolean;
  enableAnimation: boolean;
}

/**
 * 默认对齐选项
 */
export const DEFAULT_ALIGNMENT_OPTIONS: AlignmentOptions = {
  snapToGrid: true,
  gridSize: 20,
  snapToNodes: true,
  snapDistance: 10,
  alignToEdges: true,
  alignToCenter: true,
  enableAnimation: true
};

/**
 * 查找最近的对齐位置
 */
export function findNearestAlignment(
  draggedNode: Node,
  otherNodes: Node[],
  options: AlignmentOptions = DEFAULT_ALIGNMENT_OPTIONS
): { x: number; y: number; alignedTo?: string } {
  let bestX = draggedNode.position.x;
  let bestY = draggedNode.position.y;
  let alignedTo: string | undefined;

  // 网格对齐
  if (options.snapToGrid) {
    bestX = Math.round(bestX / options.gridSize) * options.gridSize;
    bestY = Math.round(bestY / options.gridSize) * options.gridSize;
    alignedTo = 'grid';
  }

  // 节点对齐
  if (options.snapToNodes && otherNodes.length > 0) {
    const nodeAlignment = findNodeAlignment(draggedNode, otherNodes, options);
    if (nodeAlignment.alignedTo) {
      bestX = nodeAlignment.x;
      bestY = nodeAlignment.y;
      alignedTo = nodeAlignment.alignedTo;
    }
  }

  return { x: bestX, y: bestY, alignedTo };
}

/**
 * 查找节点对齐
 */
function findNodeAlignment(
  draggedNode: Node,
  otherNodes: Node[],
  options: AlignmentOptions
): { x: number; y: number; alignedTo?: string } {
  let bestX = draggedNode.position.x;
  let bestY = draggedNode.position.y;
  let minDistance = Infinity;
  let alignedTo: string | undefined;

  const draggedCenterX = draggedNode.position.x + 90; // 节点宽度的一半
  const draggedCenterY = draggedNode.position.y + 20; // 节点高度的一半

  otherNodes.forEach(node => {
    if (node.id === draggedNode.id) return;

    const nodeCenterX = node.position.x + 90;
    const nodeCenterY = node.position.y + 20;

    // 水平对齐检查
    if (options.alignToCenter) {
      // 中心对齐
      const centerDistanceY = Math.abs(draggedCenterY - nodeCenterY);
      if (centerDistanceY <= options.snapDistance && centerDistanceY < minDistance) {
        bestY = nodeCenterY - 20; // 调整到中心对齐
        minDistance = centerDistanceY;
        alignedTo = `center-${node.id}`;
      }
    }

    if (options.alignToEdges) {
      // 顶部对齐
      const topDistance = Math.abs(draggedNode.position.y - node.position.y);
      if (topDistance <= options.snapDistance && topDistance < minDistance) {
        bestY = node.position.y;
        minDistance = topDistance;
        alignedTo = `top-${node.id}`;
      }

      // 底部对齐
      const bottomDistance = Math.abs(
        (draggedNode.position.y + 40) - (node.position.y + 40)
      );
      if (bottomDistance <= options.snapDistance && bottomDistance < minDistance) {
        bestY = node.position.y;
        minDistance = bottomDistance;
        alignedTo = `bottom-${node.id}`;
      }
    }

    // 垂直对齐检查
    if (options.alignToCenter) {
      // 中心对齐
      const centerDistanceX = Math.abs(draggedCenterX - nodeCenterX);
      if (centerDistanceX <= options.snapDistance && centerDistanceX < minDistance) {
        bestX = nodeCenterX - 90; // 调整到中心对齐
        minDistance = centerDistanceX;
        alignedTo = `center-${node.id}`;
      }
    }

    if (options.alignToEdges) {
      // 左边对齐
      const leftDistance = Math.abs(draggedNode.position.x - node.position.x);
      if (leftDistance <= options.snapDistance && leftDistance < minDistance) {
        bestX = node.position.x;
        minDistance = leftDistance;
        alignedTo = `left-${node.id}`;
      }

      // 右边对齐
      const rightDistance = Math.abs(
        (draggedNode.position.x + 180) - (node.position.x + 180)
      );
      if (rightDistance <= options.snapDistance && rightDistance < minDistance) {
        bestX = node.position.x;
        minDistance = rightDistance;
        alignedTo = `right-${node.id}`;
      }
    }
  });

  return { x: bestX, y: bestY, alignedTo };
}

/**
 * 批量对齐多个节点
 */
export function alignMultipleNodes(
  selectedNodes: Node[],
  alignType: 'left' | 'right' | 'top' | 'bottom' | 'centerX' | 'centerY' | 'distributeX' | 'distributeY'
): Node[] {
  if (selectedNodes.length < 2) return selectedNodes;

  const alignedNodes = [...selectedNodes];

  switch (alignType) {
    case 'left':
      const leftmostX = Math.min(...selectedNodes.map(n => n.position.x));
      alignedNodes.forEach(node => { node.position.x = leftmostX; });
      break;

    case 'right':
      const rightmostX = Math.max(...selectedNodes.map(n => n.position.x + 180));
      alignedNodes.forEach(node => { node.position.x = rightmostX - 180; });
      break;

    case 'top':
      const topmostY = Math.min(...selectedNodes.map(n => n.position.y));
      alignedNodes.forEach(node => { node.position.y = topmostY; });
      break;

    case 'bottom':
      const bottommostY = Math.max(...selectedNodes.map(n => n.position.y + 40));
      alignedNodes.forEach(node => { node.position.y = bottommostY - 40; });
      break;

    case 'centerX':
      const avgX = selectedNodes.reduce((sum, n) => sum + n.position.x + 90, 0) / selectedNodes.length;
      alignedNodes.forEach(node => { node.position.x = avgX - 90; });
      break;

    case 'centerY':
      const avgY = selectedNodes.reduce((sum, n) => sum + n.position.y + 20, 0) / selectedNodes.length;
      alignedNodes.forEach(node => { node.position.y = avgY - 20; });
      break;

    case 'distributeX':
      alignedNodes.sort((a, b) => a.position.x - b.position.x);
      const totalWidthX = alignedNodes[alignedNodes.length - 1].position.x - alignedNodes[0].position.x;
      const spacingX = totalWidthX / (alignedNodes.length - 1);
      alignedNodes.forEach((node, index) => {
        if (index > 0 && index < alignedNodes.length - 1) {
          node.position.x = alignedNodes[0].position.x + spacingX * index;
        }
      });
      break;

    case 'distributeY':
      alignedNodes.sort((a, b) => a.position.y - b.position.y);
      const totalHeightY = alignedNodes[alignedNodes.length - 1].position.y - alignedNodes[0].position.y;
      const spacingY = totalHeightY / (alignedNodes.length - 1);
      alignedNodes.forEach((node, index) => {
        if (index > 0 && index < alignedNodes.length - 1) {
          node.position.y = alignedNodes[0].position.y + spacingY * index;
        }
      });
      break;
  }

  return alignedNodes;
}

/**
 * 连线穿越检测和优化
 */

/**
 * 线段相交检测
 */
export function doLinesIntersect(
  line1Start: { x: number; y: number },
  line1End: { x: number; y: number },
  line2Start: { x: number; y: number },
  line2End: { x: number; y: number }
): boolean {
  const x1 = line1Start.x, y1 = line1Start.y;
  const x2 = line1End.x, y2 = line1End.y;
  const x3 = line2Start.x, y3 = line2Start.y;
  const x4 = line2End.x, y4 = line2End.y;

  const denom = (x1 - x2) * (y3 - y4) - (y1 - y2) * (x3 - x4);
  if (Math.abs(denom) < 1e-10) return false; // 平行线

  const t = ((x1 - x3) * (y3 - y4) - (y1 - y3) * (x3 - x4)) / denom;
  const u = -((x1 - x2) * (y1 - y3) - (y1 - y2) * (x1 - x3)) / denom;

  return t >= 0 && t <= 1 && u >= 0 && u <= 1;
}

/**
 * 检测连线是否穿越节点
 */
export function doesEdgeCrossNode(
  edgeStart: { x: number; y: number },
  edgeEnd: { x: number; y: number },
  node: Node,
  nodeWidth: number = 180,
  nodeHeight: number = 40
): boolean {
  const nodeLeft = node.position.x;
  const nodeRight = node.position.x + nodeWidth;
  const nodeTop = node.position.y;
  const nodeBottom = node.position.y + nodeHeight;

  // 检查连线是否与节点边界相交
  const nodeEdges = [
    { start: { x: nodeLeft, y: nodeTop }, end: { x: nodeRight, y: nodeTop } }, // 上边
    { start: { x: nodeRight, y: nodeTop }, end: { x: nodeRight, y: nodeBottom } }, // 右边
    { start: { x: nodeRight, y: nodeBottom }, end: { x: nodeLeft, y: nodeBottom } }, // 下边
    { start: { x: nodeLeft, y: nodeBottom }, end: { x: nodeLeft, y: nodeTop } } // 左边
  ];

  return nodeEdges.some(nodeEdge =>
    doLinesIntersect(edgeStart, edgeEnd, nodeEdge.start, nodeEdge.end)
  );
}

/**
 * 检测DAG中的连线穿越问题
 */
export interface EdgeCrossingInfo {
  edgeId: string;
  sourceNodeId: string;
  targetNodeId: string;
  crossingNodes: string[];
  severity: 'low' | 'medium' | 'high';
}

// 移除重复的calculateNodeLevels函数定义，使用已有的export版本

export function detectEdgeCrossings(
  nodes: Node[], 
  edges: Edge[],
  nodeWidth: number = 180,
  nodeHeight: number = 40
): EdgeCrossingInfo[] {
  const crossings: EdgeCrossingInfo[] = [];
  const nodeMap = new Map(nodes.map(n => [n.id, n]));
  
  // 🎯 关键优化：计算节点层级，只对跨层级连线进行穿越检测
  const nodeLevels = calculateNodeLevels(nodes, edges);

  edges.forEach(edge => {
    const sourceNode = nodeMap.get(edge.source);
    const targetNode = nodeMap.get(edge.target);
    
    if (!sourceNode || !targetNode) return;
    
    // 获取源节点和目标节点的层级
    const sourceLevel = nodeLevels.get(edge.source) || 0;
    const targetLevel = nodeLevels.get(edge.target) || 0;
    const levelSpan = Math.abs(targetLevel - sourceLevel);
    
    // ⭐ 核心优化：只对真正跨越2个或更多层级的连线进行穿越检测
    // 相邻层级连接（levelSpan <= 1）保持直接连线，不进行绕行优化
    // 这样可以避免正常的父子关系连线被误判为需要优化
    if (levelSpan <= 1) {
      // 静默跳过，不输出日志避免控制台污染
      return; // 跳过相邻层级的连线
    }

    console.log(`🔍 检测跨层级连线: ${edge.source}(L${sourceLevel}) -> ${edge.target}(L${targetLevel}), 跨度: ${levelSpan}`);

    // 计算连线的起点和终点（节点中心）
    const edgeStart = {
      x: sourceNode.position.x + nodeWidth / 2,
      y: sourceNode.position.y + nodeHeight / 2
    };
    const edgeEnd = {
      x: targetNode.position.x + nodeWidth / 2,
      y: targetNode.position.y + nodeHeight / 2
    };

    // 检查哪些节点被此连线穿越
    const crossingNodes: string[] = [];
    nodes.forEach(node => {
      if (node.id === edge.source || node.id === edge.target) return;
      
      const nodeLevel = nodeLevels.get(node.id) || 0;
      const minLevel = Math.min(sourceLevel, targetLevel);
      const maxLevel = Math.max(sourceLevel, targetLevel);
      
      // 只检查位于源节点和目标节点层级之间的节点
      if (nodeLevel > minLevel && nodeLevel < maxLevel) {
        if (doesEdgeCrossNode(edgeStart, edgeEnd, node, nodeWidth, nodeHeight)) {
          crossingNodes.push(node.id);
        }
      }
    });

    if (crossingNodes.length > 0) {
      // 根据穿越节点数量确定严重程度
      let severity: 'low' | 'medium' | 'high' = 'low';
      if (crossingNodes.length >= 3) severity = 'high';
      else if (crossingNodes.length >= 2) severity = 'medium';

      crossings.push({
        edgeId: edge.id,
        sourceNodeId: edge.source,
        targetNodeId: edge.target,
        crossingNodes,
        severity
      });
      
      console.log(`  ⚠️ 发现穿越: 穿越${crossingNodes.length}个节点 [${crossingNodes.join(', ')}], 严重程度: ${severity}`);
    }
  });

  return crossings;
}

/**
 * 优化布局以减少连线穿越
 */
export function optimizeLayoutForEdgeCrossings(
  nodes: Node[], 
  edges: Edge[], 
  _options: LayoutOptions = DEFAULT_LAYOUT_OPTIONS
): Node[] {
  // 检测连线穿越仅用于分析，不再进行位置调整
  const crossings = detectEdgeCrossings(nodes, edges);
  
  if (crossings.length > 0) {
    console.log(`检测到 ${crossings.length} 个跨层级连线穿越问题，但不进行自动调整以保持布局清晰`);
  }

  // 直接返回原节点，不进行任何调整
  return nodes;
}


/**
 * 复杂DAG连线穿越分析
 */
export interface DAGAnalysisResult {
  totalNodes: number;
  totalEdges: number;
  crossingEdges: EdgeCrossingInfo[];
  severitySummary: {
    high: number;
    medium: number;
    low: number;
  };
  suggestions: string[];
}

export function analyzeComplexDAG(nodes: Node[], edges: Edge[]): DAGAnalysisResult {
  const crossings = detectEdgeCrossings(nodes, edges);
  
  const severitySummary = {
    high: crossings.filter(c => c.severity === 'high').length,
    medium: crossings.filter(c => c.severity === 'medium').length,
    low: crossings.filter(c => c.severity === 'low').length
  };

  const suggestions: string[] = [];
  
  if (severitySummary.high > 0) {
    suggestions.push(`发现 ${severitySummary.high} 个严重连线穿越问题，建议调整布局方向或节点位置`);
  }
  
  if (crossings.length > nodes.length * 0.3) {
    suggestions.push('连线穿越过多，建议使用分层布局或增加节点间距');
  }
  
  if (nodes.length > 20) {
    suggestions.push('节点数量较多，建议使用分组展示或折叠部分节点');
  }

  return {
    totalNodes: nodes.length,
    totalEdges: edges.length,
    crossingEdges: crossings,
    severitySummary,
    suggestions
  };
}

/**
 * 增强的连线穿越优化算法 - 针对复杂DAG
 */
export function optimizeComplexDAGLayout(
  nodes: Node[], 
  edges: Edge[], 
  options: LayoutOptions = DEFAULT_LAYOUT_OPTIONS
): Node[] {
  // 1. 分析当前问题
  const analysis = analyzeComplexDAG(nodes, edges);
  
  console.log('复杂DAG分析结果:', analysis);
  
  // 2. 仅使用基础智能布局，不进行额外的穿越优化
  const optimizedNodes = calculateSmartLayout(nodes, edges, options);
  
  // 直接返回基础布局结果，不进行节点位置调整
  return optimizedNodes;
}


/**
 * 平滑动画移动节点到目标位置
 */
export function animateNodeToPosition(
  node: Node,
  targetPosition: { x: number; y: number },
  _duration: number = 300
): Promise<Node> {
  // 简化版本：直接返回目标位置的节点
  return Promise.resolve({
    ...node,
    position: targetPosition
  });
}
