import path from 'node:path';
export const evidencePath=(name:string)=>path.join(process.env.FILEX_E2E_OUTPUT_DIR || '.cache/evidence',name);
