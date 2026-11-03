// Keeps a bearer token and the demo password out of anything the capture scripts print.

/** A full JWT: header, payload and signature. */
export const TOKEN_PATTERN = /eyJ[\w-]{6,}\.[\w-]+\.[\w-]+/;

const EVERY_TOKEN = new RegExp(TOKEN_PATTERN.source, 'g');

export const TOKEN_PLACEHOLDER = '[redacted token]';
export const PASSWORD_PLACEHOLDER = '[redacted password]';

/**
 * Returns the text with every JWT and every occurrence of the demo password replaced by a placeholder. The
 * password is read from DEMO_PASSWORD on each call, and is only masked when it is set.
 */
export const redact = value => {
  let text = String(value);
  text = text.replace(EVERY_TOKEN, TOKEN_PLACEHOLDER);
  const password = process.env.DEMO_PASSWORD ?? '';
  if (password !== '') text = text.split(password).join(PASSWORD_PLACEHOLDER);
  return text;
};

/** The first line of an error message, redacted. */
export const firstLine = error => redact(String(error?.message ?? error).split('\n')[0]);

/** The whole error with its stack, redacted. Used by SHOWCASE_DEBUG. */
export const fullError = error => redact(error?.stack ?? error?.message ?? error);
