// Self-check for redact.mjs. Run from e2e/showcase/: node --test redact.test.mjs
//
// The token below is a made-up string in the shape of a JWT, not a real one, and the password is a throwaway
// fixture set only for the duration of these tests.

import assert from 'node:assert/strict';
import { test } from 'node:test';
import { PASSWORD_PLACEHOLDER, TOKEN_PLACEHOLDER, firstLine, fullError, redact } from './redact.mjs';

const FAKE_JWT = 'eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJkZW1vIn0.c2lnbmF0dXJlLXZhbHVl';
const FIXTURE_PASSWORD = 'fixture-pass-12345';

test('redact masks a JWT', () => {
  const masked = redact(`login answered 200 with ${FAKE_JWT} in the body`);
  assert.ok(!masked.includes(FAKE_JWT));
  assert.ok(!masked.includes('eyJhbGci'));
  assert.ok(masked.includes(TOKEN_PLACEHOLDER));
});

test('redact masks every token in a string', () => {
  const masked = redact(`Bearer ${FAKE_JWT} and again ${FAKE_JWT}`);
  assert.equal(masked, `Bearer ${TOKEN_PLACEHOLDER} and again ${TOKEN_PLACEHOLDER}`);
});

test('redact masks the demo password when it is set', () => {
  const before = process.env.DEMO_PASSWORD;
  process.env.DEMO_PASSWORD = FIXTURE_PASSWORD;
  try {
    const masked = redact(`password ${FIXTURE_PASSWORD} was sent, then ${FIXTURE_PASSWORD}`);
    assert.ok(!masked.includes(FIXTURE_PASSWORD));
    assert.equal(masked, `password ${PASSWORD_PLACEHOLDER} was sent, then ${PASSWORD_PLACEHOLDER}`);
  } finally {
    if (before === undefined) delete process.env.DEMO_PASSWORD;
    else process.env.DEMO_PASSWORD = before;
  }
});

test('redact leaves text alone when the password is unset and there is no token', () => {
  const before = process.env.DEMO_PASSWORD;
  delete process.env.DEMO_PASSWORD;
  try {
    assert.equal(redact('GET /tasks answered 500'), 'GET /tasks answered 500');
  } finally {
    if (before !== undefined) process.env.DEMO_PASSWORD = before;
  }
});

test('the first line and the full error, stack included, are redacted', () => {
  const error = new Error(`boom ${FAKE_JWT}\nsecond line ${FAKE_JWT}`);
  assert.ok(!firstLine(error).includes(FAKE_JWT));
  assert.ok(!fullError(error).includes(FAKE_JWT));
  assert.ok(fullError(error).includes(TOKEN_PLACEHOLDER));
});
