import {defineConfig, devices} from '@playwright/test';
const output = process.env.FILEX_E2E_OUTPUT_DIR || '.cache/evidence/browser/default';
export default defineConfig({
  testDir:'scripts/e2e', timeout:90000, fullyParallel:false, workers:1, retries:0,
  reporter:[['list'],['json',{outputFile:`${output}/results.json`}],['html',{outputFolder:`${output}/report`,open:'never'}]],
  outputDir:`${output}/results`,
  snapshotPathTemplate:`{testDir}/visual-baselines/${process.env.FILEX_BROWSER_PLATFORM || process.platform}/{projectName}/{arg}{ext}`,
  use:{connectOptions:process.env.FILEX_BROWSER_WS?{wsEndpoint:process.env.FILEX_BROWSER_WS,exposeNetwork:'<loopback>'}:undefined,baseURL:process.env.FILEX_VISUAL_BASE_URL || 'http://localhost:8090',headless:true,acceptDownloads:true,trace:'retain-on-failure',screenshot:'only-on-failure',locale:'ru-RU',timezoneId:'UTC',colorScheme:'light',reducedMotion:'reduce'},
  projects:[
    {name:'chromium',use:{...devices['Desktop Chrome']}},
    {name:'firefox',use:{...devices['Desktop Firefox']}},
    {name:'webkit',use:{...devices['Desktop Safari']}},
    {name:'mobile-chromium',use:{...devices['Pixel 7']}},
    {name:'mobile-webkit',use:{...devices['iPhone 13']}},
  ],
});
