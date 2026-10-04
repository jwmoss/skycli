import { test, beforeAll, afterAll } from 'e2e';
import assert from 'node:assert/strict';
import { writeFile, readFile, stat, readdir } from 'node:fs/promises';
import { join } from 'node:path';
import http from 'node:http';
import type { AddressInfo } from 'node:net';
import { fixture, json, ok, bad, configBytes, root, token, newToken, refresh, password,
  entity, chore, reward, frame, list, recipe, sitting, calendar, setup, teardown } from './support.js';

beforeAll(setup);
afterAll(teardown);

const reads: [string[], string, Record<string, string>, string][] = [
  [['frames', 'list'], '/api/frames', {}, 'Fixture frame'],
  [['frames', 'show', '--id', '123'], root, {}, 'Fixture frame'],
  [['frames', 'devices'], `${root}/devices`, {}, 'Display'],
  [['frames', 'device', '--device-id', '2'], `${root}/devices/2`, {}, 'Display'],
  [['frames', 'alarms', '--device-id', '2'], `${root}/devices/2/alarms`, {}, 'Wake'],
  [['frames', 'household-config'], `${root}/household_config`, {}, 'Fixture household'],
  [['frames', 'notifications', '--type', 'event'], `${root}/event_notification_settings`, {}, 'event_notifications'],
  [['frames', 'notifications', '--type', 'task'], `${root}/task_notification_settings`, {}, 'task_notifications'],
  [['frames', 'month-reviews'], '/api/month_in_reviews', {}, 'September'],
  [['frames', 'reminder-profile'], '/api/reminder_profile', {}, '08:00'],
  [['frames', 'avatars'], '/api/avatars', {}, 'Cat'],
  [['frames', 'colors'], '/api/colors', {}, '#123456'],
  [['frames', 'nudges', '--after', '2026-09-28T00:00:00Z', '--before', '2026-10-05T00:00:00Z'], `${root}/nudges`, { after: '2026-09-28T00:00:00Z', before: '2026-10-05T00:00:00Z' }, 'task'],
  [['categories'], `${root}/categories`, {}, 'Test child'],
  [['chores', 'list', '--date', '2026-09-28', '--assignee-id', '7', '--status', 'complete'], `${root}/chores`, { date: '2026-09-28', after: '2026-09-28', before: '2026-09-28', assignee_id: '7', status: 'complete', include_late: 'true', include_up_for_grabs: 'true' }, 'Sweep'],
  [['chores', 'search', '--query', 'Sweep', '--limit', '8', '--ended-lookback-days', '14'], `${root}/chores/search`, { search_query: 'Sweep', limit: '8', ended_chore_lookback_days: '14', include_up_for_grabs: 'true' }, 'Sweep'],
  [['rewards', 'list'], `${root}/rewards`, {}, 'TV ticket'],
  [['rewards', 'points'], `${root}/reward_points`, {}, '13'],
  [['calendar', 'list', '--start-date', '2026-09-28', '--end-date', '2026-10-04'], `${root}/calendar_events`, { date_min: '2026-09-28', date_max: '2026-10-04' }, 'Dance'],
  [['calendar', 'sources'], `${root}/source_calendars`, {}, 'Family calendar'],
  [['calendar', 'search', '--query', 'Dance'], `${root}/calendar_events/search`, { search_query: 'Dance', timezone: 'UTC', include: 'categories' }, 'Dance'],
  [['calendar', 'countdowns', '--start-date', '2026-09-28', '--end-date', '2026-10-04'], `${root}/calendar_events/countdowns`, { date_min: '2026-09-28', date_max: '2026-10-04', timezone: 'UTC', include: 'categories' }, 'Birthday'],
  [['calendar', 'recent-invites'], `${root}/calendar_events/recent_invited_emails`, {}, 'guest@example.test'],
  [['lists', 'list'], `${root}/lists`, {}, 'Groceries'],
  [['lists', 'show', '--list-id', '20'], `${root}/lists/20`, {}, 'Eggs'],
  [['lists', 'task-box-items'], `${root}/task_box/items`, {}, 'To do'],
  [['grocery', 'list'], `${root}/lists`, {}, 'Groceries'],
  [['grocery', 'show', '--list-id', '20'], `${root}/lists/20`, {}, 'Milk'],
  [['meals', 'categories'], `${root}/meals/categories`, {}, 'Dinner'],
  [['meals', 'recipes'], `${root}/meals/recipes`, {}, 'Soup'],
  [['meals', 'recipe-info', '--recipe-id', '30'], `${root}/meals/recipes/30`, {}, 'Soup'],
  [['meals', 'sittings', '--date-min', '2026-09-28', '--date-max', '2026-10-04'], `${root}/meals/sittings`, { date_min: '2026-09-28', date_max: '2026-10-04' }, 'Dinner'],
  [['photos', 'list', '--page-token', 'cursor +/2'], `${root}/messages`, { page_token: 'cursor +/2' }, 'Photo'],
  [['photos', 'show', '--message-id', '90'], `${root}/messages/90`, {}, 'Photo'],
  [['photos', 'likes', '--message-id', '90'], `${root}/messages/90/all_likes`, {}, 'Parent'],
  [['photos', 'comments', '--message-id', '90', '--page', '2'], `${root}/messages/90/comments`, { page: '2' }, 'Nice'],
  [['albums', 'list'], `${root}/albums`, {}, 'Family'],
  [['albums', 'messages', '--album-id', '70', '--page', '3'], `${root}/albums/70/messages`, { page: '3' }, 'Photo'],
  [['albums', 'message-ids', '--album-id', '70'], `${root}/albums/70/messages/all_ids`, {}, '91'],
  [['routines', 'list'], `${root}/routines`, {}, 'Morning'],
  [['sidekick', 'history'], `${root}/auto_creation_intents`, {}, 'done'],
];

for (const [args, path, query, value] of reads) {
  test(`${args.join(' ')} delivers data and request filters under readonly`, async () => {
    await fixture(async ({ run, requests }) => {
      const result = await run([...args, '--readonly', '--json', '--frame', '123']);
      assert.ok(JSON.stringify(json(result)).includes(value), result.stdout);
      assert.equal(requests.length, 1);
      assert.equal(requests[0].method, 'GET'); assert.equal(requests[0].path, path);
      for (const [key, expected] of Object.entries(query)) assert.equal(requests[0].query.get(key), expected);
    });
  });
}

