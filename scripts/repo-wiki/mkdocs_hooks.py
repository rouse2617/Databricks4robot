"""MkDocs build hooks for the repo-wiki.

Keeps the Qoder markdown source unchanged while making it render well in
Material for MkDocs:

- `<cite>...</cite>` blocks become `<div class="cite" markdown>` so the
  Markdown list inside renders.

Referenced by mkdocs.yml `hooks:`. Source of truth stays docs/repo-wiki/*.md.
"""
from __future__ import annotations


def on_page_markdown(markdown: str, *, page=None, config=None, files=None) -> str:
    if "<cite>" in markdown:
        markdown = markdown.replace("<cite>", '<div class="cite" markdown>').replace(
            "</cite>", "</div>"
        )
    return markdown
