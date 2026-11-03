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
// The run checks that the demo users log in and own the seeded tasks, purges
// both titles before it starts and again on SIGINT, SIGTERM or SIGHUP, so an
// interrupted run leaves nothing behind. The forbidden shot is rejected
// by the API and writes nothing, and it re-reads its target afterwards and
// restores it, loudly, if it changed.
//
// A cleanup failure never hides the error of the shot and is reported on its
// own line, and any failure, capture or cleanup, makes the exit code non-zero.
//
// A bearer token never reaches an image. The token is pasted into a masked
// field, every `pre` block that would print it is rewritten to a truncated
// form before it is painted, and the page is checked for a full token after
// every step that renders one.
//
// Usage, from e2e/showcase/ (see README.md for the variables):
//   node capture.mjs                  every still
//   SLUGS=overview,forbidden node capture.mjs
//   node capture.mjs hero             the video, for gif.mjs (a slug may also be an argument)

import { AsyncLocalStorage } from 'node:async_hooks';
import { copyFile, mkdir, rm, stat, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { chromium } from 'playwright';
import { TOKEN_PATTERN, firstLine, fullError, redact } from './redact.mjs';

const here = path.dirname(fileURLToPath(import.meta.url));
const REPO = path.resolve(here, '..', '..');

const CREATOR_EMAIL = process.env.DEMO_CREATOR_EMAIL ?? '';
const ASSIGNEE_EMAIL = process.env.DEMO_ASSIGNEE_EMAIL ?? '';
const PASSWORD = process.env.DEMO_PASSWORD ?? '';
const RAW_BASE_URL = process.env.BASE_URL ?? 'http://localhost:8080';
/** How long one API call may take, in milliseconds. Not a credential, so it has a default. */
const FETCH_TIMEOUT_MS = Number(process.env.SHOWCASE_FETCH_TIMEOUT_MS ?? 15_000);

// Set by requireEnvironment, which parses BASE_URL so that an invalid value ends in a FAIL line and not a stack trace.
/** The API origin plus any path prefix, without a trailing slash. */
let BASE = '';
let DOCS = '';
/** The address the browser frame shows, derived from BASE_URL: host, port, path prefix and the documentation path. */
let ADDRESS = '';
const FRAME_FILE = path.join(here, 'frame.html');

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
// Everything the script prints goes through redact, so a token or the password cannot reach a log.
const log = (...parts) => console.log(...parts.map(redact));
const logError = (...parts) => console.error(...parts.map(redact));

// -- interruption ---------------------------------------------------------------------------------

const SIGNAL_CODES = { SIGINT: 130, SIGTERM: 143, SIGHUP: 129 };
/** The name of the first signal received, or null. Set by the handler and read by main, which stops on it. */
let interruptedBy = null;
const interrupt = new AbortController();
const isInterrupted = () => interruptedBy !== null;

/** Marks the code that undoes the run's writes. Those calls are not cut short by an interrupt, they finish it. */
const cleaning = new AsyncLocalStorage();

/** Runs cleanup steps one at a time, so a shot's cleanup and the interrupt's cleanup never run together. */
let cleanupChain = Promise.resolve();
const exclusive = step => {
  const run = cleanupChain.then(() => cleaning.run(true, step));
  cleanupChain = run.then(
    () => {},
    () => {}
  );
  return run;
};

// -- the API, used for tokens, for the ids a shot needs and for cleaning up ----------------------

const api = async (route, { method = 'GET', token, body } = {}) => {
  // Every call times out, and a call that is not part of a cleanup also stops on an interrupt.
  const signals = [AbortSignal.timeout(FETCH_TIMEOUT_MS)];
  if (!cleaning.getStore()) signals.push(interrupt.signal);

  let response;
  let text;
  try {
    response = await fetch(`${BASE}${route}`, {
      method,
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: AbortSignal.any(signals),
    });
    text = await response.text();
  } catch (error) {
    if (error?.name === 'TimeoutError') throw new Error(`${method} ${route} timed out after ${FETCH_TIMEOUT_MS} ms`);
    if (isInterrupted() && error?.name === 'AbortError') throw new Error(`${method} ${route} stopped: the run was interrupted`);
    throw error;
  }
  // A body that is not JSON (a proxy page, an empty error) leaves json null, and the caller checks the status.
  let json = null;
  if (text) {
    try {
      json = JSON.parse(text);
    } catch {
      json = null;
    }
  }
  return { status: response.status, json };
};

const login = async email => {
  const { status, json } = await api('/auth/login', { method: 'POST', body: { email, password: PASSWORD } });
  if (status !== 200) throw new Error(`login as ${email} answered ${status}: is the demo seeded with this password?`);
  if (typeof json?.token !== 'string' || json.token === '') throw new Error(`login as ${email} answered 200 without a token`);
  return json.token;
};

/** The tasks the token can see. Fails on any status but 200 and on a body that is not a list. */
const listTasks = async token => {
  const { status, json } = await api('/tasks', { token });
  if (status !== 200) throw new Error(`GET /tasks answered ${status}`);
  if (!Array.isArray(json)) throw new Error('GET /tasks did not answer a list');
  return json;
};

/**
 * Deletes every task of the token's user with this title, which is how a flow undoes its own write. Every
 * DELETE must answer 2xx or 404 (a task that is already gone is the goal), and the list is read again
 * afterwards: a title that is still there fails loudly.
 */
const deleteTasksTitled = async (token, title) => {
  const problems = [];
  for (const task of (await listTasks(token)).filter(item => item.title === title)) {
    const { status } = await api(`/tasks/${task.id}`, { method: 'DELETE', token });
    if (status === 404) continue;
    if (status < 200 || status >= 300) problems.push(`DELETE /tasks/${task.id} answered ${status}`);
  }

  const left = (await listTasks(token)).filter(item => item.title === title);
  if (left.length > 0) problems.push(`${left.length} task(s) titled "${title}" remain`);
  if (problems.length > 0) throw new Error(`cleanup of "${title}" failed: ${problems.join('; ')}`);
};

/** Runs a step and returns its error instead of throwing it, so a cleanup can never mask the error before it. */
const attempt = async step => {
  try {
    await step();
    return null;
  } catch (error) {
    return error;
  }
};

/** True once the demo accounts were checked. Nothing is purged or created before that. */
let demoVerified = false;

/**
 * Checks that the two demo users log in and that the creator owns the seeded demo tasks, so the purge only ever
 * runs against the seeded demo account and never against a user's own tasks.
 */
const verifyDemoAccounts = async () => {
  const seedHint = 'run the seed first: APP_ENV=development DEMO_PASSWORD=<password> go run ./cmd seed';
  let creatorToken;
  try {
    creatorToken = await login(CREATOR_EMAIL);
    await login(ASSIGNEE_EMAIL);
  } catch (error) {
    throw new Error(`the demo users are not usable (${firstLine(error)}): ${seedHint}`);
  }
  const owned = await listTasks(creatorToken);
  if (!owned.some(task => task.title === KNOWN_HIGH_TITLE)) {
    throw new Error(`${CREATOR_EMAIL} does not own the seeded demo tasks (no task titled "${KNOWN_HIGH_TITLE}"), so nothing was purged or created: ${seedHint}`);
  }
  demoVerified = true;
};

/** Deletes the tasks both flows create. Only ever called once the demo accounts are verified. */
const purgeCreatedTasks = async () => {
  if (!demoVerified) throw new Error('refusing to purge before the demo accounts were verified');
  const token = await login(CREATOR_EMAIL);
  for (const { title } of [STILL_TASK, HERO_TASK]) await deleteTasksTitled(token, title);
};

/** The fields of a task the forbidden shot records and compares. Title is not editable, so it is only compared. */
const WATCHED_FIELDS = ['title', 'description', 'due_date', 'priority', 'status'];

/**
 * Re-reads the task a shot recorded and, when a watched field changed, puts it back and fails loudly. The
 * creator owns the demo tasks, so the creator restores them.
 */
const restoreTask = async original => {
  if (!original) return;
  const token = await login(CREATOR_EMAIL);
  const read = async () => {
    const { status, json } = await api(`/tasks/${original.id}`, { token });
    if (status !== 200 || !json) throw new Error(`could not re-read task ${original.id}: status ${status}`);
    return WATCHED_FIELDS.filter(key => json[key] !== original.fields[key]);
  };

  const changed = await read();
  if (changed.length === 0) return;

  const { title: _title, ...editable } = original.fields;
  const { status } = await api(`/tasks/${original.id}`, { method: 'PUT', token, body: editable });
  const stillChanged = status === 200 ? await read() : changed;
  const outcome = stillChanged.length === 0 ? 'restored' : `NOT restored (${stillChanged.join(', ')} still differ, PUT answered ${status})`;
  throw new Error(`the forbidden shot changed ${changed.join(', ')} of task ${original.id}: ${outcome}`);
};

// -- what an interrupted run must still undo ------------------------------------------------------

/** Browsers that are open, and the forbidden-shot states that still have a task to check. */
const openBrowsers = new Set();
const activeRestores = new Set();

/** Every cleanup a run owes, as a list of errors. Used when the run is interrupted. */
const runAllCleanups = async () => {
  const errors = [];
  for (const browser of [...openBrowsers]) {
    const error = await attempt(() => browser.close());
    if (error) errors.push(error);
  }
  if (demoVerified) {
    const purge = await attempt(purgeCreatedTasks);
    if (purge) errors.push(purge);
  }
  for (const state of [...activeRestores]) {
    const error = await attempt(() => restoreTask(state.original));
    if (error) errors.push(error);
  }
  return errors;
};

/** The one cleanup of an interrupted run. Every caller gets the same promise, so it runs once. */
let sharedCleanup = null;
const cleanupOnce = () => {
  sharedCleanup ??= exclusive(runAllCleanups);
  return sharedCleanup;
};

/**
 * The signal handler only records the interrupt, stops the calls that are in flight and starts the shared cleanup.
 * main notices the flag, launches nothing more, awaits the cleanup and exits. A second signal exits at once with
 * the code of the first.
 */
const installSignalHandlers = () => {
  for (const [signal, code] of Object.entries(SIGNAL_CODES)) {
    process.on(signal, () => {
      if (isInterrupted()) process.exit(SIGNAL_CODES[interruptedBy] ?? code);
      interruptedBy = signal;
      logError(`\n${signal}: undoing what this run wrote before exiting`);
      interrupt.abort();
      cleanupOnce();
    });
  }
};

/** Awaits the shared cleanup, reports what it could not undo and exits with the code of the signal. */
const exitInterrupted = async () => {
  for (const error of await cleanupOnce()) logError(`FAIL cleanup  ${firstLine(error)}`);
  process.exit(SIGNAL_CODES[interruptedBy]);
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

/**
 * Fails the shot when a token is on screen or something is still loading. Three places are checked: the visible
 * text, the contents of every `pre` block, and the value of every `input` and `textarea`.
 *
 * The Authorize field legitimately holds `Bearer <token>` as its raw value while it is rendered as dots through
 * `-webkit-text-security: disc`. What reaches the image is the rendering, so a masked field is checked by its
 * rendering: it passes only when the computed style is `disc` (or the field is a password input), and the raw
 * value is not compared. Any other field has its value compared with the token pattern.
 */
const assertClean = async page => {
  const leaks = await page.evaluate(pattern => {
    const token = new RegExp(pattern);
    const found = [];
    if (token.test(document.body.innerText)) found.push('the visible text');
    for (const block of document.querySelectorAll('pre')) {
      if (token.test(block.textContent)) found.push('a pre block');
    }
    for (const field of document.querySelectorAll('input, textarea')) {
      const masked = field.type === 'password' || getComputedStyle(field).webkitTextSecurity === 'disc';
      if (!masked && token.test(field.value)) found.push(`the value of a visible ${field.tagName.toLowerCase()}`);
    }
    return found;
  }, TOKEN_PATTERN.source);
  if (leaks.length > 0) throw new Error(`a full bearer token is visible in ${[...new Set(leaks)].join(', ')}`);

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
 * runs in every case, success or failure, and undoes whatever the shot wrote. `run` and `cleanup` share a
 * `state` object, which is how a shot hands what it recorded to its cleanup.
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
    run: async (page, state) => {
      const assigneeToken = await login(ASSIGNEE_EMAIL);
      // The API does not say who created a task, so pick the one the assignee holds that the creator owns.
      const target = (await listTasks(assigneeToken)).find(task => task.title === 'Promote a task from pending to in_progress');
      if (!target) throw new Error('the assignee has no seeded task to try to edit: is the demo seeded?');
      // Recorded before the request, so the cleanup can tell whether the rejected edit changed anything.
      state.original = { id: target.id, fields: Object.fromEntries(WATCHED_FIELDS.map(key => [key, target[key]])) };
      activeRestores.add(state);

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
    cleanup: async state => {
      try {
        await restoreTask(state.original);
      } finally {
        activeRestores.delete(state);
      }
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
    address: ADDRESS,
  });
  return framePage.locator('#stage').screenshot({ type: 'png' });
};

const kilobytes = async file => `${Math.round((await stat(file)).size / 1024)} KB`;

/**
 * Takes one still. A capture failure is thrown, with the cleanup error (if the cleanup failed too) attached as
 * `cleanupError`. When the still was written and only the cleanup failed, the cleanup error is returned and not
 * thrown, so the caller can report the shot as captured and the cleanup as failed.
 */
const takeStill = async (shot, { browser, framePage }) => {
  const state = {};
  let captureError = null;
  let context;
  try {
    context = await openContext(browser);
    await prepare(context);
    const page = await context.newPage();
    await shot.run(page, state);
    const raw = await page.screenshot({ type: 'png' });
    const framed = await frameShot(framePage, raw, VIEWPORT.width);

    await writeFile(path.join(WORK, `${shot.slug}.raw.png`), raw);
    await writeFile(path.join(OUT, `${shot.slug}.png`), framed);
  } catch (error) {
    captureError = error;
  }

  await context?.close().catch(() => {});
  const cleanupError = await attempt(() => exclusive(() => shot.cleanup?.(state)));

  if (captureError) {
    if (cleanupError) captureError.cleanupError = cleanupError;
    throw captureError;
  }
  return { cleanupError };
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

/**
 * The script owns SIGINT, SIGTERM and SIGHUP. Playwright must not install its own handlers for them, or it would
 * close the browser and exit while the run still has tasks to undo.
 */
const SIGNAL_OPTIONS = { handleSIGINT: false, handleSIGTERM: false, handleSIGHUP: false };
const LAUNCH_OPTIONS = SIGNAL_OPTIONS;
const LAUNCH_OPTIONS_HERO = { channel: 'chromium', ...SIGNAL_OPTIONS };

/**
 * Records the hero. The browser is closed in every case, and the task the flow creates is deleted afterwards.
 * Errors follow the same contract as takeStill: a failed recording is thrown with `cleanupError` attached when
 * the cleanup failed too, and a cleanup failure after a good recording is returned.
 */
const recordHero = async () => {
  const dir = path.join(WORK, 'video');
  await rm(dir, { recursive: true, force: true });
  // A recording and its cut points from an earlier run must never be mistaken for this one's, so a failed
  // recording leaves nothing for gif.mjs to publish.
  await rm(path.join(WORK, 'hero.webm'), { force: true });
  await rm(path.join(WORK, 'hero.json'), { force: true });

  let captureError = null;
  const browser = await chromium.launch(LAUNCH_OPTIONS_HERO);
  openBrowsers.add(browser);
  try {
    await recordHeroFlow(browser, dir);
  } catch (error) {
    captureError = error;
  }

  await browser.close().catch(() => {});
  openBrowsers.delete(browser);
  // A fresh login keeps this working when the flow failed before it had a token.
  const cleanupError = await attempt(() => exclusive(async () => deleteTasksTitled(await login(CREATOR_EMAIL), HERO_TASK.title)));

  if (captureError) {
    if (cleanupError) captureError.cleanupError = cleanupError;
    throw captureError;
  }
  return { cleanupError };
};

const recordHeroFlow = async (browser, dir) => {
  const context = await openContext(browser, { deviceScaleFactor: 1, recordVideo: { dir, size: VIEWPORT } });
  await prepare(context);
  await context.addInitScript(CURSOR_SCRIPT);
  // The login body on screen shows dots for the password. The request that leaves the browser carries the real one,
  // so the password never appears in the recording and the login still works.
  await context.route('**/auth/login', route => {
    const body = JSON.parse(route.request().postData() ?? '{}');
    return route.continue({ postData: JSON.stringify({ ...body, password: PASSWORD }) });
  });

  // The video clock starts when the page is created, so the timestamp is taken just before it. The cut points
  // written to hero.json are measured from here and match the video.
  const recordingStarted = Date.now();
  const page = await context.newPage();
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
    await assertClean(page);
    await beat(page, 1800);
    await assertClean(page);

    // Authorize with Bearer and the token. The field shows dots.
    await scrollTo(page, page.locator('.scheme-container'), 0);
    await press(page, page.locator('button.authorize').first());
    await beat(page, 500);
    const modal = page.locator('.auth-container');
    await modal.locator('input[type="text"]').fill(`Bearer ${token}`);
    await assertClean(page);
    await beat(page, 900);
    await press(page, modal.locator('button.authorize'));
    await modal.locator('button', { hasText: 'Logout' }).waitFor();
    await assertClean(page);
    await beat(page, 900);
    await press(page, page.locator('.modal-ux button.btn-done'));
    await beat(page, 600);
    await assertClean(page);

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
    await assertClean(page);
    await scrollTo(page, create.locator('.responses-wrapper'), 160);
    await beat(page, 2200);
    await assertClean(page);

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
    await assertClean(page);

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

  if (!Number.isInteger(FETCH_TIMEOUT_MS) || FETCH_TIMEOUT_MS <= 0) {
    throw new Error('SHOWCASE_FETCH_TIMEOUT_MS must be a positive whole number of milliseconds');
  }

  let url;
  try {
    url = new URL(RAW_BASE_URL);
  } catch {
    throw new Error('BASE_URL is not a valid URL. Use the form http://localhost:8080, with an optional path prefix.');
  }
  if (url.protocol !== 'http:' && url.protocol !== 'https:') throw new Error('BASE_URL must start with http:// or https://');
  if (url.username !== '' || url.password !== '') throw new Error('BASE_URL must not carry credentials');
  if (url.search !== '' || url.hash !== '') throw new Error('BASE_URL must not carry a query or a fragment');

  // The path prefix, when the API is served under one, is part of every URL the script builds.
  const prefix = url.pathname.replace(/\/+$/, '');
  BASE = `${url.origin}${prefix}`;
  DOCS = `${BASE}/documentation/index.html`;
  ADDRESS = `${url.host}${prefix}/documentation/`;
};

const main = async () => {
  requireEnvironment();

  const slugs = selectSlugs();
  const stills = SHOTS.filter(shot => slugs.includes(shot.slug));

  await mkdir(WORK, { recursive: true });
  await mkdir(OUT, { recursive: true });
  await stat(FRAME_FILE).catch(() => {
    throw new Error(`the browser frame ${FRAME_FILE} is missing: restore frame.html next to capture.mjs`);
  });

  installSignalHandlers();
  // Nothing is purged or created until the demo users log in and the creator owns the seeded tasks.
  await verifyDemoAccounts();
  // An earlier run that was killed may have left the tasks the flows create. Remove them before anything else.
  await exclusive(purgeCreatedTasks);
  if (isInterrupted()) return exitInterrupted();

  const browser = await chromium.launch(LAUNCH_OPTIONS);
  openBrowsers.add(browser);
  const failures = [];
  const cleanupFailures = [];
  const report = (slug, ok, detail) => {
    log(`${ok ? 'OK  ' : 'FAIL'} ${slug}${detail ? `  ${detail}` : ''}`);
    if (!ok) failures.push(slug);
  };
  const reportCleanup = (slug, error) => {
    log(`FAIL cleanup ${slug}  ${firstLine(error)}`);
    cleanupFailures.push(slug);
    if (process.env.SHOWCASE_DEBUG) logError(fullError(error));
  };

  try {
    const frameContext = await browser.newContext({ viewport: { width: 2000, height: 1500 }, deviceScaleFactor: FRAME_SCALE });
    const framePage = await frameContext.newPage();
    await framePage.goto(pathToFileURL(FRAME_FILE).href);

    for (const shot of stills) {
      // An interrupt stops the run: no new shot is launched, and the shared cleanup finishes below.
      if (isInterrupted()) break;
      try {
        const { cleanupError } = await takeStill(shot, { browser, framePage });
        report(shot.slug, true, await kilobytes(path.join(OUT, `${shot.slug}.png`)));
        if (cleanupError) reportCleanup(shot.slug, cleanupError);
      } catch (error) {
        report(shot.slug, false, firstLine(error));
        if (process.env.SHOWCASE_DEBUG) logError(fullError(error));
        if (error.cleanupError) reportCleanup(shot.slug, error.cleanupError);
      }
    }

    if (slugs.includes('hero') && !isInterrupted()) {
      try {
        const { cleanupError } = await recordHero();
        report('hero', true, 'recorded, run gif.mjs to convert it');
        if (cleanupError) reportCleanup('hero', cleanupError);
      } catch (error) {
        report('hero', false, firstLine(error));
        if (process.env.SHOWCASE_DEBUG) logError(fullError(error));
        if (error.cleanupError) reportCleanup('hero', error.cleanupError);
      }
    }
  } finally {
    await browser.close().catch(() => {});
    openBrowsers.delete(browser);
  }

  if (isInterrupted()) return exitInterrupted();

  if (failures.length > 0 || cleanupFailures.length > 0) {
    if (failures.length > 0) log(`\n${failures.length} capture failure(s): ${failures.join(', ')}`);
    if (cleanupFailures.length > 0) log(`${cleanupFailures.length} cleanup failure(s): ${cleanupFailures.join(', ')}. The demo data may need a reseed.`);
    process.exit(1);
  }
  log('\nAll shots captured.');
};

main().catch(async error => {
  logError(`FAIL ${firstLine(error)}`);
  if (process.env.SHOWCASE_DEBUG) logError(fullError(error));
  // An interrupt that ended main with an error still owes its cleanup and its exit code.
  if (isInterrupted()) await exitInterrupted();
  process.exit(1);
});
