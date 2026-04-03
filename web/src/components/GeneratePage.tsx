import { useState } from 'react';
import { Play, Download, Folder, FileCode, Check, AlertCircle, RefreshCw, ChevronDown, ChevronRight } from 'lucide-react';
import { generateConfig, fetchOutput, downloadOutput, fetchOutputFile } from '../api/config';
import type { GenerateResult, OutputResult } from '../api/config';

export function GeneratePage() {
  const [generating, setGenerating] = useState(false);
  const [result, setResult] = useState<GenerateResult | null>(null);
  const [output, setOutput] = useState<OutputResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
  const [selectedFile, setSelectedFile] = useState<{ path: string; content: string } | null>(null);
  
  const handleGenerate = async () => {
    setGenerating(true);
    setError(null);
    setResult(null);
    setOutput(null);
    
    try {
      const res = await generateConfig();
      setResult(res);
      
      if (res.success) {
        // 获取文件列表
        const outputRes = await fetchOutput();
        setOutput(outputRes);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '生成配置失败');
    } finally {
      setGenerating(false);
    }
  };
  
  const handleDownload = () => {
    downloadOutput();
  };
  
  const toggleNode = (ip: string) => {
    const newExpanded = new Set(expandedNodes);
    if (newExpanded.has(ip)) {
      newExpanded.delete(ip);
    } else {
      newExpanded.add(ip);
    }
    setExpandedNodes(newExpanded);
  };
  
  const handleViewFile = async (ip: string, file: string) => {
    const path = `${ip}/${file}`;
    try {
      const result = await fetchOutputFile(path);
      setSelectedFile(result);
    } catch (err) {
      console.error('获取文件内容失败:', err);
    }
  };
  
  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-gray-800">生成配置</h2>
          <p className="text-gray-500 mt-1">根据 config.yaml 配置生成部署脚本</p>
        </div>
        <div className="flex gap-2">
          <button 
            className="btn btn-primary"
            onClick={handleGenerate}
            disabled={generating}
          >
            {generating ? (
              <>
                <RefreshCw className="w-4 h-4 animate-spin" />
                生成中...
              </>
            ) : (
              <>
                <Play className="w-4 h-4" />
                生成配置
              </>
            )}
          </button>
          
          {output && output.total > 0 && (
            <button className="btn btn-success" onClick={handleDownload}>
              <Download className="w-4 h-4" />
              下载配置包
            </button>
          )}
        </div>
      </div>
      
      {/* 错误提示 */}
      {error && (
        <div className="card mb-6" style={{ borderColor: 'var(--danger)' }}>
          <div className="card-body">
            <div className="flex items-center gap-2 text-danger">
              <AlertCircle className="w-5 h-5" />
              <span>{error}</span>
            </div>
          </div>
        </div>
      )}
      
      {/* 生成结果 */}
      {result && (
        <div className="card mb-6">
          <div className="card-header">
            <div className="flex items-center gap-2">
              {result.success ? (
                <Check className="w-5 h-5 text-success" />
              ) : (
                <AlertCircle className="w-5 h-5 text-danger" />
              )}
              <span className="font-semibold text-gray-800">
                {result.success ? '生成成功' : '生成完成（有错误）'}
              </span>
            </div>
            <span className="badge badge-gray">
              共 {result.stats.generated} 个文件
            </span>
          </div>
          <div className="card-body">
            <div className="flex gap-4 mb-4">
              <div className="flex-1 text-center p-3 bg-gray-50 rounded-lg">
                <p className="text-2xl font-bold text-gray-800">{result.stats.nodes}</p>
                <p className="text-sm text-gray-500">节点数</p>
              </div>
              <div className="flex-1 text-center p-3 bg-gray-50 rounded-lg">
                <p className="text-2xl font-bold text-gray-800">{result.stats.services}</p>
                <p className="text-sm text-gray-500">服务数</p>
              </div>
              <div className="flex-1 text-center p-3 bg-gray-50 rounded-lg">
                <p className="text-2xl font-bold text-gray-800">{result.stats.generated}</p>
                <p className="text-sm text-gray-500">生成文件</p>
              </div>
              <div className="flex-1 text-center p-3 bg-gray-50 rounded-lg">
                <p className="text-2xl font-bold text-warning">{result.stats.skipped}</p>
                <p className="text-sm text-gray-500">跳过服务</p>
              </div>
            </div>
            
            {result.errors.length > 0 && (
              <div className="mt-4">
                <h4 className="font-medium text-danger mb-2">错误信息：</h4>
                <ul className="text-sm text-danger space-y-1">
                  {result.errors.map((err, idx) => (
                    <li key={idx} className="flex items-start gap-2">
                      <span>•</span>
                      <span>{err}</span>
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {result.warnings.length > 0 && (
              <div className="mt-4">
                <h4 className="font-medium text-warning mb-2">警告信息：</h4>
                <ul className="text-sm text-warning space-y-1">
                  {result.warnings.map((warn, idx) => (
                    <li key={idx} className="flex items-start gap-2">
                      <span>•</span>
                      <span>{warn}</span>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        </div>
      )}
      
      {/* 生成的文件列表 */}
      {output && output.total > 0 && (
        <div className="card">
          <div className="card-header">
            <div className="flex items-center gap-2">
              <Folder className="w-5 h-5 text-primary" />
              <span className="font-semibold text-gray-800">生成的文件</span>
            </div>
            <span className="text-sm text-gray-500">output/</span>
          </div>
          <div className="card-body p-0">
            <div className="max-h-96 overflow-y-auto">
              {Object.entries(output.files).map(([ip, files]) => (
                <div key={ip} className="border-b last:border-b-0">
                  <button
                    className="w-full px-4 py-3 flex items-center gap-2 hover:bg-gray-50 text-left"
                    onClick={() => toggleNode(ip)}
                  >
                    {expandedNodes.has(ip) ? (
                      <ChevronDown className="w-4 h-4 text-gray-400" />
                    ) : (
                      <ChevronRight className="w-4 h-4 text-gray-400" />
                    )}
                    <Folder className="w-4 h-4 text-primary" />
                    <span className="font-medium">{ip}</span>
                    <span className="text-sm text-gray-500">({files.length} 个文件)</span>
                  </button>
                  
                  {expandedNodes.has(ip) && (
                    <div className="pl-10 pb-2">
                      {files.map((file) => (
                        <button
                          key={file}
                          className="w-full px-4 py-2 flex items-center gap-2 hover:bg-gray-100 text-left text-sm"
                          onClick={() => handleViewFile(ip, file)}
                        >
                          <FileCode className="w-4 h-4 text-gray-400" />
                          <span>{file}</span>
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
      
      {/* 文件内容预览 */}
      {selectedFile && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setSelectedFile(null)}>
          <div className="bg-white rounded-lg max-w-4xl w-full mx-4 max-h-[80vh] flex flex-col" onClick={e => e.stopPropagation()}>
            <div className="px-4 py-3 border-b flex items-center justify-between">
              <div className="flex items-center gap-2">
                <FileCode className="w-5 h-5 text-primary" />
                <span className="font-medium">{selectedFile.path}</span>
              </div>
              <button className="text-gray-400 hover:text-gray-600" onClick={() => setSelectedFile(null)}>×</button>
            </div>
            <div className="flex-1 overflow-auto p-4">
              <pre className="text-sm font-mono bg-gray-50 p-4 rounded-lg whitespace-pre-wrap">{selectedFile.content}</pre>
            </div>
          </div>
        </div>
      )}
      
      {/* 使用说明 */}
      {!result && (
        <div className="card">
          <div className="card-header">
            <span className="font-semibold text-gray-800">使用说明</span>
          </div>
          <div className="card-body">
            <div className="space-y-3 text-sm text-gray-600">
              <p>1. 在「节点管理」页面添加集群节点（IP、主机名）</p>
              <p>2. 在「服务配置」页面配置服务拓扑和参数</p>
              <p>3. 点击「生成配置」按钮，根据模板生成部署脚本</p>
              <p>4. 下载生成的配置包，分发到各节点执行部署</p>
              
              <div className="mt-4 p-3 bg-blue-50 rounded-lg text-primary">
                <p className="font-medium">提示</p>
                <p className="mt-1">也可以直接修改 config.yaml 文件，然后点击「重新加载」按钮从文件读取配置。</p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
