import { useEffect, useState } from 'react';
import Editor from '@monaco-editor/react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import {
  Play,
  Download,
  Folder,
  FileCode,
  Check,
  AlertCircle,
  RefreshCw,
  ChevronDown,
  ChevronRight,
  X,
} from 'lucide-react';
import { downloadOutput, fetchOutput, fetchOutputFile, generateConfig } from '../api/config';
import type { GenerateResult, OutputResult } from '../api/config';

const GENERATE_CACHE_KEY = 'config-generator:generate-page-cache';

interface PersistedGenerateState {
  result: GenerateResult | null;
  output: OutputResult | null;
  expandedNodes: string[];
  generatedAt: string | null;
}

interface FilePresentation {
  label: string;
  badgeClassName: string;
  previewClassName: string;
  previewMode: 'code' | 'markdown';
  language: string;
}

function readPersistedGenerateState(): PersistedGenerateState {
  try {
    const raw = window.localStorage.getItem(GENERATE_CACHE_KEY);
    if (!raw) {
      return {
        result: null,
        output: null,
        expandedNodes: [],
        generatedAt: null,
      };
    }

    const parsed = JSON.parse(raw) as Partial<PersistedGenerateState>;
    return {
      result: parsed.result ?? null,
      output: parsed.output ?? null,
      expandedNodes: Array.isArray(parsed.expandedNodes) ? parsed.expandedNodes : [],
      generatedAt: typeof parsed.generatedAt === 'string' ? parsed.generatedAt : null,
    };
  } catch (error) {
    console.warn('[GeneratePage] 恢复生成结果缓存失败:', error);
    return {
      result: null,
      output: null,
      expandedNodes: [],
      generatedAt: null,
    };
  }
}

function getFilePresentation(path: string): FilePresentation {
  const lowerPath = path.toLowerCase();

  if (lowerPath.endsWith('.sh')) {
    return {
      label: 'Shell',
      badgeClassName: 'badge-file-shell',
      previewClassName: 'preview-code-shell',
      previewMode: 'code',
      language: 'shell',
    };
  }

  if (lowerPath.endsWith('.conf')) {
    return {
      label: 'CONF',
      badgeClassName: 'badge-file-conf',
      previewClassName: 'preview-code-conf',
      previewMode: 'code',
      language: 'ini',
    };
  }

  if (lowerPath.endsWith('.json')) {
    return {
      label: 'JSON',
      badgeClassName: 'badge-file-json',
      previewClassName: 'preview-code-json',
      previewMode: 'code',
      language: 'json',
    };
  }

  if (lowerPath.endsWith('.properties')) {
    return {
      label: 'Props',
      badgeClassName: 'badge-file-properties',
      previewClassName: 'preview-code-properties',
      previewMode: 'code',
      language: 'ini',
    };
  }

  if (lowerPath.endsWith('.yaml') || lowerPath.endsWith('.yml')) {
    return {
      label: 'YAML',
      badgeClassName: 'badge-file-yaml',
      previewClassName: 'preview-code-yaml',
      previewMode: 'code',
      language: 'yaml',
    };
  }

  if (lowerPath.endsWith('.md')) {
    return {
      label: 'Markdown',
      badgeClassName: 'badge-file-md',
      previewClassName: 'preview-code-md',
      previewMode: 'markdown',
      language: 'markdown',
    };
  }

  return {
    label: 'File',
    badgeClassName: 'badge-file-default',
    previewClassName: 'preview-code-default',
    previewMode: 'code',
    language: 'plaintext',
  };
}

function formatGeneratedAt(value: string | null): string | null {
  if (!value) {
    return null;
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return null;
  }

  return new Intl.DateTimeFormat('zh-CN', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date);
}

interface GeneratePageProps {
  hidePageTitle?: boolean;
}

