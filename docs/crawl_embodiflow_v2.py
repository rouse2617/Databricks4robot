#!/usr/bin/env python3
"""
Crawl EmbodiFlow docs with Selenium (headless Chrome) to capture
JS-rendered content and images.

Output:
  docs/embodiflow_crawl/pages/*.md
  docs/embodiflow_crawl/images/*.png / *.webp / *.jpg
  docs/embodiflow_crawl/index.json
"""

import json
import os
import re
import time
import hashlib
import requests
from pathlib import Path
from urllib.parse import urljoin, urlparse

from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from webdriver_manager.chrome import ChromeDriverManager

# ── Config ──────────────────────────────────────────────────────
BASE = "https://io-ai.tech"

# Focus on the Modules pages (Chinese) which have the screenshots
PAGES = [
    # Chinese guides (main target — has screenshots)
    "/platform/guides/",
    "/platform/guides/Modules/",
    "/platform/guides/Modules/overview/",
    "/platform/guides/Modules/data/",
    "/platform/guides/Modules/upload/",
    "/platform/guides/Modules/tasks/",
    "/platform/guides/Modules/export/",
    "/platform/guides/Modules/charts/",
    "/platform/guides/Modules/robots/",
    "/platform/guides/Modules/collections/",
    "/platform/guides/Modules/dictionaries/",
    "/platform/guides/Modules/projects/",
    "/platform/guides/Modules/users/",
    "/platform/guides/Modules/storage/",
    "/platform/guides/Modules/system/",
    "/platform/guides/Modules/trash/",
    "/platform/guides/Modules/skills/",
    "/platform/guides/Modules/plugins/",
    "/platform/guides/Modules/labs/",
    "/platform/guides/Modules/import/",
    "/platform/guides/Modules/signin/",
    "/platform/guides/Modules/training/",
    "/platform/guides/Modules/inference/",
    "/platform/guides/Annotation/",
    "/platform/guides/Pipeline/Annotation/",
    "/platform/guides/Pipeline/QC/",
    "/platform/guides/Pipeline/VideoQC/",
    "/platform/guides/Structure/",
    "/platform/guides/Workflow/",
    "/platform/guides/FAQ/",
    # English guides
    "/platform/en/guides/",
    "/platform/en/guides/Modules/",
    "/platform/en/guides/Modules/overview/",
    "/platform/en/guides/Modules/data/",
    "/platform/en/guides/Modules/upload/",
    "/platform/en/guides/Modules/tasks/",
    "/platform/en/guides/Modules/export/",
    "/platform/en/guides/Modules/charts/",
    "/platform/en/guides/Modules/robots/",
    "/platform/en/guides/Annotation/",
    "/platform/en/guides/Pipeline/Annotation/",
    "/platform/en/guides/Structure/",
    "/platform/en/guides/Workflow/",
    "/platform/en/guides/DaaS/",
    "/platform/en/guides/FAQ/",
    "/platform/en/plans/",
]

OUT_DIR = Path(__file__).parent / "embodiflow_crawl"
PAGES_DIR = OUT_DIR / "pages"
IMAGES_DIR = OUT_DIR / "images"

# ── Helpers ─────────────────────────────────────────────────────
http = requests.Session()
http.headers["User-Agent"] = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
    "AppleWebKit/537.36 Chrome/120 Safari/537.36"
)


def slug(url: str) -> str:
    p = urlparse(url).path.strip("/")
    return re.sub(r"[^a-zA-Z0-9]+", "_", p) or "index"


def download_image(img_url: str) -> str | None:
    """Download an image via requests, return local filename."""
    try:
        parsed = urlparse(img_url)
        ext = Path(parsed.path).suffix or ".png"
        h = hashlib.md5(img_url.encode()).hexdigest()[:12]
        name = slug(img_url)[-60:] + "_" + h + ext
        dest = IMAGES_DIR / name
        if dest.exists() and dest.stat().st_size > 500:
            return name
        r = http.get(img_url, timeout=10, stream=True)
        if r.status_code == 200:
            content = b""
            for chunk in r.iter_content(chunk_size=65536):
                content += chunk
                if len(content) > 5 * 1024 * 1024:  # 5MB max
                    break
            if len(content) > 500:
                dest.write_bytes(content)
                print(f"    📷 {name} ({len(content)//1024}KB)")
                return name
    except Exception as e:
        print(f"    ⚠️  skip: {img_url[:60]} — {type(e).__name__}")
    return None