type Write = { args: string[]; method: string; path: string; payload?: any; response?: any; query?: Record<string, string> };
const writes: Write[] = [
  { args: ['chores', 'create', '--category', '7', '--summary', 'Sweep', '--start', '2026-09-28', '--points', '3', '--recurrence', 'weekly:MO,FR'], method: 'POST', path: `${root}/chores`, payload: { summary: 'Sweep', category_id: 7, start: '2026-09-28', reward_points: 3, recurrence_set: ['RRULE:FREQ=WEEKLY;BYDAY=MO,FR'], up_for_grabs: false }, response: { data: chore } },
  { args: ['chores', 'create-up-for-grabs', '--summary', 'Bonus', '--points', '4'], method: 'POST', path: `${root}/chores/create_multiple`, payload: { summary: 'Bonus', up_for_grabs: true, reward_points: 4 }, response: { data: [entity('6', { summary: 'Bonus', up_for_grabs: true })] } },
  { args: ['chores', 'update', '--id', '5', '--summary', 'Mop', '--points', '0'], method: 'PUT', path: `${root}/chores/5`, payload: { summary: 'Mop', reward_points: 0 }, response: { data: entity('5', { summary: 'Mop', reward_points: 0 }) } },
  { args: ['chores', 'claim', '--id', '5-2026-09-28', '--category', '7'], method: 'PUT', path: `${root}/chores/5`, payload: { category_id: 7 }, response: { data: chore } },
  { args: ['chores', 'complete', '--id', '5-2026-09-28'], method: 'PUT', path: `${root}/chores/5/completions`, payload: { status: 'complete', instance_date: '2026-09-28' }, response: { data: chore } },
  { args: ['chores', 'skip', '--id', '5'], method: 'PUT', path: `${root}/chores/5/completions`, payload: { status: 'skipped' }, response: { data: entity('5', { summary: 'Sweep', status: 'skipped' }) } },
  { args: ['chores', 'delete', '--id', '5', '--apply-to', 'this_only'], method: 'DELETE', path: `${root}/chores/5`, query: { apply_to: 'this_only' } },
  { args: ['rewards', 'create', '--name', 'TV ticket', '--points', '10', '--categories', '7,8', '--respawn'], method: 'POST', path: `${root}/rewards`, payload: { name: 'TV ticket', point_value: 10, category_ids: [7, 8], respawn_on_redemption: true }, response: { data: [reward, entity('10', { name: 'TV ticket', point_value: 10 })] } },
  { args: ['rewards', 'update', '--id', '9', '--points', '0', '--no-respawn'], method: 'PATCH', path: `${root}/rewards/9`, payload: { point_value: 0, respawn_on_redemption: false }, response: { data: entity('9', { name: 'TV ticket', point_value: 0 }) } },
  { args: ['rewards', 'redeem', '--id', '9'], method: 'POST', path: `${root}/rewards/9/redeem` },
  { args: ['rewards', 'unredeem', '--id', '9'], method: 'POST', path: `${root}/rewards/9/unredeem` },
  { args: ['rewards', 'delete', '--id', '9'], method: 'DELETE', path: `${root}/rewards/9` },
  { args: ['calendar', 'create', '--title', 'Dance', '--start-at', '2026-09-28T18:00:00-04:00', '--end-at', '2026-09-28T19:00:00-04:00', '--category', '7'], method: 'POST', path: `${root}/calendar_events`, payload: { summary: 'Dance', starts_at: '2026-09-28T18:00:00-04:00', ends_at: '2026-09-28T19:00:00-04:00', category_id: '7' }, response: { data: calendar } },
  { args: ['calendar', 'create-countdown', '--title', 'Birthday', '--date', '2026-10-04'], method: 'POST', path: `${root}/calendar_events`, payload: { summary: 'Birthday', starts_at: '2026-10-04', all_day: true, event_type: 'countdown' }, response: { data: entity('62', { summary: 'Birthday' }) } },
  { args: ['calendar', 'update', '--event-id', '60', '--title', 'Class', '--all-day=false'], method: 'PUT', path: `${root}/calendar_events/60`, payload: { summary: 'Class', all_day: false }, response: { data: entity('60', { summary: 'Class' }) } },
  { args: ['calendar', 'delete', '--event-id', '60'], method: 'DELETE', path: `${root}/calendar_events/60` },
  { args: ['lists', 'create', '--title', 'Pack', '--kind', 'to_do', '--hide-from-frame'], method: 'POST', path: `${root}/lists`, payload: { label: 'Pack', kind: 'to_do', color: '#2178AF', hide_from_frame: true }, response: { data: entity('23', { label: 'Pack', kind: 'to_do' }) } },
  { args: ['lists', 'update', '--list-id', '20', '--title', '', '--hide-from-frame=false'], method: 'PUT', path: `${root}/lists/20`, payload: { label: '', hide_from_frame: false }, response: { data: list } },
  { args: ['lists', 'delete', '--list-id', '20'], method: 'DELETE', path: `${root}/lists/20` },
  { args: ['lists', 'add-item', '--list-id', '20', '--title', 'Milk', '--position', '0', '--completed'], method: 'POST', path: `${root}/lists/20/list_items`, payload: { label: 'Milk', position: 0, status: 'completed' }, response: { data: entity('21', { label: 'Milk' }) } },
  { args: ['lists', 'update-item', '--list-id', '20', '--item-id', '21', '--pending'], method: 'PUT', path: `${root}/lists/20/list_items/21`, payload: { status: 'pending' }, response: { data: entity('21', { label: 'Milk', status: 'pending' }) } },
  { args: ['lists', 'delete-item', '--list-id', '20', '--item-id', '21'], method: 'DELETE', path: `${root}/lists/20/list_items/21` },
  { args: ['lists', 'organize', '--list-id', '20'], method: 'POST', path: `${root}/lists/20/organize`, response: { data: { organized: true } } },
  { args: ['lists', 'order', '--list-id', '20', '--retailer', 'fixture-store'], method: 'POST', path: `${root}/lists/20/order`, payload: { retailer: 'fixture-store' }, response: { data: { order_url: 'https://example.test/order' } } },
  { args: ['lists', 'task-box-item', '--title', 'Read'], method: 'POST', path: `${root}/task_box/items`, payload: { task_box_item: { title: 'Read' } }, response: { data: entity('24', { title: 'Read' }) } },
  { args: ['grocery', 'create', '--title', 'Food'], method: 'POST', path: `${root}/lists`, payload: { label: 'Food', kind: 'grocery' }, response: { data: entity('25', { label: 'Food', kind: 'grocery' }) } },
  { args: ['grocery', 'organize', '--list-id', '20'], method: 'POST', path: `${root}/lists/20/organize`, response: { data: { organized: true } } },
  { args: ['grocery', 'order', '--list-id', '20', '--retailer', 'fixture-store'], method: 'POST', path: `${root}/lists/20/order`, payload: { retailer: 'fixture-store' }, response: { data: { order_url: 'https://example.test/order' } } },
  { args: ['grocery', 'add-recipe', '--recipe-id', '30'], method: 'POST', path: `${root}/meals/recipes/30/add_to_grocery_list`, response: { data: { added: 1 } } },
  { args: ['meals', 'create-recipe', '--title', 'Soup', '--ingredients', 'Beans,Water', '--meal-category-id', '40'], method: 'POST', path: `${root}/meals/recipes`, payload: { summary: 'Soup', ingredients: ['Beans', 'Water'], meal_category_id: '40' }, response: { data: recipe } },
  { args: ['meals', 'update-recipe', '--recipe-id', '30', '--description', '', '--ingredients', 'Rice'], method: 'PATCH', path: `${root}/meals/recipes/30`, payload: { description: '', ingredients: ['Rice'] }, response: { data: recipe } },
  { args: ['meals', 'delete-recipe', '--recipe-id', '30'], method: 'DELETE', path: `${root}/meals/recipes/30` },
  { args: ['meals', 'create-sitting', '--recipe-id', '30', '--date', '2026-09-28', '--meal-category-id', '40'], method: 'POST', path: `${root}/meals/sittings`, payload: { meal_recipe_id: '30', date: '2026-09-28', meal_category_id: '40' }, response: { data: sitting } },
  { args: ['meals', 'delete-sitting', '--sitting-id', '50', '--date', '2026-09-28'], method: 'DELETE', path: `${root}/meals/sittings/50/instances/2026-09-28` },
  { args: ['meals', 'add-to-grocery', '--recipe-id', '30'], method: 'POST', path: `${root}/meals/recipes/30/add_to_grocery_list`, response: { data: { added: 1 } } },
  { args: ['routines', 'create', '--title', 'Morning', '--assignee-id', '7', '--steps', 'Brush,Read'], method: 'POST', path: `${root}/routines`, payload: { title: 'Morning', assignee_id: '7', steps: ['Brush', 'Read'] }, response: { data: entity('80', { title: 'Morning' }) } },
  { args: ['routines', 'update', '--routine-id', '80', '--steps', 'Read', '--title', 'Evening'], method: 'PUT', path: `${root}/routines/80`, payload: { title: 'Evening', steps: ['Read'] }, response: { data: entity('80', { title: 'Evening' }) } },
  { args: ['routines', 'delete', '--routine-id', '80'], method: 'DELETE', path: `${root}/routines/80` },
  { args: ['routines', 'reorder', '--routine-ids', '81,80'], method: 'PATCH', path: `${root}/routines/reorder`, payload: { ids: ['81', '80'] }, response: { data: [entity('81', { title: 'First' }), entity('80', { title: 'Second' })] } },
  { args: ['photos', 'delete', '--message-ids', '90,91'], method: 'DELETE', path: `${root}/messages/destroy_multiple`, payload: { message_ids: [90, 91] } },
];

