async function runQuery() {
  return fetch('/api/ds/query', { method: 'POST' });
}
