#!/usr/bin/env python3
"""Extract logging UUID IDs from Go logging statements.

Scans repository Go files for:
  logging.Infof("uuid", ...)
  logging.Errorf("uuid", ...)
  logging.Fatalf("uuid", ...)

Writes extracted UUIDs (one per line, including duplicates in source order) to:
  extracted_logging_uuids.txt
"""

from __future__ import annotations

import re
import os
from pathlib import Path


LOGGING_CALL_UUID_RE = re.compile(
    r'logging\.(?:Infof|Errorf|Fatalf)\(\s*"'
    r'([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})"'
)


def main() -> None:
    script_dir = Path(__file__).resolve().parent
    repo_root = script_dir.parent
    output_path = script_dir / "extracted_logging_uuids.txt"
    duplicate_output_path = script_dir / "duplicate_logging_uuids.txt"

    # Always start from clean output files for a fresh extraction run.
    output_path.write_text("", encoding="utf-8")
    duplicate_output_path.write_text("", encoding="utf-8")

    extracted: list[str] = []

    for go_file in iter_go_files(repo_root):
        text = go_file.read_text(encoding="utf-8", errors="ignore")
        for match in LOGGING_CALL_UUID_RE.finditer(text):
            extracted.append(match.group(1).lower())

    output_path.write_text(
        "\n".join(extracted) + ("\n" if extracted else ""), encoding="utf-8"
    )
    print(f"wrote {len(extracted)} UUID entries to {output_path}")


def iter_go_files(repo_root: Path) -> list[Path]:
    files: list[Path] = []
    for dirpath, dirnames, filenames in os.walk(repo_root):
        # Skip VCS metadata and nested git dirs.
        dirnames[:] = [name for name in dirnames if name != ".git"]
        base = Path(dirpath)
        for filename in filenames:
            # Hard guard: extract from Go source files only.
            if not filename.endswith(".go"):
                continue
            files.append(base / filename)
    files.sort()
    return files


if __name__ == "__main__":
    main()
