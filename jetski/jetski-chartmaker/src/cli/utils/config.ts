import fs from 'fs';

export interface JetskiConfig {
  version: number;
  defaultTimeout: number;
  screenshotOnFailure: boolean;
  sessionDir?: string;
  chartsDir?: string;
}

const DEFAULT_CONFIG: JetskiConfig = {
  version: 1,
  defaultTimeout: 5000,
  screenshotOnFailure: true,
  sessionDir: 'sessions',
  chartsDir: 'charts',
};

export class ConfigManager {
  private configPath: string;
  private config: JetskiConfig;

  constructor(configPath: string = 'jetski.config.json') {
    this.configPath = configPath;
    this.config = this.loadConfig();
  }

  private loadConfig(): JetskiConfig {
    if (!fs.existsSync(this.configPath)) {
      return { ...DEFAULT_CONFIG };
    }

    try {
      const fileContent = fs.readFileSync(this.configPath, 'utf-8');
      const userConfig = JSON.parse(fileContent);
      return { ...DEFAULT_CONFIG, ...userConfig };
    } catch (error) {
      console.warn(`Failed to load config from ${this.configPath}, using defaults`);
      return { ...DEFAULT_CONFIG };
    }
  }

  getConfig(): JetskiConfig {
    return this.config;
  }

  saveConfig(config: Partial<JetskiConfig>): void {
    this.config = { ...this.config, ...config };
    fs.writeFileSync(
      this.configPath,
      JSON.stringify(this.config, null, 2),
      'utf-8'
    );
  }

  getSessionDir(): string {
    return this.config.sessionDir || DEFAULT_CONFIG.sessionDir || 'sessions';
  }

  getChartsDir(): string {
    return this.config.chartsDir || DEFAULT_CONFIG.chartsDir || 'charts';
  }
}
