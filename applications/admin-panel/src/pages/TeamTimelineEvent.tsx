import React from 'react';
import {
  PlusCircle,
  XCircle,
  UserPlus,
  UserMinus,
  Crown,
  ArrowRightLeft,
  Repeat,
  Clock,
} from 'lucide-react';
import type { TeamEvent } from './TeamTimelinePage';

const EVENT_STYLES: Record<string, { bg: string; text: string; icon: React.FC<{ className?: string }> }> = {
  team_created:     { bg: 'bg-green-500/20',  text: 'text-green-400',  icon: PlusCircle },
  team_dissolved:   { bg: 'bg-red-500/20',    text: 'text-red-400',    icon: XCircle },
  member_added:     { bg: 'bg-blue-500/20',   text: 'text-blue-400',   icon: UserPlus },
  member_removed:   { bg: 'bg-blue-500/20',   text: 'text-blue-400',   icon: UserMinus },
  role_assigned:    { bg: 'bg-yellow-500/20', text: 'text-yellow-400', icon: Crown },
  delegation_sent:  { bg: 'bg-purple-500/20', text: 'text-purple-400', icon: ArrowRightLeft },
  handoff_complete: { bg: 'bg-orange-500/20', text: 'text-orange-400', icon: Repeat },
};

interface TeamTimelineEventProps {
  event: TeamEvent;
  expanded: boolean;
  onToggle: () => void;
}

export function TeamTimelineEvent({ event, expanded, onToggle }: TeamTimelineEventProps) {
  const style = EVENT_STYLES[event.event_type] ?? { bg: 'bg-gray-500/20', text: 'text-gray-400', icon: Clock };
  const Icon = style.icon;

  return (
    <div
      className="p-4 hover:bg-gray-700/30 transition-colors cursor-pointer"
      onClick={onToggle}
    >
      <div className="flex items-start gap-3">
        <div className={`p-2 rounded-lg ${style.bg}`}>
          <Icon className={`w-4 h-4 ${style.text}`} />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <code className="text-sm font-mono text-blue-300">
              {event.event_type.replace(/_/g, ' ')}
            </code>
            <span className="text-xs text-gray-500">team: {event.team_id}</span>
          </div>
          <div className="flex items-center gap-4 text-xs text-gray-400 mb-1">
            <span>Agent: {event.agent_id}</span>
            {event.role_name && <span>Role: {event.role_name}</span>}
            <span className="flex items-center gap-1">
              <Clock className="w-3 h-3" />
              {new Date(event.timestamp).toLocaleString()}
            </span>
          </div>
          {expanded && event.details && Object.keys(event.details).length > 0 && (
            <div className="mt-2 pl-3 border-l-2 border-gray-600 space-y-1">
              {Object.entries(event.details).map(([key, value]) => (
                <div key={key} className="text-xs">
                  <span className="text-gray-500">{key}:</span>{' '}
                  <span className="text-gray-300">{value}</span>
                </div>
              ))}
            </div>
          )}
        </div>
        <div className="text-right text-xs">
          <div className={`px-2 py-0.5 rounded ${style.bg} ${style.text}`}>
            {event.event_type.replace(/_/g, ' ')}
          </div>
        </div>
      </div>
    </div>
  );
}

export default TeamTimelineEvent;
