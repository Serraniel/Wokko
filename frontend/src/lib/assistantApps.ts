import { t } from '#lib/i18n/index.svelte.ts';

/**
 * Each assistant app's own way in. Apps that connect by address log in through Wokko and need no
 * token. The others get a snippet that fills in the token once it's created.
 */
export type AssistantApp = {
	id: string;
	name: string;
	byAddress?: boolean;
	steps: number;
	snippet?: (token: string) => string;
	link?: (token: string) => string;
	doc?: string;
};

export function assistantApps(address: string): AssistantApp[] {
	// What Cursor and VS Code store for a server that's reached over HTTP with a token.
	const remote = (tk: string) => ({ url: address, headers: { Authorization: `Bearer ${tk}` } });
	return [
		{ id: 'claude', name: 'Claude', byAddress: true, steps: 3, doc: 'https://www.claude.com/docs/connectors/custom/remote-mcp', link: () => `https://claude.ai/customize/connectors?${new URLSearchParams({ modal: 'add-custom-connector', connectorName: 'Wokko', connectorUrl: address })}` },
		{ id: 'chatgpt', name: 'ChatGPT', byAddress: true, steps: 3 },
		{ id: 'claude_code', name: 'Claude Code', steps: 1, snippet: (tk) => `claude mcp add --transport http wokko ${address} --header "Authorization: Bearer ${tk}"` },
		{ id: 'codex', name: 'Codex', steps: 1, doc: 'https://developers.openai.com/codex/mcp', snippet: (tk) => `export WOKKO_TOKEN=${tk}\ncodex mcp add wokko --url ${address} --bearer-token-env-var WOKKO_TOKEN` },
		{ id: 'gemini', name: 'Gemini CLI', steps: 1, snippet: (tk) => `gemini mcp add --transport http --header "Authorization: Bearer ${tk}" wokko ${address}` },
		{ id: 'cursor', name: 'Cursor', steps: 1, doc: 'https://cursor.com/docs/mcp/install-links', snippet: (tk) => JSON.stringify({ mcpServers: { wokko: remote(tk) } }, null, 2), link: (tk) => `cursor://anysphere.cursor-deeplink/mcp/install?name=wokko&config=${encodeURIComponent(btoa(JSON.stringify(remote(tk))))}` },
		{ id: 'vscode', name: 'VS Code', steps: 1, snippet: (tk) => JSON.stringify({ servers: { wokko: { type: 'http', ...remote(tk) } } }, null, 2), link: (tk) => `vscode:mcp/install?${encodeURIComponent(JSON.stringify({ name: 'wokko', type: 'http', ...remote(tk) }))}` },
		{ id: 'openclaw', name: 'OpenClaw', steps: 1, doc: 'https://docs.openclaw.ai/gateway/configuration-reference', snippet: (tk) => JSON.stringify({ mcp: { servers: { wokko: { url: address, transport: 'streamable-http', headers: { Authorization: `Bearer ${tk}` } } } } }, null, 2) },
		{ id: 'hermes', name: 'Hermes Agent', steps: 1, doc: 'https://hermes-agent.nousresearch.com/docs/user-guide/features/mcp', snippet: (tk) => `mcp_servers:\n  wokko:\n    url: "${address}"\n    headers:\n      Authorization: "Bearer ${tk}"` },
		{ id: 'other', name: t('assistant.app_other'), steps: 2, snippet: (tk) => `Authorization: Bearer ${tk}` }
	];
}