for (const spec of writes) {
  test(`${spec.args.slice(0, 2).join(' ')} sends its mutation and honors both safety guards`, async () => {
    await fixture(async ({ run, requests }) => {
      json(await run([...spec.args, '--json']));
      assert.equal(requests.length, 1);
      const r = requests[0]; assert.equal(r.method, spec.method); assert.equal(r.path, spec.path);
      for (const [key, value] of Object.entries(spec.payload ?? {})) assert.deepEqual(r.body[key], value, key);
      for (const [key, value] of Object.entries(spec.query ?? {})) assert.equal(r.query.get(key), value);
      bad(await run([...spec.args, '--readonly', '--json']), 1, /readonly/);
      bad(await run(['--dry-run', ...spec.args, '--json']), 1, /dry-run/);
      assert.equal(requests.length, 1, 'a safety flag permits a write');
    }, r => r.method === spec.method && r.path === spec.path ? { data: spec.response ?? { ok: true } } : undefined);
  });
}

test('auth login performs CSRF/cookie OAuth flow and saves tokens without output leakage', async () => {
  await fixture(async ({ run, requests, config }) => {
    const receipt = json(await run(['auth', 'login', '--email', 'parent@example.test', '--password-stdin', '--fingerprint', 'fixture-device', '--json', '--trace-http'], { input: password + '\n' }));
    assert.equal(receipt.logged_in, true); assert.equal(receipt.refresh_token_saved, true);
    assert.deepEqual(requests.map(r => `${r.method} ${r.path}`), ['GET /auth/session/new', 'POST /auth/session', 'GET /oauth/authorize', 'POST /oauth/token']);
    const session = requests[1]; assert.equal(session.body.password, password); assert.equal(session.body.email, 'parent@example.test');
    assert.equal(session.body.authenticity_token, 'fixture-csrf'); assert.equal(session.headers.cookie, 'session=fixture-cookie');
    assert.equal(requests[3].body.code, 'fixture-code'); assert.equal(requests[3].body.grant_type, 'authorization_code');
    const saved = JSON.parse(await configBytes(config)); assert.equal(saved.access_token, newToken); assert.equal(saved.refresh_token, 'fixture-rotated-refresh');
    assert.equal((await stat(config)).mode & 0o777, 0o600);
    assert.equal(json(await run(['auth', 'status', '--json'])).has_token, true);
    json(await run(['frames', 'list', '--json']));
    assert.equal(requests.at(-1)!.headers.authorization, `Bearer ${newToken}`);
  }, r => {
    if (r.path === '/auth/session/new') return { text: '<input name="authenticity_token" value="fixture-csrf">', headers: { 'set-cookie': 'session=fixture-cookie; Path=/' } };
    if (r.path === '/auth/session') return { status: 302, headers: { location: '/dashboard' }, text: '' };
    if (r.path === '/oauth/authorize') return { status: 302, headers: { location: 'https://ourskylight.com/welcome?code=fixture-code' }, text: '' };
    if (r.path === '/oauth/token') return { data: { access_token: newToken, refresh_token: 'fixture-rotated-refresh', expires_in: 3600 } };
  });
});

