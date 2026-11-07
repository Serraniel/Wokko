# UI overhaul: fast, crisp, modern

Goal, from the owner: Wokko should look and feel like a blazing-fast modern web app, on the phone first.

## What was measured before changing anything

- JavaScript and CSS were sent uncompressed: 524 KB, against 141 KB gzipped.
- Every page change waited on a 110 ms cross-fade, and page entrances faded in. An interface can't feel instant while it plays an animation first.
- Opening a recipe always waited for its data, though the card had been under the pointer for a while.
- Route code loaded only on demand, so the first visit to each page paid for it.

## Speed

1. Serve gzip for all text assets, compressed once and kept in memory.
2. No page transitions. Navigation swaps content immediately.
3. Load a recipe's data when a card is pressed or hovered, so it's ready when the page opens.
4. Load route code straight after startup. It's cached for good after that.
5. Skip rendering off-screen cards (`content-visibility`), and load more recipes while scrolling.
6. Command palette (Ctrl/⌘+K, or `/`) that reaches any recipe or page from the keyboard. `n` starts a new recipe.

Deliberately left out for now: a service worker. Offline shells can serve stale code after an upgrade, and stability comes first.

## Look

- Neutral, crisp surfaces instead of the warm beige. Near-white and near-black, hairline borders, red only for what matters.
- Inter, self-hosted and embedded in the binary. No third-party font requests.
- Tighter type and spacing. 44 px touch targets stay on touch screens, and controls get more compact with a mouse.
- Photo-first recipe cards. A recipe without a photo gets a tinted tile with its initial, not a repeated grey logo.
- Recipe page: on large screens the ingredients stay beside the steps while scrolling, which is what you want while cooking.
- New logo: a plain wok and one flame. It replaces the character with face, cheeks and legs.

## Rules that stay

Colors only through tokens, both themes, every string in the message files, reduced motion respected, no horizontal page scroll at phone width.
