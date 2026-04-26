#!/usr/bin/env python3
"""
Crawl RoboxStudio v2 — improved login handling.
"""

import json
import os
import time
from pathlib import Path

from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from webdriver_manager.chrome import ChromeDriverManager

BASE = "https://roboxstudio.baai.ac.cn"
PROJECT_ID = "431"

OUT_DIR = Path(__file__).parent / "roboxstudio_crawl"
SCREENSHOTS_DIR = OUT_DIR / "screenshots"
PAGES_DIR = OUT_DIR / "pages"

PAGES = [
    (f"/dashboard/collection?projectId={PROJECT_ID}", "collection"),
    (f"/dashboard/annotation?projectId={PROJECT_ID}", "annotation"),
    (f"/dashboard/data?projectId={PROJECT_ID}", "data"),
    (f"/dashboard/task?projectId={PROJECT_ID}", "task"),
    (f"/dashboard/export?projectId={PROJECT_ID}", "export"),
    (f"/dashboard/statistics?projectId={PROJECT_ID}", "statistics"),
    (f"/dashboard/setting?projectId={PROJECT_ID}", "setting"),
    (f"/dashboard/member?projectId={PROJECT_ID}", "member"),
    (f"/dashboard/device?projectId={PROJECT_ID}", "device"),
    (f"/dashboard/model?projectId={PROJECT_ID}", "model"),
    (f"/dashboard/overview?projectId={PROJECT_ID}", "overview"),
    (f"/dashboard/quality?projectId={PROJECT_ID}", "quality"),
    (f"/dashboard/review?projectId={PROJECT_ID}", "review"),
    (f"/dashboard/scene?projectId={PROJECT_ID}", "scene"),
    (f"/dashboard/robot?projectId={PROJECT_ID}", "robot"),
    (f"/dashboard/skill?projectId={PROJECT_ID}", "skill"),
    (f"/dashboard?projectId={PROJECT_ID}", "dashboard_home"),
]


