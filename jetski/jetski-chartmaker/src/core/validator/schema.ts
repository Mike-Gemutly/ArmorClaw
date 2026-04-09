import Ajv from 'ajv';
import addFormats from 'ajv-formats';
import { readFileSync } from 'fs';
import { logger } from '../../cli/utils/logger';
import type { NavChart } from '../recorder/state-compiler';

export type { NavChart };

const ajv = new Ajv({ allErrors: true, strict: false });
addFormats(ajv);

const navChartSchema = JSON.parse(
  readFileSync(__dirname + '/../../../schemas/nav-chart.json', 'utf-8')
);

export const validateNavChart = ajv.compile(navChartSchema);

export function isValidNavChart(data: unknown): boolean {
  const valid = validateNavChart(data);
  if (!valid) {
    logger.error('Nav-Chart validation errors:');
    if (validateNavChart.errors) {
      for (const error of validateNavChart.errors) {
        logger.error(`  - ${error.instancePath}: ${error.message}`);
      }
    }
  }
  return valid === true;
}