export function GeneratePage({ hidePageTitle = false }: GeneratePageProps) {
  const [generating, setGenerating] = useState(false);
  const [cacheState, setCacheState] = useState<PersistedGenerateState>(() => readPersistedGenerateState());
  const [error, setError] = useState<string | null>(null);
  const [selectedFile, setSelectedFile] = useState<{ path: string; content: string } | null>(null);

  useEffect(() => {
    window.localStorage.setItem(GENERATE_CACHE_KEY, JSON.stringify(cacheState));
  }, [cacheState]);

  const expandedNodes = new Set(cacheState.expandedNodes);
  const formattedGeneratedAt = formatGeneratedAt(cacheState.generatedAt);

  const handleGenerate = async () => {
    setGenerating(true);
    setError(null);

    try {
      const result = await generateConfig();
      const output = await fetchOutput();

      setCacheState((previous) => {
        const persistedExpandedNodes = previous.expandedNodes.filter((node) => output.files[node]);
        const nextExpandedNodes =
          persistedExpandedNodes.length > 0
            ? persistedExpandedNodes
            : Object.keys(output.files).slice(0, 1);

        return {
          result,
          output,
          expandedNodes: nextExpandedNodes,
          generatedAt: new Date().toISOString(),
        };
      });
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
    setCacheState((previous) => {
      const nextExpandedNodes = new Set(previous.expandedNodes);
      if (nextExpandedNodes.has(ip)) {
        nextExpandedNodes.delete(ip);
      } else {
        nextExpandedNodes.add(ip);
      }

      return {
        ...previous,
        expandedNodes: Array.from(nextExpandedNodes),
      };
    });
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

  const result = cacheState.result;
  const output = cacheState.output;
  const selectedFilePresentation = selectedFile ? getFilePresentation(selectedFile.path) : null;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        {!hidePageTitle && <div>
          <h2 className="text-2xl font-bold text-gray-800">生成配置</h2>
          <p className="text-gray-500 mt-1">根据 config.yaml 配置生成部署脚本</p>
          {formattedGeneratedAt && (
            <p className="text-xs text-gray-500 mt-2">
              已保留最近一次生成结果 · {formattedGeneratedAt}
            </p>
          )}
        </div>}
        <div className="flex gap-2">
          <button className="btn btn-primary" onClick={handleGenerate} disabled={generating}>
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
            <div className="flex items-center gap-2">
              {formattedGeneratedAt && <span className="badge badge-gray">{formattedGeneratedAt}</span>}
              <span className="badge badge-gray">共 {result.stats.generated} 个文件</span>
            </div>
          </div>
          <div className="card-body">
            <p className="text-sm text-gray-600 mb-4">{result.message}</p>

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
                  {result.errors.map((item, index) => (
                    <li key={index} className="flex items-start gap-2">
                      <span>•</span>
                      <span>{item}</span>
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {result.warnings.length > 0 && (
              <div className="mt-4">
                <h4 className="font-medium text-warning mb-2">警告信息：</h4>
                <ul className="text-sm text-warning space-y-1">
                  {result.warnings.map((item, index) => (
                    <li key={index} className="flex items-start gap-2">
                      <span>•</span>
                      <span>{item}</span>
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {result.skippedServices.length > 0 && (
              <div className="mt-4">
                <h4 className="font-medium text-warning mb-2">跳过的服务：</h4>
                <ul className="text-sm text-gray-700 space-y-1">
                  {result.skippedServices.map((item, index) => (
                    <li key={`${item.nodeIp}-${item.service}-${index}`} className="flex items-start gap-2">
                      <span>•</span>
                      <span>
                        {item.nodeIp} / {item.service} / {item.reason}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        </div>
      )}

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
            <div>
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
                    <div className="pl-10 pr-4 pb-3 space-y-1.5">
                      {files.map((file) => {
                        const filePresentation = getFilePresentation(file);
                        return (
                          <button
                            key={file}
                            className="generate-file-item"
                            onClick={() => handleViewFile(ip, file)}
                          >
                            <div className="generate-file-item__meta">
                              <FileCode className="w-4 h-4 text-gray-400" />
                              <span className="generate-file-item__name">{file}</span>
                            </div>
                            <span className={`badge ${filePresentation.badgeClassName}`}>{filePresentation.label}</span>
                          </button>
                        );
                      })}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {result && output && output.total === 0 && (
        <div className="card">
          <div className="card-body">
            <div className="flex items-center gap-2 text-gray-600">
              <AlertCircle className="w-5 h-5 text-warning" />
              <span>本次生成没有产生可展示的输出文件。</span>
            </div>
          </div>
        </div>
      )}

      {selectedFile && selectedFilePresentation && (
        <div className="modal-overlay modal-preview-overlay" onClick={() => setSelectedFile(null)}>
          <div className="modal modal-preview" onClick={(event) => event.stopPropagation()}>
            <div className="modal-header">
              <div className="flex items-center gap-3 min-w-0">
                <FileCode className="w-5 h-5 text-primary shrink-0" />
                <div className="min-w-0">
                  <div className="font-medium truncate">{selectedFile.path}</div>
                </div>
                <span className={`badge ${selectedFilePresentation.badgeClassName}`}>
                  {selectedFilePresentation.label}
                </span>
              </div>
              <button className="modal-close" onClick={() => setSelectedFile(null)} aria-label="关闭">
                <X className="w-4 h-4" />
              </button>
            </div>
            <div className="modal-body modal-preview-body">
              {selectedFilePresentation.previewMode === 'markdown' ? (
                <div className={`generate-markdown-preview ${selectedFilePresentation.previewClassName}`}>
                  <ReactMarkdown remarkPlugins={[remarkGfm]}>{selectedFile.content}</ReactMarkdown>
                </div>
              ) : (
                <div className={`generate-preview-editor ${selectedFilePresentation.previewClassName}`}>
                  <Editor
                    height="100%"
                    defaultLanguage={selectedFilePresentation.language}
                    value={selectedFile.content}
                    options={{
                      readOnly: true,
                      minimap: { enabled: false },
                      fontSize: 13,
                      lineNumbers: 'on',
                      scrollBeyondLastLine: false,
                      wordWrap: 'on',
                      renderLineHighlight: 'all',
                      theme: 'vs',
                      padding: { top: 18, bottom: 18 },
                    }}
                  />
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {!result && !output && (
        <div className="card">
          <div className="card-header">
            <span className="font-semibold text-gray-800">使用说明</span>
          </div>
          <div className="card-body">
            <div className="space-y-3 text-sm text-gray-600">
              <p>1. 在“节点管理”页面添加集群节点（IP、主机名）</p>
              <p>2. 在“服务配置”页面配置服务拓扑和参数</p>
              <p>3. 点击“生成配置”按钮，根据模板生成部署脚本</p>
              <p>4. 下载生成的配置包，分发到各节点执行部署</p>

              <div className="mt-4 p-3 bg-blue-50 rounded-lg text-primary">
                <p className="font-medium">提示</p>
                <p className="mt-1">切换到其他页面后，最近一次生成结果会继续保留；只有再次点击“生成配置”才会更新。</p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
