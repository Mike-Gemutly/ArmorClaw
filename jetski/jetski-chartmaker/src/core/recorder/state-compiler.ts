import { URL } from 'url';

export interface RecordedAction {
  action_type: string;
  selector?: {
    primary_css: string;
    secondary_xpath?: string;
    fallback_js?: string;
  };
  value?: string;
  url?: string;
  frame_routing?: {
    selector?: string;
    name?: string;
    origin?: string;
  };
  timestamp: number;
}

export interface NavChart {
  version: number;
  target_domain: string;
  metadata: {
    generated_by: string;
    timestamp: string;
    session_id?: string;
    total_actions?: number;
  };
  action_map: Record<string, CompiledAction>;
}

export interface CompiledAction {
  action_type: 'click' | 'input' | 'navigate' | 'wait' | 'assert';
  selector?: {
    primary_css: string;
    secondary_xpath?: string;
    fallback_js?: string;
  };
  value?: string;
  url?: string;
  frame_routing?: {
    selector?: string;
    name?: string;
    origin?: string;
  };
  post_action_wait?: {
    type: 'waitForVisible' | 'waitForHidden' | 'waitForTimeout' | 'waitForSelector';
    selector?: {
      primary_css: string;
    };
    timeout?: number;
  };
  assertion?: {
    type: 'equals' | 'includes' | 'matches' | 'exists';
    expected: string | number | boolean;
  };
}

export interface CompileOptions {
  sessionId?: string;
  addPostActionWaits?: boolean;
  waitTimeout?: number;
}

export class StateCompiler {
  private defaultWaitTimeout = 5000;

  compileEvents(events: RecordedAction[], options: CompileOptions = {}): NavChart {
    if (!events || events.length === 0) {
      throw new Error('No events to compile');
    }

    const targetDomain = this.extractTargetDomain(events[0]);
    const actionMap = this.compileActionMap(events, options);

    return {
      version: 1,
      target_domain: targetDomain,
      metadata: {
        generated_by: '@armorclaw/jetski-chartmaker',
        timestamp: new Date().toISOString(),
        ...(options.sessionId && { session_id: options.sessionId }),
        total_actions: events.length
      },
      action_map: actionMap
    };
  }

  private extractTargetDomain(event: RecordedAction): string {
    if (event.url) {
      try {
        const url = new URL(event.url);
        return `${url.protocol}//${url.host}`;
      } catch {
        return 'https://unknown';
      }
    }
    return 'https://unknown';
  }

  private compileActionMap(events: RecordedAction[], options: CompileOptions): Record<string, CompiledAction> {
    const actionMap: Record<string, CompiledAction> = {};

    events.forEach((event, index) => {
      const actionId = `action_${index + 1}`;
      actionMap[actionId] = this.compileSingleAction(event, options);
    });

    return actionMap;
  }

  private compileSingleAction(event: RecordedAction, options: CompileOptions): CompiledAction {
    const compiledAction: CompiledAction = {
      action_type: event.action_type as CompiledAction['action_type'],
    };

    if (event.selector) {
      compiledAction.selector = {
        primary_css: event.selector.primary_css,
        ...(event.selector.secondary_xpath && { secondary_xpath: event.selector.secondary_xpath }),
        ...(event.selector.fallback_js && { fallback_js: event.selector.fallback_js })
      };
    }

    if (event.value !== undefined) {
      compiledAction.value = event.value;
    }

    if (event.url && event.action_type === 'navigate') {
      compiledAction.url = event.url;
    }

    if (event.frame_routing) {
      compiledAction.frame_routing = {
        ...(event.frame_routing.selector && { selector: event.frame_routing.selector }),
        ...(event.frame_routing.name && { name: event.frame_routing.name }),
        ...(event.frame_routing.origin && { origin: event.frame_routing.origin })
      };
    }

    if (options.addPostActionWaits !== false) {
      compiledAction.post_action_wait = this.generatePostActionWait(event, options);
    }

    return compiledAction;
  }

  private generatePostActionWait(event: RecordedAction, options: CompileOptions): CompiledAction['post_action_wait'] {
    const waitTimeout = options.waitTimeout || this.defaultWaitTimeout;

    if (event.action_type === 'click') {
      return {
        type: 'waitForVisible',
        ...(event.selector && { selector: { primary_css: event.selector.primary_css } }),
        timeout: waitTimeout
      };
    }

    if (event.action_type === 'input') {
      return {
        type: 'waitForTimeout',
        timeout: 500
      };
    }

    if (event.action_type === 'navigate') {
      return {
        type: 'waitForSelector',
        selector: { primary_css: 'body' },
        timeout: waitTimeout
      };
    }

    return undefined;
  }

  validateFrameRouting(event: RecordedAction): boolean {
    if (!event.frame_routing) {
      return false;
    }

    const hasSelector = !!event.frame_routing.selector;
    const hasName = !!event.frame_routing.name;
    const hasOrigin = !!event.frame_routing.origin;

    return hasSelector || hasName || hasOrigin;
  }

  static async saveToFile(navChart: NavChart, outputPath: string): Promise<void> {
    const fs = await import('fs');
    const path = await import('path');
    const { validateNavChart } = await import('../validator/schema');
    const { logger } = await import('../../cli/utils/logger');

    if (!validateNavChart(navChart)) {
      const errors = validateNavChart.errors || [];
      const errorMsg = `Invalid Nav-Chart: ${errors.map(e => e.message).join(', ')}`;

      logger.error('🧭 Jetski Schema Validation Failed:', errorMsg);

      throw new Error(errorMsg);
    }

    const tempPath = `${outputPath}.tmp`;
    const outputDir = path.dirname(outputPath);

    fs.mkdirSync(outputDir, { recursive: true });

    fs.writeFileSync(tempPath, JSON.stringify(navChart, null, 2));

    fs.renameSync(tempPath, outputPath);
  }
}
