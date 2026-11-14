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
await step('import pasted text into a draft, nothing saved until Save', async () => {
  await p.goto(base + '/recipes/new'); await p.waitForSelector('#import-url'); await shot('20-import-panel');
  await p.click('button:has-text("Paste the text instead")');
  await p.fill('#import-text', 'Linsensuppe\n\nZutaten\n200 g Linsen\n2 Zwiebeln\n\nZubereitung\n1. Kochen.\n2. Pürieren.');
  await p.click('section button:has-text("Import")'); await p.waitForSelector('text=Imported. Please check it');
  if (await p.inputValue('#name') !== 'Linsensuppe') throw new Error('name not filled');
  if (await p.locator('#import-url, #import-text').count()) throw new Error('panel should hide after importing');
  if (await p.locator('input[aria-label="Ingredient"]').count() !== 2) throw new Error('ingredients not filled');
  await shot('21-import-draft');
  if (await p.inputValue('#visibility') !== 'private') throw new Error('imports must start private');
  await p.click('form button:has-text("Save")'); await p.waitForURL(/recipes\/\d+$/); await p.waitForSelector('h1:has-text("Linsensuppe")');
  await p.click('button[aria-label="More"]'); await p.click('dialog[open] button:has-text("Delete")'); await p.waitForSelector('text=Moved to trash'); await p.waitForURL('**/recipes');
});
await step('typing hides the import panel so it cannot overwrite work', async () => {
  await p.goto(base + '/recipes/new'); await p.waitForSelector('#import-url'); await p.fill('#name', 'Mine');
  await p.waitForSelector('#import-url', { state: 'detached' }); await p.evaluate(() => { for (const k of Object.keys(localStorage)) if (k.includes('draft')) localStorage.removeItem(k); });
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
  await p.click('button[aria-label="More"]'); await p.click('dialog[open] button:has-text("Delete")'); await p.waitForSelector('text=Moved to trash'); await p.waitForURL('**/recipes'); await p.click('button:has-text("Undo")'); await p.waitForSelector('h1:has-text("My version")');
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
  await p.goto(base + '/settings'); await p.click('button:has-text("Light")');
});
await step('admin health and backup', async () => {
  await p.goto(base + '/admin'); await p.waitForSelector('text=Data safety'); await p.click('button:has-text("Back up now")'); await p.waitForSelector('text=Everything looks good', { timeout: 20000 }); await shot('13-admin');
  await p.waitForSelector('h2:has-text("Starter catalog updates")'); await p.waitForSelector('text=Nothing to decide.'); await p.waitForSelector('text=Last update');
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
  await p.goto(base + '/recipes'); await p.waitForSelector('a.recipe-card img'); await p.waitForTimeout(300); await shot('16-recipes');
});
await step('desktop layout and command palette', async () => {
  await p.setViewportSize({ width: 1280, height: 800 }); await p.goto(base + '/recipes'); await p.waitForSelector('aside'); await p.waitForTimeout(200); await shot('17-desktop');
  await p.keyboard.press('Control+k'); await p.waitForSelector('dialog[open] [role=combobox]'); await p.keyboard.type('My ver'); await p.waitForSelector('[role=option]:has-text("My version")'); await p.waitForTimeout(100); await shot('22-palette');
  await p.keyboard.press('Enter'); await p.waitForURL(/recipes\/\d+$/); await p.waitForSelector('h1:has-text("My version")'); await p.waitForTimeout(150); await shot('23-recipe-desktop');
  await p.keyboard.press('/'); await p.waitForSelector('dialog[open] [role=combobox]'); await p.keyboard.type('ingred'); await p.keyboard.press('Enter'); await p.waitForURL('**/ingredients');
  await p.goto(base + '/recipes'); await p.waitForSelector('aside'); await p.keyboard.press('n'); await p.waitForURL('**/recipes/new');
});
await step('meal plan, shopping and books work together', async () => {
  await p.setViewportSize({ width: 390, height: 844 });
  await p.goto(base + '/recipes'); await p.click('text=My version'); await p.waitForURL(/recipes\/\d+$/);
  await p.click('button[aria-label="More"]'); await p.waitForSelector('dialog[open] >> text=Add to meal plan'); await p.click('dialog[open] a:has-text("Add to meal plan")'); await p.waitForURL(/plan/);
  await p.waitForSelector('dialog[open] #plan-date'); await p.check('dialog[open] input[type=checkbox]'); await shot('24-plan-sheet'); await p.click('dialog[open] button:has-text("Save")');
  await p.waitForSelector('section button:has-text("My version")'); await shot('25-plan');
  await p.click('nav a[href="/shopping"]'); await p.waitForSelector('text=onions'); await p.fill('#shopping-line', '2 kg potatoes'); await p.keyboard.press('Enter'); await p.waitForSelector('button:has-text("potatoes")'); await shot('26-shopping');
  await p.click('[role=checkbox][aria-label*="potato"]'); await p.waitForSelector('text=In the cart');
  await p.click('button:has-text("Remove bought items")'); await p.waitForSelector('text=In the cart', { state: 'detached' });
  await p.goto(base + '/books'); await p.click('button:has-text("New book")'); await p.fill('#book-name', 'Weeknight'); await p.click('dialog[open] button:has-text("Save")'); await p.waitForSelector('a:has-text("Weeknight")');
  await p.goto(base + '/recipes'); await p.click('text=My version'); await p.waitForURL(/recipes\/\d+$/); await p.click('button[aria-label="More"]'); await p.click('dialog[open] button:has-text("Add to a book")'); await p.check('dialog[open] label:has-text("Weeknight") input');
  await p.goto(base + '/books'); await p.click('a:has-text("Weeknight")'); await p.waitForSelector('a.recipe-card:has-text("My version")');
});
await step('adding a planned week to shopping twice does not double it', async () => {
  await p.goto(base + '/shopping'); await p.waitForSelector('[role=checkbox]'); const before = await p.locator('[role=checkbox]').count();
  await p.goto(base + '/plan'); await p.click('button:has-text("Add this week to shopping")'); await p.waitForSelector('text=already on the list');
  await p.goto(base + '/shopping'); await p.waitForSelector('[role=checkbox]'); if (await p.locator('[role=checkbox]').count() !== before) throw new Error('shopping list grew');
});
await step('narrow phones never scroll sideways', async () => {
  await p.setViewportSize({ width: 320, height: 568 });
  for (const u of ['/recipes', '/plan', '/shopping', '/settings', '/recipes/new']) { await p.goto(base + u); await p.waitForLoadState('networkidle'); const o = await p.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth); if (o > 0) throw new Error(u + ' overflows by ' + o); }
  await p.setViewportSize({ width: 390, height: 844 });
});
await step('cooking mode, history and fractions', async () => {
  await p.goto(base + '/settings'); await p.selectOption('#amounts', 'fraction');
  await p.goto(base + '/recipes'); await p.click('text=My version'); await p.waitForURL(/recipes\/\d+$/);
  await p.waitForSelector('section.card li:has-text("1½")');
  await p.click('a:has-text("Cook")'); await p.waitForURL(/cook\?/); await p.waitForSelector('text=Step 1 of 2'); await shot('27-cook');
  await p.click('button:has-text("Next step")'); await p.waitForSelector('text=Step 2 of 2'); await p.click('button:has-text("Done")'); await p.waitForSelector('dialog[open] >> text=Your rating'); await p.click('dialog[open] button[aria-label="Close"]');
  await p.click('button[aria-label="More"]'); await p.click('dialog[open] button:has-text("Version history")'); await p.waitForSelector('dialog[open] button:has-text("Restore")'); await shot('28-history'); await p.click('dialog[open] button[aria-label="Close"]');
  await p.goto(base + '/settings'); await p.selectOption('#amounts', 'decimal');
  await p.click('button:has-text("Create link")'); await p.waitForSelector('input[aria-label="Link to your recipes"]');
  const link = await p.inputValue('input[aria-label="Link to your recipes"]'); const anon = await b.newContext({ viewport: { width: 390, height: 844 } }); const a = await anon.newPage();
  await a.goto(base + new URL(link, base).pathname); await a.waitForSelector('a.recipe-card:has-text("My version")'); await anon.close();
});
await step('live cooking and printing with only a share link', async () => {
  await p.goto(base + '/recipes'); await p.click('text=My version'); await p.waitForURL(/recipes\/\d+$/);
  await p.click('button[aria-label="More"]'); await p.click('button:has-text("Export")'); await p.waitForSelector('a:has-text("For another Wokko")');
  if (!/\/api\/v1\/recipes\/\d+\/export\?format=wokko$/.test(await p.getAttribute('a:has-text("For another Wokko")', 'href'))) throw new Error('export link');
  await p.click('dialog[open] button[aria-label="Close"]'); await p.click('button[aria-label="Share"]'); await p.waitForSelector('text=Who can see this');
  await p.click('button:has-text("Create link")'); await p.waitForSelector('input[aria-label="Share link"]'); const url = await p.inputValue('input[aria-label="Share link"]'); await p.click('dialog button[aria-label="Close"]');
  const anon = await b.newContext({ viewport: { width: 390, height: 844 }, locale: 'en-US', hasTouch: true, isMobile: true }); const a = await anon.newPage();
  const path = new URL(url, base).pathname;
  await a.goto(base + path); await a.waitForSelector('button:has-text("Print")'); await a.click('button[aria-label="More servings"]'); await a.emulateMedia({ media: 'print' }); await a.waitForTimeout(100); await a.screenshot({ path: `${shots}/32-print.png`, fullPage: true });
  if (await a.locator('button:has-text("Print")').isVisible()) throw new Error('the print button should not print');
  await a.emulateMedia({ media: 'screen' });
  // Someone with only the link can take the recipe to their own Wokko or another app.
  await a.click('button:has-text("Export")'); await a.waitForSelector('text=For another Wokko'); await a.screenshot({ path: `${shots}/33-export-shared.png` });
  for (const label of ['For another Wokko', 'For other apps']) {
    const href = await a.getAttribute(`a:has-text("${label}")`, 'href'); const res = await a.request.get(base + href);
    if (!res.ok() || res.headers()['content-type'] !== 'application/zip' || (await res.body()).length < 100) throw new Error(`${label}: ${res.status()}`);
  }
  await a.click('dialog button[aria-label="Close"]');
  await a.click('a:has-text("Cook")'); await a.waitForURL(/\/cook\?servings=5/); await a.waitForSelector('text=Step 1 of 2'); await a.waitForSelector('button:has-text("2.5")');
  await a.click('button:has-text("onion")'); await a.click('button[aria-label="All ingredients"]'); await a.waitForSelector('input[type=checkbox]');
  await a.click('button[aria-label="All ingredients"]'); await a.click('button:has-text("Next step")'); await a.waitForSelector('text=Step 2 of 2'); await a.screenshot({ path: `${shots}/33-cook-shared.png` });
  await a.reload(); await a.waitForSelector('text=Step 2 of 2');
  await a.click('button:has-text("Done")'); await a.waitForURL(new RegExp(path + '$')); await anon.close();
});
await step('footer, changelog and licenses without an account', async () => {
  const anon = await b.newContext({ viewport: { width: 390, height: 844 }, locale: 'en-US' }); const a = await anon.newPage();
  await a.goto(base + '/login'); await a.waitForSelector('footer a:has-text("GitHub")'); await a.click('footer a:has-text("Third-party notices")'); await a.waitForURL('**/licenses'); await a.waitForSelector('pre:has-text("EUPL-1.2")'); await a.waitForSelector('pre:has-text("go-webauthn")'); await a.waitForSelector('pre:has-text("Permission is hereby granted")'); await a.waitForSelector('pre:has-text("svelte")');
  await a.goto(base + '/changelog'); await a.waitForSelector('h2:has-text("0.1.0")'); await a.waitForSelector('footer'); await a.screenshot({ path: `${shots}/34-changelog.png` }); await anon.close();
});
await step('share to Wokko and the bookmark import from a link', async () => {
  const seen = []; await p.route('**/api/v1/import', (r) => { seen.push(r.request().postDataJSON()); r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ recipe: { name: 'Geteilt', description: '', work_minutes: 0, wait_minutes: 0, servings: 2, servings_text: '', source_url: 'https://e.com/r', visibility: 'private', keywords: [], steps: [{ name: '', instruction: 'Kochen.', minutes: 0, ingredients: [] }], base_version: 0 }, image_data: '', image_type: '', guessed: false }) }); });
  await p.goto(base + '/recipes/new?' + new URLSearchParams({ title: 'Toll', text: 'Schau mal https://e.com/r.' })); await p.waitForSelector('text=Imported. Please check it');
  if (seen[0]?.url !== 'https://e.com/r') throw new Error('shared link: ' + JSON.stringify(seen)); if (await p.inputValue('#name') !== 'Geteilt') throw new Error('draft not filled'); if (new URL(p.url()).search) throw new Error('reload would import again');
  await p.unroute('**/api/v1/import'); await p.evaluate(() => { for (const k of Object.keys(localStorage)) if (k.includes('draft')) localStorage.removeItem(k); });
  await p.goto(base + '/recipes/new?' + new URLSearchParams({ title: 'Suppe', text: 'Zutaten\n2 Karotten\n1 l Wasser\nZubereitung\nKochen.' })); await p.waitForSelector('text=Imported. Please check it'); if (await p.inputValue('#name') !== 'Suppe') throw new Error('text share');
  await p.evaluate(() => { for (const k of Object.keys(localStorage)) if (k.includes('draft')) localStorage.removeItem(k); });
});
await step('import another app\'s file, check first, undo', async () => {
  const recipes = [{ '@type': 'Recipe', name: 'Aus Datei eins', recipeIngredient: ['200 g Mehl', '2 Eier'], recipeInstructions: ['Mischen.'] }, { '@type': 'Recipe', name: 'My version', recipeIngredient: ['1 Salz'] }];
  await p.goto(base + '/more'); await p.click('a:has-text("Integrations")'); await p.waitForURL('**/integrations');
  await p.setInputFiles('#import-file', { name: 'export.json', mimeType: 'application/json', buffer: Buffer.from(JSON.stringify(recipes)) });
  await p.waitForSelector('text=Found 2 recipes'); await p.waitForSelector('li:has-text("My version") >> text=Already here'); await shot('35-file-preview');
  await p.goto(base + '/recipes'); await p.waitForSelector('a.recipe-card'); if (await p.locator('text=Aus Datei eins').count()) throw new Error('a preview saved something');
  // The old address still works.
  await p.goto(base + '/import'); await p.waitForURL('**/integrations'); await p.setInputFiles('#import-file', { name: 'export.json', mimeType: 'application/json', buffer: Buffer.from(JSON.stringify(recipes)) }); await p.click('button:has-text("Import 1 recipe")'); await p.waitForSelector('text=Imported 1 recipe'); await p.waitForSelector('text=Skipped 1');
  await p.goto(base + '/recipes'); await p.waitForSelector('text=Aus Datei eins');
  await p.goto(base + '/integrations'); await p.click('label:has-text("All I can see")'); await p.waitForSelector('text=To move a whole server'); await p.locator('#export-title').scrollIntoViewIfNeeded(); await shot('36-export-all');
  for (const label of ['For another Wokko', 'For other apps']) {
    const href = await p.getAttribute(`#export-title ~ div a:has-text("${label}")`, 'href'); if (!href.includes('scope=all')) throw new Error('export scope: ' + href);
    const res = await p.request.get(base + href); if (!res.ok() || res.headers()['content-type'] !== 'application/zip') throw new Error(`${label}: ${res.status()}`);
  }
  await p.goto(base + '/integrations'); await p.setInputFiles('#import-file', { name: 'bad.txt', mimeType: 'text/plain', buffer: Buffer.from('hello') }); await p.waitForSelector('[role=alert]:has-text("No recipes found")'); await p.waitForSelector('input[aria-label="From any web page"]');
});
await step('offline: saved recipes open, shopping queues and syncs', async () => {
  const off = await b.newContext({ viewport: { width: 390, height: 844 }, locale: 'en-US', hasTouch: true, isMobile: true }); const o = await off.newPage(); o.setDefaultTimeout(15000);
  await o.goto(base + '/login'); await o.fill('#handle', 'anna'); await o.fill('#password', 'password123'); await o.click('form button:has-text("Log in")'); await o.waitForURL('**/recipes');
  await o.evaluate(() => navigator.serviceWorker.ready); await o.waitForFunction(() => navigator.serviceWorker.controller); await o.reload(); await o.waitForSelector('a.recipe-card:has-text("My version")');
  await o.click('a.recipe-card:has-text("My version")'); await o.waitForSelector('h1:has-text("My version")'); await o.waitForTimeout(300);
  await o.goto(base + '/shopping'); await o.waitForSelector('#shopping-line'); await o.waitForTimeout(800); await o.goto(base + '/recipes'); await o.waitForSelector('a.recipe-card'); await o.waitForTimeout(800);
  await off.setOffline(true);
  await o.reload(); await o.waitForSelector('a.recipe-card:has-text("My version")'); await o.waitForSelector('text=You\'re offline'); await o.screenshot({ path: `${shots}/36-offline.png` });
  await o.click('a.recipe-card:has-text("My version")'); await o.waitForSelector('h1:has-text("My version")'); await o.waitForSelector('article img');
  await o.goto(base + '/shopping'); await o.waitForSelector('#shopping-line'); await o.fill('#shopping-line', '3 Offline-Tomaten'); await o.keyboard.press('Enter'); await o.waitForSelector('button:has-text("Offline-Tomaten")'); await o.fill('#shopping-line', 'Offline-Gurken'); await o.keyboard.press('Enter'); await o.waitForSelector('button:has-text("Offline-Gurken")'); await o.waitForSelector('button:has-text("Offline-Tomaten")'); await o.waitForSelector('text=waiting to be sent'); await o.screenshot({ path: `${shots}/37-offline-shopping.png` });
  await o.reload(); await o.waitForSelector('button:has-text("Offline-Tomaten")'); await o.waitForSelector('button:has-text("Offline-Gurken")');
  await off.setOffline(false); await o.waitForSelector('text=waiting to be sent', { state: 'detached' });
  const back = await o.evaluate(async () => (await (await fetch('/api/v1/shopping')).json()).filter((i) => i.food.name.includes('Offline-Tomaten')).length); if (back !== 1) throw new Error('sent ' + back + ' times');
  // The server saves an item, but its answer never arrives. The app sends it again.
  let lost = 0; await o.route('**/api/v1/shopping', async (r) => { if (r.request().method() === 'POST' && !lost++) { await r.fetch(); return r.abort(); } return r.continue(); });
  await o.fill('#shopping-line', 'Verlorene-Antwort'); await o.keyboard.press('Enter'); await o.waitForSelector('text=waiting to be sent'); await o.waitForSelector('text=waiting to be sent', { state: 'detached', timeout: 12000 }); await o.unroute('**/api/v1/shopping');
  const once = await o.evaluate(async () => (await (await fetch('/api/v1/shopping')).json()).filter((i) => i.food.name.includes('Verlorene-Antwort')).length); if (lost < 2 || once !== 1) throw new Error(`lost answer: sent ${lost} times, ${once} on the list`);
  await o.goto(base + '/settings'); await o.locator('button:has-text("Log out")').last().click(); await o.waitForURL('**/login**'); await o.waitForTimeout(300);
  const left = await o.evaluate(async () => (await caches.keys()).filter((k) => k === 'wokko-data' || k === 'wokko-media')); if (left.length) throw new Error('kept after logout: ' + left);
  await off.close();
});
await step('assistant access: admin switch, token and app approval', async () => {
  await p.goto(base + '/settings'); await p.click('a:has-text("Integrations")'); await p.waitForSelector('text=Turn this on in the admin settings first');
  await p.goto(base + '/admin'); await p.check('#mcp-enabled'); await p.waitForTimeout(200);
  const rows = await p.locator('section:has(#mcp-enabled) input[type=checkbox]').evaluateAll((els) => els.map((e) => Math.round(e.getBoundingClientRect().right))); if (rows.length !== 3 || new Set(rows).size !== 1) throw new Error('admin settings out of line: ' + rows);
  await p.goto(base + '/integrations'); await p.waitForSelector('text=Add custom connector'); await p.click('button[aria-pressed]:has-text("Claude Code")'); await p.fill('#token-name', 'Claude test'); await p.click('button:has-text("Create access token")');
  await p.waitForSelector('text=Copy this token now'); await p.waitForSelector('li:has-text("Claude test")'); await p.click('button[aria-pressed]:has-text("Don\'t allow")'); await p.waitForSelector('text=A recipe with an unknown one isn\'t saved'); const cmd = await p.inputValue('input[aria-label="Claude Code"]'); if (!cmd.startsWith('claude mcp add --transport http wokko ') || !cmd.includes('Bearer wokko_')) throw new Error('command without token: ' + cmd); await shot('29-assistant');
  await p.click('button[aria-pressed]:has-text("Codex")'); if (!(await p.textContent('pre')).includes('<token>')) throw new Error('another app must not get the Claude Code token');
  await p.click('button[aria-pressed]:has-text("Claude")'); const add = new URL(await p.getAttribute('a:has-text("Add to Claude")', 'href')); if (add.searchParams.get('connectorUrl') !== base + '/mcp' || add.searchParams.get('modal') !== 'add-custom-connector') throw new Error('Claude link: ' + add);
  await p.click('button[aria-pressed]:has-text("Cursor")'); if (await p.locator('a:has-text("Add to Cursor")').count()) throw new Error('Cursor link before a token');
  await p.fill('#token-name', 'Cursor test'); await p.click('button:has-text("Create access token")'); const cur = new URL(await p.getAttribute('a:has-text("Add to Cursor")', 'href'));
  const cfg = JSON.parse(Buffer.from(cur.searchParams.get('config'), 'base64').toString()); if (cfg.url !== base + '/mcp' || !cfg.headers.Authorization.startsWith('Bearer wokko_')) throw new Error('Cursor link: ' + JSON.stringify(cfg));
  await p.click('button[aria-pressed]:has-text("VS Code")'); await p.fill('#token-name', 'VS Code test'); await p.click('button:has-text("Create access token")'); const vs = await p.getAttribute('a:has-text("Add to VS Code")', 'href');
  const vcfg = JSON.parse(decodeURIComponent(vs.slice('vscode:mcp/install?'.length))); if (vcfg.type !== 'http' || vcfg.url !== base + '/mcp' || !vcfg.headers.Authorization.startsWith('Bearer wokko_')) throw new Error('VS Code link: ' + vs);
  const reg = await p.evaluate(async () => (await (await fetch('/oauth/register', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ client_name: 'Claude', redirect_uris: ['http://127.0.0.1:33333/callback'] }) })).json()));
  await p.route('http://127.0.0.1:33333/**', (r) => r.fulfill({ status: 200, contentType: 'text/html', body: 'ok' }));
  const q = new URLSearchParams({ client_id: reg.client_id, redirect_uri: 'http://127.0.0.1:33333/callback', response_type: 'code', code_challenge: 'abc', code_challenge_method: 'S256', scope: 'write', state: 'xyz' });
  await p.goto(base + '/oauth/authorize?' + q); await p.waitForSelector('h1:has-text("Allow access?")'); await p.waitForSelector('text=Claude wants to use Wokko'); await shot('30-consent');
  if (!(await p.isChecked('input[name=scope][value=write]'))) throw new Error('consent should start at what the app asked for'); await p.check('input[name=scope][value=read]'); const approval = p.waitForRequest((r) => r.url().endsWith('/oauth/approve')); await p.click('button.btn-primary:has-text("Allow")'); if ((await approval).postDataJSON().scope !== 'read') throw new Error('chosen access level not sent'); await p.waitForURL(/33333\/callback\?code=.+state=xyz|33333\/callback\?.*code=/);
  await p.goto(base + '/oauth/authorize?' + q); await p.waitForSelector('button:has-text("Don\'t allow")'); await p.click('button:has-text("Don\'t allow")'); await p.waitForURL(/error=access_denied/);
});
await step('passkeys with a virtual authenticator', async () => {
  const local = base.replace('127.0.0.1', 'localhost');
  const c3 = await b.newContext({ viewport: { width: 390, height: 844 }, locale: 'en-US' }); const k = await c3.newPage();
  const cdp = await c3.newCDPSession(k); await cdp.send('WebAuthn.enable');
  await cdp.send('WebAuthn.addVirtualAuthenticator', { options: { protocol: 'ctap2', transport: 'internal', hasResidentKey: true, hasUserVerification: true, isUserVerified: true, automaticPresenceSimulation: true } });
  await k.goto(local + '/login'); await k.fill('#handle', 'anna'); await k.fill('#password', 'password123'); await k.click('form button:has-text("Log in"):not(:has-text("passkey"))'); await k.waitForURL('**/recipes');
  await k.goto(local + '/settings'); await k.fill('input[aria-label="Name for this passkey"]', 'Test key'); await k.click('button:has-text("Add a passkey")'); await k.waitForSelector('li:has-text("Test key")'); await k.locator('li:has-text("Test key")').scrollIntoViewIfNeeded(); await k.screenshot({ path: `${shots}/31-passkey.png` });
  await c3.clearCookies(); await k.goto(local + '/login'); await k.click('button:has-text("Log in with a passkey")'); await k.waitForURL('**/recipes'); await c3.close();
});
await step('kitchen knowledge explains a recipe', async () => {
  const ing = (food_name, amount, unit_name = '') => ({ food_name, unit_name, amount, note: '', food_id: 0, unit_id: 0 });
  const res = await p.request.post(base + '/api/v1/recipes', { data: { name: 'Asparagus with hollandaise', description: '', work_minutes: 20, wait_minutes: 0, servings: 2, servings_text: '', source_url: '', visibility: 'private', keywords: [], base_version: 0, steps: [
    { name: '', minutes: 8, instruction: 'Whisk the egg yolks with lemon juice over a water bath, then slowly beat in the melted butter.', ingredients: [ing('egg yolk', 3), ing('lemon juice', 1, 'tablespoon'), ing('butter', 150, 'gram')] },
    { name: '', minutes: 10, instruction: 'Cook the asparagus and serve with the sauce.', ingredients: [ing('asparagus', 500, 'gram')] }] } });
  if (res.status() !== 201) throw new Error('create: ' + res.status() + ' ' + (await res.text()));
  const id = (await res.json()).id;
  await p.goto(base + '/recipes/' + id); await p.waitForSelector('h2:has-text("Kitchen knowledge")');
  await p.waitForSelector('text=Vegetarian'); await p.waitForSelector('text=Allergens:');
  await p.click('summary:has-text("Hollandaise")'); await p.waitForSelector('text=above about 65 °C');
  await p.locator('#insights-title').scrollIntoViewIfNeeded(); await shot('37-kitchen-knowledge');
});
await step('unknown page', async () => { await p.goto(base + '/does/not/exist'); await p.waitForSelector('text=Nothing here'); await shot('18-404'); });
await step('log out protects pages', async () => {
  await p.setViewportSize({ width: 1280, height: 800 }); await p.goto(base + '/settings'); await p.click('aside button:has-text("Log out")'); await p.waitForURL('**/login**'); await p.goto(base + '/recipes'); await p.waitForURL('**/login**'); await shot('19-login');
});
console.log('browser errors:', errors.length ? errors : 'none');
if (errors.length) process.exitCode = 1;
await b.close();
