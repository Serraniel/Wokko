import { chromium } from 'playwright-core';
// Drives a fresh Wokko instance in a phone-sized browser. See `make e2e`.
const base = process.env.WOKKO_URL ?? 'http://127.0.0.1:8765';
const shots = process.env.E2E_SHOTS ?? 'e2e/screenshots';
const photo = new URL('./photo.png', import.meta.url).pathname;
const b = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH ?? '/usr/bin/chromium', args: ['--no-sandbox'] });
const ctx = await b.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 2, hasTouch: true, isMobile: true, colorScheme: 'light', locale: 'en-US' });
const p = await ctx.newPage();
const errors = [];
p.on('pageerror', e => errors.push('pageerror: ' + e.message));
// Expected answers such as a 401 before login or a 409 conflict also log a line, so only
// script errors count here. Server errors are caught by the response check below.
p.on('console', m => { if (m.type() === 'error' && !m.text().startsWith('Failed to load resource')) errors.push('console: ' + m.text()); });
p.on('response', r => { if (r.status() >= 500) errors.push('5xx: ' + r.url()); });
const shot = n => p.screenshot({ path: `${shots}/${n}.png` });
const step = async (n, fn) => { const t = Date.now(); await fn(); console.log('ok', n, (Date.now() - t) + 'ms'); };

