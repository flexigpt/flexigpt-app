python3 - <<'PY'
from pathlib import Path

# Preserve persisted package-kind values and legacy filename aliases. The
# mechanical Go identifier rename above must not rewrite wire/storage strings.
restores = {
    '"agent-plugin"': '"agent-collection"',
    '"skill-plugin"': '"skill-collection"',
    '"mcp-plugin"': '"mcp-collection"',
    '"tool-plugin"': '"tool-collection"',
    '"plugin.yaml"': '"plugin.yaml"',
    '"plugin.yml"': '"plugin.yml"',
    '"plugin.json"': '"plugin.json"',
}
for path in Path("internal").rglob("*.go"):
    value = path.read_text()
    updated = value
    for old, new in restores.items():
        updated = updated.replace(old, new)
    if updated != value:
        path.write_text(updated)
PY