test('normal auto-refresh rotates persisted credentials; explicit token bypasses refresh', async () => {
  await fixture(async ({ run, requests, config, save }) => {
    const original = await configBytes(config);
    json(await run(['frames', 'list', '--token', 'override-token', '--json']));
    assert.equal(await configBytes(config), original); assert.equal(requests.length, 1);
    assert.equal(requests[0].headers.authorization, 'Bearer override-token');
    const concurrent = await Promise.all([run(['frames', 'list', '--json']), run(['frames', 'list', '--json'])]);
    concurrent.forEach(json);
    assert.equal(requests[1].path, '/oauth/token'); assert.equal(requests[1].body.grant_type, 'refresh_token');
    assert.equal(requests[1].body.refresh_token, refresh); assert.equal(requests[1].body.skylight_api_client_device_fingerprint, 'fixture-device');
    assert.equal(requests[2].headers.authorization, `Bearer ${newToken}`);
    assert.equal(requests[3].headers.authorization, `Bearer ${newToken}`);
    assert.equal(requests.filter(r => r.path === '/oauth/token').length, 1, 'concurrent callers must not rotate the same refresh token twice');
    const saved = JSON.parse(await configBytes(config)); assert.equal(saved.access_token, newToken); assert.equal(saved.refresh_token, 'fixture-rotated-refresh');
    json(await run(['auth', 'refresh', '--json']));
    assert.equal(requests.at(-1)!.body.refresh_token, 'fixture-rotated-refresh');
    await save({ access_token: '' });
    json(await run(['frames', 'list', '--json'], { env: { SKYLIGHT_ACCESS_TOKEN: 'override-env-token', SKYLIGHT_FRAME_ID: '123' } }));
    assert.equal(requests.at(-1)!.headers.authorization, 'Bearer override-env-token');
  }, r => r.path === '/oauth/token' ? { data: { access_token: newToken, refresh_token: 'fixture-rotated-refresh', expires_in: 3600 } } : undefined,
  { refresh_token: refresh, device_fingerprint: 'fixture-device', access_token_expires_at: '2000-01-01T00:00:00Z' });
});

test('readonly and dry-run never refresh expired tokens or change config', async () => {
  await fixture(async ({ run, requests, config }) => {
    const before = await configBytes(config);
    for (const flag of ['--readonly', '--dry-run']) {
      bad(await run([flag, 'frames', 'list', '--json']), 1, /readonly|dry-run/);
      assert.equal(requests.length, 0, 'token refresh sends a POST under a safety flag');
      assert.equal(await configBytes(config), before, 'token refresh changes stored credentials');
      json(await run([flag, '--token', 'override-token', 'frames', 'list', '--json']));
      assert.equal(requests.at(-1)!.method, 'GET'); requests.length = 0;
    }
  }, r => r.path === '/oauth/token' ? { data: { access_token: newToken, refresh_token: 'fixture-rotated-refresh', expires_in: 3600 } } : undefined,
  { refresh_token: refresh, device_fingerprint: 'fixture-device', access_token_expires_at: '2000-01-01T00:00:00Z' });
});

test('readonly and dry-run auth login do not create a remote session or save tokens', async () => {
  await fixture(async ({ run, requests, config, home }) => {
    const before = await configBytes(config);
    const files = (await readdir(home, { recursive: true })).sort();
    for (const flag of ['--readonly', '--dry-run']) {
      bad(await run([flag, 'auth', 'login', '--email', 'parent@example.test', '--password-stdin', '--json'], { input: password }), 1, new RegExp(flag.slice(2)));
      assert.equal(requests.length, 0); assert.equal(await configBytes(config), before);
      assert.deepEqual((await readdir(home, { recursive: true })).sort(), files, 'a safety flag creates credential files');
    }
  }, r => {
    if (r.path === '/auth/session/new') return { text: '<input name="authenticity_token" value="fixture-csrf">' };
    if (r.path === '/auth/session') return { text: 'ok' };
    if (r.path === '/oauth/authorize') return { status: 302, headers: { location: 'https://ourskylight.com/welcome?code=fixture-code' }, text: '' };
    if (r.path === '/oauth/token') return { data: { access_token: newToken, refresh_token: refresh, expires_in: 3600 } };
  });
});

test('readonly and dry-run refuse local credential/config writes and frame selection changes', async () => {
  await fixture(async ({ run, config, requests, home }) => {
    const mmkv = join(home, 'mmkv.default');
    await writeFile(mmkv, JSON.stringify({ state: { userId: '1', accessToken: token, refreshToken: refresh, uniqueId: 'fixture-device' }, version: 1 }));
    const before = await configBytes(config);
    const files = (await readdir(home, { recursive: true })).sort();
    for (const flag of ['--readonly', '--dry-run']) {
      for (const args of [['auth', 'set-token'], ['auth', 'import-mac', '--mmkv', mmkv],
        ['config', 'set', 'frame', '999'], ['config', 'unset', 'access_token'], ['config', 'edit'], ['frames', 'set-default', '999']]) {
        bad(await run([flag, ...args, '--json'], { input: newToken, env: { EDITOR: 'true' } }), 1, new RegExp(flag.slice(2)));
        assert.equal(requests.length, 0); assert.equal(await configBytes(config), before);
        assert.deepEqual((await readdir(home, { recursive: true })).sort(), files, 'a safety flag creates credential files');
      }
    }
  });
});

test('config set/get/unset, auth set-token/import-mac, frame selection and editor persist across processes', async () => {
  await fixture(async ({ run, config, home, requests }) => {
    json(await run(['config', 'set', 'api_version', '2026-04-15', '--json']));
    assert.equal(json(await run(['config', 'get', 'api_version', '--json'])).value, '2026-04-15');
    json(await run(['frames', 'set-default', '456', '--json']));
    assert.equal(json(await run(['config', 'get', 'frame', '--json'])).value, 456);
    json(await run(['config', 'unset', 'frame', '--json']));
    assert.equal(json(await run(['config', 'get', 'frame', '--json'])).value, 0);
    json(await run(['auth', 'set-token', '--json'], { input: `Basic ${token}\n` }));
    const shown = json(await run(['config', 'show', '--json']));
    assert.equal(shown.access_token, 'fixt****'); assert.equal(shown.auth_scheme, 'Basic');
    assert.match(ok(await run(['config', 'get', 'access_token', '--plain'])), /fixt\*\*\*\*/);
    const mmkv = join(home, 'mmkv.default');
    await writeFile(mmkv, `prefix${JSON.stringify({ state: { userId: '1', accessToken: token, refreshToken: refresh, accessTokenExpiry: 4102444800000, uniqueId: 'fixture-device' }, version: 1 })}suffix`);
    assert.equal(json(await run(['auth', 'import-mac', '--mmkv', mmkv, '--json'])).imported, true);
    assert.equal(JSON.parse(await configBytes(config)).refresh_token, refresh);
    const editor = join(home, 'editor.mjs');
    await writeFile(editor, 'import fs from "node:fs";const p=process.argv.at(-1);const c=JSON.parse(fs.readFileSync(p));c.default_frame_id=123;fs.writeFileSync(p,JSON.stringify(c));');
    ok(await run(['config', 'edit'], { env: { EDITOR: `"${process.execPath}" "${editor}"` } }));
    assert.equal(json(await run(['config', 'get', 'frame', '--json'])).value, 123);
    assert.equal(requests.length, 0);
  });
});