# ── Main ────────────────────────────────────────────────────────
def main():
    PAGES_DIR.mkdir(parents=True, exist_ok=True)
    IMAGES_DIR.mkdir(parents=True, exist_ok=True)

    # Setup headless Chrome
    opts = Options()
    opts.add_argument("--headless=new")
    opts.add_argument("--no-sandbox")
    opts.add_argument("--disable-dev-shm-usage")
    opts.add_argument("--window-size=1920,1080")
    opts.add_argument("--disable-gpu")

    print("🚀 Starting headless Chrome...")
    svc = Service(ChromeDriverManager().install())
    driver = webdriver.Chrome(service=svc, options=opts)
    driver.set_page_load_timeout(30)

    index = {}
    visited = set()

    for path in PAGES:
        url = BASE + path
        if url in visited:
            continue
        visited.add(url)

        print(f"\n🔍 {path}")
        try:
            driver.get(url)
            # Wait for content to render
            time.sleep(3)
            # Try to wait for main content
            try:
                WebDriverWait(driver, 8).until(
                    EC.presence_of_element_located((By.CSS_SELECTOR, "article, main, .theme-doc-markdown"))
                )
            except Exception:
                pass
            # Scroll to bottom to trigger lazy-loaded images
            driver.execute_script("window.scrollTo(0, document.body.scrollHeight);")
            time.sleep(2)
            driver.execute_script("window.scrollTo(0, 0);")
            time.sleep(1)
        except Exception as e:
            print(f"  ❌ load fail: {e}")
            continue

        # Get page title
        title = driver.title or ""

        # Collect all images
        page_images = []
        img_elements = driver.find_elements(By.TAG_NAME, "img")
        print(f"  Found {len(img_elements)} <img> tags")

        for img in img_elements:
            src = img.get_attribute("src") or img.get_attribute("data-src") or ""
            if not src or src.startswith("data:"):
                continue
            img_url = urljoin(url, src)
            local = download_image(img_url)
            if local:
                alt = img.get_attribute("alt") or ""
                page_images.append({
                    "local": local,
                    "src": img_url,
                    "alt": alt,
                })

        # Also check for images in <picture><source> tags
        sources = driver.find_elements(By.CSS_SELECTOR, "picture source")
        for source in sources:
            srcset = source.get_attribute("srcset") or ""
            if srcset:
                img_url = urljoin(url, srcset.split(",")[0].strip().split(" ")[0])
                local = download_image(img_url)
                if local:
                    page_images.append({"local": local, "src": img_url, "alt": ""})

        # Extract text content
        text_parts = []
        for tag in ["h1", "h2", "h3", "h4", "p", "li", "td", "th"]:
            for el in driver.find_elements(By.TAG_NAME, tag):
                t = el.text.strip()
                if t and len(t) > 1:
                    prefix = {"h1": "# ", "h2": "## ", "h3": "### ",
                              "h4": "#### ", "li": "- "}.get(tag, "")
                    text_parts.append(prefix + t)

        # Save markdown
        page_slug = slug(url)
        md = f"# {title}\n\nSource: {url}\n\n"
        md += "\n\n".join(dict.fromkeys(text_parts))  # dedupe while preserving order
        if page_images:
            md += "\n\n---\n## Screenshots\n\n"
            for img in page_images:
                md += f"![{img['alt']}](../images/{img['local']})\n\n"

        (PAGES_DIR / f"{page_slug}.md").write_text(md, encoding="utf-8")

        index[page_slug] = {
            "url": url,
            "title": title,
            "image_count": len(page_images),
            "images": page_images,
        }

        print(f"  ✅ {title} — {len(page_images)} images, {len(text_parts)} text blocks")

    driver.quit()

    # Save index
    (OUT_DIR / "index.json").write_text(
        json.dumps(index, indent=2, ensure_ascii=False), encoding="utf-8"
    )

    total_imgs = sum(v["image_count"] for v in index.values())
    print(f"\n🏁 Done: {len(index)} pages, {total_imgs} total image refs")
    print(f"   Unique images: {len(list(IMAGES_DIR.glob('*')))}")
    print(f"   Pages: {PAGES_DIR}")
    print(f"   Images: {IMAGES_DIR}")


if __name__ == "__main__":
    main()
