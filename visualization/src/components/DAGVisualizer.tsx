import React, { useState, useCallback, useRef } from 'react';
import ReactFlow, {         
  MiniMap, 
  Controls, 
  ControlButton,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  MarkerType,
  useReactFlow,
  addEdge,
  ConnectionLineType,
  ConnectionMode,
  Position
} from 'reactflow';
import type { Node, Edge, Connection } from 'reactflow';
import 'reactflow/dist/style.css';
import { useApp } from '../context/AppContext';
import NodeCreationDialog from './NodeCreationDialog';
import { DEFAULT_NODE_TYPES, ColorManager } from '../utils/nodeTypeManager';



import { 
  calculateSmartLayout, 
  alignNodesToGrid, 
  DEFAULT_LAYOUT_OPTIONS, 
  LAYOUT_DIRECTIONS,
  findNearestAlignment,
  DEFAULT_ALIGNMENT_OPTIONS,
  detectEdgeCrossings,
  optimizeLayoutForEdgeCrossings,
  optimizeComplexDAGLayout,
  analyzeComplexDAG
} from '../utils/layoutUtils';
import { flattenParams, restoreParams } from '../utils/dagDataProcessor';
import type { LayoutOptions, AlignmentOptions } from '../utils/layoutUtils';
import {
  calculateUniformNodeSize,
  DEFAULT_NODE_TEXT_CONFIG
} from '../utils/textUtils';
// 移除连线重叠优化相关导入，功能已被智能布局囊括
// import {
//   optimizeEdges,
//   DEFAULT_EDGE_OPTIMIZATION_OPTIONS
// } from '../utils/edgeOptimization';
// import type { EdgeOptimizationOptions } from '../utils/edgeOptimization';

