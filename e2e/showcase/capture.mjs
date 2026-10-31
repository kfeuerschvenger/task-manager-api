// Captures the screenshots and the hero animation used in the README.
//
// It drives a running API that holds the demo data (`go run ./cmd seed`), takes
// each shot in a real browser through the Swagger UI, and writes two files per
// shot: the raw screenshot, and the same screenshot inside a browser frame. The
// hero slug records a video instead, which gif.mjs turns into a GIF and an MP4.
//
// Nothing here confirms a dialog that writes to the database. Two shots create
// a task on purpose (create-task and the hero), and both delete what they
// created through the API when they finish, so the demo data ends as it began.
// The forbidden shot is rejected by the API and writes nothing.
//
// A bearer token never reaches an image. The token is pasted into a masked
// field, every `pre` block that would print it is rewritten to a truncated
// form before it is painted, and each shot fails when the visible text of the
// page still holds a full token.
//
// Usage, from e2e/showcase/ (see README.md for the variables):
//   node capture.mjs                  every still
//   SLUGS=overview,forbidden node capture.mjs
//   node capture.mjs hero             the video, for gif.mjs (a slug may also be an argument)

import { copyFile, mkdir, readFile, rm, stat, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { chromium } from 'playwright';

const here = path.dirname(fileURLToPath(import.meta.url));
const REPO = path.resolve(here, '..', '..');

const BASE = (process.env.BASE_URL ?? 'http://localhost:8080').replace(/\/$/, '');
const CREATOR_EMAIL = process.env.DEMO_CREATOR_EMAIL ?? '';
const ASSIGNEE_EMAIL = process.env.DEMO_ASSIGNEE_EMAIL ?? '';
const PASSWORD = process.env.DEMO_PASSWORD ?? '';

const DOCS = `${BASE}/documentation/index.html`;

/** Framed shots, which are committed. */
const OUT = process.env.SHOWCASE_OUT ? path.resolve(process.env.SHOWCASE_OUT) : path.join(REPO, 'docs', 'assets', 'screens');
/** Raw shots and the hero video, which are not. */
const WORK = path.join(here, 'out');

const VIEWPORT = { width: 1600, height: 1000 };
/** Raw shots are taken at twice the pixel density, for crisp text. */
const RAW_SCALE = 2;
/** The framed composite is rendered at this density. */
const FRAME_SCALE = Number(process.env.SHOWCASE_FRAME_SCALE ?? 1.25);

/** The demo task the filtered list must contain: it proves the shot shows the seeded data. */
const KNOWN_HIGH_TITLE = 'Write the README before the due date';
/** Titles of the tasks the flows create. Neither exists in the seed, so a cleanup can match on the title. */
const STILL_TASK = {
  title: 'Draft the changelog before the release',
  description: 'List what changed since the last tag, in the order a reader cares about.',
  due_date: '2026-11-09T09:00:00Z',
  priority: 'high',
  status: 'pending',
};
const HERO_TASK = {
  title: 'Publish the Swagger docs before lunch',
  description: 'Regenerate the spec, check the try-it-out flow and tell the team where it lives.',
  due_date: '2026-11-09T09:00:00Z',
  priority: 'high',
  status: 'pending',
};
/** A full JWT: header, payload and signature. */
const TOKEN_PATTERN = /eyJ[\w-]{6,}\.[\w-]+\.[\w-]+/;

const log = (...parts) => console.log(...parts);

// -- the API, used for tokens, for the ids a shot needs and for cleaning up ----------------------

const api = async (route, { method = 'GET', token, body } = {}) => {
  const response = await fetch(`${BASE}${route}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await response.text();
  return { status: response.status, json: text ? JSON.parse(text) : null };
};

const login = async email => {
  const { status, json } = await api('/auth/login', { method: 'POST', body: { email, password: PASSWORD } });
  if (status !== 200) throw new Error(`login as ${email} answered ${status}: is the demo seeded with this password?`);
  return json.token;
};

/** Deletes every task of the creator with this title, which is how a flow undoes its own write. */
const deleteTasksTitled = async (token, title) => {
  const { json } = await api('/tasks', { token });
  for (const task of json ?? []) {
    if (task.title === title) await api(`/tasks/${task.id}`, { method: 'DELETE', token });
  }
};

// -- scripts that run inside every page ---------------------------------------------------------

/**
 * Keeps a bearer token off the screen. The Authorize field shows dots, and a
 * `pre` block that prints a token (the curl command, the login response) is
 * rewritten to the first characters of it, in the same task as the render and
 * so before the next paint. The Swagger highlighter splits a token into one
 * span per punctuation mark, so the match runs on the block, not on a text node.
 */
const MASK_SCRIPT = pattern => {
  const token = new RegExp(pattern);
  const everyToken = new RegExp(pattern, 'g');
  const scrub = () => {
    for (const block of document.querySelectorAll('pre')) {
      if (token.test(block.textContent)) block.textContent = block.textContent.replace(everyToken, found => `${found.slice(0, 12)}...`);
    }
  };
  window.addEventListener('DOMContentLoaded', () => {
    const style = document.createElement('style');
    style.textContent = '.auth-container input { -webkit-text-security: disc; }';
    document.head.append(style);
    new MutationObserver(scrub).observe(document.body, { subtree: true, childList: true, characterData: true });
  });
};

/** A dot that follows the pointer, since a recording does not show the system cursor. */
const CURSOR_SCRIPT = () => {
  window.addEventListener('DOMContentLoaded', () => {
    const dot = document.createElement('div');
    dot.style.cssText =
      'position:fixed;left:0;top:0;width:22px;height:22px;margin:-11px 0 0 -11px;border-radius:50%;' +
      'background:rgb(0 173 216 / 0.85);border:3px solid #fff;box-shadow:0 2px 8px rgb(15 23 42 / 0.4);' +
      'pointer-events:none;z-index:2147483647;transition:transform 0.08s ease-out;transform:translate(-100px,-100px)';
    document.body.append(dot);
    window.addEventListener('mousemove', event => {
      dot.style.transform = `translate(${event.clientX}px, ${event.clientY}px)`;
    }, true);
    window.addEventListener('mousedown', () => (dot.style.scale = '0.8'), true);
    window.addEventListener('mouseup', () => (dot.style.scale = '1'), true);
  });
};

/** Lets a response body show every line. Swagger caps it at 400 pixels and scrolls, which would cut the list. */
const SHOW_WHOLE_RESPONSE_CSS = `
  .highlight-code > .microlight, .microlight { max-height: none !important; }
  textarea.body-param__text { min-height: 0 !important; height: 130px !important; }
  .responses-inner > div > h4, .responses-inner .responses-table:not(.live-responses-table) { display: none !important; }
`;

const openContext = (browser, options = {}) =>
  browser.newContext({
    viewport: VIEWPORT,
    deviceScaleFactor: RAW_SCALE,
    colorScheme: 'light',
    locale: 'en-US',
    timezoneId: 'UTC',
    ...options,
  });

const prepare = async context => {
  await context.addInitScript(MASK_SCRIPT, TOKEN_PATTERN.source);
};

// -- Swagger UI helpers --------------------------------------------------------------------------

const openDocs = async page => {
  await page.goto(DOCS);
  await page.locator('.opblock-tag').first().waitFor();
  await page.addStyleTag({ content: SHOW_WHOLE_RESPONSE_CSS });
};

const expandTag = async (page, tag) => {
  const heading = page.locator(`#operations-tag-${tag}`);
  if ((await heading.getAttribute('data-is-open')) !== 'true') await heading.click();
  await page.locator(`.opblock-tag-section.is-open #operations-tag-${tag}`).waitFor();
};

/** Opens an operation and turns on Try it out. The id looks like `tasks-post_tasks`. */
const openOperation = async (page, id) => {
  const block = page.locator(`#operations-${id}`);
  await block.locator('.opblock-summary-control').click();
  await block.locator('button.try-out__btn').click();
  return block;
};

const authorize = async (page, token) => {
  await page.locator('button.authorize').first().click();
  const modal = page.locator('.auth-container');
  await modal.locator('input[type="text"]').fill(`Bearer ${token}`);
  await modal.locator('button.authorize').click();
  await modal.locator('button', { hasText: 'Logout' }).waitFor();
  await page.locator('.modal-ux button.btn-done').click();
  await page.locator('.modal-ux').waitFor({ state: 'detached' });
};

/** The status code of the live response of an operation, once it has appeared. */
const waitForStatus = async block => {
  const status = block.locator('.live-responses-table .response .response-col_status').first();
  await status.waitFor();
  const code = (await status.innerText()).trim().match(/^\d{3}/)?.[0];
  if (!code) throw new Error('the response panel shows no status code');
  return Number(code);
};

/** Presses Execute and returns the status code the live response shows. */
const execute = async (block, { page }) => {
  await block.locator('button.execute').click();
  const code = await waitForStatus(block);
  await page.waitForFunction(() => document.querySelectorAll('.loading-container, .btn.execute.loading').length === 0);
  return code;
};

const responseBody = block => block.locator('.live-responses-table .response .response-col_description pre.microlight').first();

/**
 * Brings the block to the top of the window, and grows the window when the whole exchange does not fit, so no
 * response is cut. The width stays 1600.
 */
const frameBlock = async (page, block) => {
  const height = await block.evaluate(element => element.getBoundingClientRect().height);
  const needed = Math.ceil(height) + 48;
  if (needed > VIEWPORT.height) await page.setViewportSize({ width: VIEWPORT.width, height: needed });
  await block.evaluate(element => window.scrollTo(0, element.getBoundingClientRect().top + window.scrollY - 24));
};

const waitForCalm = page =>
  page.waitForFunction(
    () => document.getAnimations().every(animation => animation.effect?.getComputedTiming().iterations === Infinity || animation.playState !== 'running'),
    undefined,
    { timeout: 15_000 }
  );

/** Fails the shot when a token is on screen or something is still loading. */
const assertClean = async page => {
  const visible = await page.evaluate(() => document.body.innerText);
  if (TOKEN_PATTERN.test(visible)) throw new Error('a full bearer token is visible on the page');

  const busy = await page.locator('.loading-container, .btn.execute.loading').count();
  if (busy > 0) throw new Error(`a loading indicator is still on screen (${busy})`);
};

const settle = async page => {
  await page.waitForLoadState('networkidle');
  await page.mouse.move(VIEWPORT.width / 2, 2);
  await waitForCalm(page);
  await page.waitForTimeout(400);
  await assertClean(page);
};

const expectStatus = (actual, expected) => {
  if (actual !== expected) throw new Error(`expected status ${expected}, the page shows ${actual}`);
};

const expectText = async (locator, text) => {
  const shown = await locator.innerText();
  if (!shown.includes(text)) throw new Error(`the response does not contain "${text}"`);
};

// -- the inventory -------------------------------------------------------------------------------

/**
 * A shot receives a fresh page on the Swagger UI and, in `run`, brings it to the state to photograph. `cleanup`
 * runs in every case, success or failure, and undoes whatever the shot wrote.
 */
const SHOTS = [
  {
    slug: 'overview',
    run: async page => {
      await openDocs(page);
      await expandTag(page, 'auth');
      await expandTag(page, 'tasks');
      await expandTag(page, 'healthcheck');

      const groups = await page.locator('.opblock-tag').allInnerTexts();
      for (const group of ['auth', 'tasks', 'healthcheck']) {
        if (!groups.some(text => text.trim().startsWith(group))) throw new Error(`the ${group} group is missing`);
      }
      const operations = await page.locator('.opblock').count();
      if (operations !== 8) throw new Error(`expected 8 operations, found ${operations}`);
      await settle(page);
    },
  },
  {
    slug: 'create-task',
    run: async page => {
      await openDocs(page);
      await authorize(page, await login(CREATOR_EMAIL));
      await expandTag(page, 'tasks');
      const block = await openOperation(page, 'tasks-post_tasks');
      await block.locator('textarea.body-param__text').fill(JSON.stringify(STILL_TASK, null, 2));
      expectStatus(await execute(block, { page }), 201);
      await expectText(responseBody(block), STILL_TASK.title);
      await frameBlock(page, block);
      await settle(page);
    },
    cleanup: async () => deleteTasksTitled(await login(CREATOR_EMAIL), STILL_TASK.title),
  },
  {
    slug: 'filter-high',
    run: async page => {
      await openDocs(page);
      await authorize(page, await login(CREATOR_EMAIL));
      await expandTag(page, 'tasks');
      const block = await openOperation(page, 'tasks-get_tasks');
      await block.locator('tr[data-param-name="priority"] input').fill('high');
      expectStatus(await execute(block, { page }), 200);

      const body = responseBody(block);
      await expectText(body, KNOWN_HIGH_TITLE);
      const shown = await body.innerText();
      if (/"priority":\s*"(low|medium)"/.test(shown)) throw new Error('the filtered list holds a task that is not high priority');
      await frameBlock(page, block);
      await settle(page);
    },
  },
  {
    slug: 'forbidden',
    run: async page => {
      const assigneeToken = await login(ASSIGNEE_EMAIL);
      // The API does not say who created a task, so pick the one the assignee holds that the creator owns.
      const { json } = await api('/tasks', { token: assigneeToken });
      const target = (json ?? []).find(task => task.title === 'Promote a task from pending to in_progress');
      if (!target) throw new Error('the assignee has no seeded task to try to edit: is the demo seeded?');

      await openDocs(page);
      await authorize(page, assigneeToken);
      await expandTag(page, 'tasks');
      const block = await openOperation(page, 'tasks-put_tasks__id_');
      await block.locator('tr[data-param-name="id"] input').fill(target.id);
      await block.locator('textarea.body-param__text').fill(JSON.stringify({ status: 'complete' }, null, 2));
      expectStatus(await execute(block, { page }), 403);
      await expectText(responseBody(block), 'Unauthorized to update task');
      await frameBlock(page, block);
      await settle(page);
    },
  },
];

/** Every slug that can be asked for: the stills and the hero video. */
const SLUGS = [...SHOTS.map(shot => shot.slug), 'hero'];

/** The slugs this run takes. The hero is only recorded when it is named. */
const selectSlugs = () => {
  const requested = [process.env.SLUGS ?? '', ...process.argv.slice(2)]
    .flatMap(list => list.split(','))
    .map(slug => slug.trim())
    .filter(Boolean);

  const unknown = requested.filter(slug => !SLUGS.includes(slug));
  if (unknown.length > 0) {
    throw new Error(`Unknown slug ${unknown.join(', ')}. Known slugs: ${SLUGS.join(', ')}`);
  }

  return requested.length > 0 ? requested : SHOTS.map(shot => shot.slug);
};

/** Puts a raw screenshot in the browser frame and returns the framed PNG. */
const frameShot = async (framePage, raw, width) => {
  await framePage.evaluate(shot => window.placeShot(shot), {
    src: `data:image/png;base64,${raw.toString('base64')}`,
    width,
    address: 'localhost:8080/documentation/',
  });
  return framePage.locator('#stage').screenshot({ type: 'png' });
};

const kilobytes = async file => `${Math.round((await stat(file)).size / 1024)} KB`;

const takeStill = async (shot, { browser, framePage }) => {
  const context = await openContext(browser);
  await prepare(context);
  const page = await context.newPage();
  try {
    await shot.run(page);
    const raw = await page.screenshot({ type: 'png' });
    const framed = await frameShot(framePage, raw, VIEWPORT.width);

    await writeFile(path.join(WORK, `${shot.slug}.raw.png`), raw);
    await writeFile(path.join(OUT, `${shot.slug}.png`), framed);
  } finally {
    await context.close().catch(() => {});
    await shot.cleanup?.();
  }
};

// -- the hero --------------------------------------------------------------------------------------

/** Waits the way a person looks at a screen. */
const PACE = 0.4;
const beat = (page, ms = 700) => page.waitForTimeout(ms * PACE);

/** Moves the pointer to the middle of an element along a smooth path, which is what the recording shows. */
const glide = async (page, locator, steps = 10) => {
  await locator.scrollIntoViewIfNeeded();
  const box = await locator.boundingBox();
  if (!box) throw new Error('an element the hero flow needs is not rendered');
  const x = box.x + Math.min(box.width / 2, 240);
  const y = box.y + box.height / 2;
  const from = await page.evaluate(() => window.__pointer ?? { x: 800, y: 40 });
  for (let step = 1; step <= steps; step += 1) {
    const t = step / steps;
    const eased = t < 0.5 ? 2 * t * t : 1 - (-2 * t + 2) ** 2 / 2;
    await page.mouse.move(from.x + (x - from.x) * eased, from.y + (y - from.y) * eased);
    // The recorder keeps 25 frames a second: one pointer move every 24 ms or so gives it a new picture for each.
    await page.waitForTimeout(24);
  }
  await page.evaluate(point => (window.__pointer = point), { x, y });
};

const press = async (page, locator) => {
  await glide(page, locator);
  await page.waitForTimeout(120);
  await locator.click();
};

/** Scrolls the window smoothly so the element sits near the top, as a hand on a wheel would. */
const scrollTo = async (page, locator, offset = 90) => {
  await locator.evaluate(
    (element, gap) => window.scrollTo({ top: element.getBoundingClientRect().top + window.scrollY - gap, behavior: 'instant' }),
    offset
  );
  await page.waitForTimeout(350);
};

const MASKED_PASSWORD = '************';

const recordHero = async () => {
  const dir = path.join(WORK, 'video');
  await rm(dir, { recursive: true, force: true });

  const browser = await chromium.launch({ channel: 'chromium' });
  const context = await openContext(browser, { deviceScaleFactor: 1, recordVideo: { dir, size: VIEWPORT } });
  await prepare(context);
  await context.addInitScript(CURSOR_SCRIPT);
  // The login body on screen shows dots for the password. The request that leaves the browser carries the real one,
  // so the password never appears in the recording and the login still works.
  await context.route('**/auth/login', route => {
    const body = JSON.parse(route.request().postData() ?? '{}');
    return route.continue({ postData: JSON.stringify({ ...body, password: PASSWORD }) });
  });

  const page = await context.newPage();
  const recordingStarted = Date.now();
  let token = '';

  try {
    await openDocs(page);
    await page.mouse.move(800, 40);
    await page.evaluate(() => (window.__pointer = { x: 800, y: 40 }));
    await settle(page);

    // The footage before this moment is the page loading, and gif.mjs cuts it.
    const startAt = (Date.now() - recordingStarted) / 1000 + 0.3;
    await beat(page, 1200);

    // Log in through the API and read the token from the response.
    await press(page, page.locator('#operations-tag-auth'));
    await beat(page);
    const signIn = page.locator('#operations-auth-post_auth_login');
    await press(page, signIn.locator('.opblock-summary-control'));
    await beat(page, 500);
    await press(page, signIn.locator('button.try-out__btn'));
    await beat(page, 500);
    await signIn.locator('textarea.body-param__text').fill(JSON.stringify({ email: CREATOR_EMAIL, password: MASKED_PASSWORD }, null, 2));
    await beat(page, 900);
    const answered = page.waitForResponse(response => response.url().endsWith('/auth/login'));
    await press(page, signIn.locator('button.execute'));
    const loginResponse = await answered;
    token = (await loginResponse.json()).token;
    expectStatus(await waitForStatus(signIn), 200);
    await beat(page, 1800);

    // Authorize with Bearer and the token. The field shows dots.
    await scrollTo(page, page.locator('.scheme-container'), 0);
    await press(page, page.locator('button.authorize').first());
    await beat(page, 500);
    const modal = page.locator('.auth-container');
    await modal.locator('input[type="text"]').fill(`Bearer ${token}`);
    await beat(page, 900);
    await press(page, modal.locator('button.authorize'));
    await modal.locator('button', { hasText: 'Logout' }).waitFor();
    await beat(page, 900);
    await press(page, page.locator('.modal-ux button.btn-done'));
    await beat(page, 600);

    // Put the login away, so the next calls have the page to themselves.
    await press(page, page.locator('#operations-tag-auth'));
    await beat(page, 500);

    // Create a task.
    await press(page, page.locator('#operations-tag-tasks'));
    await beat(page, 500);
    const create = page.locator('#operations-tasks-post_tasks');
    await press(page, create.locator('.opblock-summary-control'));
    await beat(page, 400);
    await press(page, create.locator('button.try-out__btn'));
    await beat(page, 400);
    await create.locator('textarea.body-param__text').fill(JSON.stringify(HERO_TASK, null, 2));
    await beat(page, 800);
    await scrollTo(page, create.locator('.opblock-summary'), 24);
    await press(page, create.locator('button.execute'));
    expectStatus(await waitForStatus(create), 201);
    await scrollTo(page, create.locator('.responses-wrapper'), 160);
    await beat(page, 2200);

    // Close it and read the list, filtered to high priority.
    await scrollTo(page, create.locator('.opblock-summary'), 24);
    await press(page, create.locator('.opblock-summary-control'));
    await beat(page, 500);
    const list = page.locator('#operations-tasks-get_tasks');
    await press(page, list.locator('.opblock-summary-control'));
    await beat(page, 400);
    await press(page, list.locator('button.try-out__btn'));
    await beat(page, 400);
    const priority = list.locator('tr[data-param-name="priority"] input');
    await press(page, priority);
    await priority.pressSequentially('high', { delay: 110 });
    await beat(page, 700);
    await press(page, list.locator('button.execute'));
    expectStatus(await waitForStatus(list), 200);

    const body = responseBody(list);
    await expectText(body, KNOWN_HIGH_TITLE);
    await expectText(body, HERO_TASK.title);
    await scrollTo(page, list.locator('.responses-wrapper'), 120);
    await beat(page, 3600);
    await assertClean(page);

    const endAt = (Date.now() - recordingStarted) / 1000;
    await page.mouse.move(VIEWPORT.width / 2, 2);
    await beat(page, 300);

    await context.close();
    const video = await page.video().path();
    await copyFile(video, path.join(WORK, 'hero.webm'));
    await writeFile(path.join(WORK, 'hero.json'), JSON.stringify({ start: Number(startAt.toFixed(2)), end: Number(endAt.toFixed(2)) }));
  } finally {
    await context.close().catch(() => {});
    await browser.close().catch(() => {});
    // Undo the write. A fresh login keeps this working when the flow failed before it had a token.
    await deleteTasksTitled(await login(CREATOR_EMAIL), HERO_TASK.title);
  }
};

// -- main ----------------------------------------------------------------------------------------

const requireEnvironment = () => {
  const missing = [
    ['DEMO_CREATOR_EMAIL', CREATOR_EMAIL],
    ['DEMO_ASSIGNEE_EMAIL', ASSIGNEE_EMAIL],
    ['DEMO_PASSWORD', PASSWORD],
  ]
    .filter(([, value]) => value.length === 0)
    .map(([name]) => name);

  if (missing.length > 0) {
    throw new Error(`${missing.join(', ')} not set. They are the two demo users and the password the demo was seeded with (go run ./cmd seed). None has a default.`);
  }
};

const main = async () => {
  requireEnvironment();

  const slugs = selectSlugs();
  const stills = SHOTS.filter(shot => slugs.includes(shot.slug));

  await mkdir(WORK, { recursive: true });
  await mkdir(OUT, { recursive: true });
  await readFile(path.join(here, 'frame.html'));

  const browser = await chromium.launch();
  const failures = [];
  const report = (slug, ok, detail) => {
    log(`${ok ? 'OK  ' : 'FAIL'} ${slug}${detail ? `  ${detail}` : ''}`);
    if (!ok) failures.push(slug);
  };

  try {
    const frameContext = await browser.newContext({ viewport: { width: 2000, height: 1500 }, deviceScaleFactor: FRAME_SCALE });
    const framePage = await frameContext.newPage();
    await framePage.goto(pathToFileURL(path.join(here, 'frame.html')).href);

    for (const shot of stills) {
      try {
        await takeStill(shot, { browser, framePage });
        report(shot.slug, true, await kilobytes(path.join(OUT, `${shot.slug}.png`)));
      } catch (error) {
        report(shot.slug, false, error.message.split('\n')[0]);
        if (process.env.SHOWCASE_DEBUG) console.error(error);
      }
    }

    if (slugs.includes('hero')) {
      try {
        await recordHero();
        report('hero', true, 'recorded, run gif.mjs to convert it');
      } catch (error) {
        report('hero', false, error.message.split('\n')[0]);
      }
    }
  } finally {
    await browser.close();
  }

  if (failures.length > 0) {
    log(`\n${failures.length} failed: ${failures.join(', ')}`);
    process.exit(1);
  }
  log('\nAll shots captured.');
};

main().catch(error => {
  console.error(`FAIL ${error.message}`);
  process.exit(1);
});
