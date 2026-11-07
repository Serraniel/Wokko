# AI integrations

Status: **built**, on 2026-11-07. Tested with Claude Code and the official MCP client, over HTTP and the stdio bridge. A hosted connector on claude.ai needs a public HTTPS address, so that path has been tested only against the OAuth flow itself. The owner uses Claude and wants both local and network use.

## Principle

Wokko stays free of any built-in AI. It doesn't call a model, hold an API key or send recipes anywhere. AI tools come to Wokko through a standard interface, the Model Context Protocol (MCP), and only if the person turns it on. This keeps the app stable and private, and lets people use whichever assistant they already have.

## What it makes possible

- "What can I cook with leeks and potatoes in 30 minutes?"
- "Save this recipe to Wokko", from a chat with any MCP-capable assistant.
- "Translate this recipe to German" or "scale this for 12 people and save a copy".
- Later, once those features exist: "Plan next week's dinners" and "put what's missing on my shopping list".

## Design

**Server.** An MCP endpoint at `/mcp` in the same binary, off by default. Admins enable it for the instance and each person enables it for their account. It calls the `app` package like the web API and kitshn do, so permissions and data rules exist only once.

**Access.**
1. Personal access tokens first: created in settings, stored hashed, shown once, revocable, read-only or read-write, with an expiry.
2. OAuth for clients that require it (issue #21) as a second step.

Tokens act as their owner and can never see more than the owner can.

**Tools, first version.**

| Tool | Scope | Notes |
|---|---|---|
| `search_recipes` | read | Same search as the app, only what the person may see |
| `get_recipe` | read | Full recipe with scaling |
| `list_foods`, `list_units`, `list_tags` | read | So new recipes reuse the catalog |
| `create_recipe` | write | Private by default |
| `update_recipe` | write | Needs the version it started from, so conflicts are caught |
| `import_recipe_from_url` | read | Returns a draft through the existing importer, saves nothing |
| `delete_recipe` | write, off by default | Moves to the trash only |

Meal plan and shopping list tools follow once those features exist.

**Not offered, ever:** searching or listing users, changing sharing or visibility to public, account and admin actions, permanent deletion. The rule that people can't be discovered holds here too.

## Safety

- **Recipe text is untrusted.** Pages and shared recipes could contain text written to manipulate an assistant. Tool results mark it as data, and the risky actions above simply don't exist as tools, which limits what a manipulated assistant can do.
- **Every write is recoverable.** One transaction, a version snapshot, trash instead of deletion. An assistant's mistake is undone like a person's.
- **Visible.** A per-token activity log in settings shows what changed and when. Tools are rate limited per token.

## Possibly later: AI inside Wokko

Things like importing a recipe from a photo need a model. The owner ruled out AI import for now, so this stays parked. If it returns, it must be optional, use the person's own provider or local model, and show a draft for review like the current importer.
