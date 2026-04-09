import fs from 'fs';
import path from 'path';
import { logger } from '../utils/logger';

export async function fetchCommand(domain: string, options: { blessed?: boolean; version?: string }) {
  logger.info(`🕯️ [LIGHTHOUSE] Searching for Nav-Charts: ${domain}...`);

  try {
    let url: URL;

    if (options.blessed) {
      url = new URL('http://localhost:8080/charts/blessed');
      url.searchParams.append('domain', domain);
    } else if (options.version) {
      url = new URL(`http://localhost:8080/charts/${domain}/${options.version}`);
    } else {
      url = new URL('http://localhost:8080/charts');
      url.searchParams.append('domain', domain);
    }

    const response = await fetch(url.toString());

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const chart = await response.json() as {
      version: string;
      author: string;
      blessed: boolean;
      chartData: string;
    };

    if (!chart) {
      logger.error(`❌ [LIGHTHOUSE]: No charts found for domain: ${domain}`);
      return;
    }

    const chartsDir = './charts';
    if (!fs.existsSync(chartsDir)) {
      fs.mkdirSync(chartsDir, { recursive: true });
    }

    const filePath = path.join(chartsDir, `${domain.replace(/\./g, '_')}.acsb.json`);

    if (typeof chart.chartData === 'string') {
      const parsedChartData = JSON.parse(chart.chartData);
      fs.writeFileSync(filePath, JSON.stringify(parsedChartData, null, 2));
    } else {
      fs.writeFileSync(filePath, JSON.stringify(chart.chartData, null, 2));
    }

    logger.success(`✅ [LIGHTHOUSE]: Chart secured at ${filePath}`);
    logger.info(`   Version: ${chart.version}`);
    logger.info(`   Author: ${chart.author}`);
    logger.info(`   Signature: ${chart.blessed ? 'Verified ✓' : 'Unsigned ⚠️'}`);

    if (!chart.blessed) {
      logger.warn(`⚠️  [LIGHTHOUSE]: Warning: This is a community chart. Use with caution.`);
    }

  } catch (error) {
    logger.error(`Failed to fetch chart: ${error instanceof Error ? error.message : String(error)}`);
    process.exit(1);
  }
}
