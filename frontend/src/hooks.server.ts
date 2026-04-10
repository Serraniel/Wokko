import type { Handle } from '@sveltejs/kit';

const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:8080';

export const handle: Handle = async ({ event, resolve }) => {
  if (event.url.pathname.startsWith('/api/')) {
    const target = BACKEND_URL.replace(/\/$/, '') + event.url.pathname + event.url.search;

    const headers = new Headers(event.request.headers);
    headers.delete('host');

    const response = await fetch(target, {
      method: event.request.method,
      headers,
      body: ['GET', 'HEAD'].includes(event.request.method) ? undefined : event.request.body,
      // @ts-ignore — needed for streaming bodies
      duplex: 'half',
    });

    return new Response(response.body, {
      status: response.status,
      headers: response.headers,
    });
  }

  return resolve(event);
};
