import fs from 'fs';
import path from 'path';

export interface SessionState {
  sessionId: string;
  startTime: number;
  lastActionTime: number;
  actionCount: number;
  recordedActions: RecordedAction[];
  metadata: {
    url?: string;
    userAgent?: string;
    viewport?: { width: number; height: number };
  };
}

export interface RecordedAction {
  id: string;
  action_type: string;
  selector?: {
    primary_css: string;
    secondary_xpath?: string;
    fallback_js?: string;
  };
  value?: string;
  url?: string;
  timestamp: number;
  frameRouting?: {
    selector?: string;
    name?: string;
    origin?: string;
  };
}

export class SessionManager {
  private sessionId: string;
  private state: SessionState;
  private sessionDir: string;

  constructor(sessionId: string) {
    this.sessionId = sessionId;
    this.sessionDir = path.join('sessions', sessionId);
    this.state = this.loadState() || this.createInitialState();
  }

  private createInitialState(): SessionState {
    return {
      sessionId: this.sessionId,
      startTime: Date.now(),
      lastActionTime: Date.now(),
      actionCount: 0,
      recordedActions: [],
      metadata: {},
    };
  }

  private loadState(): SessionState | null {
    const stateFile = path.join(this.sessionDir, 'state.json');
    if (!fs.existsSync(stateFile)) {
      return null;
    }

    try {
      const content = fs.readFileSync(stateFile, 'utf-8');
      return JSON.parse(content);
    } catch (error) {
      return null;
    }
  }

  saveState(): void {
    fs.mkdirSync(this.sessionDir, { recursive: true });
    const stateFile = path.join(this.sessionDir, 'state.json');
    fs.writeFileSync(stateFile, JSON.stringify(this.state, null, 2));
  }

  recordAction(action: RecordedAction): void {
    this.state.recordedActions.push(action);
    this.state.actionCount++;
    this.state.lastActionTime = Date.now();
    this.saveState();
  }

  getActions(): RecordedAction[] {
    return this.state.recordedActions;
  }

  getSessionId(): string {
    return this.sessionId;
  }

  updateMetadata(metadata: Partial<SessionState['metadata']>): void {
    this.state.metadata = { ...this.state.metadata, ...metadata };
    this.saveState();
  }

  clearActions(): void {
    this.state.recordedActions = [];
    this.state.actionCount = 0;
    this.state.lastActionTime = Date.now();
    this.saveState();
  }

  exportNavChart(): any {
    return {
      version: 1,
      target_domain: this.state.metadata.url || '',
      metadata: {
        generated_by: '@armorclaw/jetski-chartmaker',
        timestamp: new Date().toISOString(),
        session_id: this.sessionId,
      },
      action_map: this.state.recordedActions.reduce((map, action, index) => {
        map[`action-${index + 1}`] = {
          action_type: action.action_type,
          selector: action.selector,
          value: action.value,
          url: action.url,
          frame_routing: action.frameRouting,
        };
        return map;
      }, {} as Record<string, any>),
    };
  }
}