test('table commands preserve TSV rows and JSON format conflicts fail', async () => {
  await fixture(async ({ run }) => {
    for (const [args, title] of [[['frames', 'list'], 'Fixture frame'], [['categories'], 'Test child'], [['chores', 'list'], 'Sweep'],
      [['rewards', 'list'], 'TV ticket'], [['rewards', 'points'], '13']] as const) {
      assert.ok(ok(await run([...args, '--plain'])).includes(title));
      assert.ok(ok(await run([...args])).includes(title));
    }
    bad(await run(['--json', '--plain', 'version']), 2, /choose only one/);
  });
});

test('safe reads do not migrate secrets; normal file storage persists encrypted credentials across processes', async () => {
  await fixture(async ({ run, home, config }) => {
    const env = { SKYCLI_FILE_SECRET_KEY: 'fixture-file-encryption-key' };
    const before = await configBytes(config);
    for (const flag of ['--dry-run', '--readonly']) {
      json(await run([flag, 'frames', 'list', '--json'], { env }));
      assert.equal(await configBytes(config), before, 'a safe read silently migrates credentials');
      assert.deepEqual(await readdir(home), ['config.json']);
    }
    json(await run(['auth', 'set-token', '--json'], { input: newToken, env }));
    const saved = JSON.parse(await configBytes(config));
    assert.ok(!saved.access_token && !saved.refresh_token);
    const configRoot = process.platform === 'darwin' ? join(home, 'Library', 'Application Support') : join(home, '.config');
    const secretFile = join(configRoot, 'skycli', 'secrets.json.enc');
    const encrypted = await readFile(secretFile);
    assert.ok(!encrypted.includes(Buffer.from(newToken))); assert.equal((await stat(secretFile)).mode & 0o777, 0o600);
    assert.equal(json(await run(['auth', 'status', '--json'], { env })).has_token, true);
    json(await run(['frames', 'list', '--readonly', '--json'], { env }));
  }, undefined, { secrets_backend: 'file' });
});

test('raw external GET omits account authorization and API headers', async () => {
  let headers: http.IncomingHttpHeaders | undefined;
  const external = http.createServer((req, res) => { headers = req.headers; res.end('{"public":"fixture"}'); });
  await new Promise<void>(resolve => external.listen(0, '127.0.0.1', resolve));
  try {
    await fixture(async ({ run, requests }) => {
      const url = `http://127.0.0.1:${(external.address() as AddressInfo).port}/public`;
      assert.equal(json(await run(['raw', url, '--readonly', '--json'])).public, 'fixture');
      assert.equal(headers!.authorization, undefined); assert.equal(headers!['skylight-api-version'], undefined);
      assert.equal(requests.length, 0);
    });
  } finally { external.closeAllConnections(); await new Promise<void>(resolve => external.close(() => resolve())); }
});

test('week views, reports, doctor, and Plus status combine the correct resources', async () => {
  await fixture(async ({ run, requests }) => {
    const week = json(await run(['calendar', 'week', '--date', '2026-09-30', '--json']));
    assert.equal(week.length, 7); assert.equal(week[0].date, '2026-09-28');
    assert.equal(week[0].events[0].attributes.starts_at, '2026-09-28T20:30:00-04:00');
    assert.equal(json(await run(['chores', 'week', '--date', '2026-09-30', '--json']))[0].chores[0].id, '5');
    assert.ok(Array.isArray(json(await run(['chores', 'streak', '--days', '7', '--json']))));
    const status = json(await run(['status', '--json'])); assert.equal(status.frame.id, '123'); assert.equal(status.points[0].balance, 13);
    const analytics = json(await run(['analytics', '--days', '7', '--json'])); assert.ok(JSON.stringify(analytics).includes('Test child'));
    const home = json(await run(['home', '--date', '2026-09-30', '--json'])); assert.equal(home.week_start, '2026-09-28'); assert.equal(home.lists[0].id, '20');
    const count = requests.length;
    json(await run(['home', '--date', '2026-09-30', '--no-tasks', '--no-lists', '--json']));
    assert.deepEqual(requests.slice(count).map(r => r.path), [root, `${root}/calendar_events`]);
    assert.equal(json(await run(['--readonly', '--doctor', '--json'])).ok, true);
    const plus = json(await run(['sidekick', 'status', '--json'])); assert.equal(plus.active_calendar_plus, true); assert.equal(plus.active_subscription_count, 1);
    const bounty = await run(['bounties', 'list', '--json']); assert.deepEqual(json(bounty), []); assert.match(bounty.stderr, /no verified bounty links/);
    assert.ok(requests.every(r => r.method === 'GET'));
  });
});

test('chore/reward bulk writes validate each item and report partial failures', async () => {
  await fixture(async ({ run, requests, home }) => {
    const chores = [{ summary: 'Sweep', category_id: 7 }, { summary: '', category_id: 7 }];
    const c = await run(['chores', 'bulk', '--sleep', '0s', '--json'], { input: JSON.stringify(chores) });
    bad(c, 1, /summary and category_id/); assert.equal(JSON.parse(c.stdout).failures, 1); assert.equal(requests.length, 1);
    const bulkFile = join(home, 'rewards.json');
    await writeFile(bulkFile, JSON.stringify([{ name: 'TV ticket', point_value: 10, category_ids: [7] }, { name: 'Bad', point_value: 0 }]));
    const r = await run(['rewards', 'bulk', '--file', bulkFile, '--json']); bad(r, 1, /point_value/);
    assert.equal(JSON.parse(r.stdout).results[0].ok, true); assert.equal(requests.length, 2);
    bad(await run(['--dry-run', 'chores', 'bulk', '--sleep', '0s', '--json'], { input: JSON.stringify([chores[0]]) }), 1, /dry-run/);
    assert.equal(requests.length, 2);
  }, r => r.method === 'POST' && r.path === `${root}/chores` ? { data: { data: chore } } :
    r.method === 'POST' && r.path === `${root}/rewards` ? { data: { data: [reward] } } : undefined);
});

