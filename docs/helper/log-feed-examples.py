# rss_collector.py
"""
Simple RSS/Atom feed collector
--------------------------------
Runs continuously, polling a set of publicly‑available service‑health feeds (or any feeds you like) and saves NEW or UPDATED
entries to plain‑text files so you can build a corpus of real‑world examples.

Works with both Atom and RSS 2.0 via the `feedparser` library.

Usage:
    1. Install dependencies (only feedparser is required):

        pip install feedparser

    2. Adjust FEEDS and POLL_INTERVAL as needed.
    3. Run the script (e.g. with `python rss_collector.py`) and leave it running.
    4. New entries are written to ./data/<provider>/YYYY-MM-DD/HHMMSS_<slug>.txt
       Each file contains the full XML fragment of the entry plus a plain‑text summary.

This creates a lightweight local archive you can review later or feed into test cases.
"""

import feedparser
import json
from pathlib import Path
import re
import textwrap
import time
from datetime import datetime, timezone
from typing import Dict, Set

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

# Feeds to monitor (provider name -> URL)
FEEDS: Dict[str, str] = {
    "gcp": "https://status.cloud.google.com/feed.atom",
    "azure": "https://azurestatuscdn.azureedge.net/en-us/status/feed/",
    "aws": "https://status.aws.amazon.com/rss/all.rss",
    "genesys": "https://status.mypurecloud.com/history.atom",
}

# How often to poll each feed (seconds)
POLL_INTERVAL = 300  # 5 minutes

# Folder where collected examples will be stored
DATA_DIR = Path("data")

# File to persist the set of already‑seen entry IDs / guids so we don't duplicate
STATE_FILE = Path("seen_entries.json")

# ---------------------------------------------------------------------------
# Helper functions
# ---------------------------------------------------------------------------

_slug_re = re.compile(r"[^A-Za-z0-9_-]+")

def slugify(text: str, max_len: int = 60) -> str:
    """Create a filesystem‑safe slug from text."""
    slug = _slug_re.sub("-", text).strip("-")
    return slug[:max_len] or "entry"


def load_state() -> Dict[str, Set[str]]:
    """Load set of seen entry IDs per provider from STATE_FILE."""
    if STATE_FILE.exists():
        with STATE_FILE.open("r", encoding="utf-8") as f:
            raw = json.load(f)
        return {k: set(v) for k, v in raw.items()}
    return {p: set() for p in FEEDS}


def save_state(state: Dict[str, Set[str]]):
    with STATE_FILE.open("w", encoding="utf-8") as f:
        json.dump({k: sorted(list(v)) for k, v in state.items()}, f, indent=2)


def write_entry(provider: str, entry):
    """Write a feed entry to a timestamped TXT file."""
    dt = datetime.now(timezone.utc)
    date_folder = DATA_DIR / provider / dt.strftime("%Y-%m-%d")
    date_folder.mkdir(parents=True, exist_ok=True)

    title = entry.get("title", "no-title")
    slug = slugify(title)
    filename = f"{dt.strftime('%H%M%S')}_{slug}.txt"
    filepath = date_folder / filename

    # Build a simple text representation
    summary = entry.get("summary", entry.get("description", ""))
    published = entry.get("published", entry.get("updated", ""))

    content = textwrap.dedent(
        f"""
        Provider : {provider}
        Title    : {title}
        Published: {published}
        Link     : {entry.get('link', '')}
        ID       : {entry.get('id', entry.get('guid', ''))}
        """
    ).strip()

    with filepath.open("w", encoding="utf-8") as f:
        f.write(content + "\n\n--- SUMMARY / DESCRIPTION ---\n")
        f.write(summary.strip() + "\n\n")
        f.write("--- FULL RAW ENTRY ---\n")
        # feedparser doesn't provide raw XML; we reconstruct important bits
        f.write(json.dumps(entry, indent=2, default=str))

    print(f"[+] Saved new entry for {provider}: {filename}")


# ---------------------------------------------------------------------------
# Main polling loop
# ---------------------------------------------------------------------------

def main():
    state = load_state()

    # Ensure every provider has a set in state
    for prov in FEEDS:
        state.setdefault(prov, set())

    try:
        while True:
            for provider, url in FEEDS.items():
                print(f"Checking {provider} …", flush=True)
                feed = feedparser.parse(url)
                if feed.bozo:
                    print(f"  ⚠️  Could not parse {url}: {feed.bozo_exception}")
                    continue

                for entry in feed.entries:
                    entry_id = entry.get("id") or entry.get("guid") or entry.get("link")
                    if not entry_id:
                        # Fallback: hash title + published
                        entry_id = f"{entry.get('title')}-{entry.get('published')}"

                    if entry_id not in state[provider]:
                        write_entry(provider, entry)
                        state[provider].add(entry_id)

            save_state(state)
            print(f"Sleeping {POLL_INTERVAL} seconds…\n", flush=True)
            time.sleep(POLL_INTERVAL)
    except KeyboardInterrupt:
        print("\nExiting. State saved.")
        save_state(state)


if __name__ == "__main__":
    main()