const DAGVisualizer: React.FC = () => {
  const { state, dispatch, loadDAGData, setReactFlowInstance, isVarsEditorOpen, setIsVarsEditorOpen } = useApp();
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const reactFlowInstance = useReactFlow();

  // 将ReactFlow实例传递给AppContext
  React.useEffect(() => {
    if (reactFlowInstance) {
      setReactFlowInstance(reactFlowInstance);
    }
  }, [reactFlowInstance, setReactFlowInstance]);

  // 节点创建对话框状态
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [dialogPosition, setDialogPosition] = useState<{ x: number; y: number }>({ x: 0, y: 0 });
  const [contextMenuPosition, setContextMenuPosition] = useState<{ x: number; y: number } | null>(null);
  
  // 节点编辑状态
  const [editingNodeId, setEditingNodeId] = useState<string | null>(null);
  const [editingNodeLabel, setEditingNodeLabel] = useState<string>('');
  // 节点的业务名称（对应 JSON 里的 task.name）
  const [editingNodeName, setEditingNodeName] = useState<string>('');
  const [editingNodeType, setEditingNodeType] = useState<string>('');
  const [editingNodeColor, setEditingNodeColor] = useState<string>('');
  const [editingIsCustomType, setEditingIsCustomType] = useState<boolean>(false);
  const [editingCustomNodeType, setEditingCustomNodeType] = useState<string>('');
  const [editingLoadedNodeTypes, setEditingLoadedNodeTypes] = useState<import('../types').NodeTypeDefinition[]>([]);
  const [editingSelectedLoadedType, setEditingSelectedLoadedType] = useState<string>('');
  const [editingNodeParams, setEditingNodeParams] = useState<any[]>([]);
  const [editingNodeOutputs, setEditingNodeOutputs] = useState<any[]>([]);
  const [editingNodePreChecks, setEditingNodePreChecks] = useState<any[]>([]);
  const [editingNodeTimeoutSecs, setEditingNodeTimeoutSecs] = useState<number>(3600);

  // 前置检查展开状态（按索引记录哪些 pre_check 展开条件）
  const [expandedPreCheckIndexes, setExpandedPreCheckIndexes] = useState<Set<number>>(new Set());
  
  // 抽屉宽度状态
  const [drawerWidth, setDrawerWidth] = useState<number>(480);
  const [isResizing, setIsResizing] = useState<boolean>(false);
  
  // 变量编辑状态
  const [vars, setVars] = useState<Array<{ name: string; desc?: string; default_value?: any }>>([]);
  const [varsHasChanges, setVarsHasChanges] = useState(false);
  
  // 工作流基础信息状态
  const [workflowInfo, setWorkflowInfo] = useState<{
    dag_id?: string;
    name?: string;
    desc?: string;
  }>({});

  // 参数值变量选择器状态
  const [paramVarPicker, setParamVarPicker] = useState<{
    open: boolean;
    paramIndex: number | null;
    field: 'paramValue' | 'paramKey' | 'outputValue' | null;
    anchorRect: DOMRect | null;
    options: { label: string; value: string }[];
    searchTerm: string; // 新增：搜索过滤词
  }>({ open: false, paramIndex: null, field: null, anchorRect: null, options: [], searchTerm: '' });
  
  // 批量颜色控制状态
  const [batchColorMode, setBatchColorMode] = useState<boolean>(false);
  
  // 连线删除提示状态
  const [showDeleteHint, setShowDeleteHint] = useState<boolean>(false);

  // 智能布局状态
  const [isLayouting, setIsLayouting] = useState<boolean>(false);
  const [layoutOptions, setLayoutOptions] = useState<LayoutOptions>({
    ...DEFAULT_LAYOUT_OPTIONS,
    direction: 'TB' // 默认使用纵向布局
  });
  
  // 节点对齐状态
  const [alignmentOptions, setAlignmentOptions] = useState<AlignmentOptions>(DEFAULT_ALIGNMENT_OPTIONS);
  
  // 使用ref来存储智能布局函数，避免依赖循环
  const smartLayoutRef = useRef<(() => void) | null>(null);

  // 文本自适应布局状态 (T=固定, U=统一)
  const [textAdaptiveMode, setTextAdaptiveMode] = useState<'uniform' | 'fixed'>('fixed');
  const textConfig = DEFAULT_NODE_TEXT_CONFIG; // 使用默认配置

  // 移除连线重叠优化状态，功能已被智能布局囊括
  // const [edgeOptimizationOptions, setEdgeOptimizationOptions] = useState<EdgeOptimizationOptions>(DEFAULT_EDGE_OPTIMIZATION_OPTIONS);

  // 连线选中和节点高亮状态
  const [selectedEdge, setSelectedEdge] = useState<string | null>(null);
  const [highlightedNodes, setHighlightedNodes] = useState<Set<string>>(new Set());
  
  // DAG分析状态
  const [showAnalysisModal, setShowAnalysisModal] = useState<boolean>(false);
  const [analysisResult, setAnalysisResult] = useState<any>(null);

  // 参数变量选择弹窗的ref，用于检测点击外部
  const paramVarPickerRef = useRef<HTMLDivElement>(null);

  // 当DAG数据变化时更新节点和边
  React.useEffect(() => {
    if (state.dagData) {
      // 检查是否有自定义位置 - 检查原始position字段
      const hasPositions = state.dagData.nodes.some(node => 
        node.position && 
        typeof node.position.x === 'number' && 
        typeof node.position.y === 'number' &&
        // 确保不是默认的自动计算位置（比如y=0或y=120这样的规律）
        node.data?.original?.position !== undefined
      );
      console.log(`DAGVisualizer检测: ${hasPositions ? '有' : '无'}自定义位置`);
      if (hasPositions) {
        console.log('第一个节点位置:', state.dagData.nodes[0]?.position);
        console.log('第一个节点原始数据中的位置:', state.dagData.nodes[0]?.data?.original?.position);
      }
      
      const reactFlowNodes: Node[] = state.dagData.nodes.map(node => {
        // 根据actionName获取固定颜色
        const nodeTaskType = node.data.taskType || 'default';
        const actionName = (node.data as any).actionName || '';
        const nodeColor = ColorManager.getColorByActionName(actionName);
        
        // 调试：打印节点位置
        console.log(`创建ReactFlow节点 ${node.id}, position:`, node.position);
        
        return {
          id: node.id,
          type: 'default',
          position: node.position, // 数据加载时使用原始位置
          data: {
            label: node.label,
            original: node.data.original,
            taskType: nodeTaskType, // 保持一致的数据结构
            actionName: (node.data as any).actionName,
            inputCount: node.data.inputCount,
            outputCount: node.data.outputCount,
            color: nodeColor, // 记录随机颜色，便于双击编辑时读取
            textColor: ColorManager.getContrastTextColor(nodeColor),
            timeoutSecs: (node.data as any).timeoutSecs,
            preChecks: (node.data as any).preChecks,
            params: (node.data.original as any)?.params,
            outputs: (node.data.original as any)?.outputs,
            isCustomType: (node.data as any).isCustomType || false
          },
          style: {
            // 关键：设置节点背景为半透明，允许连线显示在节点上方
            backgroundColor: `${nodeColor}dd`, // 添加透明度（dd = 87%不透明度）
            color: ColorManager.getContrastTextColor(nodeColor),
            border: '2px solid #ffffff',
            borderRadius: '8px',
            fontSize: '13px', // 默认字体大小，文本模式会单独处理
            fontWeight: 'bold',
            width: '220px', // 默认尺寸，文本模式会单独处理
            height: '60px',
            padding: '12px 16px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            textAlign: 'center',
            whiteSpace: 'nowrap' as const, // 默认不换行，文本模式会单独处理
            wordBreak: 'normal' as const,
            overflow: 'hidden',
            lineHeight: '1.2',
            transition: 'all 0.3s ease',
            // 确保节点内容可见但连线可以穿过
            boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)'
          },
          sourcePosition: layoutOptions.direction === 'LR' ? Position.Right : Position.Bottom,
          targetPosition: layoutOptions.direction === 'LR' ? Position.Left : Position.Top
        };
      });
      
      let reactFlowEdges: Edge[] = state.dagData.edges.map(edge => {
        // 判断是否为条件边
        const isConditional = edge.condition?.type === 'conditional';
        const edgeColor = isConditional ? '#f59e0b' : '#94a3b8'; // 条件边用橙色，普通边用灰色
        const edgeStyle = isConditional ? 'dashed' : 'solid'; // 条件边用虚线
        
        return {
          id: edge.id,
          source: edge.source,
          target: edge.target,
          type: 'smoothstep',
          animated: true,
          label: edge.label, // 添加条件标签
          labelStyle: {
            backgroundColor: isConditional ? '#fef3c7' : '#f1f5f9',
            color: isConditional ? '#d97706' : '#475569',
            fontSize: '12px',
            fontWeight: '500',
            padding: '4px 8px',
            borderRadius: '4px',
            border: isConditional ? '1px solid #f59e0b' : 'none'
          },
          style: { 
            stroke: edgeColor,
            strokeWidth: isConditional ? 2.5 : 2,
            opacity: 0.8,
            strokeDasharray: edgeStyle === 'dashed' ? '5,5' : '0' // 虚线样式
          },
          sourcePosition: layoutOptions.direction === 'LR' ? Position.Right : Position.Bottom,
          targetPosition: layoutOptions.direction === 'LR' ? Position.Left : Position.Top,
          markerEnd: {
            type: MarkerType.ArrowClosed,
            color: edgeColor,
            width: 16,
            height: 16,
          }
        };
      });

      // 移除连线重叠优化应用，功能已被智能布局囊括
      // if (edgeOptimizationOptions.enabled && reactFlowNodes.length > 0) {
      //   reactFlowEdges = optimizeEdges(
      //     reactFlowEdges,
      //     reactFlowNodes,
      //     layoutOptions.direction === 'LR' ? Position.Right : Position.Bottom,
      //     layoutOptions.direction === 'LR' ? Position.Left : Position.Top,
      //     edgeOptimizationOptions
      //   );
      // }

      setNodes(reactFlowNodes);
      setEdges(reactFlowEdges);
      
      // 无论是否有自定义位置，都需要调整视口以显示节点
      setTimeout(() => {
        if (reactFlowInstance && reactFlowInstance.fitView) {
          if (hasPositions) {
            // 有自定义位置：适配视图但保持位置关系
            console.log('有自定义位置，调整视口以完整显示所有节点');
            reactFlowInstance.fitView({
              padding: 0.1,
              includeHiddenNodes: false,
              duration: 0 // 不要动画，立即显示
            });
          } else {
            // 无自定义位置：正常fitView
            console.log('无自定义位置，执行标准fitView');
            reactFlowInstance.fitView({
              padding: 0.2,
              includeHiddenNodes: false,
              duration: 200
            });
          }
        }
      }, 50);
    } else {
      // 当dagData为null时，清空ReactFlow的节点和边
      setNodes([]);
      setEdges([]);
      
      // 重置其他相关状态
      setSelectedEdge(null);
      setHighlightedNodes(new Set());
      setEditingNodeId(null);
      setIsDialogOpen(false);
      setShowAnalysisModal(false);
      
      console.log('✅ 画布已清空：节点和边已移除');
    }
  }, [state.dagData, setNodes, setEdges, layoutOptions.direction]);

  // 加载变量数据和工作流基础信息
  React.useEffect(() => {
    if (isVarsEditorOpen && state.jsonText) {
      try {
        const jsonData = JSON.parse(state.jsonText);
        const loadedVars = jsonData.vars || [];
        setVars(loadedVars.length > 0 ? loadedVars : []);
        
        // 加载工作流基础信息
        setWorkflowInfo({
          dag_id: jsonData.dag_id || '',
          name: jsonData.name || '',
          desc: jsonData.desc || ''
        });
        
        setVarsHasChanges(false);
      } catch (error) {
        console.error('解析JSON失败:', error);
        setVars([]);
        setWorkflowInfo({});
      }
    }
  }, [isVarsEditorOpen, state.jsonText]);

  // 变量编辑处理函数
  const handleAddVar = () => {
    setVars([...vars, { name: '', default_value: '', desc: '' }]);
    setVarsHasChanges(true);
  };

  const handleRemoveVar = (index: number) => {
    const newVars = vars.filter((_, i) => i !== index);
    setVars(newVars);
    setVarsHasChanges(true);
  };

  const handleVarChange = (index: number, field: 'name' | 'desc' | 'default_value', value: any) => {
    const newVars = [...vars];
    newVars[index] = { ...newVars[index], [field]: value };
    setVars(newVars);
    setVarsHasChanges(true);
  };

  const handleWorkflowInfoChange = (field: 'dag_id' | 'name' | 'desc', value: string) => {
    setWorkflowInfo(prev => ({ ...prev, [field]: value }));
    setVarsHasChanges(true);
  };

  const handleSaveVars = async () => {
    if (!state.jsonText) return;

    try {
      const jsonData = JSON.parse(state.jsonText);
      const updatedData = {
        ...jsonData,
        // 更新工作流基础信息
        dag_id: workflowInfo.dag_id || jsonData.dag_id,
        name: workflowInfo.name || jsonData.name,
        desc: workflowInfo.desc || jsonData.desc,
        // 更新变量
        vars: vars.filter(v => v.name.trim() !== '')
      };
      const updatedJsonText = JSON.stringify(updatedData, null, 2);
      dispatch({ type: 'SET_JSON_TEXT', payload: updatedJsonText });
      await loadDAGData(updatedData, false);
      setVarsHasChanges(false);
      // 保存成功后不关闭抽屉，让用户可以继续编辑
    } catch (error) {
      console.error('保存失败:', error);
      alert('保存失败: ' + (error instanceof Error ? error.message : '未知错误'));
    }
  };

  // 使用ref来跟踪上次的textAdaptiveMode，避免无限循环
  const lastTextModeRef = useRef<'uniform' | 'fixed'>('fixed');

  // 文本模式变化时，只更新节点样式，保持位置不变 - 使用函数式更新
  React.useEffect(() => {
    // 只有当textAdaptiveMode真正改变时才更新
    if (lastTextModeRef.current !== textAdaptiveMode) {
      lastTextModeRef.current = textAdaptiveMode;
      
      setNodes(prevNodes => {
        if (prevNodes.length === 0) return prevNodes;
        
        // 计算节点尺寸（根据文本自适应模式）
        let uniformSize: { width: number; height: number; fontSize: number } | null = null;
        if (textAdaptiveMode === 'uniform' && state.dagData) {
          const allTexts = state.dagData.nodes.map(node => node.label);
          uniformSize = calculateUniformNodeSize(allTexts, textConfig);
        }

        return prevNodes.map(node => {
          let nodeSize = { width: 220, height: 60, fontSize: 13 };
          
          if (textAdaptiveMode === 'uniform' && uniformSize) {
            nodeSize = uniformSize;
          }

          return {
            ...node,
            style: {
              ...node.style,
              fontSize: `${nodeSize.fontSize}px`,
              width: textAdaptiveMode === 'fixed' ? '220px' : `${nodeSize.width}px`,
              height: textAdaptiveMode === 'fixed' ? '60px' : `${nodeSize.height}px`,
              whiteSpace: textAdaptiveMode === 'fixed' ? 'nowrap' as const : 'pre-wrap' as const,
              wordBreak: textAdaptiveMode === 'fixed' ? 'normal' as const : 'break-word' as const,
            }
          };
        });
      });
    }
  }, [textAdaptiveMode, setNodes, state.dagData]);

  // 单独处理节点高亮状态更新 - 使用函数式更新避免依赖冲突
  React.useEffect(() => {
    setNodes(prevNodes => {
      if (prevNodes.length === 0) return prevNodes;
      
      return prevNodes.map(node => {
        const isHighlighted = highlightedNodes.has(node.id);
        const currentlyHighlighted = node.style?.border === '3px solid #fbbf24';
        
        // 只有状态真正改变时才更新
        if (isHighlighted === currentlyHighlighted) {
          return node;
        }
        
        return {
          ...node,
          // 明确保持原始位置不变
          position: node.position,
          style: {
            ...node.style,
            border: isHighlighted ? '3px solid #fbbf24' : '2px solid #ffffff',
            boxShadow: isHighlighted ? '0 0 20px rgba(251, 191, 36, 0.6)' : '0 2px 8px rgba(0, 0, 0, 0.1)'
            // 移除 transform 和 zIndex 避免影响交互
          }
        };
      });
    });
  }, [highlightedNodes, setNodes]);

  // 单独处理边的高亮状态更新
  React.useEffect(() => {
    setEdges(prevEdges => {
      if (prevEdges.length === 0) return prevEdges;
      
      return prevEdges.map(edge => {
        const isSelected = selectedEdge === edge.id;
        const currentlySelected = edge.style?.stroke === '#fbbf24';
        
        // 只有状态真正改变时才更新
        if (isSelected === currentlySelected) {
          return edge;
        }
        
        // 获取原始颜色（条件边或普通边）
        const isConditional = (edge as any).condition?.type === 'conditional';
        const originalColor = isConditional ? '#f59e0b' : '#94a3b8';
        const selectedColor = '#fbbf24';
        
        return {
          ...edge,
          style: {
            ...edge.style,
            stroke: isSelected ? selectedColor : originalColor,
            strokeWidth: isSelected ? 4 : (isConditional ? 2.5 : 2),
            opacity: isSelected ? 1 : 0.8,
            filter: isSelected ? 'drop-shadow(0 0 8px rgba(251, 191, 36, 0.8))' : undefined
          },
          markerEnd: {
            type: MarkerType.ArrowClosed,
            color: isSelected ? selectedColor : originalColor,
            width: 16,
            height: 16,
          }
        };
      });
    });
  }, [selectedEdge, setEdges]);

  // 监听全局点击事件，点击弹窗外部时关闭参数变量选择弹窗
  React.useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (paramVarPicker.open && paramVarPickerRef.current) {
        const target = event.target;
        if (!target) return;
        
        // 检查点击是否在弹窗外部（使用 DOM Node 类型）
        if (!paramVarPickerRef.current.contains(target as globalThis.Node)) {
          // 同时检查是否点击了触发按钮（{x}按钮），如果是则不关闭
          const targetElement = target as HTMLElement;
          const isPickerButton = targetElement.closest('button[title="选择预定义参数"], button[title="选择变量 (vars/share_data)"], button[title="选择预定义输出"]');
          
          if (!isPickerButton) {
            setParamVarPicker({ open: false, paramIndex: null, field: null, anchorRect: null, options: [], searchTerm: '' });
          }
        }
      }
    };

    if (paramVarPicker.open) {
      // 延迟添加监听器，避免立即触发
      setTimeout(() => {
        document.addEventListener('mousedown', handleClickOutside);
      }, 0);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [paramVarPicker.open]);



  // 更新JSON中的连线信息
  const updateConnectionsInJSON = useCallback(async (connection: Connection) => {
    if (!state.jsonText || !connection.source || !connection.target) return;
    
    try {
      const currentJson = JSON.parse(state.jsonText);

      // 检查是否是DAGDefinition格式
      if (currentJson.tasks && Array.isArray(currentJson.tasks)) {
        // DAGDefinition格式
        const updatedJson = {
          ...currentJson,
          tasks: currentJson.tasks.map((task: any) => {
            if (task.id === connection.target) {
              const depend_on = task.depend_on || [];
              if (!depend_on.includes(connection.source)) {
                return {
                  ...task,
                  depend_on: [...depend_on, connection.source]
                };
              }
            }
            return task;
          })
        };

        const jsonString = JSON.stringify(updatedJson, null, 2);
        dispatch({
          type: 'SET_JSON_TEXT',
          payload: jsonString
        });

        // 重新加载DAG数据，不触发自动布局
        await loadDAGData(updatedJson, false);
      } else if (Array.isArray(currentJson)) {
        // 数组格式
        const updatedJson = currentJson.map((task: any) => {
          if (task.id === connection.target) {
            const depend_on = task.depend_on || [];
            if (!depend_on.includes(connection.source)) {
              return {
                ...task,
                depend_on: [...depend_on, connection.source]
              };
            }
          }
          return task;
        });
        
        const jsonString = JSON.stringify(updatedJson, null, 2);
        dispatch({
          type: 'SET_JSON_TEXT',
          payload: jsonString
        });
        
        // 重新加载DAG数据，不触发自动布局
        await loadDAGData(updatedJson, false);
      }
    } catch (error) {
      console.error('更新连线失败:', error);
    }
  }, [state.jsonText, dispatch, loadDAGData]);

  // 处理连线删除
  const onEdgesDelete = useCallback(async (edgesToDelete: Edge[]) => {
    setEdges((eds) => eds.filter((edge) => !edgesToDelete.find(e => e.id === edge.id)));
    
    // 显示删除提示
    setShowDeleteHint(true);
    setTimeout(() => setShowDeleteHint(false), 3000);

    // 更新JSON配置，移除对应的depend_on
    if (state.jsonText) {
      try {
        const currentJson = JSON.parse(state.jsonText);
            
        // 检查是否是DAGDefinition格式
        if (currentJson.tasks && Array.isArray(currentJson.tasks)) {
          // DAGDefinition格式
          const updatedJson = {
            ...currentJson,
            tasks: currentJson.tasks.map((task: any) => {
              let updatedTask = { ...task };

              edgesToDelete.forEach(edge => {
                if (task.id === edge.target) {
                  const depend_on = task.depend_on || [];
                  updatedTask = {
                    ...updatedTask,
                    depend_on: depend_on.filter((dep: string) => dep !== edge.source)
                  };
                }
              });

              return updatedTask;
            })
          };

          const jsonString = JSON.stringify(updatedJson, null, 2);
          dispatch({
            type: 'SET_JSON_TEXT',
            payload: jsonString
          });

          // 重新加载DAG数据，不触发自动布局
          await loadDAGData(updatedJson, false);
        } else if (Array.isArray(currentJson)) {
          // 数组格式
          const updatedJson = currentJson.map((task: any) => {
            let updatedTask = { ...task };

            edgesToDelete.forEach(edge => {
              if (task.id === edge.target) {
                const depend_on = task.depend_on || [];
                updatedTask = {
                  ...updatedTask,
                  depend_on: depend_on.filter((dep: string) => dep !== edge.source)
                };
              }
            });

            return updatedTask;
          });
          
          const jsonString = JSON.stringify(updatedJson, null, 2);
          dispatch({
            type: 'SET_JSON_TEXT',
            payload: jsonString
          });
          
          // 重新加载DAG数据，不触发自动布局
          await loadDAGData(updatedJson, false);
        }
      } catch (error) {
        console.error('删除连线失败:', error);
      }
    }
  }, [state.jsonText, dispatch, loadDAGData]);



  // 处理连线
  const onConnect = useCallback((connection: Connection) => {
    const newEdge = {
      ...connection,
      id: `${connection.source}-${connection.target}`,
      type: 'smoothstep',
      animated: true,
      style: { stroke: '#94a3b8', strokeWidth: 2 },
      sourcePosition: layoutOptions.direction === 'LR' ? Position.Right : Position.Bottom,
      targetPosition: layoutOptions.direction === 'LR' ? Position.Left : Position.Top,
      markerEnd: {
        type: MarkerType.ArrowClosed,
        color: '#94a3b8',
        width: 16,
        height: 16,
      }
    };
    
    setEdges((eds) => addEdge(newEdge, eds));
    
    // 更新JSON配置中的dependencies
    updateConnectionsInJSON(connection);
  }, [setEdges, updateConnectionsInJSON, layoutOptions.direction]);

  // 处理节点双击编辑
  const onNodeDoubleClick = useCallback((event: React.MouseEvent, node: Node) => {
    event.stopPropagation();
    setEditingNodeId(node.id);
    setEditingNodeLabel(node.id); // 编辑taskId，所以初始值是node.id
    setEditingNodeName((node.data as any)?.original?.name || '');
    
    // 获取当前节点的实际颜色（从node.data.color或node.style获取）
    const currentColor = node.data?.color || node.style?.backgroundColor || '#64748b';
    setEditingNodeColor(currentColor);
    
    // 获取当前节点的类型信息（优先使用 actionName / action_name，其次 taskType）
    const originalData: any = (node.data as any)?.original || {};
    const currentNodeType = node.data?.actionName || originalData.action_name || node.data?.taskType || 'PROMPT_BUILD';
    const isCustomType = node.data?.isCustomType || false;
    
    // 获取 params、outputs、pre_checks、timeout_secs
    const originalNode = (node.data as any)?.original;
    // params 可能包含 sub_params，这里统一扁平化为 key / value 形式
    setEditingNodeParams(flattenParams(originalNode?.params));
    setEditingNodeOutputs(originalNode?.outputs || []);
    setEditingNodePreChecks(originalNode?.pre_checks || []);
    setEditingNodeTimeoutSecs(originalNode?.timeout_secs || 3600);

    // 初始化时加载已导入的节点类型定义
    try {
      const stored = localStorage.getItem('custom_node_types');
      if (stored) {
        const parsed = JSON.parse(stored) as import('../types').NodeTypeDefinition[];
        if (Array.isArray(parsed)) {
          setEditingLoadedNodeTypes(parsed);
        }
      }
    } catch (e) {
      console.warn('读取节点类型定义失败:', e);
    }
    // 这里使用 action_name 作为选中值
    setEditingSelectedLoadedType(currentNodeType || '');
    
    if (isCustomType) {
      // 自定义类型节点，回显自定义信息
      setEditingIsCustomType(true);
      setEditingNodeType('CUSTOM');
      setEditingCustomNodeType(currentNodeType);
    } else {
      // 预定义类型节点，检查是否在预定义类型列表中
      const predefinedType = DEFAULT_NODE_TYPES.find(type => 
        type.id === currentNodeType || 
        type.template.taskType === currentNodeType
      );
      
      if (predefinedType) {
        setEditingIsCustomType(false);
        setEditingNodeType(predefinedType.id);
        setEditingCustomNodeType('');
      } else {
        // 虽然isCustomType为false，但不在预定义类型中，当作自定义类型处理
        setEditingIsCustomType(true);
        setEditingNodeType('CUSTOM');
        setEditingCustomNodeType(currentNodeType);
      }
    }
  }, []);

  // 处理节点单击刷新弹窗内容
  const onNodeClick = useCallback((event: React.MouseEvent, node: Node) => {
    event.stopPropagation();
    // 如果已经打开了编辑抽屉，点击节点时刷新内容
    if (editingNodeId) {
      // 关键修复：切换点击节点时，立即更新 editingNodeId 为当前点击的节点 ID
      // 否则 saveNodeLabelEdit 会一直把原来打开的那个节点当作“旧节点”去处理
      setEditingNodeId(node.id);
      
      setEditingNodeLabel(node.id);
      setEditingNodeName((node.data as any)?.original?.name || '');
      
      const currentColor = node.data?.color || node.style?.backgroundColor || '#64748b';
      setEditingNodeColor(currentColor);
      
      const originalData: any = (node.data as any)?.original || {};
      const currentNodeType = node.data?.actionName || originalData.action_name || node.data?.taskType || 'PROMPT_BUILD';
      const isCustomType = node.data?.isCustomType || false;
      
      const originalNode = (node.data as any)?.original;
      setEditingNodeParams(flattenParams(originalNode?.params));
      setEditingNodeOutputs(originalNode?.outputs || []);
      setEditingNodePreChecks(originalNode?.pre_checks || []);
      setEditingNodeTimeoutSecs(originalNode?.timeout_secs || 3600);

      try {
        const stored = localStorage.getItem('custom_node_types');
        if (stored) {
          const parsed = JSON.parse(stored) as import('../types').NodeTypeDefinition[];
          if (Array.isArray(parsed)) {
            setEditingLoadedNodeTypes(parsed);
          }
        }
      } catch (e) {
        console.warn('读取节点类型定义失败:', e);
      }
      setEditingSelectedLoadedType(currentNodeType || '');
      
      if (isCustomType) {
        setEditingIsCustomType(true);
        setEditingNodeType('CUSTOM');
        setEditingCustomNodeType(currentNodeType);
      } else {
        const predefinedType = DEFAULT_NODE_TYPES.find(type => 
          type.id === currentNodeType || 
          type.template.taskType === currentNodeType
        );
        
        if (predefinedType) {
          setEditingIsCustomType(false);
          setEditingNodeType(predefinedType.id);
          setEditingCustomNodeType('');
        } else {
          setEditingIsCustomType(true);
          setEditingNodeType('CUSTOM');
          setEditingCustomNodeType(currentNodeType);
        }
      }
    }
  }, [editingNodeId]);

  // 批量更新同类型节点颜色功能已移除（颜色每次随机生成），保留此注释避免未使用函数

  // 保存节点标签编辑（同时更新 id / name / action_name）
  const saveNodeLabelEdit = useCallback(async () => {
    if (!editingNodeId || !editingNodeLabel.trim()) {
      setEditingNodeId(null);
      setEditingNodeLabel('');
      return;
    }

    const newTaskId = editingNodeLabel.trim();
    const oldTaskId = editingNodeId;

    // 从导入文件选择的 action_name
    const selectedActionName = editingSelectedLoadedType && editingSelectedLoadedType.trim();

    // 验证taskId格式
    if (!/^[a-zA-Z0-9_-]+$/.test(newTaskId)) {
      alert('节点ID只能包含字母、数字、下划线和短横线');
      return;
    }

    // 如果taskId变化，检查新的taskId是否已存在
    if (newTaskId !== oldTaskId && state.jsonText) {
      try {
        const currentJson = JSON.parse(state.jsonText);
        let tasks: any[] = [];
        if (currentJson.tasks && Array.isArray(currentJson.tasks)) {
          tasks = currentJson.tasks;
        } else if (Array.isArray(currentJson)) {
          tasks = currentJson;
        }

        // 查找是否已经有其他节点占用了 newTaskId (排除正在编辑的 oldTaskId 本身)
        const conflictTask = tasks.find((t: any) => t.id === newTaskId && t.id !== oldTaskId);

        if (conflictTask) {
          alert(`该节点ID "${newTaskId}" 已被其他任务占用，请使用其他名称`);
          return;
        }
      } catch (error) {
        // JSON解析失败，继续编辑
      }
    }

    // 更新JSON配置：id / name / action_name / depend_on
    if (state.jsonText) {
      try {
        const currentJson = JSON.parse(state.jsonText);

        const applyUpdate = (tasks: any[]): { updatedTasks: any[]; hit: boolean } => {
          let hit = false;
          const updatedTasks = tasks.map((task: any) => {
            if (task.id === oldTaskId) {
              hit = true;
              
              // 获取节点当前位置
              const currentNode = nodes.find(n => n.id === oldTaskId);
              
              const updatedTask: any = {
                ...task,
                id: newTaskId,
                // 无论是否已有 name，都覆盖为当前编辑值
                name: editingNodeName || newTaskId,
                // 将编辑中的 params / outputs / pre_checks / timeout_secs 写回 JSON
                params: restoreParams(editingNodeParams),
                outputs: editingNodeOutputs,
                pre_checks: editingNodePreChecks,
                timeout_secs: editingNodeTimeoutSecs,
                // 保存节点位置
                position: currentNode ? currentNode.position : task.position,
              };
              if (selectedActionName) {
                updatedTask.action_name = selectedActionName;
              }
              return updatedTask;
            }

            // 更新其他任务的 depend_on
            if (task.depend_on && task.depend_on.includes(oldTaskId)) {
              return {
                ...task,
                depend_on: task.depend_on.map((dep: string) => dep === oldTaskId ? newTaskId : dep)
              };
            }

            return task;
          });
          return { updatedTasks, hit };
        };

        let updatedJson: any = currentJson;
        let hit = false;

        // DAGDefinition: { tasks: [...] }
        if (currentJson.tasks && Array.isArray(currentJson.tasks)) {
          const result = applyUpdate(currentJson.tasks);
          hit = result.hit;
          updatedJson = {
            ...currentJson,
            tasks: result.updatedTasks
          };
        }
        // 数组: [ {...}, {...} ]
        else if (Array.isArray(currentJson)) {
          const result = applyUpdate(currentJson);
          hit = result.hit;
          updatedJson = result.updatedTasks;
        } else {
          console.warn('[saveNodeLabelEdit] 未识别的 JSON 结构，跳过更新');
        }

        if (!hit) {
          console.warn('[saveNodeLabelEdit] 没找到 id =', oldTaskId, '的任务，无法更新');
        }

        const jsonString = JSON.stringify(updatedJson, null, 2);
        dispatch({
          type: 'SET_JSON_TEXT',
          payload: jsonString
        });

        // 关键逻辑：更新编辑状态中的 editingNodeId 为保存后的新 ID
        // 这样在“弹窗不关闭”的情况下，下次保存时的 oldTaskId 才是正确的
        setEditingNodeId(newTaskId);

        // 检查是否有位置信息，如果有就不执行自动布局
        let hasPositionInfo = false;
        let tasks: any[] = [];
        
        if (updatedJson && typeof updatedJson === 'object' && updatedJson.tasks && Array.isArray(updatedJson.tasks)) {
          tasks = updatedJson.tasks;
        } else if (Array.isArray(updatedJson)) {
          tasks = updatedJson;
        }
        
        // 如果至少有一个任务有位置信息，就不自动布局
        if (tasks.length > 0) {
          hasPositionInfo = tasks.some((task: any) => 
            task.position && 
            typeof task.position.x === 'number' && 
            typeof task.position.y === 'number'
          );
        }
        
        // 重新加载DAG数据以更新显示，如果有位置信息就不执行自动布局
        await loadDAGData(updatedJson, !hasPositionInfo);
      } catch (error) {
        console.error('更新节点失败:', error);
        alert('更新失败，请检查JSON格式');
      }
    }

    // 注释掉清理逻辑，实现保存后不关闭弹窗
    /*
    setEditingNodeId(null);
    setEditingNodeLabel('');
    setEditingNodeName('');
    setEditingNodeType('');
    setEditingNodeColor('');
    setEditingIsCustomType(false);
    setEditingCustomNodeType('');
    setBatchColorMode(false);
    setEditingNodeParams([]);
    setEditingNodeOutputs([]);
    setEditingNodePreChecks([]);
    setEditingNodeTimeoutSecs(3600);
    setExpandedPreCheckIndexes(new Set());
    */
  }, [
    editingNodeId,
    editingNodeLabel,
    editingNodeName,
    editingNodeType,
    editingNodeColor,
    editingIsCustomType,
    editingCustomNodeType,
    batchColorMode,
    editingNodeParams,
    editingNodeOutputs,
    editingNodePreChecks,
    editingNodeTimeoutSecs,
    editingSelectedLoadedType,
    state.jsonText,
    dispatch,
    loadDAGData
  ]);

  // 取消节点标签编辑
  const cancelNodeLabelEdit = useCallback(() => {
    setEditingNodeId(null);
    setEditingNodeLabel('');
    setEditingNodeName('');
    setEditingNodeType('');
    setEditingNodeColor('');
    setEditingIsCustomType(false);
    setEditingCustomNodeType('');
    setBatchColorMode(false);
    setEditingNodeParams([]);
    setEditingNodeOutputs([]);
    setEditingNodePreChecks([]);
    setEditingNodeTimeoutSecs(3600);
    setExpandedPreCheckIndexes(new Set());
    setEditingLoadedNodeTypes([]);
    setEditingSelectedLoadedType('');
  }, []);

  // 处理键盘事件
  const handleEditKeyDown = useCallback((event: React.KeyboardEvent) => {
    if (event.key === 'Enter') {
      saveNodeLabelEdit();
    } else if (event.key === 'Escape') {
      cancelNodeLabelEdit();
    }
  }, [saveNodeLabelEdit, cancelNodeLabelEdit]);

  // 获取节点颜色的函数（用于迷你地图）
  const getNodeColor = (node: Node) => {
    return node.style?.backgroundColor || '#64748b';
  };

  // 迷你地图样式
  const miniMapStyle = {
    backgroundColor: '#f9fafb',
    border: '1px solid #e5e7eb'
  };

  // 处理右键点击事件
  const handlePaneContextMenu = useCallback((event: React.MouseEvent) => {
    event.preventDefault();
    
    // 获取ReactFlow画布的点击位置
    const position = reactFlowInstance.screenToFlowPosition({
      x: event.clientX,
      y: event.clientY,
    });

    setDialogPosition(position);
    setContextMenuPosition({ x: event.clientX, y: event.clientY });
  }, [reactFlowInstance]);

  // 处理新节点创建
  const handleCreateNode = useCallback(async (newNode: Node) => {
    // 新增节点一律使用随机ID，避免依赖已有结构
    const taskId = newNode.id;

    // 检查 ID 是否冲突
    if (state.jsonText) {
      try {
        const currentJson = JSON.parse(state.jsonText);
        let tasks: any[] = [];
        if (currentJson.tasks && Array.isArray(currentJson.tasks)) {
          tasks = currentJson.tasks;
        } else if (Array.isArray(currentJson)) {
          tasks = currentJson;
        }
        if (tasks.find((t: any) => t.id === taskId)) {
          alert('ID 冲突，请重试');
          return;
        }
      } catch (error) {}
    }

    // 更新节点 ID
    const nodeToCreate = {
      ...newNode,
      id: taskId,
      data: {
        ...newNode.data,
        id: taskId,
        label: newNode.data?.label || taskId,
        taskId: taskId
      }
    };

    // 添加节点到 ReactFlow
    setNodes((currentNodes) => [...currentNodes, nodeToCreate]);

    // 创建新任务
    const newTask: any = {
      id: taskId,
      name: nodeToCreate.data?.label || taskId,
      action_name: nodeToCreate.data?.taskType || 'default',
      depend_on: [],
      params: nodeToCreate.data?.input?.map((i: any) => ({ key: i.fieldName, value: i.value })) || [],
      outputs: nodeToCreate.data?.output?.map((o: any) => ({ key: o.fieldName, value: o.path })) || [],
      position: nodeToCreate.position // 保存节点位置
    };

    // 更新AppContext中的JSON文本
    try {
      let updatedJson;
      if (state.jsonText) {
        const currentJson = JSON.parse(state.jsonText);
        if (currentJson.tasks && Array.isArray(currentJson.tasks)) {
          updatedJson = { ...currentJson, tasks: [...currentJson.tasks, newTask] };
        } else if (Array.isArray(currentJson)) {
          updatedJson = [...currentJson, newTask];
        } else {
          updatedJson = [newTask];
        }
      } else {
        updatedJson = [newTask];
      }
      
      const jsonString = JSON.stringify(updatedJson, null, 2);
      dispatch({ type: 'SET_JSON_TEXT', payload: jsonString });
      // 创建新节点时不触发自动布局
      await loadDAGData(updatedJson, false);
    } catch (error) {
      console.error('更新JSON配置失败:', error);
    }
    setContextMenuPosition(null);
  }, [setNodes, state.jsonText, dispatch, loadDAGData]);

  // 快速添加节点（在页面上）
  const handleQuickAddNode = useCallback(async () => {
    const nodeId = prompt('请输入节点ID (唯一且只能包含字母、数字、下划线、短横线):');
    
    if (nodeId === null) return; // 用户点击取消
    
    const trimmedId = nodeId.trim();
    if (!trimmedId) {
      alert('节点ID不能为空');
      return;
    }

    if (!/^[a-zA-Z0-9_-]+$/.test(trimmedId)) {
      alert('节点ID格式不正确（仅限字母、数字、下划线、短横线）');
      return;
    }

    try {
      if (state.jsonText) {
        const currentJson = JSON.parse(state.jsonText);
        let tasks = [];
        if (currentJson.tasks && Array.isArray(currentJson.tasks)) {
          tasks = currentJson.tasks;
        } else if (Array.isArray(currentJson)) {
          tasks = currentJson;
        }
        
        const exists = tasks.some((t: any) => t.id === trimmedId);
        if (exists) {
          alert(`节点ID "${trimmedId}" 已存在，请输入一个唯一的ID`);
          return;
        }
      }

      // 创建新任务
      const newTask = {
        id: trimmedId,
        name: trimmedId,
        action_name: 'default',
        depend_on: [],
        params: [],
        outputs: []
      };

      let updatedJson;
      if (state.jsonText) {
        const currentJson = JSON.parse(state.jsonText);
        if (currentJson.tasks && Array.isArray(currentJson.tasks)) {
          updatedJson = { ...currentJson, tasks: [...currentJson.tasks, newTask] };
        } else if (Array.isArray(currentJson)) {
          updatedJson = [...currentJson, newTask];
        } else {
          updatedJson = [newTask];
        }
      } else {
        updatedJson = [newTask];
      }
      
      const jsonString = JSON.stringify(updatedJson, null, 2);
      dispatch({ type: 'SET_JSON_TEXT', payload: jsonString });
      // 快速添加节点时不触发自动布局
      await loadDAGData(updatedJson, false);
    } catch (error) {
      console.error('添加节点失败:', error);
      alert('添加节点失败，请检查JSON内容');
    }

    setContextMenuPosition(null);
  }, [state.jsonText, dispatch, loadDAGData]);



  // 智能布局处理函数
  const handleSmartLayout = useCallback(async () => {
    if (isLayouting || nodes.length === 0) return;
    
    setIsLayouting(true);
    
    try {
      console.log('开始智能布局...');
      
      // 计算基础智能布局
      const layoutedNodes = calculateSmartLayout(nodes, edges, layoutOptions);
      
      // 检测连线穿越问题
      const crossings = detectEdgeCrossings(layoutedNodes, edges);
      console.log(`检测到 ${crossings.length} 个连线穿越问题`);
      
      // 根据DAG复杂度选择优化策略
      let optimizedNodes = layoutedNodes;
      if (crossings.length > 0) {
        console.log('应用连线穿越优化...', crossings);
        
        // 对于复杂DAG (节点数>10 或 穿越数>5) 使用增强算法
        if (nodes.length > 10 || crossings.length > 5) {
          console.log('使用复杂DAG优化算法...');
          optimizedNodes = optimizeComplexDAGLayout(layoutedNodes, edges, layoutOptions);
          
          // 输出详细分析结果
          const analysis = analyzeComplexDAG(optimizedNodes, edges);
          console.log('复杂DAG优化分析:', analysis);
          
          if (analysis.suggestions.length > 0) {
            console.log('优化建议:', analysis.suggestions.join('; '));
          }
        } else {
          // 简单DAG使用基础算法
          optimizedNodes = optimizeLayoutForEdgeCrossings(layoutedNodes, edges, layoutOptions);
        }
        
        // 再次检测优化效果
        const optimizedCrossings = detectEdgeCrossings(optimizedNodes, edges);
        console.log(`优化后剩余 ${optimizedCrossings.length} 个连线穿越问题`);
        
        // 输出穿越问题的详细信息
        if (optimizedCrossings.length > 0) {
          const problemEdges = optimizedCrossings.map(c => 
            `${c.sourceNodeId}→${c.targetNodeId} (${c.severity}, 穿越${c.crossingNodes.length}个节点)`
          ).join(', ');
          console.log('剩余穿越问题:', problemEdges);
        }
      }
      
      // 对齐到网格
      const alignedNodes = alignNodesToGrid(optimizedNodes, 20);
      
      // 应用新的节点位置
      setNodes(alignedNodes);
      
      // 保存所有节点位置到JSON数据中
      if (state.jsonText) {
        try {
          const currentJson = JSON.parse(state.jsonText);
          let tasks: any[] = [];
          
          // 获取tasks数组
          if (currentJson && typeof currentJson === 'object' && currentJson.tasks && Array.isArray(currentJson.tasks)) {
            tasks = currentJson.tasks;
          } else if (Array.isArray(currentJson)) {
            tasks = currentJson;
          }
          
          // 更新所有任务的位置
          const updatedTasks = tasks.map(task => {
            const node = alignedNodes.find(n => n.id === task.id);
            if (node) {
              return { ...task, position: node.position };
            }
            return task;
          });
          
          // 更新JSON文本
          let updatedJsonText: string;
          if (currentJson && typeof currentJson === 'object' && currentJson.tasks) {
            updatedJsonText = JSON.stringify({ ...currentJson, tasks: updatedTasks }, null, 2);
          } else {
            updatedJsonText = JSON.stringify(updatedTasks, null, 2);
          }
          
          dispatch({ type: 'SET_JSON_TEXT', payload: updatedJsonText });
          console.log('智能布局后节点位置已保存');
        } catch (error) {
          console.error('保存智能布局位置失败:', error);
        }
      }
      
      // 布局完成后自动适配视图
      if (reactFlowInstance && reactFlowInstance.fitView) {
        setTimeout(() => {
          reactFlowInstance.fitView({
            padding: 0.1,
            includeHiddenNodes: false,
            duration: 800 // 添加平滑动画
          });
        }, 100);
      }
      
      console.log('智能布局完成');
    } catch (error) {
      console.error('智能布局失败:', error);
    } finally {
      setIsLayouting(false);
    }
  }, [isLayouting, nodes, edges, layoutOptions, setNodes, reactFlowInstance, state.jsonText, dispatch]);

  // 将智能布局函数存储到ref中
  React.useEffect(() => {
    smartLayoutRef.current = handleSmartLayout;
  }, [handleSmartLayout]);

  // 当 JSON 更新并请求自动布局时，在 DAG 渲染完后执行一次智能布局
  React.useEffect(() => {
    if (!state.autoLayoutRequested || !state.dagData || !smartLayoutRef.current) return;
    const timer = setTimeout(() => {
      smartLayoutRef.current && smartLayoutRef.current();
      // 布局完成后清除标记
      dispatch({ type: 'SET_AUTO_LAYOUT_REQUESTED', payload: false });
    }, 50);
    return () => clearTimeout(timer);
  }, [state.autoLayoutRequested, state.dagData, dispatch]);

  // 监听智能布局请求（用于保存节点后自动布局）
  React.useEffect(() => {
    if (!state.smartLayoutRequested || !state.dagData || !smartLayoutRef.current) return;
    const timer = setTimeout(() => {
      console.log('保存更改后自动执行智能纵向布局...');
      smartLayoutRef.current && smartLayoutRef.current();
      // 布局完成后清除标记
      dispatch({ type: 'SET_SMART_LAYOUT_REQUESTED', payload: false });
    }, 50);
    return () => clearTimeout(timer);
  }, [state.smartLayoutRequested, state.dagData, dispatch]);


  // 切换布局方向
  const toggleLayoutDirection = useCallback(() => {
    if (isLayouting || nodes.length === 0) return;
    
    const directions: LayoutOptions['direction'][] = ['TB', 'LR'];
    const currentIndex = directions.indexOf(layoutOptions.direction);
    const nextIndex = (currentIndex + 1) % directions.length;
    const nextDirection = directions[nextIndex];
    
    setLayoutOptions(prev => ({
      ...prev,
      direction: nextDirection
    }));
    
    console.log(`布局方向切换为: ${LAYOUT_DIRECTIONS[nextDirection].name}`);
    
    // 延迟执行智能布局，确保状态更新完成
    setTimeout(() => {
      if (smartLayoutRef.current) {
        smartLayoutRef.current();
      }
    }, 100);
  }, [layoutOptions.direction, isLayouting, nodes.length]);

  // 节点拖动结束处理
  const handleNodeDragStop = useCallback((_event: React.MouseEvent, node: Node) => {
    // 先处理对齐
    let finalPosition = node.position;
    
    if (alignmentOptions.snapToGrid || alignmentOptions.snapToNodes) {
      // 获取其他节点
      const otherNodes = nodes.filter(n => n.id !== node.id);
      
      // 查找最近的对齐位置
      const alignment = findNearestAlignment(node, otherNodes, alignmentOptions);
      
      // 如果位置有变化，应用对齐
      if (alignment.x !== node.position.x || alignment.y !== node.position.y) {
        finalPosition = { x: alignment.x, y: alignment.y };
        
        const updatedNodes = nodes.map(n => 
          n.id === node.id 
            ? { ...n, position: finalPosition }
            : n
        );
        
        setNodes(updatedNodes);
        
        if (alignment.alignedTo) {
          console.log(`节点 ${node.id} 已对齐到: ${alignment.alignedTo}`);
        }
      }
    }
    
    // 保存节点位置到JSON数据中
    if (state.jsonText) {
      try {
        const currentJson = JSON.parse(state.jsonText);
        let tasks: any[] = [];
        
        // 获取tasks数组
        if (currentJson && typeof currentJson === 'object' && currentJson.tasks && Array.isArray(currentJson.tasks)) {
          tasks = currentJson.tasks;
        } else if (Array.isArray(currentJson)) {
          tasks = currentJson;
        }
        
        // 更新对应任务的位置
        const taskIndex = tasks.findIndex(t => t.id === node.id);
        if (taskIndex !== -1) {
          tasks[taskIndex].position = finalPosition;
          
          // 更新JSON文本
          let updatedJsonText: string;
          if (currentJson && typeof currentJson === 'object' && currentJson.tasks) {
            updatedJsonText = JSON.stringify({ ...currentJson, tasks }, null, 2);
          } else {
            updatedJsonText = JSON.stringify(tasks, null, 2);
          }
          
          dispatch({ type: 'SET_JSON_TEXT', payload: updatedJsonText });
          console.log(`节点 ${node.id} 位置已保存:`, finalPosition);
        }
      } catch (error) {
        console.error('保存节点位置失败:', error);
      }
    }
  }, [nodes, alignmentOptions, setNodes, state.jsonText, dispatch]);

  // 切换对齐选项
  const toggleSnapToGrid = useCallback(() => {
    setAlignmentOptions(prev => ({
      ...prev,
      snapToGrid: !prev.snapToGrid
    }));
    console.log(`网格对齐: ${alignmentOptions.snapToGrid ? '关闭' : '开启'}`);
  }, [alignmentOptions.snapToGrid]);

  // 切换文本自适应模式 (T/U)
  const toggleTextAdaptiveMode = useCallback(() => {
    console.log(`文本模式切换前: 布局方向=${layoutOptions.direction}, 文本模式=${textAdaptiveMode}`);
    
    const modes: Array<'fixed' | 'uniform'> = ['fixed', 'uniform'];
    const currentIndex = modes.indexOf(textAdaptiveMode);
    const nextIndex = (currentIndex + 1) % modes.length;
    const nextMode = modes[nextIndex];
    
    setTextAdaptiveMode(nextMode);
    
    const modeNames = {
      'fixed': 'T (固定尺寸)',
      'uniform': 'U (统一尺寸)'
    };
    
    console.log(`文本布局模式切换为: ${modeNames[nextMode]}`);
    
    // 延迟检查布局方向是否被意外改变
    setTimeout(() => {
      console.log(`文本模式切换后: 布局方向=${layoutOptions.direction}, 文本模式=${nextMode}`);
    }, 50);
  }, [textAdaptiveMode, layoutOptions.direction]);

  // 移除连线重叠优化切换函数，功能已被智能布局囊括
  // const toggleEdgeOptimization = useCallback(() => {
  //   setEdgeOptimizationOptions(prev => ({
  //     ...prev,
  //     enabled: !prev.enabled
  //   }));
  //   
  //   console.log(`连线重叠优化: ${edgeOptimizationOptions.enabled ? '关闭' : '开启'}`);
  // }, [edgeOptimizationOptions.enabled]);

  // 连线点击事件处理
  const handleEdgeClick = useCallback((event: React.MouseEvent, edge: Edge) => {
    event.stopPropagation();
    console.log(`点击连线: ${edge.id}, 起点: ${edge.source}, 终点: ${edge.target}`);
    
    // 显示删除提示
    if (!showDeleteHint) {
      setShowDeleteHint(true);
      setTimeout(() => setShowDeleteHint(false), 3000);
    }
    
    if (selectedEdge === edge.id) {
      // 取消选中
      setSelectedEdge(null);
      setHighlightedNodes(new Set());
    } else {
      // 选中新连线
      setSelectedEdge(edge.id);
      setHighlightedNodes(new Set([edge.source, edge.target]));
    }
  }, [selectedEdge, showDeleteHint]);
  
  // DAG分析处理函数
  const handleAnalyzeDAG = useCallback(() => {
    if (nodes.length === 0) {
      alert('没有节点数据可供分析');
      return;
    }
    
    const analysis = analyzeComplexDAG(nodes, edges);
    setAnalysisResult(analysis);
    setShowAnalysisModal(true);
    
    console.log('DAG分析结果:', analysis);
  }, [nodes, edges]);

  // 关闭节点创建对话框
  const closeNodeCreationDialog = useCallback(() => {
    setIsDialogOpen(false);
  }, []);

  // 点击画布时关闭右键菜单和取消连线选中
  const handlePaneClick = useCallback(() => {
    setContextMenuPosition(null);
    // 取消连线选中
    setSelectedEdge(null);
    setHighlightedNodes(new Set());
  }, []);

  // 处理节点删除
  const onNodesDelete = useCallback(async (nodesToDelete: Node[]) => {
    if (!state.jsonText) return;
    
    try {
      const currentJson = JSON.parse(state.jsonText);
      const deletedIds = nodesToDelete.map(n => n.id);

      const applyUpdate = (tasks: any[]): { updatedTasks: any[] } => {
        // 1. 过滤掉被删除的任务
        let updatedTasks = tasks.filter((task: any) => !deletedIds.includes(task.id));
        
        // 2. 更新剩余任务的 depend_on，移除对已删除任务的依赖
        updatedTasks = updatedTasks.map((task: any) => {
          if (task.depend_on && Array.isArray(task.depend_on)) {
            const newDependOn = task.depend_on.filter((dep: string) => !deletedIds.includes(dep));
            if (newDependOn.length !== task.depend_on.length) {
              return { ...task, depend_on: newDependOn };
            }
          }
          return task;
        });
        
        return { updatedTasks };
      };

      let updatedJson: any = currentJson;
      if (currentJson.tasks && Array.isArray(currentJson.tasks)) {
        const result = applyUpdate(currentJson.tasks);
        updatedJson = { ...currentJson, tasks: result.updatedTasks };
      } else if (Array.isArray(currentJson)) {
        const result = applyUpdate(currentJson);
        updatedJson = result.updatedTasks;
      }

      const jsonString = JSON.stringify(updatedJson, null, 2);
      dispatch({ type: 'SET_JSON_TEXT', payload: jsonString });

      // 如果当前编辑的节点被删除了，关闭编辑抽屉
      if (editingNodeId && deletedIds.includes(editingNodeId)) {
        cancelNodeLabelEdit();
      }

      // 重新加载数据更新视图，不触发自动布局
      await loadDAGData(updatedJson, false);
    } catch (error) {
      console.error('删除节点失败:', error);
    }
  }, [state.jsonText, dispatch, loadDAGData, editingNodeId, cancelNodeLabelEdit]);

  return (
    <div className={`visualizer-container ${editingNodeId || isVarsEditorOpen ? 'with-drawer' : ''}`}>
      {state.isLoading && (
        <div className="loading-overlay">
          <div className="loading-spinner"></div>
          <div className="loading-text">正在处理DAG数据...</div>
        </div>
      )}
      
      <div className="visualizer-main">
        <div className="reactflow-wrapper">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onNodeDragStop={handleNodeDragStop}
          onPaneContextMenu={handlePaneContextMenu}
          onPaneClick={handlePaneClick}
          onNodeDoubleClick={onNodeDoubleClick}
          onNodeClick={onNodeClick}
          onConnect={onConnect}
          onNodesDelete={onNodesDelete}
          onEdgesDelete={onEdgesDelete}
          onEdgeClick={handleEdgeClick}
          connectionLineType={layoutOptions.direction === 'LR' ? ConnectionLineType.Straight : ConnectionLineType.SmoothStep}
          connectionLineStyle={{ stroke: '#3b82f6', strokeWidth: 2 }}
          connectionMode={ConnectionMode.Loose}
          snapToGrid={alignmentOptions.snapToGrid}
          snapGrid={[alignmentOptions.gridSize, alignmentOptions.gridSize]}
          deleteKeyCode={['Backspace', 'Delete']}
          multiSelectionKeyCode={'Shift'}
          nodesDraggable={true}
          nodesConnectable={true}
          edgesUpdatable={true}
          edgesFocusable={true}
          attributionPosition="bottom-left"
          minZoom={0.1}
          maxZoom={2}
          defaultViewport={{ x: 0, y: 0, zoom: 1 }}
        >
          <Controls 
            position="top-right"
            showZoom={true}
            showFitView={true}
            showInteractive={true}
          >
            {/* 智能布局按钮 */}
            <ControlButton
              onClick={handleSmartLayout}
              title={`智能布局 (${LAYOUT_DIRECTIONS[layoutOptions.direction].name})`}
              disabled={isLayouting || nodes.length === 0}
            >
              {isLayouting ? '⟳' : '⊞'}
            </ControlButton>
            
            {/* 布局方向切换按钮 */}
            <ControlButton
              onClick={toggleLayoutDirection}
              title={`布局方向: ${LAYOUT_DIRECTIONS[layoutOptions.direction].name}`}
              disabled={isLayouting}
            >
              {LAYOUT_DIRECTIONS[layoutOptions.direction].icon}
            </ControlButton>
            
            {/* 网格对齐开关按钮 */}
            <ControlButton
              onClick={toggleSnapToGrid}
              title={`网格对齐: ${alignmentOptions.snapToGrid ? '开启' : '关闭'}`}
              style={{
                backgroundColor: alignmentOptions.snapToGrid ? '#4CAF50' : undefined,
                color: alignmentOptions.snapToGrid ? 'white' : undefined
              }}
            >
              {alignmentOptions.snapToGrid ? '⊞' : '⊡'}
            </ControlButton>
            
            {/* 文本自适应模式切换按钮 */}
            <ControlButton
              onClick={toggleTextAdaptiveMode}
              title={`文本布局: ${
                textAdaptiveMode === 'fixed' ? 'T (固定尺寸)' : 'U (统一尺寸)'
              }`}
              style={{
                backgroundColor: textAdaptiveMode !== 'fixed' ? '#2196F3' : undefined,
                color: textAdaptiveMode !== 'fixed' ? 'white' : undefined
              }}
            >
              {textAdaptiveMode === 'fixed' ? 'T' : 'U'}
            </ControlButton>
            
            {/* 移除连线重叠优化按钮，功能已被智能布局囊括 */}
            
            {/* DAG分析按钮 */}
            <ControlButton
              onClick={handleAnalyzeDAG}
              title="分析DAG连线穿越问题"
              disabled={nodes.length === 0}
              style={{ 
                backgroundColor: 'white',
                color: nodes.length === 0 ? '#9ca3af' : '#374151',
                cursor: nodes.length === 0 ? 'not-allowed' : 'pointer'
              }}
            >
              📊
            </ControlButton>
          </Controls>
          {/* 只在有数据时显示小地图 */}
          {state.dagData && (
            <MiniMap 
              position="bottom-right"
              nodeColor={getNodeColor}
              style={miniMapStyle}
              ariaLabel="DAG 迷你地图"
              pannable={true}
              zoomable={true}
              inversePan={false}
            />
          )}
          <Background 
            gap={30}
            size={1}
            color="#f0f0f0"
            variant={BackgroundVariant.Lines}
            style={{ opacity: 0.6 }}
          />
        </ReactFlow>
      </div>
      </div>

      {/* 参数变量选择弹层 */}
      {paramVarPicker.open && paramVarPicker.anchorRect && paramVarPicker.paramIndex !== null && (
        <div
          ref={paramVarPickerRef}
          style={{
            position: 'fixed',
            top: paramVarPicker.anchorRect.bottom + 4,
            left: paramVarPicker.anchorRect.left,
            zIndex: 2000,
            backgroundColor: 'white',
            border: '1px solid #e5e7eb',
            borderRadius: '6px',
            boxShadow: '0 4px 12px rgba(0,0,0,0.12)',
            maxHeight: '300px',
            display: 'flex',
            flexDirection: 'column',
            minWidth: Math.max(paramVarPicker.anchorRect.width, 220),
            overflow: 'hidden'
          }}
        >
          {/* 搜索框 */}
          <div style={{ padding: '8px', borderBottom: '1px solid #f3f4f6' }}>
            <input 
              autoFocus
              type="text"
              placeholder="搜索..."
              value={paramVarPicker.searchTerm}
              onChange={(e) => setParamVarPicker(prev => ({ ...prev, searchTerm: e.target.value }))}
              style={{
                width: '100%',
                padding: '4px 8px',
                fontSize: '12px',
                border: '1px solid #d1d5db',
                borderRadius: '4px',
                outline: 'none'
              }}
            />
          </div>

          {/* 列表区域 */}
          <div style={{ overflowY: 'auto', flex: 1 }}>
            {paramVarPicker.options
              .filter(opt => 
                opt.label.toLowerCase().includes(paramVarPicker.searchTerm.toLowerCase()) || 
                opt.value.toLowerCase().includes(paramVarPicker.searchTerm.toLowerCase())
              )
              .map((opt, i) => (
                <div
                  key={i}
                  onClick={() => {
                    const idx = paramVarPicker.paramIndex!;
                    const field = paramVarPicker.field;
                    if (field === 'paramKey') {
                      const newParams = [...editingNodeParams];
                      newParams[idx] = { ...newParams[idx], key: opt.value };
                      setEditingNodeParams(newParams);
                    } else if (field === 'paramValue') {
                      const newParams = [...editingNodeParams];
                      newParams[idx] = { ...newParams[idx], value: opt.value };
                      setEditingNodeParams(newParams);
                    } else if (field === 'outputValue') {
                      const newOutputs = [...editingNodeOutputs];
                      newOutputs[idx] = { ...newOutputs[idx], value: opt.value };
                      setEditingNodeOutputs(newOutputs);
                    }

                    setParamVarPicker({ open: false, paramIndex: null, field: null, anchorRect: null, options: [], searchTerm: '' });
                  }}
                  style={{
                    padding: '8px 10px',
                    fontSize: '12px',
                    cursor: 'pointer',
                    borderBottom: '1px solid #f9fafb',
                    backgroundColor: 'white',
                  }}
                  onMouseEnter={(e) => {
                    (e.currentTarget as HTMLDivElement).style.backgroundColor = '#f3f4f6';
                  }}
                  onMouseLeave={(e) => {
                    (e.currentTarget as HTMLDivElement).style.backgroundColor = 'white';
                  }}
                >
                  <div style={{ fontWeight: '500' }}>{opt.label}</div>
                  <div style={{ color: '#6b7280', fontSize: '10px' }}>{opt.value}</div>
                </div>
              ))}
            {paramVarPicker.options.filter(opt => 
              opt.label.toLowerCase().includes(paramVarPicker.searchTerm.toLowerCase()) || 
              opt.value.toLowerCase().includes(paramVarPicker.searchTerm.toLowerCase())
            ).length === 0 && (
              <div style={{ padding: '12px', textAlign: 'center', color: '#9ca3af', fontSize: '12px' }}>
                未找到匹配项
              </div>
            )}
          </div>
        </div>
      )}

      {/* 连线删除提示 */}
      {showDeleteHint && (
        <div className="edge-delete-hint">
          ℹ 选中连线后按 Delete 或 Backspace 键删除
        </div>
      )}

      {/* 节点编辑抽屉 - 水平排列 */}
      {editingNodeId && (
        <div 
          className="node-edit-drawer"
          style={{
            position: 'relative',
            width: `${drawerWidth}px`,
            flexShrink: 0,
            height: '100%',
            display: 'flex',
            flexDirection: 'column'
          }}
        >
          {/* 调整大小的拖动条 */}
          <div
            onMouseDown={() => setIsResizing(true)}
            style={{
              position: 'absolute',
              left: 0,
              top: 0,
              bottom: 0,
              width: '4px',
              cursor: 'col-resize',
              backgroundColor: isResizing ? '#3b82f6' : 'transparent',
              transition: isResizing ? 'none' : 'background-color 0.2s',
              zIndex: 1101
            }}
            onMouseEnter={(e) => {
              if (!isResizing) {
                (e.currentTarget as HTMLElement).style.backgroundColor = '#d1d5db';
              }
            }}
            onMouseLeave={(e) => {
              if (!isResizing) {
                (e.currentTarget as HTMLElement).style.backgroundColor = 'transparent';
              }
            }}
          />
            
            {/* 抽屉头部 */}
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
              }}>编辑节点</h4>
              <button
                onClick={cancelNodeLabelEdit}
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
            
            {/* 抽屉内容 */}
            <div className="node-edit-drawer-content">
            {/* 节点类型（来自节点类型定义） */}
            {editingLoadedNodeTypes.length > 0 && (
              <div className="edit-form-section">
                <label htmlFor="editLoadedNodeType">节点类型</label>
                <input
                  id="editLoadedNodeType"
                  className="edit-input"
                  list="edit-node-type-list"
                  placeholder="搜索或选择节点类型"
                  value={editingSelectedLoadedType}
                  onChange={(e) => {
                    const value = e.target.value;
                    setEditingSelectedLoadedType(value);
                    const found = editingLoadedNodeTypes.find(t => t.action_name === value);
                    if (found) {
                      // 选中节点类型后，仅自动用 desc 填充 name，不再修改 ID
                      if (found.desc) {
                        setEditingNodeName(found.desc);
                      }
                    }
                  }}
                />
                <datalist id="edit-node-type-list">
                  {editingLoadedNodeTypes.map((t) => (
                    <option key={t.action_name} value={t.action_name}>
                      {t.desc ? `${t.action_name} - ${t.desc}` : t.action_name}
                    </option>
                  ))}
                </datalist>
              </div>
            )}

            {/* 节点ID输入 - 设为只读，不允许修改 */}
            <div className="edit-form-section">
              <label htmlFor="editNodeId">节点ID（不可修改）</label>
              <input
                id="editNodeId"
                type="text"
                value={editingNodeId}
                disabled
                className="edit-input disabled-input"
                style={{ backgroundColor: '#f3f4f6', cursor: 'not-allowed', color: '#6b7280' }}
              />
            </div>

            {/* 节点名称输入 */}
            <div className="edit-form-section">
              <label htmlFor="editNodeName">name</label>
              <input
                id="editNodeName"
                type="text"
                value={editingNodeName}
                onChange={(e) => setEditingNodeName(e.target.value)}
                onKeyDown={handleEditKeyDown}
                placeholder="默认填充所选节点类型的描述"
                className="edit-input"
              />
            </div>

            {/* 超时时间输入 */}
            <div className="edit-form-section">
              <label htmlFor="editNodeTimeoutSecs">timeout_secs</label>
              <input
                id="editNodeTimeoutSecs"
                type="number"
                value={editingNodeTimeoutSecs}
                onChange={(e) => setEditingNodeTimeoutSecs(Math.max(0, parseInt(e.target.value) || 0))}
                onKeyDown={handleEditKeyDown}
                placeholder="3600"
                className="edit-input"
                min="0"
              />
            </div>

            {/* 颜色选择 */}
            {/* 节点颜色：不再在编辑界面中手动修改，颜色在每次加载时随机生成 */}

            {/* 参数信息（可编辑） */}
            <div className="edit-form-section">
              <label>输入参数 ({editingNodeParams.length})</label>
              <div
                style={{
                  backgroundColor: '#f9fafb',
                  border: '1px solid #e5e7eb',
                  borderRadius: '6px',
                  padding: '8px',
                  maxHeight: '180px',
                  overflowY: 'auto',
                  fontSize: '12px',
                }}
              >
                {editingNodeParams.length === 0 && (
                  <div style={{ color: '#9ca3af', fontSize: '12px', padding: '4px 0' }}>
                    暂无参数，点击下方“新增参数”添加
                  </div>
                )}

                {editingNodeParams.map((param: any, idx: number) => (
                  <div
                    key={idx}
                    style={{
                      display: 'grid',
                      gridTemplateColumns: '1fr 1fr auto',
                      gap: '4px',
                      alignItems: 'center',
                      padding: '4px 0',
                      borderBottom:
                        idx < editingNodeParams.length - 1 ? '1px solid #e5e7eb' : 'none',
                    }}
                  >
                    {/* key 从节点类型定义的 input 字段中选择 */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: '2px' }}>
                      <button
                        type="button"
                        title="选择预定义参数"
                        onClick={(e) => {
                          const typeDef = editingLoadedNodeTypes.find(
                            (t) => t.action_name === editingSelectedLoadedType
                          );
                          const inputs = (typeDef as any)?.input || (typeDef as any)?.inputs || [];
                          const options: { label: string; value: string }[] = [];
                          const inputOptions: Array<{ name: string; desc?: string }> = [];
                          if (Array.isArray(inputs)) {
                            inputs.forEach((item: any) => {
                              if (typeof item === 'string') {
                                inputOptions.push({ name: item, desc: item });
                              } else if (typeof item === 'object' && item.name) {
                                inputOptions.push({ name: item.name, desc: item.desc || item.name });
                              }
                            });
                          }
                          if (inputOptions.length > 0) {
                            inputOptions.forEach((opt) => options.push({ label: opt.desc || opt.name, value: opt.name }));
                          } else {
                            const existingKeys = editingNodeParams
                              .map((p) => p.key)
                              .filter((k: any) => typeof k === 'string' && k.length > 0);
                            existingKeys.forEach((k) => options.push({ label: k, value: k }));
                          }
                          if (options.length === 0) return;
                          const rect = e.currentTarget.getBoundingClientRect();
                          setParamVarPicker({
                            open: true,
                            paramIndex: idx,
                            field: 'paramKey',
                            anchorRect: rect,
                            options,
                            searchTerm: ''
                          });
                        }}
                        style={{
                          padding: '2px',
                          background: '#f3f4f6',
                          border: '1px solid #d1d5db',
                          borderRadius: '4px',
                          cursor: 'pointer',
                          fontSize: '10px',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center'
                        }}
                      >
                        {'{x}'}
                      </button>
                      <input
                        type="text"
                        value={param.key ?? ''}
                        onChange={(e) => {
                          const newParams = [...editingNodeParams];
                          newParams[idx] = { ...newParams[idx], key: e.target.value };
                          setEditingNodeParams(newParams);
                        }}
                        placeholder="参数名"
                        className="edit-input"
                        style={{ fontSize: '12px', padding: '4px 6px', flex: 1 }}
                      />
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '2px' }}>
                      <button
                        type="button"
                        title="选择变量 (vars/share_data)"
                        onClick={(e) => {
                          const options: { label: string; value: string }[] = [];
                          try {
                            if (state.jsonText) {
                              const json = JSON.parse(state.jsonText);
                              const dagVars = json.vars as Array<{ name: string }> | undefined;
                              if (Array.isArray(dagVars)) {
                                dagVars.forEach((v: any) => {
                                  if (v && typeof v.name === 'string') {
                                    options.push({ label: `vars.${v.name}`, value: `{{${v.name}}}` });
                                  }
                                });
                              }
                            }
                          } catch {}

                          if (state.dagData && editingNodeId) {
                            const nodeMap = new Map(state.dagData.nodes.map((n: any) => [n.id, n]));
                            const edgeList = state.dagData.edges;
                            const upstream = new Set<string>();
                            const stack: string[] = [editingNodeId];
                            while (stack.length > 0) {
                              const current = stack.pop()!;
                              edgeList.filter((e: any) => e.target === current).forEach((e: any) => {
                                if (!upstream.has(e.source)) {
                                  upstream.add(e.source);
                                  stack.push(e.source);
                                }
                              });
                            }
                            upstream.forEach((nid) => {
                              const n = nodeMap.get(nid);
                              const original = (n as any)?.data?.original;
                              const outs = original?.outputs as Array<{ key: string; value: string }> | undefined;
                              if (Array.isArray(outs)) {
                                outs.forEach((o) => {
                                  if (o && typeof o.key === 'string') {
                                    options.push({ label: `share_data.${nid}.${o.key}`, value: `{{.shareData.${o.key}}}` });
                                  }
                                });
                              }
                            });
                          }
                          if (options.length === 0) return;
                          const rect = e.currentTarget.getBoundingClientRect();
                          setParamVarPicker({
                            open: true,
                            paramIndex: idx,
                            field: 'paramValue',
                            anchorRect: rect,
                            options,
                            searchTerm: ''
                          });
                        }}
                        style={{
                          padding: '2px',
                          background: '#f3f4f6',
                          border: '1px solid #d1d5db',
                          borderRadius: '4px',
                          cursor: 'pointer',
                          fontSize: '10px',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center'
                        }}
                      >
                        {'{x}'}
                      </button>
                      <input
                        type="text"
                        value={
                          typeof param.value === 'string'
                            ? param.value
                            : param.value != null
                            ? JSON.stringify(param.value)
                            : ''
                        }
                        onChange={(e) => {
                          const text = e.target.value;
                          const value: any = text;
                          const newParams = [...editingNodeParams];
                          newParams[idx] = { ...newParams[idx], value };
                          setEditingNodeParams(newParams);
                        }}
                        placeholder="参数值"
                        className="edit-input"
                        style={{ fontSize: '12px', padding: '4px 6px', flex: 1 }}
                      />
                    </div>
                    <button
                      type="button"
                      onClick={() => {
                        const newParams = editingNodeParams.filter((_, i) => i !== idx);
                        setEditingNodeParams(newParams);
                      }}
                      style={{
                        marginLeft: '4px',
                        padding: '2px 6px',
                        fontSize: '12px',
                        color: '#b91c1c',
                        background: 'transparent',
                        border: '1px solid #fecaca',
                        borderRadius: '4px',
                        cursor: 'pointer',
                      }}
                    >
                      删除
                    </button>
                  </div>
                ))}
              </div>

              <button
                type="button"
                onClick={() =>
                  setEditingNodeParams([
                    ...editingNodeParams,
                    { key: '', value: '' },
                  ])
                }
                style={{
                  marginTop: '6px',
                  padding: '4px 8px',
                  fontSize: '12px',
                  borderRadius: '4px',
                  border: '1px dashed #94a3b8',
                  backgroundColor: '#f9fafb',
                  cursor: 'pointer',
                  color: '#374151',
                }}
              >
                + 新增参数
              </button>
            </div>

            {/* 输出信息（可编辑） */}
            <div className="edit-form-section">
              <label>输出结果 ({editingNodeOutputs.length})</label>
              <div style={{ color: '#6b7280', fontSize: '11px', marginBottom: '4px', fontStyle: 'italic' }}>
                注：以 _ 开头且以 _ 结尾的 key（如 _temp_）不会包含在 Kafka 消息体中
              </div>
              <div
                style={{
                  backgroundColor: '#f9fafb',
                  border: '1px solid #e5e7eb',
                  borderRadius: '6px',
                  padding: '8px',
                  maxHeight: '180px',
                  overflowY: 'auto',
                  fontSize: '12px',
                }}
              >
                {editingNodeOutputs.length === 0 && (
                  <div style={{ color: '#9ca3af', fontSize: '12px', padding: '4px 0' }}>
                    暂无输出，点击下方“新增输出”添加
                  </div>
                )}

                {editingNodeOutputs.map((output: any, idx: number) => (
                  <div
                    key={idx}
                    style={{
                      display: 'grid',
                      gridTemplateColumns: '1fr 1fr auto',
                      gap: '4px',
                      alignItems: 'center',
                      padding: '4px 0',
                      borderBottom:
                        idx < editingNodeOutputs.length - 1 ? '1px solid #e5e7eb' : 'none',
                    }}
                  >
                    <input
                      type="text"
                      value={output.key ?? ''}
                      onChange={(e) => {
                        const newOutputs = [...editingNodeOutputs];
                        newOutputs[idx] = { ...newOutputs[idx], key: e.target.value };
                        setEditingNodeOutputs(newOutputs);
                      }}
                      placeholder="输出名 (key)"
                      className="edit-input"
                      style={{ fontSize: '12px', padding: '4px 6px' }}
                    />
                    <div style={{ display: 'flex', alignItems: 'center', gap: '2px' }}>
                      <button
                        type="button"
                        title="选择预定义输出"
                        onClick={(e) => {
                          const typeDef = editingLoadedNodeTypes.find(
                            (t) => t.action_name === editingSelectedLoadedType
                          );
                          const outs = (typeDef as any)?.outputs || (typeDef as any)?.output || [];
                          const options: { label: string; value: string }[] = [];
                          const outputOptions: Array<{ name: string; desc?: string }> = [];

                          if (Array.isArray(outs)) {
                            outs.forEach((item: any) => {
                              if (typeof item === 'string') {
                                outputOptions.push({ name: item, desc: item });
                              } else if (typeof item === 'object' && item.name) {
                                outputOptions.push({ name: item.name, desc: item.desc || item.name });
                              }
                            });
                          }

                          if (outputOptions.length > 0) {
                            outputOptions.forEach((opt) => options.push({ label: opt.desc || opt.name, value: opt.name }));
                          } else {
                            const existingValues = editingNodeOutputs
                              .map((p) => p.value)
                              .filter((v: any) => typeof v === 'string' && v.length > 0);
                            existingValues.forEach((v) => options.push({ label: v, value: v }));
                          }

                          if (options.length === 0) return;
                          const rect = e.currentTarget.getBoundingClientRect();
                          setParamVarPicker({
                            open: true,
                            paramIndex: idx,
                            field: 'outputValue',
                            anchorRect: rect,
                            options,
                            searchTerm: ''
                          });
                        }}
                        style={{
                          padding: '2px',
                          background: '#f3f4f6',
                          border: '1px solid #d1d5db',
                          borderRadius: '4px',
                          cursor: 'pointer',
                          fontSize: '10px',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center'
                        }}
                      >
                        {'{x}'}
                      </button>
                      {/* value 从节点类型定义的 outputs 字段中选择 */}
                      <input
                        type="text"
                        value={output.value ?? ''}
                        onChange={(e) => {
                          const newOutputs = [...editingNodeOutputs];
                          newOutputs[idx] = { ...newOutputs[idx], value: e.target.value };
                          setEditingNodeOutputs(newOutputs);
                        }}
                        placeholder="输出路径"
                        className="edit-input"
                        style={{ fontSize: '12px', padding: '4px 6px', flex: 1 }}
                      />
                    </div>
                    <button
                      type="button"
                      onClick={() => {
                        const newOutputs = editingNodeOutputs.filter((_, i) => i !== idx);
                        setEditingNodeOutputs(newOutputs);
                      }}
                      style={{
                        marginLeft: '4px',
                        padding: '2px 6px',
                        fontSize: '12px',
                        color: '#b91c1c',
                        background: 'transparent',
                        border: '1px solid #fecaca',
                        borderRadius: '4px',
                        cursor: 'pointer',
                      }}
                    >
                      删除
                    </button>
                  </div>
                ))}
              </div>

              <button
                type="button"
                onClick={() =>
                  setEditingNodeOutputs([
                    ...editingNodeOutputs,
                    { key: '', value: '' },
                  ])
                }
                style={{
                  marginTop: '6px',
                  padding: '4px 8px',
                  fontSize: '12px',
                  borderRadius: '4px',
                  border: '1px dashed #94a3b8',
                  backgroundColor: '#f9fafb',
                  cursor: 'pointer',
                  color: '#374151',
                }}
              >
                + 新增输出
              </button>
            </div>

            {/* 前置检查信息（可编辑） */}
            <div className="edit-form-section">
              <label>前置检查 ({editingNodePreChecks.length})</label>
              <div
                style={{
                  backgroundColor: '#f9fafb',
                  border: '1px solid #e5e7eb',
                  borderRadius: '6px',
                  padding: '8px',
                  maxHeight: '260px',
                  overflowY: 'auto',
                  fontSize: '12px',
                }}
              >
                {editingNodePreChecks.length === 0 && (
                  <div style={{ color: '#9ca3af', fontSize: '12px', padding: '4px 0' }}>
                    暂无前置检查，点击下方“新增前置检查”添加
                  </div>
                )}

                {editingNodePreChecks.map((check: any, idx: number) => {
                  const isExpanded = expandedPreCheckIndexes.has(idx);
                  const conditions = check.check?.conditions || [];

                  return (
                    <div
                      key={idx}
                      style={{
                        borderBottom:
                          idx < editingNodePreChecks.length - 1
                            ? '1px solid #e5e7eb'
                            : 'none',
                        padding: '6px 0',
                      }}
                    >
                      {/* 头部：名称 + active_action + 展开/删除 */}
                      <div
                        style={{
                          display: 'grid',
                          gridTemplateColumns: '1.6fr 1.1fr auto auto',
                          gap: '4px',
                          alignItems: 'center',
                        }}
                      >
                        {/* 前置检查名称 */}
                        <input
                          type="text"
                          value={check.name ?? ''}
                          onChange={(e) => {
                            const newChecks = [...editingNodePreChecks];
                            newChecks[idx] = { ...newChecks[idx], name: e.target.value };
                            setEditingNodePreChecks(newChecks);
                          }}
                          placeholder="检查名称"
                          className="edit-input"
                          style={{ fontSize: '12px', padding: '4px 6px' }}
                        />

                        {/* active_action：默认 skip，仅在内部使用，不再展示“可选”提示 */}
                        <select
                          value={check.check?.active_action ?? 'skip'}
                          onChange={(e) => {
                            const newChecks = [...editingNodePreChecks];
                            const prev = newChecks[idx] || {};
                            const prevCheck = prev.check || {};
                            newChecks[idx] = {
                              ...prev,
                              check: {
                                ...prevCheck,
                                active_action: e.target.value || 'skip',
                                conditions: prevCheck.conditions || [
                                  { source: 'vars', key: '', value: [""], operator: 'in' }
                                ],
                              },
                            };
                            setEditingNodePreChecks(newChecks);
                          }}
                          className="edit-input"
                          style={{ fontSize: '12px', padding: '4px 6px' }}
                        >
                          <option value="skip">skip</option>
                          <option value="block">block</option>
                        </select>

                        {/* 展开/收起条件 */}
                        <button
                          type="button"
                          onClick={() => {
                            setExpandedPreCheckIndexes((prev) => {
                              const next = new Set(prev);
                              if (next.has(idx)) {
                                next.delete(idx);
                              } else {
                                next.add(idx);
                              }
                              return next;
                            });
                          }}
                          style={{
                            padding: '2px 6px',
                            fontSize: '12px',
                            borderRadius: '4px',
                            border: '1px solid #d1d5db',
                            backgroundColor: '#f9fafb',
                            cursor: 'pointer',
                            color: '#374151',
                          }}
                        >
                          {isExpanded ? '收起条件' : `条件 (${conditions.length})`}
                        </button>

                        {/* 删除前置检查 */}
                        <button
                          type="button"
                          onClick={() => {
                            const newChecks = editingNodePreChecks.filter((_, i) => i !== idx);
                            setEditingNodePreChecks(newChecks);
                            setExpandedPreCheckIndexes((prev) => {
                              const next = new Set(prev);
                              next.delete(idx);
                              return next;
                            });
                          }}
                          style={{
                            marginLeft: '4px',
                            padding: '2px 6px',
                            fontSize: '12px',
                            color: '#b91c1c',
                            background: 'transparent',
                            border: '1px solid #fecaca',
                            borderRadius: '4px',
                            cursor: 'pointer',
                          }}
                        >
                          删除
                        </button>
                      </div>

                      {/* 条件列表 */}
                      {isExpanded && (
                        <div
                          style={{
                            marginTop: '6px',
                            padding: '6px',
                            backgroundColor: '#f3f4f6',
                            borderRadius: '4px',
                          }}
                        >
                          {/* 表头 */}
                          <div
                            style={{
                              display: 'grid',
                              gridTemplateColumns: '1fr 1fr 1.2fr 1fr auto',
                              gap: '4px',
                              fontSize: '11px',
                              fontWeight: 500,
                              color: '#6b7280',
                              marginBottom: '4px',
                            }}
                          >
                            <span>Source</span>
                            <span>Key</span>
                            <span>Value</span>
                            <span>Operator</span>
                            <span></span>
                          </div>

                          {/* 条件行 */}
                          {conditions.map((cond: any, cIdx: number) => (
                            <div
                              key={cIdx}
                              style={{
                                display: 'grid',
                                gridTemplateColumns: '1fr 1fr 1.2fr 1fr auto',
                                gap: '4px',
                                alignItems: 'center',
                                marginBottom: '4px',
                              }}
                            >
                              {/* source：默认 vars */}
                              <select
                                value={cond.source || 'vars'}
                                onChange={(e) => {
                                  const newSource = e.target.value;
                                  setEditingNodePreChecks((prev) => {
                                    const next = [...prev];
                                    const prevCheck = next[idx] || {};
                                    const prevCheckInner = prevCheck.check || {};
                                    const prevConds = prevCheckInner.conditions || [];
                                    const current = prevConds[cIdx] || {};
                                    const newConds = [...prevConds];
                                    newConds[cIdx] = {
                                      ...current,
                                      source: newSource,
                                      key: '' // 切换 source 时清空 key 以触发重新选择
                                    };
                                    next[idx] = {
                                      ...prevCheck,
                                      check: {
                                        ...prevCheckInner,
                                        conditions: newConds,
                                      },
                                    };
                                    return next;
                                  });
                                }}
                                className="edit-input"
                                style={{ fontSize: '12px', padding: '4px 6px' }}
                              >
                                <option value="vars">vars</option>
                                <option value="share_data">share_data</option>
                              </select>

                              {/* key：根据 source 选择不同的下拉内容 */}
                              <select
                                value={cond.key || ''}
                                onChange={(e) => {
                                  setEditingNodePreChecks((prev) => {
                                    const next = [...prev];
                                    const prevCheck = next[idx] || {};
                                    const prevCheckInner = prevCheck.check || {};
                                    const prevConds = prevCheckInner.conditions || [];
                                    const current = prevConds[cIdx] || {};
                                    const newConds = [...prevConds];
                                    newConds[cIdx] = {
                                      ...current,
                                      key: e.target.value,
                                    };
                                    next[idx] = {
                                      ...prevCheck,
                                      check: {
                                        ...prevCheckInner,
                                        conditions: newConds,
                                      },
                                    };
                                    return next;
                                  });
                                }}
                                className="edit-input"
                                style={{ fontSize: '12px', padding: '4px 6px' }}
                              >
                                {(() => {
                                  const src = cond.source || 'vars';
                                  const options: { label: string; value: string }[] = [];

                                  // vars 源：从 DAG vars 中取 name
                                  if (src === 'vars') {
                                    try {
                                      if (state.jsonText) {
                                        const json = JSON.parse(state.jsonText);
                                        const dagVars = json.vars as Array<{ name: string }> | undefined;
                                        if (Array.isArray(dagVars)) {
                                          dagVars.forEach((v: any) => {
                                            if (v && typeof v.name === 'string') {
                                              options.push({ label: v.name, value: v.name });
                                            }
                                          });
                                        }
                                      }
                                    } catch {
                                      // ignore parse error
                                    }
                                  }

                                  // share_data 源：从所有上游节点 outputs 的 key 中收集
                                  if (src === 'share_data' && state.dagData && editingNodeId) {
                                    const nodeMap = new Map(
                                      state.dagData.nodes.map((n: any) => [n.id, n])
                                    );
                                    const edgeList = state.dagData.edges;

                                    const upstream = new Set<string>();
                                    const stack: string[] = [editingNodeId];
                                    while (stack.length > 0) {
                                      const current = stack.pop()!;
                                      edgeList
                                        .filter((e: any) => e.target === current)
                                        .forEach((e: any) => {
                                          if (!upstream.has(e.source)) {
                                            upstream.add(e.source);
                                            stack.push(e.source);
                                          }
                                        });
                                    }

                                    upstream.forEach((nid) => {
                                      const n = nodeMap.get(nid);
                                      const original = (n as any)?.data?.original;
                                      const outs = original?.outputs as
                                        | Array<{ key: string; value: string }>
                                        | undefined;
                                      if (Array.isArray(outs)) {
                                        outs.forEach((o) => {
                                          if (o && typeof o.key === 'string') {
                                            // 展示: nodeId.key，真实值: 仅 key
                                            options.push({ label: `${nid}.${o.key}`, value: o.key });
                                          }
                                        });
                                      }
                                    });
                                  }

                                  return (
                                    <>
                                      <option value="">{options.length === 0 ? '无可用 key' : '选择 key'}</option>
                                      {options.map((opt, i) => (
                                        <option key={i} value={opt.value}>
                                          {opt.label}
                                        </option>
                                      ))}
                                    </>
                                  );
                                })()}
                              </select>

                              {/* value：数组格式，直接接收用户输入的 JSON 数组 */}
                              <input
                                type="text"
                                value={
                                  Array.isArray(cond.value)
                                    ? JSON.stringify(cond.value)
                                    : typeof cond.value === 'string' && cond.value.length > 0
                                    ? cond.value
                                    : ''
                                }
                                onChange={(e) => {
                                  const text = e.target.value.trim();
                                  let value: any = text;
                                  
                                  // 尝试解析为 JSON 数组
                                  if (text.startsWith('[') && text.endsWith(']')) {
                                    try {
                                      value = JSON.parse(text);
                                      if (!Array.isArray(value)) {
                                        value = text; // 如果解析结果不是数组，保持原文本
                                      }
                                    } catch (e) {
                                      // JSON 解析失败，保持原文本
                                      value = text;
                                    }
                                  }

                                  setEditingNodePreChecks((prev) => {
                                    const next = [...prev];
                                    const prevCheck = next[idx] || {};
                                    const prevCheckInner = prevCheck.check || {};
                                    const prevConds = prevCheckInner.conditions || [];
                                    const current = prevConds[cIdx] || {};
                                    const newConds = [...prevConds];
                                    newConds[cIdx] = {
                                      ...current,
                                      value: value,
                                    };
                                    next[idx] = {
                                      ...prevCheck,
                                      check: {
                                        ...prevCheckInner,
                                        conditions: newConds,
                                      },
                                    };
                                    return next;
                                  });
                                }}
                                placeholder='输入 JSON 数组，例如: ["a", "b"] 或 [""]'
                                className="edit-input"
                                style={{ fontSize: '12px', padding: '4px 6px' }}
                              />

                              {/* operator：默认 in，仅在内部使用 */}
                              <select
                                value={cond.operator ?? 'in'}
                                onChange={(e) => {
                                  setEditingNodePreChecks((prev) => {
                                    const next = [...prev];
                                    const prevCheck = next[idx] || {};
                                    const prevCheckInner = prevCheck.check || {};
                                    const prevConds = prevCheckInner.conditions || [];
                                    const current = prevConds[cIdx] || {};
                                    const newConds = [...prevConds];
                                    newConds[cIdx] = {
                                      ...current,
                                      operator: e.target.value,
                                    };
                                    next[idx] = {
                                      ...prevCheck,
                                      check: {
                                        ...prevCheckInner,
                                        conditions: newConds,
                                      },
                                    };
                                    return next;
                                  });
                                }}
                                className="edit-input"
                                style={{ fontSize: '12px', padding: '4px 6px' }}
                              >
                                <option value="in">in</option>
                                <option value="not_in">not_in</option>
                              </select>

                              {/* 删除条件 */}
                              <button
                                type="button"
                                onClick={() => {
                                  setEditingNodePreChecks((prev) => {
                                    const next = [...prev];
                                    const prevCheck = next[idx] || {};
                                    const prevCheckInner = prevCheck.check || {};
                                    const prevConds = prevCheckInner.conditions || [];
                                    const newConds = prevConds.filter((_: any, i: number) => i !== cIdx);
                                    next[idx] = {
                                      ...prevCheck,
                                      check: {
                                        ...prevCheckInner,
                                        conditions: newConds,
                                      },
                                    };
                                    return next;
                                  });
                                }}
                                style={{
                                  padding: '2px 6px',
                                  fontSize: '12px',
                                  color: '#b91c1c',
                                  background: 'transparent',
                                  border: '1px solid #fecaca',
                                  borderRadius: '4px',
                                  cursor: 'pointer',
                                }}
                              >
                                删除
                              </button>
                            </div>
                          ))}

                          {/* 新增条件 */}
                          <button
                            type="button"
                            onClick={() => {
                              setEditingNodePreChecks((prev) => {
                                const next = [...prev];
                                const prevCheck = next[idx] || {};
                                const prevCheckInner = prevCheck.check || {};
                                const prevConds = prevCheckInner.conditions || [];
                                next[idx] = {
                                  ...prevCheck,
                                  check: {
                                    ...prevCheckInner,
                                    conditions: [
                                      ...prevConds,
                                      {
                                        source: 'vars',
                                        key: '',
                                        value: [""],
                                        operator: 'in',
                                      },
                                    ],
                                  },
                                };
                                return next;
                              });
                            }}
                            style={{
                              marginTop: '4px',
                              padding: '2px 6px',
                              fontSize: '12px',
                              borderRadius: '4px',
                              border: '1px dashed #9ca3af',
                              backgroundColor: '#e5e7eb',
                              cursor: 'pointer',
                              color: '#374151',
                            }}
                          >
                            + 新增条件
                          </button>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>

              <button
                type="button"
                onClick={() =>
                  setEditingNodePreChecks([
                    ...editingNodePreChecks,
                    { name: '', check: { active_action: 'skip', conditions: [{ source: 'vars', key: '', value: [""], operator: 'in' }] } },
                  ])
                }
                style={{
                  marginTop: '6px',
                  padding: '4px 8px',
                  fontSize: '12px',
                  borderRadius: '4px',
                  border: '1px dashed #94a3b8',
                  backgroundColor: '#f9fafb',
                  cursor: 'pointer',
                  color: '#374151',
                }}
              >
                + 新增前置检查
              </button>
            </div>

            {/* 抽屉底部操作按钮 */}
            <div style={{
              display: 'flex',
              justifyContent: 'flex-end',
              gap: '12px',
              padding: '16px 20px',
              borderTop: '1px solid #e5e7eb',
              backgroundColor: '#f9fafb'
            }}>
              <button 
                onClick={cancelNodeLabelEdit}
                style={{
                  padding: '10px 20px',
                  border: '1px solid #d1d5db',
                  background: '#ffffff',
                  color: '#374151',
                  borderRadius: '6px',
                  fontSize: '14px',
                  cursor: 'pointer',
                  transition: 'all 0.2s'
                }}
                onMouseEnter={(e) => {
                  (e.currentTarget as HTMLElement).style.backgroundColor = '#f9fafb';
                  (e.currentTarget as HTMLElement).style.borderColor = '#9ca3af';
                }}
                onMouseLeave={(e) => {
                  (e.currentTarget as HTMLElement).style.backgroundColor = '#ffffff';
                  (e.currentTarget as HTMLElement).style.borderColor = '#d1d5db';
                }}
              >
                取消
              </button>
              <button 
                onClick={saveNodeLabelEdit}
                style={{
                  padding: '10px 20px',
                  border: 'none',
                  background: '#22c55e',
                  color: '#ffffff',
                  borderRadius: '6px',
                  fontSize: '14px',
                  fontWeight: '500',
                  cursor: 'pointer',
                  transition: 'all 0.2s'
                }}
                onMouseEnter={(e) => {
                  (e.currentTarget as HTMLElement).style.backgroundColor = '#16a34a';
                  (e.currentTarget as HTMLElement).style.transform = 'translateY(-1px)';
                }}
                onMouseLeave={(e) => {
                  (e.currentTarget as HTMLElement).style.backgroundColor = '#22c55e';
                  (e.currentTarget as HTMLElement).style.transform = 'translateY(0)';
                }}
              >
                保存更改
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 变量编辑抽屉 - 水平排列 */}
      {isVarsEditorOpen && (
        <div 
          className="node-edit-drawer"
          style={{
            position: 'relative',
            width: `${drawerWidth}px`,
            flexShrink: 0,
            height: '100%',
            display: 'flex',
            flexDirection: 'column'
          }}
        >
          {/* 抽屉头部 */}
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
            }}>工作流配置</h4>
            <button
              onClick={() => setIsVarsEditorOpen(false)}
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
            
          {/* 抽屉内容 */}
          <div className="node-edit-drawer-content">
            {/* 工作流基础信息编辑区域 */}
            <div className="edit-form-section">
              <label htmlFor="workflowDagId">工作流ID (dag_id)</label>
              <input
                id="workflowDagId"
                className="edit-input"
                type="text"
                value={workflowInfo.dag_id || ''}
                onChange={(e) => handleWorkflowInfoChange('dag_id', e.target.value)}
                placeholder="例如: my_workflow"
              />
            </div>

            <div className="edit-form-section">
              <label htmlFor="workflowName">工作流名称 (name)</label>
              <input
                id="workflowName"
                className="edit-input"
                type="text"
                value={workflowInfo.name || ''}
                onChange={(e) => handleWorkflowInfoChange('name', e.target.value)}
                placeholder="例如: 我的工作流"
              />
            </div>

            <div className="edit-form-section">
              <label htmlFor="workflowDesc">工作流描述 (desc)</label>
              <textarea
                id="workflowDesc"
                className="edit-input"
                value={workflowInfo.desc || ''}
                onChange={(e) => handleWorkflowInfoChange('desc', e.target.value)}
                placeholder="描述这个工作流的用途..."
                rows={2}
                style={{ fontFamily: 'inherit', resize: 'vertical' }}
              />
            </div>

            {/* 分隔线 */}
            <div style={{
              height: '1px',
              backgroundColor: '#e5e7eb',
              margin: '20px 0',
              width: '100%'
            }} />

            {/* 变量列表标题 */}
            <div className="edit-form-section" style={{ marginBottom: '12px' }}>
              <label style={{ marginBottom: '12px', fontSize: '14px', fontWeight: '600', color: '#374151' }}>工作流变量</label>
            </div>

            {vars.length === 0 ? (
              <div className="edit-form-section" style={{ 
                textAlign: 'center', 
                padding: '40px 20px', 
                backgroundColor: '#f9fafb',
                borderRadius: '8px',
                border: '2px dashed #d1d5db'
              }}>
                <div style={{ fontSize: '36px', marginBottom: '12px' }}>📝</div>
                <p style={{ fontSize: '14px', color: '#6b7280', margin: '0 0 6px 0' }}>暂无变量</p>
                <p style={{ fontSize: '12px', color: '#9ca3af', margin: 0 }}>点击下方"添加变量"按钮来创建新变量</p>
              </div>
            ) : (
              <div style={{ marginBottom: '20px' }}>
                <table style={{ 
                  width: '100%', 
                  borderCollapse: 'collapse',
                  fontSize: '13px',
                  border: '1px solid #e5e7eb',
                  borderRadius: '6px',
                  overflow: 'hidden'
                }}>
                  <thead>
                    <tr style={{ 
                      backgroundColor: '#f9fafb',
                      borderBottom: '1px solid #e5e7eb'
                    }}>
                      <th style={{ 
                        padding: '10px 12px', 
                        textAlign: 'left', 
                        fontWeight: '600',
                        color: '#374151',
                        fontSize: '13px'
                      }}>变量名 <span style={{ color: '#ef4444' }}>*</span></th>
                      <th style={{ 
                        padding: '10px 12px', 
                        textAlign: 'left', 
                        fontWeight: '600',
                        color: '#374151',
                        fontSize: '13px'
                      }}>默认值</th>
                      <th style={{ 
                        padding: '10px 12px', 
                        textAlign: 'left', 
                        fontWeight: '600',
                        color: '#374151',
                        fontSize: '13px'
                      }}>描述</th>
                      <th style={{ 
                        padding: '10px 12px', 
                        textAlign: 'center', 
                        fontWeight: '600',
                        color: '#374151',
                        fontSize: '13px',
                        width: '70px'
                      }}>操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {vars.map((variable, index) => (
                      <tr 
                        key={index}
                        style={{
                          borderBottom: index < vars.length - 1 ? '1px solid #e5e7eb' : 'none',
                          transition: 'background-color 0.2s'
                        }}
                        onMouseEnter={(e) => e.currentTarget.style.backgroundColor = '#f9fafb'}
                        onMouseLeave={(e) => e.currentTarget.style.backgroundColor = 'transparent'}
                      >
                        <td style={{ padding: '10px 12px' }}>
                          <input
                            className="edit-input"
                            type="text"
                            value={variable.name}
                            onChange={(e) => handleVarChange(index, 'name', e.target.value)}
                            placeholder="变量名"
                            style={{
                              width: '100%',
                              padding: '8px 10px',
                              fontSize: '13px'
                            }}
                          />
                        </td>
                        <td style={{ padding: '10px 12px' }}>
                          <input
                            className="edit-input"
                            type="text"
                            value={variable.default_value || ''}
                            onChange={(e) => handleVarChange(index, 'default_value', e.target.value)}
                            placeholder="默认值"
                            style={{
                              width: '100%',
                              padding: '8px 10px',
                              fontSize: '13px'
                            }}
                          />
                        </td>
                        <td style={{ padding: '10px 12px' }}>
                          <input
                            className="edit-input"
                            type="text"
                            value={variable.desc || ''}
                            onChange={(e) => handleVarChange(index, 'desc', e.target.value)}
                            placeholder="描述"
                            style={{
                              width: '100%',
                              padding: '8px 10px',
                              fontSize: '13px'
                            }}
                          />
                        </td>
                        <td style={{ padding: '10px 12px', textAlign: 'center' }}>
                          <button
                            onClick={() => handleRemoveVar(index)}
                            style={{
                              padding: '6px 12px',
                              backgroundColor: '#fee2e2',
                              color: '#dc2626',
                              border: 'none',
                              borderRadius: '4px',
                              cursor: 'pointer',
                              fontSize: '12px',
                              fontWeight: '500',
                              transition: 'all 0.2s'
                            }}
                            onMouseOver={(e) => e.currentTarget.style.backgroundColor = '#fecaca'}
                            onMouseOut={(e) => e.currentTarget.style.backgroundColor = '#fee2e2'}
                          >
                            删除
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* 抽屉底部按钮 */}
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
                onClick={() => setIsVarsEditorOpen(false)}
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
                onClick={handleSaveVars}
                disabled={!varsHasChanges}
                className="save-btn"
                style={{
                  padding: '10px 20px',
                  backgroundColor: varsHasChanges ? '#22c55e' : '#d1d5db',
                  color: varsHasChanges ? 'white' : '#9ca3af',
                  border: 'none',
                  borderRadius: '6px',
                  cursor: varsHasChanges ? 'pointer' : 'not-allowed',
                  fontSize: '14px',
                  fontWeight: '500',
                  transition: 'all 0.2s',
                  minWidth: '80px'
                }}
                onMouseOver={(e) => varsHasChanges && (e.currentTarget.style.backgroundColor = '#16a34a')}
                onMouseOut={(e) => varsHasChanges && (e.currentTarget.style.backgroundColor = '#22c55e')}
              >
                保存
              </button>
            </div>
          </div>
        </div>
      )}
      
      {/* 全局鼠标移动监听 - 用于调整抽屉大小 */}
      {isResizing && (
        <div
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            zIndex: 501,
            cursor: 'col-resize'
          }}
          onMouseMove={(e) => {
            const newWidth = window.innerWidth - e.clientX;
            if (newWidth >= 300 && newWidth <= 800) {
              setDrawerWidth(newWidth);
            }
          }}
          onMouseUp={() => setIsResizing(false)}
          onMouseLeave={() => setIsResizing(false)}
        />
      )}

      {/* 右键菜单 */}
      {contextMenuPosition && (
        <div 
          className="context-menu"
          style={{
            position: 'fixed',
            top: contextMenuPosition.y,
            left: contextMenuPosition.x,
            zIndex: 1000
          }}
          onClick={(e) => e.stopPropagation()}
        >
          <button 
            className="context-menu-item"
            onClick={handleQuickAddNode}
          >
            ➕ 快速添加节点
          </button>
        </div>
      )}

      {/* 节点创建对话框 */}
      <NodeCreationDialog
        isOpen={isDialogOpen}
        position={dialogPosition}
        onClose={closeNodeCreationDialog}
        onCreateNode={handleCreateNode}
      />
      
      {/* DAG分析结果模态框 */}
      {showAnalysisModal && analysisResult && (
        <div 
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: 'rgba(0, 0, 0, 0.5)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 1000
          }}
          onClick={() => setShowAnalysisModal(false)}
        >
          <div 
            style={{
              backgroundColor: 'white',
              borderRadius: '8px',
              padding: '24px',
              maxWidth: '32rem',
              maxHeight: '24rem',
              overflowY: 'auto',
              boxShadow: '0 10px 25px rgba(0, 0, 0, 0.1)',
              margin: '20px'
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <div style={{ 
              display: 'flex', 
              justifyContent: 'space-between', 
              alignItems: 'center', 
              marginBottom: '16px',
              borderBottom: '1px solid #e5e7eb',
              paddingBottom: '12px'
            }}>
              <h3 style={{ 
                fontSize: '18px', 
                fontWeight: '600', 
                color: '#111827',
                margin: 0
              }}>
                📊 DAG连线穿越分析报告
              </h3>
              <button 
                onClick={() => setShowAnalysisModal(false)}
                style={{
                  background: 'none',
                  border: 'none',
                  fontSize: '20px',
                  color: '#9ca3af',
                  cursor: 'pointer',
                  padding: '4px',
                  borderRadius: '4px'
                }}
                onMouseOver={(e) => (e.target as HTMLElement).style.color = '#6b7280'}
                onMouseOut={(e) => (e.target as HTMLElement).style.color = '#9ca3af'}
              >
                ✕
              </button>
            </div>
            
            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              {/* 基础统计 */}
              <div style={{ 
                display: 'grid', 
                gridTemplateColumns: '1fr 1fr', 
                gap: '16px', 
                fontSize: '14px',
                backgroundColor: '#f9fafb',
                padding: '12px',
                borderRadius: '6px'
              }}>
                <div>
                  <span style={{ fontWeight: '500', color: '#374151' }}>节点总数:</span> 
                  <span style={{ color: '#1f2937', marginLeft: '8px' }}>{analysisResult.totalNodes}</span>
                </div>
                <div>
                  <span style={{ fontWeight: '500', color: '#374151' }}>连线总数:</span> 
                  <span style={{ color: '#1f2937', marginLeft: '8px' }}>{analysisResult.totalEdges}</span>
                </div>
              </div>
              
              {/* 严重程度统计 */}
              <div>
                <h4 style={{ 
                  fontWeight: '500', 
                  color: '#374151', 
                  marginBottom: '8px',
                  fontSize: '14px',
                  margin: '0 0 8px 0'
                }}>
                  连线穿越统计
                </h4>
                <div style={{ 
                  display: 'grid', 
                  gridTemplateColumns: '1fr 1fr 1fr', 
                  gap: '8px', 
                  fontSize: '13px' 
                }}>
                  <div style={{ 
                    color: '#dc2626',
                    padding: '6px 8px',
                    backgroundColor: '#fef2f2',
                    borderRadius: '4px',
                    textAlign: 'center'
                  }}>
                    严重: {analysisResult.severitySummary.high}
                  </div>
                  <div style={{ 
                    color: '#d97706',
                    padding: '6px 8px',
                    backgroundColor: '#fffbeb',
                    borderRadius: '4px',
                    textAlign: 'center'
                  }}>
                    中等: {analysisResult.severitySummary.medium}
                  </div>
                  <div style={{ 
                    color: '#16a34a',
                    padding: '6px 8px',
                    backgroundColor: '#f0fdf4',
                    borderRadius: '4px',
                    textAlign: 'center'
                  }}>
                    轻微: {analysisResult.severitySummary.low}
                  </div>
                </div>
              </div>
              
              {/* 优化建议 */}
              {analysisResult.suggestions.length > 0 && (
                <div>
                  <h4 style={{ 
                    fontWeight: '500', 
                    color: '#374151', 
                    marginBottom: '8px',
                    fontSize: '14px',
                    margin: '0 0 8px 0'
                  }}>
                    💡 优化建议
                  </h4>
                  <ul style={{ 
                    fontSize: '13px', 
                    margin: 0, 
                    paddingLeft: '16px',
                    color: '#4b5563'
                  }}>
                    {analysisResult.suggestions.map((suggestion: string, index: number) => (
                      <li key={index} style={{ marginBottom: '4px' }}>
                        {suggestion}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
              
              {/* 问题连线详情 */}
              {analysisResult.crossingEdges.length > 0 && (
                <div>
                  <h4 style={{ 
                    fontWeight: '500', 
                    color: '#374151', 
                    marginBottom: '8px',
                    fontSize: '14px',
                    margin: '0 0 8px 0'
                  }}>
                    🔗 问题连线详情 ({analysisResult.crossingEdges.length}条)
                  </h4>
                  <div style={{ 
                    maxHeight: '120px', 
                    overflowY: 'auto', 
                    fontSize: '12px',
                    border: '1px solid #e5e7eb',
                    borderRadius: '4px',
                    padding: '8px'
                  }}>
                    {analysisResult.crossingEdges.map((crossing: any, index: number) => (
                      <div key={index} style={{ 
                        color: '#4b5563', 
                        marginBottom: '6px',
                        padding: '4px',
                        backgroundColor: index % 2 === 0 ? '#f9fafb' : 'transparent',
                        borderRadius: '2px'
                      }}>
                        <span style={{ fontWeight: '500' }}>
                          {crossing.sourceNodeId} → {crossing.targetNodeId}
                        </span>
                        <span style={{
                          marginLeft: '8px',
                          padding: '2px 6px',
                          borderRadius: '4px',
                          fontSize: '11px',
                          backgroundColor: crossing.severity === 'high' ? '#fef2f2' :
                                         crossing.severity === 'medium' ? '#fffbeb' : '#f0fdf4',
                          color: crossing.severity === 'high' ? '#dc2626' :
                                crossing.severity === 'medium' ? '#d97706' : '#16a34a'
                        }}>
                          {crossing.severity}
                        </span>
                        <span style={{ marginLeft: '4px', color: '#6b7280', fontSize: '11px' }}>
                          (穿越{crossing.crossingNodes.length}个节点)
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
            
            <div style={{ marginTop: '20px', display: 'flex', justifyContent: 'flex-end' }}>
              <button 
                onClick={() => setShowAnalysisModal(false)}
                style={{
                  padding: '8px 16px',
                  backgroundColor: '#2563eb',
                  color: 'white',
                  border: 'none',
                  borderRadius: '6px',
                  cursor: 'pointer',
                  fontSize: '14px',
                  fontWeight: '500'
                }}
                onMouseOver={(e) => (e.target as HTMLElement).style.backgroundColor = '#1d4ed8'}
                onMouseOut={(e) => (e.target as HTMLElement).style.backgroundColor = '#2563eb'}
              >
                关闭
              </button>
            </div>
          </div>
        </div>
      )}
      
      {!state.dagData && !state.isLoading && !state.error && (
        <div className="empty-state">
          <div className="empty-state-container">
            {/* Header Section */}
            <div className="empty-state-header">
              <div className="empty-state-icon">
                <div className="icon-background">
                  <svg width="48" height="48" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                    <polyline points="3.27,6.96 12,12.01 20.73,6.96" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                    <line x1="12" y1="22.08" x2="12" y2="12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                  </svg>
                </div>
              </div>
              <h1 className="empty-state-title">DAG Visualizer</h1>
              <p className="empty-state-subtitle">专业的工作流可视化工具</p>
            </div>

            {/* Main Content */}
            <div className="empty-state-content">
              <div className="content-section">
                <h3 className="section-title">快速开始</h3>
                <p className="section-description">
                  在左侧粘贴JSON工作流数据，或使用工具栏加载文件来开始可视化
                </p>
              </div>

              <div className="features-grid">
                <div className="feature-card">
                  <div className="feature-icon">🔗</div>
                  <h4>智能布局</h4>
                  <p>自动分层布局和依赖关系可视化</p>
                </div>
                <div className="feature-card">
                  <div className="feature-icon">🎨</div>
                  <h4>自定义节点</h4>
                  <p>右键创建节点，支持自定义颜色和类型</p>
                </div>
                <div className="feature-card">
                  <div className="feature-icon">⚡</div>
                  <h4>多格式支持</h4>
                  <p>支持多种节点类型和数据格式</p>
                </div>
              </div>
            </div>

            {/* Footer Section - 紧凑布局 */}
            <div className="empty-state-footer">
              <div className="footer-compact">
                <div className="footer-divider">•</div>
                <div className="tech-compact">
                  <span>⚛️ React</span>
                  <span>🔷 TypeScript</span>
                  <span>🌊 ReactFlow</span>
                </div>
                <div className="footer-divider">•</div>
              </div>
            </div>
          </div>
        </div>
      )}
      
      {state.error && (
        <div className="error-state">
          <div className="error-state-icon">⚠️</div>
          <h3 className="error-state-title">数据验证错误</h3>
          <p className="error-state-message">{state.error}</p>
          <div className="error-suggestions">
            <div className="suggestion-item">
              <span className="suggestion-icon">💡</span>
              <span>检查JSON格式是否正确</span>
            </div>
            <div className="suggestion-item">
              <span className="suggestion-icon">🔍</span>
              <span>确保每个节点包含 taskId、taskType、dependencies</span>
            </div>
            <div className="suggestion-item">
              <span className="suggestion-icon">🎯</span>
              <span>修复左侧JSON输入中的字段缺失问题</span>
            </div>
            <div className="suggestion-item">
              <span className="suggestion-icon">📋</span>
              <span>可以使用工具栏的"加载示例数据"进行测试</span>
            </div>
          </div>
          <button 
            className="error-retry-btn"
            onClick={() => {
              dispatch({ type: 'SET_ERROR', payload: null });
              dispatch({ type: 'SET_JSON_TEXT', payload: '' });
            }}
          >
            清空重新开始
          </button>
        </div>
      )}
    </div>
  );
};

export default DAGVisualizer; 