test('grocery add and clear-completed mutate only the selected items and acknowledge counts', async () => {
  const removed: string[] = [];
  await fixture(async ({ run, requests }) => {
    const added = json(await run(['grocery', 'add', '--list-id', '20', '--title', 'Milk', '--items', 'Eggs,Bread', '--json']));
    assert.equal(added.count, 3); assert.deepEqual(requests.map(r => r.body.label), ['Milk', 'Eggs', 'Bread']);
    const cleared = json(await run(['lists', 'clear-completed', '--list-id', '20', '--json']));
    assert.deepEqual(cleared.deleted, ['21']); assert.deepEqual(removed, ['21']);
    assert.equal(json(await run(['grocery', 'clear', '--list-id', '20', '--json'])).count, 0);
    const before = requests.filter(r => r.method !== 'GET').length;
    bad(await run(['--readonly', 'grocery', 'add', '--list-id', '20', '--title', 'Milk', '--json']), 1, /readonly/);
    assert.equal(requests.filter(r => r.method !== 'GET').length, before);
  }, r => {
    if (r.method === 'POST' && r.path === `${root}/lists/20/list_items`) return { data: { data: entity(String(21 + removed.length), { label: r.body.label }) } };
    if (r.method === 'DELETE' && r.path === `${root}/lists/20/list_items/21`) { removed.push('21'); return { data: {} }; }
    if (r.method === 'GET' && r.path === `${root}/lists/20` && removed.length) return { data: { data: list, included: [entity('22', { label: 'Eggs', status: 'pending' })] } };
  });
});

test('bounty create/update/delete, reward rollback, and partial update receipts reflect applied writes', async () => {
  let failReward = false;
  await fixture(async ({ run, requests }) => {
    const args = ['bounties', 'create', '--title', 'Sweep', '--points', '10', '--assignee-id', '7', '--reward-title', 'TV ticket', '--json'];
    const created = json(await run(args)); assert.equal(created.chore.id, '5'); assert.equal(created.reward.id, '9');
    json(await run(['bounties', 'update', '--chore-id', '5', '--reward-id', '9', '--title', 'Mop', '--points', '12', '--json']));
    json(await run(['bounties', 'delete', '--chore-id', '5', '--reward-id', '9', '--json']));
    failReward = true;
    const start = requests.length; bad(await run(args), 1, /create bounty reward/);
    assert.deepEqual(requests.slice(start).map(r => `${r.method} ${r.path}`), [`POST ${root}/chores`, `POST ${root}/rewards`, `DELETE ${root}/chores/5`]);
    const partial = await run(['bounties', 'update', '--chore-id', '5', '--reward-id', '9', '--title', 'Mop', '--json']);
    bad(partial, 1, /update bounty reward/); assert.equal(JSON.parse(partial.stdout).partial, true); assert.equal(JSON.parse(partial.stdout).applied.chore.id, '5');
  }, r => {
    if (r.method !== 'GET' && r.path.startsWith(`${root}/chores`)) return { data: { data: chore } };
    if (r.method !== 'GET' && r.path.startsWith(`${root}/rewards`)) return failReward ? { status: 503, data: { error: 'reward unavailable' } } : { data: { data: r.method === 'POST' ? [reward] : reward } };
  });
});

test('rotations alternate assignees and report partial creation failure', async () => {
  let failAfter = Infinity;
  const created: any[] = [];
  await fixture(async ({ run, requests }) => {
    const args = ['rotations', 'create', '--chores', 'Sweep,Mop', '--assignee-ids', '7,8', '--weeks', '2', '--start-date', '2026-09-28', '--json'];
    assert.equal(json(await run(args)).chores.length, 4);
    assert.deepEqual(created.map(c => [c.summary, c.category_id, c.start]), [['Sweep', 7, '2026-09-28'], ['Mop', 8, '2026-09-28'], ['Sweep', 8, '2026-10-05'], ['Mop', 7, '2026-10-05']]);
    failAfter = created.length + 1;
    const partial = await run(args); bad(partial, 1, /HTTP 503/); assert.equal(JSON.parse(partial.stdout).created.length, 1);
    const count = requests.length; bad(await run(['--dry-run', ...args]), 1, /dry-run/); assert.equal(requests.length, count);
  }, r => {
    if (r.method === 'POST' && r.path === `${root}/chores`) {
      if (created.length >= failAfter) return { status: 503, data: { error: 'write unavailable' } };
      created.push(r.body); return { data: { data: entity(String(created.length), { summary: r.body.summary }) } };
    }
  });
});

test('portable export persists every supported resource and import maps new recipe IDs', async () => {
  const written: any[] = [];
  await fixture(async ({ run, home, requests }) => {
    const file = join(home, 'export.json');
    assert.equal(json(await run(['export', '--output-file', file, '--days', '1', '--json'])).output_file, file);
    const data = JSON.parse(await readFile(file, 'utf8'));
    assert.equal(data.frame_id, 123); assert.equal(data.chores[0].summary, 'Sweep'); assert.equal(data.lists[0].list_items.length, 2);
    assert.equal(data.recipes[0].id, '30'); assert.equal(data.meal_sittings[0].meal_recipe_id, '30');
    assert.equal((await stat(file)).mode & 0o777, 0o600);
    const before = requests.length;
    assert.equal(json(await run(['import', '--file', file, '--dry-run', '--json'])).recipes, 1);
    assert.ok(requests.slice(before).every(r => r.method === 'GET'));
    assert.equal(json(await run(['--dry-run', 'import', '--file', file, '--json'])).recipes, 1);
    const result = json(await run(['import', '--file', file, '--json']));
    assert.deepEqual(result.failed, []); assert.equal(result.created.recipes, 1); assert.equal(result.created.sittings, 1);
    assert.equal(written.find(r => r.path.endsWith('/meals/sittings')).body.meal_recipe_id, 'new-recipe');
    await writeFile(file, JSON.stringify({ ...data, frame_id: 999 }));
    const count = requests.length; bad(await run(['import', '--file', file, '--json']), 2, /cross-frame/); assert.equal(requests.length, count);
  }, r => {
    if (r.method === 'GET') return;
    written.push(r);
    if (r.path === `${root}/rewards`) return { data: { data: [reward] } };
    if (r.path === `${root}/chores`) return { data: { data: chore } };
    if (r.path === `${root}/lists`) return { data: { data: entity('new-list', { label: 'Groceries' }) } };
    if (r.path.startsWith(`${root}/lists/new-list/list_items`)) return { data: { data: entity('new-item', { label: 'Milk' }) } };
    if (r.path === `${root}/meals/recipes`) return { data: { data: entity('new-recipe', { summary: 'Soup' }) } };
    if (r.path === `${root}/meals/sittings`) return { data: { data: sitting } };
    if (r.path === `${root}/calendar_events`) return { data: { data: calendar } };
  });
});

