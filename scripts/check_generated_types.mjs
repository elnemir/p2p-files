import fs from 'node:fs/promises';
import openapiTS, {astToString} from 'openapi-typescript';
import {format} from 'prettier';

const output = 'apps/web/lib/api-schema.ts';
const content = await format(astToString(await openapiTS(new URL('../contracts/openapi.json', import.meta.url))), {parser:'typescript'});
if (process.argv.includes('--write')) await fs.writeFile(output, content);
else if (await fs.readFile(output, 'utf8') !== content) throw new Error('Generated API types differ; run npm run generate:types');
