/**
 * 连线重叠优化工具
 * 用于检测和优化DAG中重叠的连线显示
 */

import type { Node, Edge } from 'reactflow';
import { Position } from 'reactflow';

export interface EdgePath {
  id: string;
  sourceNode: Node;
  targetNode: Node;
  sourcePosition: Position;
  targetPosition: Position;
  path: Point[];
  originalEdge: Edge;
}

export interface Point {
  x: number;
  y: number;
}

export interface EdgeGroup {
  edges: EdgePath[];
  isOverlapping: boolean;
  adjustmentType: 'offset' | 'curve' | 'bundle';
}

export interface EdgeOptimizationOptions {
  enabled: boolean;
  offsetDistance: number; // 偏移距离
  curveIntensity: number; // 曲线强度
  groupThreshold: number; // 分组阈值
  visualEnhancement: boolean; // 视觉增强
}

export const DEFAULT_EDGE_OPTIMIZATION_OPTIONS: EdgeOptimizationOptions = {
  enabled: true,
  offsetDistance: 30, // 增加偏移距离，让重叠更明显
  curveIntensity: 0.3,
  groupThreshold: 15, // 增加检测阈值，更容易识别重叠
  visualEnhancement: true
};

/**
 * 计算节点连接点的实际坐标
 */
export function getConnectionPoint(node: Node, position: Position): Point {
  const nodeX = node.position.x;
  const nodeY = node.position.y;
  const nodeWidth = node.width || 180;
  const nodeHeight = node.height || 40;

  switch (position) {
    case Position.Top:
      return { x: nodeX + nodeWidth / 2, y: nodeY };
    case Position.Bottom:
      return { x: nodeX + nodeWidth / 2, y: nodeY + nodeHeight };
    case Position.Left:
      return { x: nodeX, y: nodeY + nodeHeight / 2 };
    case Position.Right:
      return { x: nodeX + nodeWidth, y: nodeY + nodeHeight / 2 };
    default:
      return { x: nodeX + nodeWidth / 2, y: nodeY + nodeHeight / 2 };
  }
}

/**
 * 计算两点之间的直线路径
 */
export function calculateStraightPath(start: Point, end: Point): Point[] {
  return [start, end];
}

/**
 * 计算平滑路径的控制点
 */
export function calculateSmoothPath(
  start: Point, 
  end: Point, 
  sourcePosition: Position, 
  targetPosition: Position
): Point[] {
  const dx = end.x - start.x;
  const dy = end.y - start.y;
  
  // 控制点偏移量
  const controlOffset = Math.max(Math.abs(dx), Math.abs(dy)) * 0.3;
  
  let controlPoint1: Point;
  let controlPoint2: Point;
  
  if (sourcePosition === Position.Right && targetPosition === Position.Left) {
    // 水平连接
    controlPoint1 = { x: start.x + controlOffset, y: start.y };
    controlPoint2 = { x: end.x - controlOffset, y: end.y };
  } else if (sourcePosition === Position.Bottom && targetPosition === Position.Top) {
    // 垂直连接
    controlPoint1 = { x: start.x, y: start.y + controlOffset };
    controlPoint2 = { x: end.x, y: end.y - controlOffset };
  } else {
    // 其他情况
    controlPoint1 = { x: start.x + dx * 0.3, y: start.y + dy * 0.1 };
    controlPoint2 = { x: end.x - dx * 0.3, y: end.y - dy * 0.1 };
  }
  
  return [start, controlPoint1, controlPoint2, end];
}

/**
 * 检测两条连线路径是否重叠
 */
export function detectPathOverlap(path1: Point[], path2: Point[], threshold: number = 10): boolean {
  // 简化版本：检测连线中点距离
  const mid1 = getMidPoint(path1);
  const mid2 = getMidPoint(path2);
  
  const distance = Math.sqrt(
    Math.pow(mid1.x - mid2.x, 2) + Math.pow(mid1.y - mid2.y, 2)
  );
  
  return distance < threshold;
}

/**
 * 获取路径中点
 */
export function getMidPoint(path: Point[]): Point {
  if (path.length === 2) {
    // 直线
    return {
      x: (path[0].x + path[1].x) / 2,
      y: (path[0].y + path[1].y) / 2
    };
  } else if (path.length === 4) {
    // 贝塞尔曲线，取起点和终点的中点作为近似
    return {
      x: (path[0].x + path[3].x) / 2,
      y: (path[0].y + path[3].y) / 2
    };
  }
  
  // 多点路径，取中间点
  const midIndex = Math.floor(path.length / 2);
  return path[midIndex];
}

/**
 * 为重叠连线生成偏移路径
 */
export function generateOffsetPath(
  originalPath: Point[], 
  offsetDistance: number, 
  index: number,
  totalCount: number
): Point[] {
  if (originalPath.length < 2) return originalPath;
  
  // 计算偏移方向
  const start = originalPath[0];
  const end = originalPath[originalPath.length - 1];
  
  // 垂直于连线方向的单位向量
  const dx = end.x - start.x;
  const dy = end.y - start.y;
  const length = Math.sqrt(dx * dx + dy * dy);
  
  if (length === 0) return originalPath;
  
  const perpX = -dy / length;
  const perpY = dx / length;
  
  // 计算当前连线的偏移量
  const groupOffset = (index - (totalCount - 1) / 2) * offsetDistance;
  
  return originalPath.map(point => ({
    x: point.x + perpX * groupOffset,
    y: point.y + perpY * groupOffset
  }));
}

