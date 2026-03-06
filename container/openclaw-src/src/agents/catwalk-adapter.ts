import { execSync } from "child_process";
import { readFileSync } from "fs";
import { join } from "path";

export interface CatwalkProvider {
  id: string;
  name: string;
}

export interface CatwalkModel {
  id: string;
  name: string;
}

export interface CatwalkProviderMeta {
  base_url: string;
  protocol: string;
}

const CATWALK_BIN = process.env.CATWALK_PATH ?? "/usr/local/bin/catwalk";

export class CatwalkAdapter {
  private available: boolean | null = null;

  isAvailable(): boolean {
    if (this.available !== null) {
      return this.available;
    }
    try {
      execSync(`${CATWALK_BIN} --version`, { stdio: "ignore" });
      this.available = true;
    } catch {
      this.available = false;
    }
    return this.available;
  }

  listProviders(): string[] {
    if (!this.isAvailable()) {
      return this.getFallbackProviders();
    }
    try {
      const output = execSync(`${CATWALK_BIN} providers --json`, {
        encoding: "utf-8",
        timeout: 10000,
      });
      return JSON.parse(output) as string[];
    } catch (error) {
      console.warn("Failed to fetch providers from catwalk:", String(error));
      return this.getFallbackProviders();
    }
  }

  listModels(provider: string): string[] {
    if (!this.isAvailable()) {
      return this.getFallbackModels(provider);
    }
    try {
      const output = execSync(`${CATWALK_BIN} models ${provider} --json`, {
        encoding: "utf-8",
        timeout: 15000,
      });
      return JSON.parse(output) as string[];
    } catch (error) {
      console.warn(`Failed to fetch models for ${provider} from catwalk:`, String(error));
      return this.getFallbackModels(provider);
    }
  }

  getProviderMeta(provider: string): CatwalkProviderMeta | null {
    if (!this.isAvailable()) {
      return this.getFallbackMeta(provider);
    }
    try {
      const output = execSync(`${CATWALK_BIN} provider ${provider} --json`, {
        encoding: "utf-8",
        timeout: 10000,
      });
      return JSON.parse(output) as CatwalkProviderMeta;
    } catch (error) {
      console.warn(`Failed to fetch metadata for ${provider} from catwalk:`, String(error));
      return this.getFallbackMeta(provider);
    }
  }

  private getFallbackProviders(): string[] {
    return [
      "openai",
      "anthropic",
      "google",
      "xai",
      "deepseek",
      "openrouter",
      "ollama",
      "groq",
      "mistral",
      "cohere",
      "azure",
      "anthropic",
      "bedrock",
      "vertex",
      "sagemaker",
    ];
  }

  private getFallbackModels(provider: string): string[] {
    const fallbackModels: Record<string, string[]> = {
      openai: ["gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-4", "gpt-3.5-turbo"],
      anthropic: ["claude-sonnet-4-20250514", "claude-sonnet-4", "claude-3-5-sonnet-20241022", "claude-3-opus-20240229", "claude-3-haiku-20240307"],
      google: ["gemini-2.0-flash", "gemini-1.5-pro", "gemini-1.5-flash"],
      xai: ["grok-2", "grok-2-vision", "grok-beta"],
      deepseek: ["deepseek-chat", "deepseek-coder"],
      openrouter: ["anthropic/claude-3.5-sonnet", "openai/gpt-4o", "google/gemini-pro-1.5"],
      ollama: ["llama3.1", "mistral", "codellama", "phi3", "mixtral"],
      groq: ["llama-3.1-70b-versatile", "mixtral-8x7b-32768", "gemma2-9b-it"],
      mistral: ["mistral-large-latest", "mistral-small-latest", "mistral-nemo"],
      cohere: ["command-r-plus", "command-r", "command"],
      azure: ["gpt-4", "gpt-4-turbo", "gpt-35-turbo"],
      bedrock: ["anthropic.claude-3-sonnet", "anthropic.claude-3-haiku", "amazon.titan-text-express"],
      vertex: ["gemini-1.5-pro", "gemini-1.5-flash"],
      sagemaker: ["jumpstart-dft-meta-textgeneration-llama-3-1-8b"],
    };
    return fallbackModels[provider] ?? [];
  }

  private getFallbackMeta(provider: string): CatwalkProviderMeta | null {
    const fallbackMeta: Record<string, CatwalkProviderMeta> = {
      openai: { base_url: "https://api.openai.com/v1", protocol: "openai-compatible" },
      anthropic: { base_url: "https://api.anthropic.com", protocol: "anthropic" },
      google: { base_url: "https://generativelanguage.googleapis.com/v1beta", protocol: "openai-compatible" },
      xai: { base_url: "https://api.x.ai/v1", protocol: "openai-compatible" },
      deepseek: { base_url: "https://api.deepseek.com/v1", protocol: "openai-compatible" },
      openrouter: { base_url: "https://openrouter.ai/api/v1", protocol: "openai-compatible" },
      ollama: { base_url: "http://localhost:11434/v1", protocol: "openai-compatible" },
      groq: { base_url: "https://api.groq.com/openai/v1", protocol: "openai-compatible" },
      mistral: { base_url: "https://api.mistral.ai/v1", protocol: "openai-compatible" },
      cohere: { base_url: "https://api.cohere.ai/v1", protocol: "openai-compatible" },
      azure: { base_url: "${AZURE_OPENAI_ENDPOINT}", protocol: "openai-compatible" },
      bedrock: { base_url: "https://bedrock-runtime.${AWS_REGION}.amazonaws.com", protocol: "bedrock" },
      vertex: { base_url: "https://${AWS_REGION}-aiplatform.googleapis.com", protocol: "vertex" },
      sagemaker: { base_url: "https://runtime.sagemaker.${AWS_REGION}.amazonaws.com", protocol: "sagemaker" },
    };
    return fallbackMeta[provider] ?? null;
  }
}

export const catwalk = new CatwalkAdapter();
