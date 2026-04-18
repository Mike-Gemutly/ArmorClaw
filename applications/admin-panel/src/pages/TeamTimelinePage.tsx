import { useState } from 'react';
import {
  Filter,
  Search,
  Download,
  Clock,
  Activity,
} from 'lucide-react';
import { TeamTimelineEvent } from './TeamTimelineEvent';
import { ArtifactLineageView } from './ArtifactLineageView';

export interface TeamEvent {
  id: string;
  event_type: string;
  team_id: string;
  agent_id: string;
  role_name: string;
  timestamp: string;
  details: Record<string, string>;
}

export interface ArtifactNode {
  type: string;
  id: string;
  timestamp: string;
}

const EVENT_TYPES = [
  'team_created',
  'team_dissolved',
  'member_added',
  'member_removed',
  'role_assigned',
  'delegation_sent',
  'handoff_complete',
] as const;

const MOCK_EVENTS: TeamEvent[] = [
  {
    id: 'te-001',
    event_type: 'team_created',
    team_id: 'team-a1b2c3',
    agent_id: 'agent-001',
    role_name: 'team_lead',
    timestamp: '2026-04-18T10:00:00Z',
    details: { template: 'default', creator: 'admin' },
  },
  {
    id: 'te-002',
    event_type: 'member_added',
    team_id: 'team-a1b2c3',
    agent_id: 'agent-002',
    role_name: 'worker',
    timestamp: '2026-04-18T10:05:00Z',
    details: { added_by: 'agent-001' },
  },
  {
    id: 'te-003',
    event_type: 'role_assigned',
    team_id: 'team-a1b2c3',
    agent_id: 'agent-002',
    role_name: 'specialist',
    timestamp: '2026-04-18T10:30:00Z',
    details: { previous_role: 'worker', assigned_by: 'agent-001' },
  },
  {
    id: 'te-004',
    event_type: 'delegation_sent',
    team_id: 'team-a1b2c3',
    agent_id: 'agent-001',
    role_name: 'team_lead',
    timestamp: '2026-04-18T11:00:00Z',
    details: { target_agent: 'agent-003', task: 'web_research' },
  },
  {
    id: 'te-005',
    event_type: 'handoff_complete',
    team_id: 'team-a1b2c3',
    agent_id: 'agent-003',
    role_name: 'worker',
    timestamp: '2026-04-18T11:45:00Z',
    details: { from_agent: 'agent-001', status: 'success' },
  },
  {
    id: 'te-006',
    event_type: 'member_removed',
    team_id: 'team-a1b2c3',
    agent_id: 'agent-002',
    role_name: 'specialist',
    timestamp: '2026-04-18T12:00:00Z',
    details: { reason: 'task_completed', removed_by: 'agent-001' },
  },
  {
    id: 'te-007',
    event_type: 'team_dissolved',
    team_id: 'team-a1b2c3',
    agent_id: 'agent-001',
    role_name: 'team_lead',
    timestamp: '2026-04-18T12:30:00Z',
    details: { reason: 'all_tasks_complete' },
  },
];

const MOCK_LINEAGE: ArtifactNode[] = [
  { type: 'BrowserIntent', id: 'bi-001', timestamp: '2026-04-18T10:10:00Z' },
  { type: 'BrowserResult', id: 'br-001', timestamp: '2026-04-18T10:12:00Z' },
  { type: 'ExtractedChunkSet', id: 'ecs-001', timestamp: '2026-04-18T10:15:00Z' },
  { type: 'DocumentRef', id: 'dr-001', timestamp: '2026-04-18T10:20:00Z' },
  { type: 'EmailDraft', id: 'ed-001', timestamp: '2026-04-18T10:25:00Z' },
];

