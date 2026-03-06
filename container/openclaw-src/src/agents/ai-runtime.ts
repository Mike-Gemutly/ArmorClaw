import { catwalk, type CatwalkProviderMeta } from "./catwalk-adapter.js";

export interface AIRuntimeState {
  provider: string;
  model: string;
  baseUrl: string;
  apiKeyRef: string;
  protocol: string;
  lastUpdated: number;
}

export interface AIRuntimeConfig {
  stateFile: string;
}

const DEFAULT_STATE: AIRuntimeState = {
  provider: "openai",
  model: "gpt-4o-mini",
  baseUrl: "https://api.openai.com/v1",
  apiKeyRef: "openai-default",
  protocol: "openai-compatible",
  lastUpdated: Date.now(),
};

export class AIRuntimeManager {
  private state: AIRuntimeState;
  private config: AIRuntimeConfig;
  private listeners: Set<(state: AIRuntimeState) => void> = new Set();

  constructor(config: AIRuntimeConfig) {
    this.config = config;
    this.state = { ...DEFAULT_STATE };
  }

  getState(): AIRuntimeState {
    return { ...this.state };
  }

  setState(newState: Partial<AIRuntimeState>): void {
    this.state = { ...this.state, ...newState, lastUpdated: Date.now() };
    this.persistState();
    this.notifyListeners();
  }

  async switchProvider(provider: string, model?: string): Promise<{ success: boolean; error?: string }> {
    const providers = catwalk.listProviders();
    if (!providers.includes(provider)) {
      return { success: false, error: `Unknown provider: ${provider}. Available: ${providers.join(", ")}` };
    }

    const models = catwalk.listModels(provider);
    const targetModel = model ?? models[0];
    if (!models.includes(targetModel)) {
      return { success: false, error: `Unknown model: ${model} for provider ${provider}. Available: ${models.join(", ")}` };
    }

    const meta = catwalk.getProviderMeta(provider);
    if (!meta) {
      return { success: false, error: `Could not get metadata for provider: ${provider}` };
    }

    this.setState({
      provider,
      model: targetModel,
      baseUrl: meta.base_url,
      protocol: meta.protocol,
      apiKeyRef: `${provider}-default`,
    });

    return { success: true };
  }

  async switchModel(model: string): Promise<{ success: boolean; error?: string }> {
    const models = catwalk.listModels(this.state.provider);
    if (!models.includes(model)) {
      return {
        success: false,
        error: `Unknown model: ${model} for provider ${this.state.provider}. Available: ${models.join(", ")}`,
      };
    }

    this.setState({ model });
    return { success: true };
  }

  subscribe(listener: (state: AIRuntimeState) => void): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  private notifyListeners(): void {
    for (const listener of this.listeners) {
      try {
        listener(this.getState());
      } catch (error) {
        console.error("Error in AIRuntime listener:", error);
      }
    }
  }

  private persistState(): void {
    // This will be handled by the config system
    // For now, we just emit events
  }

  toConfigFormat(): string {
    return `[ai]
provider = "${this.state.provider}"
model = "${this.state.model}"

[ai.endpoint]
base_url = "${this.state.baseUrl}"

[ai.auth]
key_name = "${this.state.apiKeyRef}"
`;
  }
}
