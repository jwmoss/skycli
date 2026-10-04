import assert from 'node:assert/strict';
import { execFile, spawn } from 'node:child_process';
import { promisify } from 'node:util';
import { mkdtemp, writeFile, rm, readFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import http from 'node:http';
import type { AddressInfo } from 'node:net';

export const token = 'fixture-access-token';
export const newToken = 'fixture-new-access-token';
export const refresh = 'fixture-refresh-token';
export const password = 'fixture-password';
export const root = '/api/frames/123';
const exec = promisify(execFile);
let suiteDir: string;
let binary: string;

export async function setup() {
  suiteDir = await mkdtemp(join(tmpdir(), 'skycli-e2e-'));
  binary = join(suiteDir, 'skycli');
  await exec('go', ['build', '-trimpath', '-o', binary, '.'], { timeout: 120_000 });
}
export async function teardown() { await rm(suiteDir, { recursive: true, force: true }); }

export type Request = { method: string; path: string; query: URLSearchParams; headers: http.IncomingHttpHeaders; body: any; bytes: Buffer };
export type Reply = { status?: number; data?: any; text?: string; headers?: Record<string, string>; hang?: boolean };
export type Result = { code: number | null; stdout: string; stderr: string };
export type Handler = (r: Request) => Reply | undefined | Promise<Reply | undefined>;
export const entity = (id: string, attributes: any, relationships = {}) => ({ id, attributes, relationships });
export const chore = entity('5', { summary: 'Sweep', status: 'complete', start: '2026-09-28', completed_on: '2026-09-28', reward_points: 3, up_for_grabs: false }, { category: { data: { id: '7' } } });
export const reward = entity('9', { name: 'TV ticket', point_value: 10, redeemed_at: '2026-09-28T20:00:00Z' }, { category: { data: { id: '7' } } });
export const list = entity('20', { label: 'Groceries', kind: 'grocery', color: '#2178AF' });
export const recipe = entity('30', { summary: 'Soup', ingredients: ['Beans'], description: 'Dinner' }, { meal_category: { data: { id: '40' } } });
export const sitting = entity('50', { summary: 'Dinner', date: '2026-09-28' }, { meal_category: { data: { id: '40' } }, meal_recipe: { data: { id: '30' } } });
export const calendar = entity('60', { summary: 'Dance', starts_at: '2026-09-29T00:30:00Z', ends_at: '2026-09-29T01:30:00Z', all_day: false }, { category: { data: { id: '7' } } });
export const frame = entity('123', { name: 'Fixture frame', timezone: 'America/New_York', plus: true });

function defaults(r: Request): Reply | undefined {
  if (r.method !== 'GET') return;
  const collections: Record<string, any[]> = {
    '/api/frames': [frame], [`${root}/categories`]: [entity('7', { label: 'Test child', color: '#123456' })],
    [`${root}/chores`]: [chore], [`${root}/rewards`]: [reward], [`${root}/lists`]: [list],
    [`${root}/meals/recipes`]: [recipe], [`${root}/meals/sittings`]: [sitting],
    [`${root}/calendar_events`]: [calendar], [`${root}/devices`]: [entity('2', { name: 'Display' })],
    [`${root}/devices/2/alarms`]: [entity('3', { label: 'Wake' })], [`${root}/routines`]: [entity('80', { title: 'Morning' })],
    [`${root}/albums`]: [entity('70', { name: 'Family' })], [`${root}/messages`]: [entity('90', { caption: 'Photo' })],
    [`${root}/messages/90/comments`]: [entity('92', { text: 'Nice' })], [`${root}/messages/90/all_likes`]: [entity('93', { name: 'Parent' })],
    [`${root}/albums/70/messages`]: [entity('90', { caption: 'Photo' })], [`${root}/auto_creation_intents`]: [entity('100', { status: 'done' })],
    [`${root}/meals/categories`]: [entity('40', { label: 'Dinner' })], [`${root}/task_box/items`]: [entity('21', { title: 'To do' })],
    [`${root}/source_calendars`]: [entity('61', { name: 'Family calendar' })],
    [`${root}/calendar_events/search`]: [calendar], [`${root}/calendar_events/countdowns`]: [entity('62', { summary: 'Birthday' })],
    [`${root}/calendar_events/recent_invited_emails`]: [entity('63', { email: 'guest@example.test' })],
    [`${root}/chores/search`]: [chore], [`${root}/nudges`]: [entity('64', { type: 'task' })],
    '/api/avatars': [entity('65', { name: 'Cat' })], '/api/colors': [entity('66', { color: '#123456' })],
    '/api/month_in_reviews': [entity('67', { month: 'September' })],
  };
  if (Object.hasOwn(collections, r.path)) return { data: { data: collections[r.path], meta: { fixture: r.path } } };
  switch (r.path) {
    case '/api/user': return { data: { data: entity('1', { email: 'parent@example.test' }) } };
    case root: return { data: { data: frame } };
    case `${root}/devices/2`: return { data: { data: entity('2', { name: 'Display' }) } };
    case `${root}/household_config`: return { data: { data: { household_name: 'Fixture household' } } };
    case `${root}/event_notification_settings`: return { data: { data: { event_notifications: true } } };
    case `${root}/task_notification_settings`: return { data: { data: { task_notifications: false } } };
    case '/api/reminder_profile': return { data: { data: { reminder_time: '08:00' } } };
    case `${root}/reward_points`: return { data: [{ category_id: 7, current_point_balance: 13, lifetime_points_earned: 25 }] };
    case `${root}/lists/20`: return { data: { data: list, included: [entity('21', { label: 'Milk', status: 'completed', position: 0 }, {}), entity('22', { label: 'Eggs', status: 'pending', position: 1 }, {})] } };
    case `${root}/meals/recipes/30`: return { data: { data: recipe } };
    case `${root}/messages/90`: return { data: { data: entity('90', { caption: 'Photo' }) } };
    case `${root}/albums/70/messages/all_ids`: return { data: { data: [90, 91] } };
    case '/api/plus_access': return { data: { data: { bundle_entitlement: { available: true }, self_serve_trial_eligibility: { assistant: true },
      subscriptions: [{ attributes: { plus_type: 'cal_plus', status: 'active' } }, { attributes: { plus_type: 'other', status: 'expired' } }] } } };
  }
}

export async function fixture(body: (f: {
  run: (args: string[], opts?: { input?: string; env?: Record<string, string>; onStdout?: (data: string, child: ReturnType<typeof spawn>) => void }) => Promise<Result>;
  requests: Request[]; config: string; home: string; baseURL: string; save: (override: Record<string, any>) => Promise<void>;
}) => Promise<void>, handler?: Handler, override: Record<string, any> = {}) {
  const home = await mkdtemp(join(suiteDir, 'home-'));
  const config = join(home, 'config.json');
  const requests: Request[] = [], failures: Error[] = [];
  const server = http.createServer(async (req, res) => {
    try {
      const chunks = [];
      for await (const chunk of req) chunks.push(chunk);
      const bytes = Buffer.concat(chunks);
      const url = new URL(req.url!, 'http://fixture.test');
      const type = req.headers['content-type'] ?? '';
      const request: Request = { method: req.method!, path: url.pathname, query: url.searchParams, headers: req.headers, bytes,
        body: !bytes.length ? null : type.includes('form-urlencoded') ? Object.fromEntries(new URLSearchParams(bytes.toString())) : type.includes('json') ? JSON.parse(bytes.toString()) : bytes };
      requests.push(request);
      if (request.path.startsWith('/api/')) {
        assert.match(req.headers.authorization ?? '', /^(Bearer|Basic) (fixture-|override-)/);
        assert.equal(req.headers['skylight-api-version'], '2026-04-15');
      }
      const reply = await handler?.(request) ?? defaults(request);
      assert.ok(reply, `Unexpected request ${request.method} ${req.url}`);
      if (reply.hang) return;
      res.writeHead(reply.status ?? 200, { 'content-type': reply.text === undefined ? 'application/json' : 'text/plain', ...reply.headers });
      res.end(reply.text ?? JSON.stringify(reply.data));
    } catch (error) { failures.push(error as Error); res.writeHead(500); res.end('{"error":"fixture contract failed"}'); }
  });
  await new Promise<void>(resolve => server.listen(0, '127.0.0.1', resolve));
  const baseURL = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
  const cfg = { base_url: baseURL, secrets_backend: 'config', access_token: token, default_frame_id: 123, ...override };
  const save = async (extra: Record<string, any>) => { await writeFile(config, JSON.stringify({ ...cfg, ...extra }), { mode: 0o600 }); };
  await save({});
  try {
    const run = (args: string[], opts: { input?: string; env?: Record<string, string>; onStdout?: (data: string, child: ReturnType<typeof spawn>) => void } = {}) => new Promise<Result>((resolve, reject) => {
      const child = spawn(binary, ['--config', config, ...args], {
        env: { PATH: process.env.PATH, HOME: home, XDG_CONFIG_HOME: join(home, '.config'), TERM: 'dumb', TZ: 'UTC', ...opts.env },
        stdio: ['pipe', 'pipe', 'pipe'],
      });
      let stdout = '', stderr = '';
      const timer = setTimeout(() => { child.kill('SIGKILL'); reject(new Error(`CLI timeout ${args.join(' ')}`)); }, 10_000);
      child.stdout.on('data', data => { stdout += data; opts.onStdout?.(data.toString(), child); });
      child.stderr.on('data', data => { stderr += data; });
      child.on('error', error => { clearTimeout(timer); reject(error); });
      child.on('close', code => { clearTimeout(timer); resolve({ code, stdout, stderr }); });
      child.stdin.end(opts.input ?? '');
    });
    await body({ run, requests, config, home, baseURL, save });
    assert.deepEqual(failures, [], 'fixture protocol assertion failed');
  } finally {
    server.closeAllConnections();
    await new Promise<void>(resolve => server.close(() => resolve()));
    await rm(home, { recursive: true, force: true });
  }
}

export function ok(r: Result) {
  assert.equal(r.code, 0, r.stderr || r.stdout);
  for (const secret of [token, newToken, refresh, password, 'fixture-rotated-refresh', 'fixture-csrf']) {
    assert.ok(!(r.stdout + r.stderr).includes(secret), `output contains ${secret}`);
  }
  return r.stdout;
}
export function json(r: Result) { return JSON.parse(ok(r)); }
export function bad(r: Result, code: number, message: RegExp) {
  assert.equal(r.code, code, r.stdout + r.stderr);
  assert.match(r.stdout + r.stderr, message);
  for (const secret of [token, newToken, refresh, password]) assert.ok(!(r.stdout + r.stderr).includes(secret));
}
export async function configBytes(path: string) { return readFile(path, 'utf8'); }