def main():
    SCREENSHOTS_DIR.mkdir(parents=True, exist_ok=True)
    PAGES_DIR.mkdir(parents=True, exist_ok=True)

    opts = Options()
    opts.add_argument("--headless=new")
    opts.add_argument("--no-sandbox")
    opts.add_argument("--disable-dev-shm-usage")
    opts.add_argument("--window-size=1920,1200")
    opts.add_argument("--disable-gpu")

    print("🚀 Starting headless Chrome...")
    svc = Service(ChromeDriverManager().install())
    driver = webdriver.Chrome(service=svc, options=opts)
    driver.set_page_load_timeout(30)

    user = os.environ.get("ROBOX_USER", "")
    pwd = os.environ.get("ROBOX_PASS", "")

    # Step 1: Go to login page
    print("🔐 Navigating to login...")
    driver.get(BASE + "/userlogin")
    time.sleep(4)
    driver.save_screenshot(str(SCREENSHOTS_DIR / "00_login_page.png"))

    # Find and fill login form
    inputs = driver.find_elements(By.CSS_SELECTOR, "input")
    print(f"  Found {len(inputs)} inputs")

    username_field = None
    password_field = None
    for inp in inputs:
        t = (inp.get_attribute("type") or "").lower()
        p = (inp.get_attribute("placeholder") or "")
        if "密码" in p or t == "password":
            password_field = inp
        elif "账号" in p or "用户" in p or t == "text":
            if not username_field:
                username_field = inp

    if not username_field or not password_field:
        print("  ❌ Cannot find login fields")
        driver.quit()
        return

    # Clear and type slowly
    username_field.click()
    time.sleep(0.3)
    username_field.clear()
    for c in user:
        username_field.send_keys(c)
        time.sleep(0.05)

    password_field.click()
    time.sleep(0.3)
    password_field.clear()
    for c in pwd:
        password_field.send_keys(c)
        time.sleep(0.05)

    time.sleep(1)
    driver.save_screenshot(str(SCREENSHOTS_DIR / "01_login_filled.png"))

    # Click login button
    clicked = False
    for btn in driver.find_elements(By.CSS_SELECTOR, "button"):
        txt = btn.text.strip()
        if "登录" in txt or "login" in txt.lower():
            print(f"  Clicking: '{txt}'")
            btn.click()
            clicked = True
            break

    if not clicked:
        # Try pressing Enter
        password_field.send_keys(Keys.RETURN)
        print("  Pressed Enter to submit")

    # Wait for login to complete
    print("  Waiting for login...")
    time.sleep(8)

    driver.save_screenshot(str(SCREENSHOTS_DIR / "02_after_login.png"))
    current = driver.current_url
    print(f"  URL after login: {current}")

    # Check cookies/localStorage for auth token
    cookies = driver.get_cookies()
    print(f"  Cookies: {[c['name'] for c in cookies]}")

    local_storage = driver.execute_script(
        "var items = {}; for (var i = 0; i < localStorage.length; i++) {"
        "  var key = localStorage.key(i);"
        "  items[key] = localStorage.getItem(key).substring(0, 100);"
        "} return items;"
    )
    print(f"  LocalStorage keys: {list(local_storage.keys())}")

    # Try navigating to the target page directly
    print("\n🔍 Trying direct navigation to collection page...")
    driver.get(BASE + f"/dashboard/collection?projectId={PROJECT_ID}")
    time.sleep(5)
    driver.save_screenshot(str(SCREENSHOTS_DIR / "03_collection_attempt.png"))
    print(f"  URL: {driver.current_url}")

    # Check if we're on the actual page or redirected
    if "login" in driver.current_url.lower() or "userlogin" in driver.current_url.lower():
        print("  ❌ Still redirected to login. Login may have failed.")
        print("  Checking page source for error messages...")
        body_text = driver.find_element(By.TAG_NAME, "body").text[:500]
        print(f"  Body text: {body_text}")
        driver.quit()
        return

    print("  ✅ Login appears successful!")

    # Step 2: Crawl all pages
    index = {}
    page_num = 4

    for path, name in PAGES:
        url = BASE + path
        print(f"\n🔍 {name}: {path}")

        try:
            driver.get(url)
            time.sleep(5)

            if "login" in driver.current_url.lower() or "userlogin" in driver.current_url.lower():
                print(f"  ⚠️ Redirected to login, skipping")
                continue

            # Scroll to trigger lazy loading
            driver.execute_script("window.scrollTo(0, document.body.scrollHeight);")
            time.sleep(2)
            driver.execute_script("window.scrollTo(0, 0);")
            time.sleep(1)

            # Full page screenshot
            total_height = driver.execute_script("return document.body.scrollHeight")
            driver.set_window_size(1920, max(1200, min(total_height, 5000)))
            time.sleep(1)

            screenshot_name = f"{page_num:02d}_{name}.png"
            driver.save_screenshot(str(SCREENSHOTS_DIR / screenshot_name))
            driver.set_window_size(1920, 1200)

            title = driver.title or name

            # Extract sidebar navigation
            nav_items = []
            for sel in ["nav a", ".sidebar a", "[class*='menu'] a", "[class*='side'] a",
                        "[class*='nav'] a", ".ant-menu a", "[role='menuitem']"]:
                for el in driver.find_elements(By.CSS_SELECTOR, sel):
                    t = el.text.strip()
                    h = el.get_attribute("href") or ""
                    if t and 1 < len(t) < 50 and t not in [n[0] for n in nav_items]:
                        nav_items.append((t, h))

            # Extract text
            text_parts = []
            seen = set()
            for tag in ["h1", "h2", "h3", "h4", "h5", "p", "li", "td", "th",
                        "span", "label", "div"]:
                for el in driver.find_elements(By.TAG_NAME, tag):
                    t = el.text.strip()
                    if t and 2 < len(t) < 300 and t not in seen:
                        seen.add(t)
                        prefix = {"h1": "# ", "h2": "## ", "h3": "### ",
                                  "h4": "#### ", "li": "- "}.get(tag, "")
                        text_parts.append(prefix + t)

            # Save markdown
            md = f"# {title}\n\nSource: {url}\nScreenshot: `screenshots/{screenshot_name}`\n\n"
            if nav_items:
                md += "## Sidebar Navigation\n\n"
                for t, h in nav_items[:30]:
                    md += f"- [{t}]({h})\n"
                md += "\n"
            md += "## Page Content\n\n" + "\n\n".join(text_parts[:200])

            (PAGES_DIR / f"{name}.md").write_text(md, encoding="utf-8")

            index[name] = {
                "url": url,
                "title": title,
                "screenshot": screenshot_name,
                "text_blocks": len(text_parts),
            }

            print(f"  📸 {screenshot_name}")
            print(f"  ✅ {title} — {len(text_parts)} text blocks, {len(nav_items)} nav items")
            page_num += 1

        except Exception as e:
            print(f"  ❌ Error: {e}")

    driver.quit()

    (OUT_DIR / "index.json").write_text(
        json.dumps(index, indent=2, ensure_ascii=False), encoding="utf-8"
    )

    print(f"\n🏁 Done: {len(index)} pages crawled")


if __name__ == "__main__":
    main()
