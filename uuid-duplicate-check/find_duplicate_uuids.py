#!/usr/bin/env python3
"""Find duplicate UUIDs from extracted_logging_uuids.txt.

Reads:
  extracted_logging_uuids.txt

Writes duplicates (uuid,count) to:
  duplicate_logging_uuids.txt
"""

from __future__ import annotations

from collections import Counter
from pathlib import Path


def main() -> None:
    script_dir = Path(__file__).resolve().parent
    input_path = script_dir / "extracted_logging_uuids.txt"
    output_path = script_dir / "duplicate_logging_uuids.txt"

    if not input_path.exists():
        raise FileNotFoundError(
            f"input file not found: {input_path}. Run extract_logging_uuids.py first."
        )

    uuids = [
        line.strip().lower()
        for line in input_path.read_text(encoding="utf-8").splitlines()
        if line.strip()
    ]
    counts = Counter(uuids)

    duplicates = [(uuid, count) for uuid, count in counts.items() if count > 1]
    duplicates.sort(key=lambda item: (-item[1], item[0]))

    lines = [f"{uuid},{count}" for uuid, count in duplicates]
    output_path.write_text("\n".join(lines) + ("\n" if lines else ""), encoding="utf-8")
    print(f"wrote {len(duplicates)} duplicate UUIDs to {output_path}")


if __name__ == "__main__":
    main()
