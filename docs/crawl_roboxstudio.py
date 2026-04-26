#!/usr/bin/env python3
"""
Crawl RoboxStudio (roboxstudio.baai.ac.cn) after login.
Takes screenshots + extracts text from key pages.

Output:
  docs/roboxstudio_crawl/screenshots/*.png  — full-page screenshots
  docs/roboxstudio_crawl/pages/*.md         — extracted text
"""

import json
import os
import re
import time
from pathlib import Path

from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from webdriver_manager.chrome import ChromeDriverManager

BASE = "https://roboxstudio.baai.ac.cn"
PROJECT_ID = "431"

OUT_DIR = Path(__file__).parent / "roboxstudio_crawl"
SCREENSHOTS_DIR = OUT_DIR / "screenshots"
PAGES_DIR = OUT_DIR / "pages"

# Pages to visit after login
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
    opts.add_argument("--force-device-scale-factor=1")

    print("🚀 Starting headless Chrome...")
    svc = Service(ChromeDriverManager().install())
    driver = webdriver.Chrome(service=svc, options=opts)
    driver.set_page_load_timeout(30)

    # Step 1: Login
    print("🔐 Logging in...")
    driver.get(BASE)
    time.sleep(3)

    # Take screenshot of login page
    driver.save_screenshot(str(SCREENSHOTS_DIR / "00_login_page.png"))
    print(f"  📸 Login page screenshot saved")

    # Try to find login form
    try:
        # Look for common login form patterns
        inputs = driver.find_elements(By.CSS_SELECTOR, "input")
        print(f"  Found {len(inputs)} input fields")
        for inp in inputs:
            inp_type = inp.get_attribute("type") or ""
            inp_name = inp.get_attribute("name") or ""
            inp_placeholder = inp.get_attribute("placeholder") or ""
            print(f"    input: type={inp_type}, name={inp_name}, placeholder={inp_placeholder}")

        # Try username/password fields
        username_field = None
        password_field = None

        for inp in inputs:
            t = (inp.get_attribute("type") or "").lower()
            n = (inp.get_attribute("name") or "").lower()
            p = (inp.get_attribute("placeholder") or "").lower()
            if t == "password" or "password" in n or "密码" in p:
                password_field = inp
            elif t in ("text", "email", "") or "user" in n or "account" in n or "用户" in p or "账号" in p:
                if not username_field:
                    username_field = inp

        if username_field and password_field:
            print("  Found username and password fields")
            username_field.clear()
            username_field.send_keys(os.environ.get("ROBOX_USER", ""))
            time.sleep(0.5)
            password_field.clear()
            password_field.send_keys(os.environ.get("ROBOX_PASS", ""))
            time.sleep(0.5)

            # Take screenshot before clicking login
            driver.save_screenshot(str(SCREENSHOTS_DIR / "01_login_filled.png"))

            # Find and click login button
            buttons = driver.find_elements(By.CSS_SELECTOR, "button")
            for btn in buttons:
                txt = btn.text.strip().lower()
                if "login" in txt or "登录" in txt or "sign in" in txt or "log in" in txt:
                    print(f"  Clicking login button: '{btn.text}'")
                    btn.click()
                    break
            else:
                # Try submit
                forms = driver.find_elements(By.TAG_NAME, "form")
                if forms:
                    forms[0].submit()
                else:
                    # Click first button
                    if buttons:
                        buttons[0].click()

            time.sleep(5)
            driver.save_screenshot(str(SCREENSHOTS_DIR / "02_after_login.png"))
            print(f"  📸 After login screenshot saved")
            print(f"  Current URL: {driver.current_url}")
        else:
            print("  ⚠️ Could not find login fields, trying direct navigation")

    except Exception as e:
        print(f"  ⚠️ Login error: {e}")

    # Step 2: Navigate to each page
    index = {}
    page_num = 3

    for path, name in PAGES:
        url = BASE + path
        print(f"\n🔍 {name}: {path}")

        try:
            driver.get(url)
            time.sleep(4)

            # Scroll to load lazy content
            driver.execute_script("window.scrollTo(0, document.body.scrollHeight);")
            time.sleep(2)
            driver.execute_script("window.scrollTo(0, 0);")
            time.sleep(1)

            # Check if redirected to login
            if "login" in driver.current_url.lower() or "signin" in driver.current_url.lower():
                print(f"  ⚠️ Redirected to login, skipping")
                continue

            # Take full page screenshot
            # Get page height for full screenshot
            total_height = driver.execute_script("return document.body.scrollHeight")
            driver.set_window_size(1920, max(1200, min(total_height, 5000)))
            time.sleep(1)

            screenshot_name = f"{page_num:02d}_{name}.png"
            driver.save_screenshot(str(SCREENSHOTS_DIR / screenshot_name))
            print(f"  📸 {screenshot_name}")

            # Reset window size
            driver.set_window_size(1920, 1200)

            # Extract page title
            title = driver.title or name

            # Extract text content
            text_parts = []
            for tag in ["h1", "h2", "h3", "h4", "p", "li", "td", "th", "span", "div"]:
                for el in driver.find_elements(By.TAG_NAME, tag):
                    t = el.text.strip()
                    if t and 2 < len(t) < 500:
                        prefix = {"h1": "# ", "h2": "## ", "h3": "### ",
                                  "h4": "#### ", "li": "- "}.get(tag, "")
                        text_parts.append(prefix + t)

            # Dedupe
            seen = set()
            unique_parts = []
            for p in text_parts:
                if p not in seen:
                    seen.add(p)
                    unique_parts.append(p)

            # Extract navigation items (sidebar)
            nav_items = []
            for el in driver.find_elements(By.CSS_SELECTOR, "nav a, .sidebar a, .menu a, [class*='nav'] a, [class*='menu'] a, [class*='side'] a"):
                t = el.text.strip()
                h = el.get_attribute("href") or ""
                if t and len(t) < 50:
                    nav_items.append(f"- [{t}]({h})")

            # Save markdown
            md = f"# {title}\n\nSource: {url}\nScreenshot: `screenshots/{screenshot_name}`\n\n"
            if nav_items:
                md += "## Navigation\n\n" + "\n".join(nav_items[:30]) + "\n\n"
            md += "## Page Content\n\n" + "\n\n".join(unique_parts[:200])

            (PAGES_DIR / f"{name}.md").write_text(md, encoding="utf-8")

            index[name] = {
                "url": url,
                "title": title,
                "screenshot": screenshot_name,
                "text_blocks": len(unique_parts),
                "nav_items": len(nav_items),
            }

            print(f"  ✅ {title} — {len(unique_parts)} text blocks")
            page_num += 1

        except Exception as e:
            print(f"  ❌ Error: {e}")

    driver.quit()

    # Save index
    (OUT_DIR / "index.json").write_text(
        json.dumps(index, indent=2, ensure_ascii=False), encoding="utf-8"
    )

    print(f"\n🏁 Done: {len(index)} pages crawled")
    print(f"   Screenshots: {SCREENSHOTS_DIR}")
    print(f"   Pages: {PAGES_DIR}")


if __name__ == "__main__":
    main()