test('portable import rejects missing references and export retains existing output after read failure', async () => {
  await fixture(async ({ run, home, requests }) => {
    const file = join(home, 'export.json'); await writeFile(file, 'keep original');
    bad(await run(['export', '--resources', 'lists', '--output-file', file, '--json']), 1, /HTTP 503/);
    assert.equal(await readFile(file, 'utf8'), 'keep original');
    await writeFile(file, JSON.stringify({ frame_id: 123, chores: [{ summary: 'Missing', category_id: 999, start: '2026-09-28' }] }));
    bad(await run(['import', '--file', file, '--dry-run', '--json']), 1, /category/);
    assert.ok(requests.every(r => r.method === 'GET'));
  }, r => r.path === `${root}/lists/20` ? { status: 503, data: { error: 'items unavailable' } } : undefined);
});

test('watch persists seed state, emits each new reward once, and exits on SIGINT', async () => {
  let polls = 0;
  await fixture(async ({ run, home, requests }) => {
    const seed = json(await run(['watch', '--once', '--persist', '--resources', 'rewards', '--json'])); assert.equal(seed.seeded, true); assert.equal(seed.seen.rewards, 1);
    const state = (await readdir(home)).find(n => n.includes('.watch-'))!;
    assert.equal(JSON.parse(await readFile(join(home, state), 'utf8')).seen_reward_ids['9'], true);
    const result = await run(['watch', '--interval', '1s', '--persist', '--resources', 'rewards', '--json'], {
      onStdout: (data, child) => { if (data.includes('reward_redeemed')) child.kill('SIGINT'); },
    });
    assert.equal(JSON.parse(ok(result)).id, '10');
    const saved = JSON.parse(await readFile(join(home, state), 'utf8')); assert.equal(saved.seen_reward_ids['10'], true);
    assert.ok(requests.every(r => r.method === 'GET'));
  }, r => r.method === 'GET' && r.path === `${root}/rewards` ? { data: { data: ++polls >= 3 ? [reward, entity('10', { name: 'Book', point_value: 5, redeemed_at: '2026-10-04T10:00:00Z' })] : [reward] } } : undefined);
});

test('photo upload and atomic download transfer bytes without API credentials on storage requests', async () => {
  let uploaded = Buffer.alloc(0), base = '';
  await fixture(async ({ run, home, baseURL, requests }) => {
    base = baseURL;
    const file = join(home, 'photo.png'); await writeFile(file, 'fixture photo bytes');
    assert.equal(json(await run(['photos', 'upload', '--file', file, '--caption', 'Family', '--json'])).key, 'photo-key');
    assert.equal(uploaded.toString(), 'fixture photo bytes');
    assert.deepEqual(requests[0].body, { ext: 'png', frame_ids: ['123'], caption: 'Family' });
    const out = join(home, 'download.png');
    json(await run(['photos', 'download', '--asset-url', baseURL + '/storage/photo', '--out', out, '--readonly', '--json']));
    assert.equal(await readFile(out, 'utf8'), 'fixture photo bytes');
    bad(await run(['photos', 'download', '--asset-url', baseURL + '/storage/fail', '--out', out, '--json']), 1, /download failed/);
    assert.equal(await readFile(out, 'utf8'), 'fixture photo bytes');
    const count = requests.length; bad(await run(['--dry-run', 'photos', 'upload', '--file', file, '--json']), 1, /dry-run/); assert.equal(requests.length, count);
  }, r => {
    if (r.path === '/api/upload_url') return { data: { data: { url: base + '/storage/photo', key: 'photo-key', message_ids: [90], frame_names: ['Fixture frame'] } } };
    if (r.path.startsWith('/storage/')) {
      assert.equal(r.headers.authorization, undefined);
      if (r.path === '/storage/fail') return { status: 500, text: 'failed' };
      if (r.method === 'PUT') { uploaded = r.bytes; assert.equal(r.headers['content-type'], 'image/png'); return { status: 204, text: '' }; }
      return { text: uploaded.toString() };
    }
  });
});

test('raw preserves large integers, parses stdin payloads, and refuses cross-origin redirects', async () => {
  await fixture(async ({ run, requests, baseURL }) => {
    assert.match(ok(await run(['raw', '/precision?n=9007199254740993', '--json'])), /9007199254740993/);
    assert.equal(requests[0].query.get('n'), '9007199254740993');
    assert.deepEqual(json(await run(['raw', '--method', 'PATCH', '--body-file', '-', '/payload', '--json'], { input: '{"value":9007199254740993}' })), { accepted: true });
    assert.match(requests[1].bytes.toString(), /9007199254740993/);
    bad(await run(['raw', '/redirect', '--json']), 1, /cross-origin redirect/);
    assert.ok(!requests.some(r => r.path === '/capture'));
    bad(await run(['raw', '--method', 'POST', '/payload', '--readonly', '--json']), 1, /readonly/);
    bad(await run(['raw', '--body', '{broken', '/payload', '--json']), 2, /valid JSON/);
    bad(await run(['raw', '/precision', 'extra', '--json']), 2, /skycli raw/);
    assert.ok(baseURL.startsWith('http://127.0.0.1:'));
  }, r => {
    if (r.path === '/precision') return { text: '{"value":9007199254740993}' };
    if (r.path === '/payload') return { data: { accepted: true } };
    if (r.path === '/redirect') return { status: 307, headers: { location: 'http://localhost:1/capture' }, text: '' };
  });
});

test('allow/deny lists and readonly environment forbid writes before HTTP or config changes', async () => {
  await fixture(async ({ run, requests, config }) => {
    const before = await configBytes(config);
    json(await run(['frames', 'list', '--json'], { env: { SKYCLI_ALLOW_COMMANDS: 'frames list' } }));
    bad(await run(['categories', '--json'], { env: { SKYCLI_ALLOW_COMMANDS: 'frames list' } }), 1, /not allowed/);
    bad(await run(['chores', 'delete', '--id', '5', '--json'], { env: { SKYCLI_READONLY: 'true' } }), 1, /readonly/);
    bad(await run(['--allow-commands', 'frames', '--deny-commands', 'frames list', 'frames', 'list', '--json']), 1, /denied/);
    assert.equal(requests.length, 1); assert.equal(await configBytes(config), before);
  });
});