export function TeamTimelinePage() {
  const [eventTypeFilter, setEventTypeFilter] = useState<string>('all');
  const [teamIdSearch, setTeamIdSearch] = useState('');
  const [expandedEvent, setExpandedEvent] = useState<string | null>(null);

  const filteredEvents = MOCK_EVENTS
    .filter(event => {
      if (eventTypeFilter !== 'all' && event.event_type !== eventTypeFilter) return false;
      if (teamIdSearch && !event.team_id.toLowerCase().includes(teamIdSearch.toLowerCase())) return false;
      return true;
    })
    .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

  const stats = {
    total: MOCK_EVENTS.length,
    teams: new Set(MOCK_EVENTS.map(e => e.team_id)).size,
    activeAgents: new Set(MOCK_EVENTS.filter(e =>
      e.event_type !== 'team_dissolved' && e.event_type !== 'member_removed'
    ).map(e => e.agent_id)).size,
    handoffs: MOCK_EVENTS.filter(e => e.event_type === 'handoff_complete').length,
  };

  const exportTimeline = () => {
    const data = JSON.stringify(filteredEvents, null, 2);
    const blob = new Blob([data], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `team-timeline-${new Date().toISOString().split('T')[0]}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Team Timeline</h1>
          <p className="text-gray-400">
            Team events and artifact lineage
          </p>
        </div>
        <button
          onClick={exportTimeline}
          className="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
        >
          <Download className="w-4 h-4" />
          Export
        </button>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="bg-gray-800/50 rounded-lg p-4">
          <div className="text-2xl font-bold">{stats.total}</div>
          <div className="text-sm text-gray-400">Total Events</div>
        </div>
        <div className="bg-gray-800/50 rounded-lg p-4">
          <div className="text-2xl font-bold text-blue-400">{stats.teams}</div>
          <div className="text-sm text-gray-400">Teams</div>
        </div>
        <div className="bg-gray-800/50 rounded-lg p-4">
          <div className="text-2xl font-bold text-green-400">{stats.activeAgents}</div>
          <div className="text-sm text-gray-400">Active Agents</div>
        </div>
        <div className="bg-gray-800/50 rounded-lg p-4">
          <div className="text-2xl font-bold text-purple-400">{stats.handoffs}</div>
          <div className="text-sm text-gray-400">Handoffs</div>
        </div>
      </div>

      <div className="flex flex-col md:flex-row gap-4">
        <div className="flex-1 relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search by team ID..."
            value={teamIdSearch}
            onChange={(e) => setTeamIdSearch(e.target.value)}
            className="w-full pl-10 pr-4 py-2 bg-gray-800/50 border border-gray-700 rounded-lg focus:outline-none focus:border-blue-500"
          />
        </div>
        <div className="flex items-center gap-2">
          <Filter className="w-4 h-4 text-gray-400" />
          <select
            value={eventTypeFilter}
            onChange={(e) => setEventTypeFilter(e.target.value)}
            className="px-3 py-2 bg-gray-800/50 border border-gray-700 rounded-lg focus:outline-none focus:border-blue-500"
          >
            <option value="all">All Events</option>
            {EVENT_TYPES.map(type => (
              <option key={type} value={type}>{type.replace(/_/g, ' ')}</option>
            ))}
          </select>
        </div>
      </div>

      <div className="bg-gray-800/50 rounded-lg overflow-hidden">
        {filteredEvents.length === 0 ? (
          <div className="p-8 text-center text-gray-400">
            <Activity className="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p>No team events found</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-700">
            {filteredEvents.map(event => (
              <TeamTimelineEvent
                key={event.id}
                event={event}
                expanded={expandedEvent === event.id}
                onToggle={() => setExpandedEvent(expandedEvent === event.id ? null : event.id)}
              />
            ))}
          </div>
        )}
      </div>

      <div>
        <h2 className="text-lg font-semibold mb-3">Artifact Lineage</h2>
        <ArtifactLineageView artifacts={MOCK_LINEAGE} />
      </div>

      <div className="bg-gray-800/50 rounded-lg p-4">
        <div className="flex items-center gap-2 text-sm text-gray-400">
          <Clock className="w-4 h-4" />
          <p>
            Team timeline shows all team lifecycle events. Use the Export function to archive
            event history for compliance review.
          </p>
        </div>
      </div>
    </div>
  );
}

export default TeamTimelinePage;
