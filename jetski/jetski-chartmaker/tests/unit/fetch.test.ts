import { describe, it, expect, jest, beforeEach, afterEach } from '@jest/globals';
import { fetchCommand } from '../../src/cli/commands/fetch';
import * as fs from 'fs';
import * as path from 'path';

jest.mock('../../src/cli/utils/logger', () => ({
  logger: {
    info: jest.fn(),
    warn: jest.fn(),
    error: jest.fn(),
    success: jest.fn(),
  },
}));

(global.fetch as any) = jest.fn();

jest.spyOn(process, 'exit').mockImplementation((code?: string | number | null | undefined) => {
  throw new Error(`process.exit(${code})`);
});

describe('fetchCommand', () => {
  const mockChartsDir = './charts';
  const testDomain = 'example.com';

  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterEach(() => {
    const testFile = path.join(mockChartsDir, `${testDomain.replace(/\./g, '_')}.acsb.json`);
    if (fs.existsSync(testFile)) {
      fs.unlinkSync(testFile);
    }
  });

  it('should fetch and save a chart successfully', async () => {
    const mockChartData = {
      version: '1.0.0',
      author: 'test@example.com',
      chart_data: { navigations: [] },
      signature: 'sha256=abc123',
      blessed: true,
    };

    const mockResponse = {
      ok: true,
      json: async () => ({ charts: [mockChartData] }),
    };

    (global.fetch as any).mockResolvedValueOnce(mockResponse);

    await fetchCommand(testDomain, {});

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/charts?domain=example.com')
    );

    const expectedFile = path.join(mockChartsDir, `${testDomain.replace(/\./g, '_')}.acsb.json`);
    expect(fs.existsSync(expectedFile)).toBe(true);

    const savedContent = JSON.parse(fs.readFileSync(expectedFile, 'utf-8'));
    expect(savedContent).toEqual(mockChartData.chart_data);
  });

  it('should fetch blessed chart when --blessed flag is used', async () => {
    const mockChartData = {
      version: '1.0.0',
      author: 'test@example.com',
      chart_data: { navigations: [] },
      signature: 'sha256=abc123',
      blessed: true,
    };

    const mockResponse = {
      ok: true,
      json: async () => ({ charts: [mockChartData] }),
    };

    (global.fetch as any).mockResolvedValueOnce(mockResponse);

    await fetchCommand(testDomain, { blessed: true });

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('blessed=true')
    );
  });

  it('should fetch specific version when --version flag is used', async () => {
    const mockChartData = {
      version: '1.2.0',
      author: 'test@example.com',
      chart_data: { navigations: [] },
      signature: 'sha256=abc123',
      blessed: false,
    };

    const mockResponse = {
      ok: true,
      json: async () => ({ charts: [mockChartData] }),
    };

    (global.fetch as any).mockResolvedValueOnce(mockResponse);

    await fetchCommand(testDomain, { version: '1.2.0' });

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('version=1.2.0')
    );
  });

  it('should handle empty response from API', async () => {
    const mockResponse = {
      ok: true,
      json: async () => ({ charts: [] }),
    };

    (global.fetch as any).mockResolvedValueOnce(mockResponse);

    await fetchCommand(testDomain, {});

    const expectedFile = path.join(mockChartsDir, `${testDomain.replace(/\./g, '_')}.acsb.json`);
    expect(fs.existsSync(expectedFile)).toBe(false);
  });

  it('should handle fetch errors', async () => {
    const mockError = new Error('Network error');
    (global.fetch as any).mockRejectedValueOnce(mockError);

    await expect(fetchCommand(testDomain, {})).rejects.toThrow('process.exit(1)');
  });
});
