import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const version = {
  version: process.env.npm_package_version || '1.0.0',
  timestamp: Date.now()
};

const versionPath = path.join(__dirname, '../public/version.json');
fs.writeFileSync(versionPath, JSON.stringify(version, null, 2));

console.log('Version file generated:', version);