/**
 * 分析连线并创建EdgePath对象
 */
export function createEdgePaths(
  edges: Edge[], 
  nodes: Node[], 
  sourcePosition: Position, 
  targetPosition: Position
): EdgePath[] {
  return edges.map(edge => {
    const sourceNode = nodes.find(n => n.id === edge.source);
    const targetNode = nodes.find(n => n.id === edge.target);
    
    if (!sourceNode || !targetNode) {
      console.warn(`找不到连线 ${edge.id} 的源节点或目标节点`);
      return null;
    }
    
    const sourcePoint = getConnectionPoint(sourceNode, sourcePosition);
    const targetPoint = getConnectionPoint(targetNode, targetPosition);
    
    const path = calculateSmoothPath(sourcePoint, targetPoint, sourcePosition, targetPosition);
    
    return {
      id: edge.id,
      sourceNode,
      targetNode,
      sourcePosition,
      targetPosition,
      path,
      originalEdge: edge
    };
  }).filter(Boolean) as EdgePath[];
}

/**
 * 检测并分组重叠连线
 */
export function groupOverlappingEdges(
  edgePaths: EdgePath[], 
  options: EdgeOptimizationOptions
): EdgeGroup[] {
  const groups: EdgeGroup[] = [];
  const processed = new Set<string>();
  
  edgePaths.forEach(edgePath => {
    if (processed.has(edgePath.id)) return;
    
    const group: EdgeGroup = {
      edges: [edgePath],
      isOverlapping: false,
      adjustmentType: 'offset'
    };
    
    // 查找与当前连线重叠的其他连线
    edgePaths.forEach(otherEdgePath => {
      if (otherEdgePath.id === edgePath.id || processed.has(otherEdgePath.id)) return;
      
      if (detectPathOverlap(edgePath.path, otherEdgePath.path, options.groupThreshold)) {
        group.edges.push(otherEdgePath);
        processed.add(otherEdgePath.id);
      }
    });
    
    group.isOverlapping = group.edges.length > 1;
    processed.add(edgePath.id);
    groups.push(group);
  });
  

  
  return groups;
}

/**
 * 生成优化后的连线样式
 */
export function generateOptimizedEdges(
  groups: EdgeGroup[],
  options: EdgeOptimizationOptions
): Edge[] {
  const optimizedEdges: Edge[] = [];
  
  groups.forEach(group => {
    if (!group.isOverlapping) {
      // 无重叠，保持原样
      optimizedEdges.push(...group.edges.map(ep => ep.originalEdge));
      return;
    }
    
    // 有重叠，应用优化
    group.edges.forEach((edgePath, index) => {
      const offsetPath = generateOffsetPath(
        edgePath.path,
        options.offsetDistance,
        index,
        group.edges.length
      );
      
      // 生成优化后的连线
      const optimizedEdge: Edge = {
        ...edgePath.originalEdge,
        style: {
          ...edgePath.originalEdge.style,
          strokeWidth: options.visualEnhancement ? 2.5 + index * 0.5 : 2, // 更明显的线宽差异
          stroke: options.visualEnhancement 
            ? getEdgeColor(index, group.edges.length) 
            : '#94a3b8',
          opacity: options.visualEnhancement ? 0.9 : 0.8, // 增加不透明度
          strokeDasharray: index > 0 && options.visualEnhancement ? '5,5' : undefined // 为重叠线添加虚线样式
        },
        // 添加自定义路径信息（如果ReactFlow支持）
        data: {
          ...edgePath.originalEdge.data,
          optimizedPath: offsetPath,
          isOptimized: true,
          groupIndex: index,
          groupSize: group.edges.length
        }
      };
      
      optimizedEdges.push(optimizedEdge);
    });
  });
  
  return optimizedEdges;
}

/**
 * 为重叠连线生成不同颜色 - 增强版本，更鲜明的对比度
 */
export function getEdgeColor(index: number, _total: number): string {
  const colors = [
    '#94a3b8', // 默认灰色
    '#2563eb', // 亮蓝色 (更鲜艳)
    '#16a34a', // 亮绿色 (更鲜艳)
    '#ea580c', // 亮橙色 (更鲜艳)
    '#dc2626', // 亮红色 (更鲜艳)
    '#7c3aed', // 亮紫色 (更鲜艳)
    '#0891b2', // 亮青色 (更鲜艳)
    '#d97706'  // 亮黄橙色 (更鲜艳)
  ];
  
  return colors[index % colors.length];
}

/**
 * 主要的连线优化函数
 */
export function optimizeEdges(
  edges: Edge[],
  nodes: Node[],
  sourcePosition: Position,
  targetPosition: Position,
  options: EdgeOptimizationOptions = DEFAULT_EDGE_OPTIMIZATION_OPTIONS
): Edge[] {
  if (!options.enabled || edges.length === 0) {
    return edges;
  }
  

  
  // 1. 创建连线路径
  const edgePaths = createEdgePaths(edges, nodes, sourcePosition, targetPosition);
  
  // 2. 检测并分组重叠连线
  const groups = groupOverlappingEdges(edgePaths, options);
  
  // 3. 生成优化后的连线
  const optimizedEdges = generateOptimizedEdges(groups, options);
  

  
  return optimizedEdges;
}