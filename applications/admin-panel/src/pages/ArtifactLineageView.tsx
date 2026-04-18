import React from 'react';
import {
  Globe,
  FileText,
  Database,
  Mail,
  KeyRound,
  ChevronRight,
} from 'lucide-react';
import type { ArtifactNode } from './TeamTimelinePage';

const ARTIFACT_ICONS: Record<string, React.FC<{ className?: string }>> = {
  BrowserIntent:         Globe,
  BrowserResult:         Globe,
  DocumentRef:           FileText,
  ExtractedChunkSet:     Database,
  SecretRef:             KeyRound,
  EmailDraft:            Mail,
};

const ARTIFACT_COLORS: Record<string, string> = {
  BrowserIntent:     'border-blue-500 bg-blue-500/10',
  BrowserResult:     'border-blue-400 bg-blue-400/10',
  DocumentRef:       'border-green-500 bg-green-500/10',
  ExtractedChunkSet: 'border-purple-500 bg-purple-500/10',
  SecretRef:         'border-yellow-500 bg-yellow-500/10',
  EmailDraft:        'border-orange-500 bg-orange-500/10',
};

interface ArtifactLineageViewProps {
  artifacts: ArtifactNode[];
}

export function ArtifactLineageView({ artifacts }: ArtifactLineageViewProps) {
  if (artifacts.length === 0) {
    return (
      <div className="bg-gray-800/50 rounded-lg p-6 text-center text-gray-400">
        <FileText className="w-8 h-8 mx-auto mb-2 opacity-50" />
        <p className="text-sm">No artifact lineage data</p>
      </div>
    );
  }

  return (
    <div className="bg-gray-800/50 rounded-lg p-4">
      <div className="flex items-center gap-2 flex-wrap">
        {artifacts.map((artifact, index) => {
          const Icon = ARTIFACT_ICONS[artifact.type] ?? FileText;
          const colorClass = ARTIFACT_COLORS[artifact.type] ?? 'border-gray-500 bg-gray-500/10';

          return (
            <React.Fragment key={artifact.id}>
              <div
                className={`flex items-center gap-2 px-3 py-2 rounded-lg border ${colorClass}`}
                title={`${artifact.type} at ${artifact.timestamp}`}
              >
                <Icon className="w-4 h-4 text-gray-300" />
                <div>
                  <div className="text-xs font-medium text-gray-200">{artifact.type}</div>
                  <div className="text-[10px] text-gray-500 font-mono">{artifact.id}</div>
                </div>
              </div>
              {index < artifacts.length - 1 && (
                <ChevronRight className="w-4 h-4 text-gray-600 flex-shrink-0" />
              )}
            </React.Fragment>
          );
        })}
      </div>
    </div>
  );
}

export default ArtifactLineageView;
