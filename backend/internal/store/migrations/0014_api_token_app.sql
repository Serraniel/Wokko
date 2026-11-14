-- Which assistant app a token was made for, so the integrations page can show each app as
-- connected or not. Existing tokens named exactly like an app count as that app.
ALTER TABLE api_tokens ADD COLUMN app TEXT NOT NULL DEFAULT 'other';
UPDATE api_tokens SET app = CASE lower(trim(name))
    WHEN 'claude' THEN 'claude'
    WHEN 'chatgpt' THEN 'chatgpt'
    WHEN 'claude code' THEN 'claude_code'
    WHEN 'codex' THEN 'codex'
    WHEN 'gemini cli' THEN 'gemini'
    WHEN 'cursor' THEN 'cursor'
    WHEN 'vs code' THEN 'vscode'
    WHEN 'openclaw' THEN 'openclaw'
    WHEN 'hermes agent' THEN 'hermes'
    ELSE 'other' END;
