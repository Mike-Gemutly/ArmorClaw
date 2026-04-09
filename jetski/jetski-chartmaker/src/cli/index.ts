#!/usr/bin/env node
import { Command } from 'commander';
import { initCommand } from './commands/init';
import { mapCommand } from './commands/map';
import { verifyCommand } from './commands/verify';
import { fetchCommand } from './commands/fetch';

const program = new Command();

program
  .name('jetski-chartmaker')
  .description('🗺️  Visual navigation compiler for Jetski browser automation')
  .version('1.0.0');

program
  .command('init')
  .description('Initialize Jetski Chartmaker configuration')
  .option('--dir <path>', 'project directory', '.')
  .action(initCommand);

program
  .command('map')
  .description('Start interactive mapping session (launch browser with The Helm)')
  .option('--url <url>', 'starting URL')
  .option('--output <file>', 'output Nav-Chart file', 'site.acsb.json')
  .option('--session <id>', 'session identifier')
  .action(mapCommand);

program
  .command('verify')
  .description('Validate Nav-Chart with dry-run execution (Local Sea Trial)')
  .argument('<file>', '.acsb.json Nav-Chart file to verify')
  .option('--verbose', 'detailed output')
  .option('--headless', 'run in headless mode')
  .action(verifyCommand);

program
  .command('fetch <domain>')
  .description('Fetch Nav-Chart from Lighthouse registry')
  .option('--blessed', 'Fetch only ArmorClaw-blessed charts')
  .option('--version <version>', 'Specific version to fetch')
  .action(fetchCommand);

program.parse();