await step('first run redirects to setup', async () => { await p.goto(base + '/'); await p.waitForURL('**/setup'); await shot('01-setup'); });
await step('create admin', async () => {
  await p.fill('#handle', 'anna'); await p.fill('#display', 'Anna'); await p.fill('#password', 'password123');
  await p.click('button:has-text("Create account")'); await p.waitForURL('**/recipes'); await p.waitForSelector('text=No recipes yet'); await shot('02-empty');
});
await step('create recipe with new and existing ingredients', async () => {
  await p.click('nav a[href="/recipes/new"]'); await p.waitForURL('**/recipes/new');
  await p.fill('#name', 'Tomato soup'); await p.fill('#description', 'Warm and simple.'); await p.fill('#work', '25');
  await p.fill('input[aria-label="Qty"]', '2');
  await p.fill('input[aria-label="Ingredient"]', 'oni'); await p.waitForSelector('li[role=option] >> text=onion'); await shot('03-combobox');
  await p.click('li[role=option] >> text=onion');
  const unit = await p.inputValue('input[aria-label="Unit"]'); if (!unit) throw new Error('unit was not suggested');
  await p.click('button:has-text("Add ingredient")');
  await p.locator('input[aria-label="Qty"]').nth(1).fill('1 1/2');
  await p.locator('input[aria-label="Ingredient"]').nth(1).fill('Smoked unicorn dust');
  await p.waitForSelector('text=Create “Smoked unicorn dust”'); await p.click('text=Create “Smoked unicorn dust”');
  await p.fill('textarea[id^="ins-"]', 'Chop everything and simmer.');
  await p.click('button:has-text("Add step")');
  await p.locator('textarea[id^="ins-"]').nth(1).fill('Blend until smooth.');
  await shot('04-editor');
  await p.click('form button:has-text("Save")'); await p.waitForURL(/recipes\/\d+$/);
  await p.waitForSelector('h1:has-text("Tomato soup")'); await p.waitForSelector('text=Smoked unicorn dust'); await p.waitForSelector('text=Blend until smooth');
  await shot('05-recipe');
});
await step('edit a recipe with steps (broken in the old app)', async () => {
  await p.click('a[aria-label="Edit"]'); await p.waitForURL(/edit$/); await p.waitForSelector('#name');
  await p.fill('#name', 'Better tomato soup'); await p.click('form button:has-text("Save")'); await p.waitForURL(/recipes\/\d+$/);
  await p.waitForSelector('h1:has-text("Better tomato soup")');
  const n = await p.locator('ol > li').count(); if (n !== 2) throw new Error('steps duplicated or lost: ' + n);
});
await step('scale servings', async () => {
  const before = await p.locator('section.card li').first().innerText();
  await p.click('button[aria-label="More servings"]'); await p.waitForTimeout(50);
  const after = await p.locator('section.card li').first().innerText();
  if (before === after) throw new Error('amount did not scale'); console.log('   ', JSON.stringify(before), '->', JSON.stringify(after));
});
await step('draft survives reload', async () => {
  await p.goto(base + '/recipes/new'); await p.fill('#name', 'Unsaved idea'); await p.waitForTimeout(600); await p.reload();
  await p.waitForSelector('text=You have unsaved changes'); await shot('06-draft');
  await p.click('button:has-text("Restore")'); if (await p.inputValue('#name') !== 'Unsaved idea') throw new Error('not restored');
  await p.click('a:has-text("Cancel")');
});
await step('conflict is detected, draft kept', async () => {
  await p.goto(base + '/recipes'); await p.waitForSelector('text=Better tomato soup'); await p.click('text=Better tomato soup'); await p.waitForURL(/recipes\/\d+$/);
  const id = p.url().split('/').pop();
  await p.goto(`${base}/recipes/${id}/edit`); await p.waitForSelector('#name');
  await p.evaluate(async (id) => { const r = await (await fetch(`/api/v1/recipes/${id}`)).json(); await fetch(`/api/v1/recipes/${id}`, { method: 'PUT', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ name: 'Changed elsewhere', servings: r.servings, visibility: r.visibility, base_version: r.version, steps: [] , keywords: []}) }); }, id);
  await p.fill('#name', 'My version'); await p.click('form button:has-text("Save")');
  await p.waitForSelector('text=Someone changed this recipe'); await shot('07-conflict');
  await p.click('button:has-text("Save mine anyway")'); await p.waitForURL(/recipes\/\d+$/); await p.waitForSelector('h1:has-text("My version")');
  const v = await p.evaluate(async (id) => (await (await fetch(`/api/v1/recipes/${id}/versions`)).json()).length, id); if (v < 2) throw new Error('history missing: ' + v);
});
await step('share sheet and link', async () => {
  await p.click('button[aria-label="Share"]'); await p.waitForSelector('text=Who can see this');
  await p.fill('input[aria-label="Exact handle"]', 'nobody-here'); await p.click('dialog form button:has-text("Share")'); await p.waitForSelector('text=No user has exactly this handle');
  await p.click('button:has-text("Create link")'); await p.waitForSelector('input[aria-label="Share link"]'); await shot('08-share');
  const url = await p.inputValue('input[aria-label="Share link"]'); console.log('   ', url.replace(/s\/.*/, 's/…'));
  const path = new URL(url, base).pathname; await p.click('dialog button[aria-label="Close"]');
  const anon = await b.newContext({ viewport: { width: 390, height: 844 }, locale: 'de-DE' }); const a = await anon.newPage();
  await a.goto(base + path); await a.waitForSelector('h1:has-text("My version")');
  if (await a.locator('text=anna').count()) throw new Error('author visible'); await anon.close();
});
await step('delete and undo', async () => {
  await p.click('button[aria-label="Delete"]'); await p.waitForSelector('text=Moved to trash'); await p.waitForURL('**/recipes'); await p.click('button:has-text("Undo")'); await p.waitForSelector('h1:has-text("My version")');
});
await step('German UI and ingredient names', async () => {
  await p.goto(base + '/settings'); await p.selectOption('#language', 'de'); await p.waitForSelector('h1:has-text("Einstellungen")'); await shot('09-settings-de');
  await p.goto(base + '/ingredients'); await p.waitForSelector('h1:has-text("Zutaten")'); await p.fill('input[type=search]', 'zwie'); await p.waitForSelector('button:has-text("Zwiebel")'); await p.click('button:has-text("Zwiebel")'); await p.waitForSelector('dialog input[aria-label^="Deutsch"]'); await shot('10-ingredient-sheet'); await p.click('dialog button[aria-label="Schließen"]');
  await p.goto(base + '/settings'); await p.selectOption('#language', 'en'); await p.waitForSelector('h1:has-text("Settings")');
});
await step('dark mode', async () => {
  await p.click('button:has-text("Dark")'); const dark = await p.evaluate(() => getComputedStyle(document.body).backgroundColor); console.log('   body', dark); await shot('11-settings-dark');
  await p.goto(base + '/recipes'); await p.waitForSelector('text=My version'); await shot('12-recipes-dark');
  await p.reload(); await p.waitForSelector('text=My version'); if ((await p.evaluate(() => document.documentElement.dataset.theme)) !== 'dark') throw new Error('theme not kept');
  await p.click('nav a[href="/settings"]'); await p.click('button:has-text("Light")');
});
await step('admin health and backup', async () => {
  await p.goto(base + '/admin'); await p.waitForSelector('text=Data safety'); await p.click('button:has-text("Back up now")'); await p.waitForSelector('text=Everything looks good', { timeout: 20000 }); await shot('13-admin');
});
await step('invite, register second user, no discovery', async () => {
  await p.click('button:has-text("Create invite link")'); await p.waitForSelector('input[aria-label="Invite link"]');
  const link = await p.inputValue('input[aria-label="Invite link"]');
  const c2 = await b.newContext({ viewport: { width: 390, height: 844 } }); const u = await c2.newPage();
  await u.goto(link); await u.waitForSelector('#invite'); await u.fill('#handle', 'john.doe'); await u.fill('#password', 'password123'); await u.click('button:has-text("Create account")'); await u.waitForURL('**/recipes');
  await u.waitForSelector('nav'); if (await u.locator('a[href="/admin"]').count()) throw new Error('admin link for normal user');
  await u.goto(base + '/admin'); await u.waitForURL('**/recipes'); await c2.close();
  await p.goto(base + '/people'); await p.fill('input[aria-label="Exact handle"]', 'john'); await p.click('button:has-text("Add")'); await p.waitForSelector('text=No user has exactly this handle');
  await p.fill('input[aria-label="Exact handle"]', 'john.doe'); await p.click('button:has-text("Add")'); await p.waitForSelector('text=Waiting for an answer'); await shot('14-people');
});
await step('photo upload', async () => {
  await p.goto(base + '/recipes'); await p.click('text=My version'); await p.waitForURL(/recipes\/\d+$/); await p.click('a[aria-label="Edit"]'); await p.waitForSelector('#name');
  await p.setInputFiles('input[type=file]', photo); await p.click('form button:has-text("Save")'); await p.waitForURL(/recipes\/\d+$/); await p.waitForSelector('article img'); await p.waitForTimeout(300); await shot('15-recipe-photo');
  await p.goto(base + '/recipes'); await p.waitForSelector('a.card img'); await p.waitForTimeout(300); await shot('16-recipes');
});
await step('desktop layout', async () => {
  await p.setViewportSize({ width: 1280, height: 800 }); await p.goto(base + '/recipes'); await p.waitForSelector('aside'); await p.waitForTimeout(200); await shot('17-desktop');
});
await step('unknown page', async () => { await p.goto(base + '/does/not/exist'); await p.waitForSelector('text=Nothing here'); await shot('18-404'); });
await step('log out protects pages', async () => {
  await p.goto(base + '/settings'); await p.click('aside button:has-text("Log out")'); await p.waitForURL('**/login**'); await p.goto(base + '/recipes'); await p.waitForURL('**/login**'); await shot('19-login');
});
console.log('browser errors:', errors.length ? errors : 'none');
if (errors.length) process.exitCode = 1;
await b.close();
