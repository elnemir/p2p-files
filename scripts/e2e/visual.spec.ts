import {test,expect} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

test('keyboard navigation, error recovery, themes and 320px reflow',async({page})=>{
  await page.goto('/');
  await expect(page.getByLabel('Email',{exact:true})).toBeVisible();
  await page.keyboard.press('Tab');
  const skip=page.locator('.skip-link');
  await expect(skip).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.locator('main')).toBeFocused();
  await page.getByLabel('Email',{exact:true}).fill('missing@example.test');
  await page.keyboard.press('Tab');
  await expect(page.getByLabel('Пароль',{exact:true})).toBeFocused();
  await page.keyboard.type('Incorrect-fixture-password');
  await page.keyboard.press('Enter');
  await expect(page.locator('main').getByRole('alert')).toBeVisible();
  await page.locator('main').getByRole('alert').getByRole('button').click();
  await expect(page.locator('main').getByRole('alert')).toHaveCount(0);
  await page.setViewportSize({width:320,height:780});
  for(const locale of ['ru','en']) {
    await page.getByLabel(/Язык|Language/).selectOption(locale);
    for(const theme of ['light','dark','system']) {
      await page.getByLabel(/Тема|Theme/).selectOption(theme);
      expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
      expect((await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21aa','wcag22aa']).analyze()).violations).toEqual([]);
    }
  }
});

test('reviewed public page visual regression',async({page})=>{
  await page.goto('/help');
  await expect(page.getByLabel(/Язык|Language/)).toBeVisible();
  for(const [locale,theme] of [['ru','light'],['en','dark']]) {
    await page.getByLabel(/Язык|Language/).selectOption(locale);
    await page.getByLabel(/Тема|Theme/).selectOption(theme);
    await expect(page).toHaveScreenshot(`help-${locale}-${theme}.png`,{fullPage:true,animations:'disabled',maxDiffPixelRatio:0.001});
  }
});
