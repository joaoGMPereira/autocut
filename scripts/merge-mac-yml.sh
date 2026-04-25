#!/bin/bash
# Merges two electron-builder latest-mac-{arch}.yml files into a single latest-mac.yml
# Usage: merge-mac-yml.sh <arm64.yml> <x64.yml> <output.yml>
set -euo pipefail

ARM64_YML="${1:?Usage: $0 <arm64.yml> <x64.yml> <output.yml>}"
X64_YML="${2:?}"
OUT_YML="${3:?}"

if [[ ! -f "$ARM64_YML" ]]; then
  echo "❌ Not found: $ARM64_YML" >&2; exit 1
fi
if [[ ! -f "$X64_YML" ]]; then
  echo "❌ Not found: $X64_YML" >&2; exit 1
fi

python3 - "$ARM64_YML" "$X64_YML" "$OUT_YML" << 'PYEOF'
import sys, re

def parse_files_block(text):
    """Extract the files: list entries as raw text blocks."""
    lines = text.splitlines()
    in_files = False
    entries = []
    current = []
    for line in lines:
        if re.match(r'^files:', line):
            in_files = True
            continue
        if in_files:
            if re.match(r'^  - ', line):
                if current:
                    entries.append('\n'.join(current))
                current = [line]
            elif re.match(r'^    ', line) and current:
                current.append(line)
            else:
                if current:
                    entries.append('\n'.join(current))
                    current = []
                in_files = False
    if current:
        entries.append('\n'.join(current))
    return entries

def get_field(text, field):
    m = re.search(rf'^{field}:\s*(.+)$', text, re.MULTILINE)
    return m.group(1).strip() if m else ''

arm64_text = open(sys.argv[1]).read()
x64_text   = open(sys.argv[2]).read()
out_path   = sys.argv[3]

version     = get_field(arm64_text, 'version')
path_val    = get_field(arm64_text, 'path')
sha512_val  = get_field(arm64_text, 'sha512')
release_date = get_field(arm64_text, 'releaseDate')

arm64_entries = parse_files_block(arm64_text)
x64_entries   = parse_files_block(x64_text)

all_entries = arm64_entries + x64_entries

out = f"version: {version}\nfiles:\n"
for entry in all_entries:
    out += entry + "\n"
out += f"path: {path_val}\nsha512: {sha512_val}\nreleaseDate: {release_date}\n"

open(out_path, 'w').write(out)
print(f"✔ {out_path} ({len(all_entries)} files: {len(arm64_entries)} arm64 + {len(x64_entries)} x64)")
PYEOF
