import { expect, type Page } from '@playwright/test';

export async function createDraft(page: Page, title: string): Promise<string> {
  const created = page.waitForResponse(response =>
    new URL(response.url()).pathname === '/api/v1/transfers' &&
    response.request().method() === 'POST');
  await page.getByRole('button', { name: 'Новая передача', exact: true }).click();
  const creation = await created;
  expect(creation.status()).toBe(201);
  const { id } = await creation.json();
  expect(id).toMatch(/^[0-9a-f-]{36}$/);

  const input = page.getByLabel('Название', { exact: true });
  // Exercise keyboard input and verify the saved fixture before any backup or publish.
  await expect(input).toHaveValue('');
  await input.focus();
  await expect(input).toBeFocused();
  await input.pressSequentially(title);
  await expect(input).toHaveValue(title);
  const saved = page.waitForResponse(response =>
    new URL(response.url()).pathname === `/api/v1/transfers/${id}` &&
    response.request().method() === 'PATCH');
  await page.getByRole('button', { name: 'Сохранить название', exact: true }).click();
  const patch = await saved;
  expect(patch.request().postDataJSON()).toEqual({ title, message: '' });
  expect(patch.status()).toBe(204);
  const persisted = await page.request.get(`/api/v1/transfers/${id}`);
  expect(persisted.status()).toBe(200);
  expect(await persisted.json()).toMatchObject({ id, title, state: 'draft' });
  await expect(page.getByRole('heading', { name: title, exact: true })).toBeVisible();
  return id;
}
