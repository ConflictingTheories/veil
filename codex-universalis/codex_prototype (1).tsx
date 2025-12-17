import React, { useState, useEffect, useCallback, useRef } from 'react';
import { GitBranch, GitCommit, GitMerge, Plus, Trash2, Link, Box, Save, Download, RotateCcw, Maximize2, Minimize2, Edit3, X } from 'lucide-react';

const CODEX = () => {
  // Core state
  const [currentBranch, setCurrentBranch] = useState('main');
  const [branches, setBranches] = useState(['main', 'modern-interpretation', 'eastern-fusion']);
  const [branchData, setBranchData] = useState({});
  
  // UI state
  const [view, setView] = useState('graph');
  const [selectedNode, setSelectedNode] = useState(null);
  const [selectedEdge, setSelectedEdge] = useState(null);
  const [contextMenu, setContextMenu] = useState(null);
  const [linkMode, setLinkMode] = useState(false);
  const [linkStart, setLinkStart] = useState(null);
  const [draggedNode, setDraggedNode] = useState(null);
  const [zoomedContainer, setZoomedContainer] = useState(null);
  const [compareCommit, setCompareCommit] = useState(null);

  const canvasRef = useRef(null);

  // Generate IDs
  const uuid = () => {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
      const r = Math.random() * 16 | 0;
      return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16);
    });
  };

  const makeURI = (id) => `codex://stoic-ethics.codex/concept/${id}`;

  // Initialize
  useEffect(() => {
    const id1 = uuid();
    const id2 = uuid();
    const id3 = uuid();
    
    const initialNodes = [
      { 
        nodeId: id1,
        uri: makeURI(id1),
        label: 'Virtue', 
        type: 'concept', 
        x: 300, 
        y: 150, 
        belief: 'The only true good', 
        confidence: 0.95, 
        justification: 'Central tenet of Stoic ethics',
        isContainer: false,
        children: []
      },
      { 
        nodeId: id2,
        uri: makeURI(id2),
        label: 'External Goods', 
        type: 'concept', 
        x: 500, 
        y: 150, 
        belief: 'Indifferent to happiness', 
        confidence: 0.9, 
        justification: 'Wealth, health, reputation are preferred indifferents',
        isContainer: false,
        children: []
      },
      { 
        nodeId: id3,
        uri: makeURI(id3),
        label: 'Emotional Theory', 
        type: 'container', 
        x: 400, 
        y: 300, 
        belief: 'Emotions result from judgments', 
        confidence: 0.85, 
        justification: 'Passions arise from incorrect judgments',
        isContainer: true,
        children: []
      }
    ];

    const e1 = uuid();
    const initialEdges = [
      { id: e1, from: id1, to: id3, relation: 'governs', strength: 0.9 }
    ];

    const initialCommit = {
      id: uuid(),
      hash: 'a3f82b1', 
      author: 'Epictetus', 
      timestamp: Date.now() - 86400000,
      message: 'Initial Stoic framework',
      snapshot: { nodes: JSON.parse(JSON.stringify(initialNodes)), edges: JSON.parse(JSON.stringify(initialEdges)) }
    };

    setBranchData({
      'main': { nodes: initialNodes, edges: initialEdges, commits: [initialCommit] },
      'modern-interpretation': { nodes: JSON.parse(JSON.stringify(initialNodes)), edges: JSON.parse(JSON.stringify(initialEdges)), commits: [JSON.parse(JSON.stringify(initialCommit))] },
      'eastern-fusion': { nodes: JSON.parse(JSON.stringify(initialNodes)), edges: JSON.parse(JSON.stringify(initialEdges)), commits: [JSON.parse(JSON.stringify(initialCommit))] }
    });
  }, []);

  // Get current data
  const getCurrentData = useCallback(() => {
    return branchData[currentBranch] || { nodes: [], edges: [], commits: [] };
  }, [branchData, currentBranch]);

  const nodes = getCurrentData().nodes;
  const edges = getCurrentData().edges;
  const commits = getCurrentData().commits;

  // Update current branch
  const updateBranch = useCallback((updates) => {
    setBranchData(prev => ({
      ...prev,
      [currentBranch]: { ...prev[currentBranch], ...updates }
    }));
  }, [currentBranch]);

  // Node operations
  const addNode = useCallback((isContainer = false, parentId = null) => {
    const id = uuid();
    const newNode = {
      nodeId: id,
      uri: makeURI(id),
      label: isContainer ? 'New Container' : 'New Concept',
      type: isContainer ? 'container' : 'concept',
      x: 300 + Math.random() * 200,
      y: 200 + Math.random() * 200,
      belief: 'New belief',
      confidence: 0.7,
      justification: 'Add justification',
      isContainer,
      children: []
    };

    if (parentId) {
      const parent = nodes.find(n => n.nodeId === parentId);
      if (parent && parent.isContainer) {
        const updatedNodes = nodes.map(n => 
          n.nodeId === parentId ? { ...n, children: [...n.children, id] } : n
        );
        updateBranch({ nodes: [...updatedNodes, { ...newNode, parentId }] });
      }
    } else {
      updateBranch({ nodes: [...nodes, newNode] });
    }
  }, [nodes, updateBranch]);

  const deleteNode = useCallback((nodeId) => {
    const updatedNodes = nodes.filter(n => n.nodeId !== nodeId && n.parentId !== nodeId);
    const updatedEdges = edges.filter(e => e.from !== nodeId && e.to !== nodeId);
    
    // Remove from parent's children
    const finalNodes = updatedNodes.map(n => ({
      ...n,
      children: n.children.filter(cid => cid !== nodeId)
    }));
    
    updateBranch({ nodes: finalNodes, edges: updatedEdges });
    
    if (selectedNode?.nodeId === nodeId) setSelectedNode(null);
    setContextMenu(null);
  }, [nodes, edges, selectedNode, updateBranch]);

  const updateNode = useCallback((nodeId, updates) => {
    const updatedNodes = nodes.map(n => n.nodeId === nodeId ? { ...n, ...updates } : n);
    updateBranch({ nodes: updatedNodes });
    if (selectedNode?.nodeId === nodeId) {
      setSelectedNode({ ...selectedNode, ...updates });
    }
  }, [nodes, selectedNode, updateBranch]);

  // Edge operations
  const addEdge = useCallback((fromId, toId) => {
    if (!fromId || !toId || fromId === toId) return;
    
    const id = uuid();
    const newEdge = { id, from: fromId, to: toId, relation: 'relates-to', strength: 0.7 };
    updateBranch({ edges: [...edges, newEdge] });
  }, [edges, updateBranch]);

  const deleteEdge = useCallback((edgeId) => {
    updateBranch({ edges: edges.filter(e => e.id !== edgeId) });
    if (selectedEdge?.id === edgeId) setSelectedEdge(null);
    setContextMenu(null);
  }, [edges, selectedEdge, updateBranch]);

  const updateEdge = useCallback((edgeId, updates) => {
    const updatedEdges = edges.map(e => e.id === edgeId ? { ...e, ...updates } : e);
    updateBranch({ edges: updatedEdges });
    if (selectedEdge?.id === edgeId) {
      setSelectedEdge({ ...selectedEdge, ...updates });
    }
  }, [edges, selectedEdge, updateBranch]);

  // Version control
  const commitChanges = useCallback(() => {
    const message = prompt('Commit message:');
    if (!message) return;

    const newCommit = {
      id: uuid(),
      hash: Math.random().toString(36).substr(2, 7),
      author: 'You',
      timestamp: Date.now(),
      message,
      snapshot: { 
        nodes: JSON.parse(JSON.stringify(nodes)), 
        edges: JSON.parse(JSON.stringify(edges)) 
      }
    };

    updateBranch({ commits: [...commits, newCommit] });
    alert(`✓ Committed: ${message}`);
  }, [nodes, edges, commits, updateBranch]);

  const checkoutCommit = useCallback((commit) => {
    if (!confirm(`Restore to commit ${commit.hash}?`)) return;
    
    updateBranch({
      nodes: JSON.parse(JSON.stringify(commit.snapshot.nodes)),
      edges: JSON.parse(JSON.stringify(commit.snapshot.edges))
    });
    
    setSelectedNode(null);
    setSelectedEdge(null);
  }, [updateBranch]);

  const createBranch = useCallback(() => {
    const name = prompt('New branch name:');
    if (!name || branches.includes(name)) return;

    setBranches([...branches, name]);
    setBranchData(prev => ({
      ...prev,
      [name]: JSON.parse(JSON.stringify(prev[currentBranch]))
    }));
  }, [branches, branchData, currentBranch]);

  // Export
  const exportData = useCallback(() => {
    const data = {
      format: 'CODEX-JSON-LD',
      version: '0.3.0',
      exported: new Date().toISOString(),
      repository: 'StoicEthics.codex',
      branches: branchData,
      currentBranch
    };
    
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `codex-export-${Date.now()}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, [branchData, currentBranch]);

  // Visible nodes based on zoom
  const getVisibleNodes = useCallback(() => {
    if (zoomedContainer) {
      const container = nodes.find(n => n.nodeId === zoomedContainer);
      if (container) {
        const children = nodes.filter(n => container.children.includes(n.nodeId));
        return [container, ...children];
      }
    }
    return nodes.filter(n => !n.parentId);
  }, [nodes, zoomedContainer]);

  // Context menu
  const showContextMenu = useCallback((e, item, type) => {
    e.preventDefault();
    e.stopPropagation();
    setContextMenu({ x: e.clientX, y: e.clientY, item, type });
  }, []);

  const hideContextMenu = useCallback(() => {
    setContextMenu(null);
  }, []);

  // Mouse handlers
  const handleCanvasClick = useCallback((e) => {
    if (e.target === canvasRef.current) {
      setSelectedNode(null);
      setSelectedEdge(null);
      hideContextMenu();
    }
  }, [hideContextMenu]);

  const handleNodeMouseDown = useCallback((node, e) => {
    e.stopPropagation();
    if (linkMode) return;
    setDraggedNode({ node, offsetX: e.clientX - node.x, offsetY: e.clientY - node.y });
  }, [linkMode]);

  const handleMouseMove = useCallback((e) => {
    if (!draggedNode || !canvasRef.current) return;
    
    const rect = canvasRef.current.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    
    updateNode(draggedNode.node.nodeId, { x, y });
  }, [draggedNode, updateNode]);

  const handleMouseUp = useCallback(() => {
    setDraggedNode(null);
  }, []);

  const handleNodeClick = useCallback((node, e) => {
    e.stopPropagation();
    
    if (linkMode) {
      if (!linkStart) {
        setLinkStart(node);
      } else {
        addEdge(linkStart.nodeId, node.nodeId);
        setLinkStart(null);
        setLinkMode(false);
      }
    } else {
      setSelectedNode(node);
      setSelectedEdge(null);
    }
  }, [linkMode, linkStart, addEdge]);

  useEffect(() => {
    document.addEventListener('mouseup', handleMouseUp);
    return () => document.removeEventListener('mouseup', handleMouseUp);
  }, [handleMouseUp]);

  // Render graph
  const renderGraph = () => {
    const visibleNodes = getVisibleNodes();
    
    return (
      <div 
        ref={canvasRef}
        className="relative w-full h-full bg-gray-900 rounded-lg overflow-hidden"
        onMouseMove={handleMouseMove}
        onClick={handleCanvasClick}
        onContextMenu={(e) => e.preventDefault()}
      >
        {/* Instructions */}
        {linkMode && (
          <div className="absolute top-4 right-4 bg-green-600 px-4 py-2 rounded-lg text-white text-sm font-semibold z-20 flex items-center gap-2">
            {linkStart ? `Click target for "${linkStart.label}"` : 'Click source node'}
            <button onClick={() => { setLinkMode(false); setLinkStart(null); }}>
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {zoomedContainer && (
          <div className="absolute top-4 left-4 bg-purple-600 px-4 py-2 rounded-lg text-white text-sm font-semibold z-20 flex items-center gap-2">
            Inside: {nodes.find(n => n.nodeId === zoomedContainer)?.label}
            <button onClick={() => setZoomedContainer(null)}>
              <Minimize2 className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* SVG Layer for edges */}
        <svg className="absolute inset-0 w-full h-full pointer-events-none" style={{ zIndex: 1 }}>
          <defs>
            <marker id="arrowhead" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto">
              <polygon points="0 0, 10 3, 0 6" fill="#6b7280" />
            </marker>
          </defs>
          {edges.map(edge => {
            const from = nodes.find(n => n.nodeId === edge.from);
            const to = nodes.find(n => n.nodeId === edge.to);
            if (!from || !to) return null;
            if (!visibleNodes.find(n => n.nodeId === from.nodeId) || !visibleNodes.find(n => n.nodeId === to.nodeId)) return null;

            const selected = selectedEdge?.id === edge.id;

            return (
              <g key={edge.id}>
                <line
                  x1={from.x} y1={from.y}
                  x2={to.x} y2={to.y}
                  stroke={selected ? "#fbbf24" : "#6b7280"}
                  strokeWidth={selected ? "3" : "2"}
                  markerEnd="url(#arrowhead)"
                  opacity={edge.strength}
                />
                <circle
                  cx={(from.x + to.x) / 2}
                  cy={(from.y + to.y) / 2}
                  r="12"
                  fill="#1f2937"
                  className="cursor-pointer pointer-events-auto"
                  onClick={(e) => {
                    e.stopPropagation();
                    setSelectedEdge(edge);
                    setSelectedNode(null);
                  }}
                  onContextMenu={(e) => showContextMenu(e, edge, 'edge')}
                />
                <text
                  x={(from.x + to.x) / 2}
                  y={(from.y + to.y) / 2 - 15}
                  fill="#d1d5db"
                  fontSize="11"
                  textAnchor="middle"
                  className="pointer-events-none select-none"
                >
                  {edge.relation}
                </text>
              </g>
            );
          })}
        </svg>

        {/* Nodes */}
        {visibleNodes.map(node => {
          const isSelected = selectedNode?.nodeId === node.nodeId;
          const isLinkSource = linkStart?.nodeId === node.nodeId;
          
          return (
            <div
              key={node.nodeId}
              className={`absolute cursor-move select-none transition-shadow ${
                node.isContainer 
                  ? 'bg-gradient-to-br from-purple-600 to-purple-800' 
                  : 'bg-gradient-to-br from-blue-600 to-blue-800'
              } rounded-lg p-3 shadow-lg border-2 ${
                isSelected ? 'border-yellow-400 shadow-yellow-400/50' : 
                isLinkSource ? 'border-green-400 shadow-green-400/50' :
                'border-blue-500'
              }`}
              style={{ 
                left: node.x - 70, 
                top: node.y - 35,
                width: '140px',
                zIndex: isSelected ? 10 : 2
              }}
              onMouseDown={(e) => handleNodeMouseDown(node, e)}
              onClick={(e) => handleNodeClick(node, e)}
              onDoubleClick={(e) => {
                e.stopPropagation();
                if (node.isContainer) setZoomedContainer(node.nodeId);
              }}
              onContextMenu={(e) => showContextMenu(e, node, 'node')}
            >
              <div className="flex items-start justify-between mb-1">
                <div className="text-white text-sm font-semibold flex-1 truncate">{node.label}</div>
                {node.isContainer && (
                  <Maximize2 className="w-3 h-3 text-purple-200 flex-shrink-0" />
                )}
              </div>
              <div className="text-blue-200 text-xs">⚡ {(node.confidence * 100).toFixed(0)}%</div>
              {node.isContainer && node.children.length > 0 && (
                <div className="text-purple-200 text-xs mt-1">📦 {node.children.length}</div>
              )}
            </div>
          );
        })}

        {/* Context Menu */}
        {contextMenu && (
          <div
            className="absolute bg-gray-800 border border-gray-700 rounded-lg shadow-xl py-2 z-50"
            style={{ left: contextMenu.x, top: contextMenu.y }}
            onClick={hideContextMenu}
          >
            {contextMenu.type === 'node' && (
              <>
                <button
                  onClick={() => {
                    setSelectedNode(contextMenu.item);
                    hideContextMenu();
                  }}
                  className="w-full px-4 py-2 text-left text-white hover:bg-gray-700 flex items-center gap-2"
                >
                  <Edit3 className="w-4 h-4" />
                  Edit
                </button>
                {contextMenu.item.isContainer && (
                  <button
                    onClick={() => {
                      addNode(false, contextMenu.item.nodeId);
                      hideContextMenu();
                    }}
                    className="w-full px-4 py-2 text-left text-white hover:bg-gray-700 flex items-center gap-2"
                  >
                    <Plus className="w-4 h-4" />
                    Add Child
                  </button>
                )}
                <button
                  onClick={() => deleteNode(contextMenu.item.nodeId)}
                  className="w-full px-4 py-2 text-left text-red-400 hover:bg-gray-700 flex items-center gap-2"
                >
                  <Trash2 className="w-4 h-4" />
                  Delete
                </button>
              </>
            )}
            {contextMenu.type === 'edge' && (
              <>
                <button
                  onClick={() => {
                    setSelectedEdge(contextMenu.item);
                    hideContextMenu();
                  }}
                  className="w-full px-4 py-2 text-left text-white hover:bg-gray-700 flex items-center gap-2"
                >
                  <Edit3 className="w-4 h-4" />
                  Edit Relation
                </button>
                <button
                  onClick={() => deleteEdge(contextMenu.item.id)}
                  className="w-full px-4 py-2 text-left text-red-400 hover:bg-gray-700 flex items-center gap-2"
                >
                  <Trash2 className="w-4 h-4" />
                  Delete
                </button>
              </>
            )}
          </div>
        )}
      </div>
    );
  };

  // Render history
  const renderHistory = () => (
    <div className="bg-gray-900 rounded-lg p-6 overflow-auto h-full">
      <h3 className="text-xl font-bold text-white mb-4">Commit History - {currentBranch}</h3>
      <div className="space-y-3">
        {[...commits].reverse().map(commit => (
          <div key={commit.id} className="bg-gray-800 rounded-lg p-4 border border-gray-700">
            <div className="flex items-start gap-3">
              <GitCommit className="w-5 h-5 text-blue-400 flex-shrink-0 mt-1" />
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-2">
                  <code className="text-yellow-400 text-sm">{commit.hash}</code>
                  <span className="text-gray-500 text-xs">by {commit.author}</span>
                </div>
                <p className="text-white font-medium mb-2">{commit.message}</p>
                <div className="text-gray-400 text-xs mb-3">
                  {new Date(commit.timestamp).toLocaleString()}
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => checkoutCommit(commit)}
                    className="px-3 py-1 bg-blue-600 hover:bg-blue-700 text-white text-sm rounded flex items-center gap-1"
                  >
                    <RotateCcw className="w-3 h-3" />
                    Checkout
                  </button>
                  <button
                    onClick={() => {
                      setCompareCommit(commit);
                      setView('diff');
                    }}
                    className="px-3 py-1 bg-gray-700 hover:bg-gray-600 text-white text-sm rounded"
                  >
                    Diff
                  </button>
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );

  // Render diff
  const renderDiff = () => {
    if (!compareCommit) {
      return (
        <div className="bg-gray-900 rounded-lg p-6 h-full flex items-center justify-center">
          <p className="text-gray-400">Select a commit to compare</p>
        </div>
      );
    }

    const oldNodes = compareCommit.snapshot.nodes;
    const added = nodes.filter(n => !oldNodes.find(on => on.nodeId === n.nodeId));
    const removed = oldNodes.filter(n => !nodes.find(nn => nn.nodeId === n.nodeId));
    const modified = nodes.filter(n => {
      const old = oldNodes.find(on => on.nodeId === n.nodeId);
      return old && JSON.stringify(old) !== JSON.stringify(n);
    });

    return (
      <div className="bg-gray-900 rounded-lg p-6 overflow-auto h-full">
        <h3 className="text-xl font-bold text-white mb-4">Semantic Diff</h3>
        <div className="bg-gray-800 p-3 rounded mb-4 border border-blue-500">
          <code className="text-yellow-400 text-sm">{compareCommit.hash}</code>
          <span className="text-gray-400 text-sm ml-2">→ current</span>
        </div>

        {added.length > 0 && (
          <div className="mb-4 bg-green-900/20 border border-green-500 rounded-lg p-4">
            <div className="text-green-400 font-semibold mb-2">+ Added ({added.length})</div>
            {added.map(n => (
              <div key={n.nodeId} className="text-green-300 text-sm pl-4">• {n.label}</div>
            ))}
          </div>
        )}

        {modified.length > 0 && (
          <div className="mb-4 bg-yellow-900/20 border border-yellow-500 rounded-lg p-4">
            <div className="text-yellow-400 font-semibold mb-2">~ Modified ({modified.length})</div>
            {modified.map(n => (
              <div key={n.nodeId} className="text-yellow-300 text-sm pl-4">• {n.label}</div>
            ))}
          </div>
        )}

        {removed.length > 0 && (
          <div className="bg-red-900/20 border border-red-500 rounded-lg p-4">
            <div className="text-red-400 font-semibold mb-2">- Removed ({removed.length})</div>
            {removed.map(n => (
              <div key={n.nodeId} className="text-red-300 text-sm pl-4">• {n.label}</div>
            ))}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-950 via-blue-950 to-gray-950 p-6">
      {/* Header */}
      <div className="max-w-7xl mx-auto mb-6">
        <div className="bg-gray-900/50 backdrop-blur-sm rounded-xl p-6 border border-blue-500/30">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-4xl font-bold text-transparent bg-clip-text bg-gradient-to-r from-blue-400 to-purple-400 mb-2">
                CODEX v0.3
              </h1>
              <p className="text-gray-400 text-sm">Epistemic Version Control</p>
            </div>
            <div className="flex gap-2">
              <button onClick={exportData} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg flex items-center gap-2">
                <Download className="w-4 h-4" />
                Export
              </button>
              <button onClick={commitChanges} className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg flex items-center gap-2">
                <Save className="w-4 h-4" />
                Commit
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Main */}
      <div className="max-w-7xl mx-auto grid grid-cols-4 gap-6">
        {/* Sidebar */}
        <div className="col-span-1 space-y-4">
          <div className="bg-gray-900/50 rounded-lg p-4 border border-gray-700">
            <h3 className="text-white font-semibold mb-3">Branch</h3>
            <select 
              value={currentBranch}
              onChange={(e) => setCurrentBranch(e.target.value)}
              className="w-full bg-gray-800 text-white px-2 py-2 rounded border border-gray-700 mb-2"
            >
              {branches.map(b => <option key={b} value={b}>{b}</option>)}
            </select>
            <button onClick={createBranch} className="w-full px-3 py-1.5 bg-purple-600 hover:bg-purple-700 text-white rounded text-sm">
              New Branch
            </button>
          </div>

          <div className="bg-gray-900/50 rounded-lg p-4 border border-gray-700">
            <h3 className="text-white font-semibold mb-3">View</h3>
            <div className="space-y-2">
              {['graph', 'history', 'diff'].map(v => (
                <button
                  key={v}
                  onClick={() => setView(v)}
                  className={`w-full px-3 py-2 rounded text-sm ${
                    view === v ? 'bg-blue-600 text-white' : 'bg-gray-800 text-gray-300 hover:bg-gray-700'
                  }`}
                >
                  {v.charAt(0).toUpperCase() + v.slice(1)}
                </button>
              ))}
            </div>
          </div>

          <div className="bg-gray-900/50 rounded-lg p-4 border border-gray-700">
            <h3 className="text-white font-semibold mb-3">Actions</h3>
            <div className="space-y-2">
              <button onClick={() => addNode(false)} className="w-full px-3 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded text-sm flex items-center gap-2">
                <Plus className="w-4 h-4" />
                Concept
              </button>
              <button onClick={() => addNode(true)} className="w-full px-3 py-2 bg-purple-600 hover:bg-purple-700 text-white rounded text-sm flex items-center gap-2">
                <Box className="w-4 h-4" />
                Container
              </button>
              <button 
                onClick={() => {
                  setLinkMode(!linkMode);
                  setLinkStart(null);
                }}
                className={`w-full px-3 py-2 rounded text-sm flex items-center gap-2 ${
                  linkMode ? 'bg-green-600 text-white' : 'bg-gray-700 hover:bg-gray-600 text-gray-300'
                }`}
              >
                <Link className="w-4 h-4" />
                {linkMode ? 'Cancel Link' : 'Link Mode'}
              </button>
            </div>
          </div>

          <div className="bg-gray-900/50 rounded-lg p-4 border border-gray-700">
            <div className="text-xs text-gray-500">
              <div>{nodes.length} concepts</div>
              <div>{edges.length} relations</div>
              <div>{commits.length} commits</div>
            </div>
          </div>
        </div>

        {/* Main View */}
        <div className="col-span-3 h-[700px]">
          {view === 'graph' && renderGraph()}
          {view === 'history' && renderHistory()}
          {view === 'diff' && renderDiff()}
        </div>
      </div>

      {/* Node Inspector */}
      {selectedNode && view === 'graph' && (
        <div className="max-w-7xl mx-auto mt-6">
          <div className="bg-gray-900/50 rounded-lg p-6 border border-blue-500/50">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-xl font-bold text-white">Edit: {selectedNode.label}</h3>
              <button onClick={() => setSelectedNode(null)} className="text-gray-400 hover:text-white">
                <X className="w-5 h-5" />
              </button>
            </div>
            
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-gray-400 text-sm block mb-1">Label</label>
                <input
                  type="text"
                  value={selectedNode.label}
                  onChange={(e) => updateNode(selectedNode.nodeId, { label: e.target.value })}
                  className="w-full bg-gray-800 text-white px-3 py-2 rounded border border-gray-700 focus:border-blue-500 focus:outline-none"
                />
              </div>
              <div>
                <label className="text-gray-400 text-sm block mb-1">Type</label>
                <select
                  value={selectedNode.isContainer ? 'container' : 'concept'}
                  onChange={(e) => updateNode(selectedNode.nodeId, { isContainer: e.target.value === 'container' })}
                  className="w-full bg-gray-800 text-white px-3 py-2 rounded border border-gray-700 focus:border-blue-500 focus:outline-none"
                >
                  <option value="concept">Concept</option>
                  <option value="container">Container</option>
                </select>
              </div>
              <div>
                <label className="text-gray-400 text-sm block mb-1">Confidence: {(selectedNode.confidence * 100).toFixed(0)}%</label>
                <input
                  type="range"
                  min="0"
                  max="1"
                  step="0.05"
                  value={selectedNode.confidence}
                  onChange={(e) => updateNode(selectedNode.nodeId, { confidence: parseFloat(e.target.value) })}
                  className="w-full"
                />
              </div>
              <div>
                <label className="text-gray-400 text-sm block mb-1">Node ID</label>
                <input
                  type="text"
                  value={selectedNode.nodeId}
                  readOnly
                  className="w-full bg-gray-800 text-gray-500 px-3 py-2 rounded border border-gray-700 font-mono text-xs"
                />
              </div>
              <div className="col-span-2">
                <label className="text-gray-400 text-sm block mb-1">URI</label>
                <input
                  type="text"
                  value={selectedNode.uri}
                  readOnly
                  className="w-full bg-gray-800 text-gray-500 px-3 py-2 rounded border border-gray-700 font-mono text-xs"
                />
              </div>
              <div className="col-span-2">
                <label className="text-gray-400 text-sm block mb-1">Belief Statement</label>
                <input
                  type="text"
                  value={selectedNode.belief}
                  onChange={(e) => updateNode(selectedNode.nodeId, { belief: e.target.value })}
                  className="w-full bg-gray-800 text-white px-3 py-2 rounded border border-gray-700 focus:border-blue-500 focus:outline-none"
                />
              </div>
              <div className="col-span-2">
                <label className="text-gray-400 text-sm block mb-1">Justification</label>
                <textarea
                  value={selectedNode.justification}
                  onChange={(e) => updateNode(selectedNode.nodeId, { justification: e.target.value })}
                  className="w-full bg-gray-800 text-white px-3 py-2 rounded border border-gray-700 h-20 focus:border-blue-500 focus:outline-none resize-none"
                />
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Edge Inspector */}
      {selectedEdge && view === 'graph' && (
        <div className="max-w-7xl mx-auto mt-6">
          <div className="bg-gray-900/50 rounded-lg p-6 border border-yellow-500/50">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-xl font-bold text-white">Edit Relation</h3>
              <button onClick={() => setSelectedEdge(null)} className="text-gray-400 hover:text-white">
                <X className="w-5 h-5" />
              </button>
            </div>
            
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-gray-400 text-sm block mb-1">Relation Type</label>
                <select
                  value={selectedEdge.relation}
                  onChange={(e) => updateEdge(selectedEdge.id, { relation: e.target.value })}
                  className="w-full bg-gray-800 text-white px-3 py-2 rounded border border-gray-700 focus:border-yellow-500 focus:outline-none"
                >
                  <option value="relates-to">relates-to</option>
                  <option value="governs">governs</option>
                  <option value="influences">influences</option>
                  <option value="contrasts-with">contrasts-with</option>
                  <option value="depends-on">depends-on</option>
                  <option value="causes">causes</option>
                  <option value="enables">enables</option>
                  <option value="requires">requires</option>
                  <option value="opposes">opposes</option>
                  <option value="supports">supports</option>
                  <option value="implies">implies</option>
                  <option value="contradicts">contradicts</option>
                </select>
              </div>
              <div>
                <label className="text-gray-400 text-sm block mb-1">Strength: {(selectedEdge.strength * 100).toFixed(0)}%</label>
                <input
                  type="range"
                  min="0"
                  max="1"
                  step="0.05"
                  value={selectedEdge.strength}
                  onChange={(e) => updateEdge(selectedEdge.id, { strength: parseFloat(e.target.value) })}
                  className="w-full"
                />
              </div>
              <div className="col-span-2">
                <div className="text-sm text-gray-400">
                  From: <span className="text-white">{nodes.find(n => n.nodeId === selectedEdge.from)?.label}</span>
                  {' → '}
                  To: <span className="text-white">{nodes.find(n => n.nodeId === selectedEdge.to)?.label}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default CODEX;