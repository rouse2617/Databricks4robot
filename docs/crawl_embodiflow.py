#!/usr/bin/env python3
"""
Crawl EmbodiFlow docs (io-ai.tech/platform) — text + images.
Saves:
  docs/embodiflow_crawl/pages/*.md   — markdown text per page
  docs/embodiflow_crawl/images/*     — downloaded images
  docs/embodiflow_crawl/index.json   — page→images mapping
"""

import json
import os
import re
import time
from pathlib import Path
from urllib.parse import urljoin, urlparse

import requests
from bs4 import BeautifulSoup

BASE = "https://io-ai.tech"
SEED_URLS = [
    "/platform/guides/",
    "/platform/en/guides/",
    "/platform/en/guides/Modules/",
    "/platform/en/guides/Modules/overview/",
    "/platform/en/guides/Modules/data/",
    "/platform/en/guides/Modules/upload/",
    "/platform/en/guides/Modules/tasks/",
    "/platform/en/guides/Modules/export/",
    "/platform/en/guides/Modules/charts/",
    "/platform/en/guides/Modules/robots/",
    "/platform/en/guides/Modules/collections/",
    "/platform/en/guides/Modules/dictionaries/",
    "/platform/en/guides/Modules/projects/",
    "/platform/en/guides/Modules/users/",
    "/platform/en/guides/Modules/storage/",
    "/platform/en/guides/Modules/system/",
    "/platform/en/guides/Modules/trash/",
    "/platform/en/guides/Modules/skills/",
    "/platform/en/guides/Modules/plugins/",
    "/platform/en/guides/Modules/labs/",
    "/platform/en/guides/Modules/import/",
    "/platform/en/guides/Modules/signin/",
    "/platform/en/guides/Modules/training/",
    "/platform/en/guides/Modules/inference/",
    "/platform/en/guides/Annotation/",
    "/platform/en/guides/Pipeline/Annotation/",
    "/platform/en/guides/Pipeline/QC/",
    "/platform/en/guides/Pipeline/VideoQC/",
    "/platform/en/guides/Structure/",
    "/platform/en/guides/Workflow/",
    "/platform/en/guides/DaaS/",
    "/platform/en/guides/FAQ/",
    "/platform/en/plans/",
    # Chinese versions
    "/platform/guides/Modules/",
    "/platform/guides/Modules/overview/",
    "/platform/guides/Modules/data/",
    "/platform/guides/Modules/upload/",
    "/platform/guides/Modules/tasks/",
    "/platform/guides/Modules/export/",
    "/platform/guides/Modules/charts/",
    "/platform/guides/Modules/robots/",
    "/platform/guides/Annotation/",
    "/platform/guides/Pipeline/Annotation/",
    "/platform/guides/Structure/",
    "/platform/guides/Workflow/",
    "/platform/guides/FAQ/",
]

OUT_DIR = Path(__file__).parent / "embodiflow_crawl"
PAGES_DIR = OUT_DIR / "pages"
IMAGES_DIR = OUT_DIR / "images"

session = requests.Session()
session.headers.update({
    "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
                  "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
})

visited = set()
index = {}  # page_slug -> {url, title, images: [...]}


def slug(url: str) -> str:
    """Turn URL path into a safe filename slug."""
    p = urlparse(url).path.strip("/")
    return re.sub(r"[^a-zA-Z0-9]+", "_", p) or "index"


def download_image(img_url: str) -> str | None:
    """Download image, return local filename or None."""
    try:
        parsed = urlparse(img_url)
        ext = Path(parsed.path).suffix or ".png"
        name = slug(img_url) + ext
        dest = IMAGES_DIR / name
        if dest.exists():
            return name
        r = session.get(img_url, timeout=30)
        if r.status_code == 200 and len(r.content) > 500:
            dest.write_bytes(r.content)
            print(f"  📷 {name} ({len(r.content)//1024}KB)")
            return name
    except Exception as e:
        print(f"  ⚠️  img fail: {img_url} — {e}")
    return None


def crawl_page(url: str):
    """Fetch one page, extract text + images."""
    if url in visited:
        return
    visited.add(url)

    try:
        r = session.get(url, timeout=30)
        if r.status_code != 200:
            print(f"  ❌ {r.status_code} {url}")
            return
    except Exception as e:
        print(f"  ❌ {url} — {e}")
        return

    soup = BeautifulSoup(r.text, "html.parser")

    # Find main content area
    main = (
        soup.find("article")
        or soup.find("main")
        or soup.find("div", class_=re.compile(r"(content|docs|markdown)", re.I))
        or soup.body
    )
    if not main:
        return

    title = ""
    h1 = soup.find("h1")
    if h1:
        title = h1.get_text(strip=True)

    page_slug = slug(url)
    page_images = []

    # Extract images
    for img in main.find_all("img"):
        src = img.get("src") or img.get("data-src") or ""
        if not src:
            continue
        img_url = urljoin(url, src)
        local = download_image(img_url)
        if local:
            page_images.append({
                "local": local,
                "src": img_url,
                "alt": img.get("alt", ""),
            })

    # Also check for background images in style attrs and picture/source tags
    for source in main.find_all("source"):
        srcset = source.get("srcset", "")
        if srcset:
            img_url = urljoin(url, srcset.split(",")[0].strip().split(" ")[0])
            local = download_image(img_url)
            if local:
                page_images.append({"local": local, "src": img_url, "alt": ""})

    # Extract text content
    text_parts = []
    for el in main.find_all(["h1", "h2", "h3", "h4", "p", "li", "td", "th", "pre", "code"]):
        t = el.get_text(strip=True)
        if t:
            prefix = ""
            if el.name == "h1":
                prefix = "# "
            elif el.name == "h2":
                prefix = "## "
            elif el.name == "h3":
                prefix = "### "
            elif el.name == "h4":
                prefix = "#### "
            elif el.name == "li":
                prefix = "- "
            text_parts.append(prefix + t)

    md_content = f"# {title}\n\nSource: {url}\n\n"
    md_content += "\n\n".join(text_parts)
    if page_images:
        md_content += "\n\n## Images on this page\n\n"
        for img in page_images:
            md_content += f"- ![{img['alt']}](../images/{img['local']})\n"

    (PAGES_DIR / f"{page_slug}.md").write_text(md_content, encoding="utf-8")

    index[page_slug] = {
        "url": url,
        "title": title,
        "image_count": len(page_images),
        "images": page_images,
    }

    print(f"✅ {title or page_slug} — {len(page_images)} images")

    # Discover more links within the docs
    for a in main.find_all("a", href=True):
        href = a["href"]
        full = urljoin(url, href)
        if full.startswith(BASE) and "/platform/" in full and full not in visited:
            if not any(full.endswith(ext) for ext in [".pdf", ".zip", ".tar.gz"]):
                crawl_page(full)
                time.sleep(0.3)


def main():
    PAGES_DIR.mkdir(parents=True, exist_ok=True)
    IMAGES_DIR.mkdir(parents=True, exist_ok=True)

    print(f"🕷️  Crawling EmbodiFlow docs → {OUT_DIR}\n")

    for path in SEED_URLS:
        url = BASE + path
        crawl_page(url)
        time.sleep(0.3)

    # Save index
    (OUT_DIR / "index.json").write_text(
        json.dumps(index, indent=2, ensure_ascii=False), encoding="utf-8"
    )

    total_images = sum(v["image_count"] for v in index.values())
    print(f"\n🏁 Done: {len(index)} pages, {total_images} images")
    print(f"   Pages: {PAGES_DIR}")
    print(f"   Images: {IMAGES_DIR}")


if __name__ == "__main__":
    main()