test('invalid arguments and missing inputs fail before remote changes', async () => {
  await fixture(async ({ run, requests, config, save }) => {
    for (const args of [
      ['frames', 'notifications', '--type', 'other'], ['frames', 'nudges', '--after', 'bad', '--before', 'bad'],
      ['chores', 'create'], ['chores', 'claim', '--id', '5'], ['chores', 'search'], ['chores', 'streak', '--days', '0'],
      ['rewards', 'create'], ['rewards', 'update', '--id', '9', '--respawn', '--no-respawn'], ['calendar', 'search'],
      ['lists', 'show'], ['lists', 'update-item', '--list-id', '20', '--item-id', '21', '--completed', '--pending'],
      ['grocery', 'add', '--list-id', '20'], ['meals', 'recipe-info'], ['meals', 'delete-sitting', '--sitting-id', '50'],
      ['photos', 'comments', '--message-id', '90', '--page', '0'], ['albums', 'messages', '--album-id', '70', '--page', '0'],
      ['routines', 'update'], ['rotations', 'create'], ['watch', '--interval', '0s'], ['import'], ['auth', 'login'],
      ['config', 'get', 'unknown'], ['unknown'], ['--doctor', 'frames'],
    ]) bad(await run([...args, '--json']), 2, /required|unknown|choose only|must be|cannot be combined|parse --after/);
    bad(await run(['auth', 'set-token', '--json']), 1, /empty token/);
    bad(await run(['auth', 'login', '--email', 'parent@example.test', '--password-stdin', '--json']), 1, /empty password/);
    await save({ access_token: '', default_frame_id: 123 }); bad(await run(['frames', 'list', '--json']), 1, /no access token/);
    await save({ default_frame_id: 0 }); bad(await run(['categories', '--json']), 1, /no frame ID/);
    await writeFile(config, '{broken'); bad(await run(['frames', 'list', '--json']), 1, /parse config/);
    assert.equal(requests.length, 0);
  });
});

for (const args of [['frames', 'list'], ['chores', 'list'], ['rewards', 'list'], ['calendar', 'list'], ['lists', 'list'],
  ['meals', 'recipes'], ['photos', 'list'], ['albums', 'list'], ['routines', 'list'], ['sidekick', 'status'], ['status'], ['analytics'], ['home'], ['watch', '--once']]) {
  test(`${args.join(' ')} reports HTTP failure as JSON with a nonzero exit`, async () => {
    await fixture(async ({ run }) => {
      const result = await run([...args, '--json']); bad(result, 1, /HTTP 503/);
      assert.equal(JSON.parse(result.stdout).http_status, 503);
    }, () => ({ status: 503, data: { error: 'fixture unavailable' } }));
  });
}

test('malformed JSON, timeout, and refused connections do not report data success', async () => {
  await fixture(async ({ run, save }) => {
    bad(await run(['frames', 'list', '--json']), 1, /invalid character/);
    bad(await run(['raw', '/slow', '--timeout', '100ms', '--json']), 1, /deadline exceeded|Client.Timeout/);
    await save({ base_url: 'http://127.0.0.1:1' });
    bad(await run(['frames', 'list', '--timeout', '100ms', '--json']), 1, /connection refused/);
  }, r => r.path === '/slow' ? { hang: true } : { text: '<html>not JSON</html>' });
});

test('auth failures preserve credentials and JSON diagnostics without token disclosure', async () => {
  await fixture(async ({ run, config }) => {
    const before = await configBytes(config);
    bad(await run(['auth', 'refresh', '--json']), 1, /HTTP 401/); assert.equal(await configBytes(config), before);
    bad(await run(['auth', 'login', '--email', 'parent@example.test', '--password-stdin', '--json'], { input: password }), 1, /authenticity_token not found/);
    assert.equal(await configBytes(config), before);
  }, r => r.path === '/oauth/token' ? { status: 401, data: { error: 'invalid grant' } } : { text: '<html>no csrf</html>' },
  { refresh_token: refresh, device_fingerprint: 'fixture-device' });
});

test('command catalog, aliases, defaults, version, and literal flag delimiters work as separate processes', async () => {
  await fixture(async ({ run, requests }) => {
    const catalog = json(await run(['commands', '--json'])); assert.ok(Array.isArray(catalog.commands));
    assert.deepEqual(catalog.commands.map((c: any) => c.name).sort(), [
      'commands', 'auth', 'frames', 'categories', 'chores', 'rewards', 'calendar', 'lists', 'grocery', 'meals',
      'photos', 'albums', 'routines', 'sidekick', 'bounties', 'rotations', 'status', 'analytics', 'home', 'watch',
      'export', 'import', 'config', 'raw', 'version',
    ].sort(), 'new command families need a coverage matrix entry and executable flows');
    const covered = new Set([
      ...reads.map(([args]) => args.slice(0, 2).join(' ')), ...writes.map(({ args }) => args.slice(0, 2).join(' ')),
      'auth login', 'auth import-mac', 'auth refresh', 'auth set-token', 'auth status', 'frames set-default',
      'config show', 'config get', 'config set', 'config unset', 'config edit', 'chores week', 'chores streak', 'chores bulk',
      'rewards bulk', 'calendar week', 'lists clear-completed', 'grocery add', 'grocery clear', 'photos upload', 'photos download',
      'sidekick status', 'bounties list', 'bounties create', 'bounties update', 'bounties delete', 'rotations create',
    ]);
    for (const command of catalog.commands) {
      for (const sub of command.subcommands ?? []) assert.ok(covered.has(`${command.name} ${sub.name}`),
        `${command.name} ${sub.name} needs an executable flow and matrix entry`);
    }
    assert.deepEqual(json(await run(['--json'])), catalog);
    const version = json(await run(['version', '--json'])); assert.ok(version.version.length > 0);
    assert.ok(ok(await run(['version', '--plain'])).includes(version.version));
    for (const [alias, canonical] of [['frame', 'frames'], ['category', 'categories'], ['chore', 'chores'], ['reward', 'rewards'],
      ['list', 'lists'], ['meal', 'meals'], ['photo', 'photos'], ['album', 'albums'], ['routine', 'routines'], ['bounty', 'bounties']]) {
      assert.deepEqual(json(await run([alias, '--readonly', '--json'])), json(await run([canonical, '--readonly', '--json'])));
    }
    bad(await run(['raw', '--', '/api/frames', '--json']), 2, /skycli raw/);
    assert.ok(requests.every(r => r.method === 'GET'));
  });
});
