from __future__ import annotations

import pathlib
import re
import sys

from pypdf import PdfReader


def main() -> int:
    if len(sys.argv) < 2:
        print("Usage: extract_pelephone_spec.py <path-to-pdf>")
        return 2

    pdf_path = pathlib.Path(sys.argv[1]).resolve()
    if not pdf_path.exists():
        print(f"PDF not found: {pdf_path}")
        return 2

    repo_root = pathlib.Path(__file__).resolve().parents[1]
    out_dir = repo_root / "docs" / "_spec_extract"
    out_dir.mkdir(parents=True, exist_ok=True)

    reader = PdfReader(str(pdf_path))
    texts: list[str] = []
    for page in reader.pages:
        texts.append(page.extract_text() or "")

    full = "\n\n".join(texts)

    # Save full extracted text for manual inspection (request/response examples etc.)
    (out_dir / "pelephone_api_v1.5.2_fulltext.txt").write_text(full, encoding="utf-8")

    paths = sorted(set(re.findall(r"/ipa/apis/json/[A-Za-z0-9_/]+", full)))
    rpc_tokens = sorted(set(re.findall(r"\b(?:general|provisioning)/[A-Za-z][A-Za-z0-9_]*\b", full)))

    hook_lines: list[str] = []
    for line in full.splitlines():
        if re.search(r"hook|webhook|callback", line, re.IGNORECASE):
            line = line.strip()
            if line:
                hook_lines.append(line)

    (out_dir / "pelephone_api_v1.5.2_paths.txt").write_text("\n".join(paths), encoding="utf-8")
    (out_dir / "pelephone_api_v1.5.2_rpc_tokens.txt").write_text("\n".join(rpc_tokens), encoding="utf-8")
    (out_dir / "pelephone_api_v1.5.2_hook_lines.txt").write_text("\n".join(hook_lines), encoding="utf-8")

    print(f"pages={len(reader.pages)}")
    print(f"paths={len(paths)} rpc_tokens={len(rpc_tokens)} hook_lines={len(hook_lines)}")
    if paths:
        print("sample paths:")
        for p in paths[:50]:
            print("-", p)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
