#!/usr/bin/env python3
"""Bulk-insert 10,000 complex mock assets into PG, then ES reindex + perf tests."""

import json
import random
import string
import time
import subprocess
import concurrent.futures
import requests
from datetime import datetime, timezone

random.seed(99)

BASE_URL = "http://localhost:8080/api/v1"
TOKEN = "dev-token"
HEADERS = {"X-Databrew-Token": TOKEN, "Content-Type": "application/json"}
ADMIN_HEADERS = {"X-Databrew-Token": TOKEN, "X-Admin-Token": TOKEN}

OWNERS = [
    "alice-team", "bob-team", "charlie-team", "diana-team", "eve-team",
    "dev-team", "ml-team", "qa-team", "ops-team", "research-team"
]
REVIEWERS = [
    "reviewer-alice", "reviewer-bob", "reviewer-charlie",
    "reviewer-diana", "reviewer-eve", "reviewer-frank",
    "reviewer-grace", "reviewer-henry", "reviewer-ivy", "reviewer-jack"
]
ASSET_TYPES = ["raw_mcap", "segment"]
LIFECYCLE_STATES = ["created", "processing", "ready", "delivered", "archived"]

def rand_id():
    return ''.join(random.choices(string.ascii_letters + string.digits, k=8))

def api_post(path, data):
    resp = requests.post(f"{BASE_URL}{path}", headers=HEADERS, json=data, timeout=30)
    return resp

def admin_post(path, data):
    resp = requests.post(f"{BASE_URL}{path}", headers=ADMIN_HEADERS, json=data, timeout=60)
    return resp

# Step 1: Create mcap files via API
print("=== Step 1: Creating 26 mcap files via API ===")
MCAP_FILE_IDS = []
for i in range(26):
    mid = rand_id()
    resp = api_post("/mcap-files", {
        "file_name": f"perf_test_{i}.mcap",
        "file_size_bytes": random.randint(100000, 10000000),
        "storage_uri": f"gs://perf-bucket/perf_test_{i}.mcap",
        "source": "upload"
    })
    j = resp.json()
    if resp.status_code in (200, 201) and "mcap_file_id" in j:
        MCAP_FILE_IDS.append(j["mcap_file_id"])
        print(f"  mcap {i}: {j['mcap_file_id']}")
    else:
        print(f"  mcap {i}: FAIL {resp.status_code} {str(j)[:200]}")

print(f"  Total mcap files: {len(MCAP_FILE_IDS)}")

if not MCAP_FILE_IDS:
    print("ERROR: No mcap files created!")
    exit(1)

# Step 2: Generate 10k asset SQL
TOTAL = 10000
BATCH_SIZE = 500

print(f"\n=== Step 2: Generating {TOTAL} asset SQL ===")
all_ids = []
for i in range(TOTAL):
    aid = rand_id()
    while True:
        # Check no collision with existing assets (mcap file assets have same IDs)
        check = subprocess.run(
            ["psql", "-U", "rick", "-d", "cyber_databrew_dev", "-t", "-c",
             f"SELECT 1 FROM assets WHERE asset_id = '{aid}'"],
            capture_output=True, text=True, timeout=5
        )
        if check.stdout.strip() != "1":
            break
        aid = rand_id()
    all_ids.append(aid)

sql_lines = ["BEGIN;"]

logical_vals = []
for aid in all_ids:
    logical_vals.append(f"('{aid}', 'asset', 'mock-perf')")
# Insert logical_assets in chunks of 1000
for i in range(0, len(logical_vals), 1000):
    chunk = logical_vals[i:i+1000]
    sql_lines.append(
        "INSERT INTO logical_assets (logical_asset_id, asset_type, owner) VALUES\n" +
        ",\n".join(chunk) + ";"
    )

asset_columns = [
    "asset_id", "mcap_file_id", "start_timestamp_ns", "end_timestamp_ns",
    "segment_locator", "asset_type", "lifecycle_state", "is_deleted",
    "duration_ms", "owner", "reviewer", "storage_uri", "thumb_uri",
    "retention_tier", "asset_level", "parent_asset_id", "delivery_count",
    "metadata", "files", "tenant_id", "project_id",
    "created_at", "updated_at", "logical_asset_id"
]

now = datetime.now(timezone.utc)
asset_batches = [all_ids[i:i+BATCH_SIZE] for i in range(0, len(all_ids), BATCH_SIZE)]
segment_parents = list(MCAP_FILE_IDS)  # seed with mcap file asset IDs

for batch_ids in asset_batches:
    values = []
    for aid in batch_ids:
        atype = random.choices(ASSET_TYPES, weights=[0.7, 0.3])[0]
        owner = random.choice(OWNERS)
        reviewer = random.choice(REVIEWERS)
        mcap_id = random.choice(MCAP_FILE_IDS)
        ts_start = 1700000000000000000 + random.randint(0, 10000000000000)
        duration = random.randint(10000000000, 60000000000)
        ts_end = ts_start + duration

        lc = random.choices(LIFECYCLE_STATES, weights=[0.15, 0.05, 0.70, 0.05, 0.05])[0]
        is_deleted = "true" if lc == "archived" else "false"

        parent_id = "NULL"
        if atype == "segment" and segment_parents:
            parent_id = "'" + random.choice(segment_parents) + "'"

        metadata = {}
        if random.random() < 0.7:
            metadata = {
                "source_stream": random.choice(["left", "right", "front", "rear", "top"]),
                "resolution": random.choice(["1080p", "4K", "720p"]),
                "fps": random.choice([30, 60, 120]),
                "camera_id": random.choice(["cam-01", "cam-02", "cam-03", "cam-A", "cam-B"]),
                "frame_count": random.randint(1000, 100000),
            }
        files_j = json.dumps({atype: mcap_id}).replace("'", "''")
        meta_j = json.dumps(metadata).replace("'", "''")

        retention_tier = random.choices(["standard", "cold", "hot", "archive"], weights=[0.7, 0.1, 0.1, 0.1])[0]
        asset_level = 0 if atype == "raw_mcap" else random.randint(1, 3)
        delivery_count = random.randint(0, 10)
        tenant_id = random.choice(["", "tenant-01", "tenant-02"])
        project_id = random.choice(["", "project-alpha", "project-beta", "project-gamma"])
        tenant_sql = "'" + tenant_id + "'" if tenant_id else "NULL"
        project_sql = "'" + project_id + "'" if project_id else "NULL"

        created = now.timestamp() - random.randint(3600, 86400*30)
        created_ts = datetime.fromtimestamp(created, tz=timezone.utc).isoformat()
        updated_ts = datetime.fromtimestamp(created + random.randint(60, 3600), tz=timezone.utc).isoformat()
        locator = ''.join(random.choices('0123456789abcdef', k=40))

        values.append(
            f"('{aid}', '{mcap_id}', {ts_start}, {ts_end}, '{locator}', "
            f"'{atype}', '{lc}', {is_deleted}, {duration}, "
            f"'{owner}', '{reviewer}', '', '', '{retention_tier}', {asset_level}, "
            f"{parent_id}, {delivery_count}, "
            f"'{meta_j}'::jsonb, '{files_j}'::jsonb, "
            f"{tenant_sql}, {project_sql}, "
            f"'{created_ts}', '{updated_ts}', "
            f"'{aid}')"
        )

        if atype == "raw_mcap":
            segment_parents.append(aid)

    cols = ", ".join(asset_columns)
    sql_lines.append(
        "INSERT INTO assets (" + cols + ") VALUES\n" + ",\n".join(values) + ";"
    )

sql_lines.append("COMMIT;")

sql_path = "/tmp/mock_10k.sql"
with open(sql_path, "w") as f:
    f.write("\n".join(sql_lines))

print(f"  SQL file: {sql_path} ({len(sql_lines)} lines)")

# Execute SQL
print("\n=== Step 3: Executing SQL insert ===")
t0 = time.time()
result = subprocess.run(
    ["psql", "-U", "rick", "-d", "cyber_databrew_dev", "-f", sql_path],
    capture_output=True, text=True, timeout=120
)
t_sql = time.time() - t0
if result.returncode != 0:
    print(f"  SQL ERROR: {result.stderr[:1000]}")
    print(f"  First errors in stdout:")
    for line in result.stdout.split("\n")[:10]:
        if "ERROR" in line:
            print(f"  {line}")
else:
    print(f"  SQL OK in {t_sql:.2f}s")

# Verify PG count
print("\n=== Step 4: Verify PG count ===")
result = subprocess.run(
    ["psql", "-U", "rick", "-d", "cyber_databrew_dev", "-t", "-c",
     "SELECT COUNT(*) FROM assets"],
    capture_output=True, text=True, timeout=10
)
total_pg = int(result.stdout.strip())
print(f"  Total PG: {total_pg}")

# ES reindex
print("\n=== Step 5: ES reindex ===")
t0 = time.time()
resp = admin_post("/admin/search/reindex", {})
t_reindex = time.time() - t0
rj = resp.json()
print(f"  Reindex: indexed={rj.get('indexed','?')}, deleted={rj.get('deleted','?')}, "
      f"failed={rj.get('failed','?')}, duration={rj.get('duration_ms','?')}ms ({t_reindex:.2f}s)")

# Wait a sec and check ES
time.sleep(1)
count_resp = requests.post("http://localhost:9200/assets/_count",
                           headers={"Content-Type": "application/json"}, timeout=10)
total_es = count_resp.json().get("count", 0) if count_resp.ok else -1
print(f"  Total ES: {total_es}")
print(f"  PG-ES gap: {total_pg - total_es}")

# Performance tests
print("\n" + "="*60)
print("=== PERFORMANCE TESTS ===")
print("="*60)

def measure(label, fn, n=20):
    times = []
    for i in range(n):
        t0 = time.time()
        fn()
        times.append((time.time() - t0) * 1000)
    times.sort()
    avg = sum(times) / len(times)
    p50 = times[len(times)//2]
    p99 = times[int(len(times)*0.99)]
    print(f"  {label:45s}  avg={avg:8.1f}ms  p50={p50:8.1f}ms  p99={p99:8.1f}ms  (n={n})")

print("\n--- 7a. Single asset GET by ID (PG) ---")
test_id = random.choice(all_ids)
measure("GET /api/v1/assets/{id}", lambda: requests.get(
    f"{BASE_URL}/assets/{test_id}", headers=HEADERS, timeout=10), n=30)

print("\n--- 7b. queries/run filter (PG) ---")
measure("queries/run filter owner=dev-team", lambda: requests.post(
    f"{BASE_URL}/queries/run",
    headers=HEADERS,
    json={"schema_version": "v1", "scope": {"resource": "assets"},
          "fields": ["asset_id", "asset_type", "owner"],
          "filters": [{"field": "owner", "op": "eq", "val": "dev-team"}],
          "limit": 100}, timeout=30), n=20)

print("\n--- 7c. queries/run with facets (ES) ---")
measure("queries/run facets asset_type", lambda: requests.post(
    f"{BASE_URL}/queries/run",
    headers=HEADERS,
    json={"schema_version": "v1", "scope": {"resource": "assets"},
          "fields": ["asset_id", "asset_type"],
          "facets": [{"field": "asset_type", "type": "terms", "size": 10}],
          "limit": 20}, timeout=30), n=20)

print("\n--- 7d. queries/run multi-filter (tag-like) ---")
def multi_filter():
    rvw = random.choice(REVIEWERS)
    return requests.post(
        f"{BASE_URL}/queries/run",
        headers=HEADERS,
        json={"schema_version": "v1", "scope": {"resource": "assets"},
              "fields": ["asset_id", "owner", "reviewer", "asset_type"],
              "filters": [
                  {"field": "reviewer", "op": "eq", "val": rvw},
                  {"field": "asset_type", "op": "eq", "val": "raw_mcap"}
              ],
              "limit": 50}, timeout=30)
measure("queries/run reviewer + asset_type", multi_filter, n=20)

print("\n--- 7e. queries/run keyword search ---")
measure("queries/run keyword 'team'", lambda: requests.post(
    f"{BASE_URL}/queries/run",
    headers=HEADERS,
    json={"schema_version": "v1", "scope": {"resource": "assets"},
          "fields": ["asset_id", "owner"],
          "keyword": "team",
          "limit": 50}, timeout=30), n=20)

print("\n--- 7f. Time-range query ---")
measure("queries/run time-range filter", lambda: requests.post(
    f"{BASE_URL}/queries/run",
    headers=HEADERS,
    json={"schema_version": "v1", "scope": {"resource": "assets"},
          "fields": ["asset_id", "created_at"],
          "filters": [
              {"field": "created_at", "op": "gte", "val": "2026-05-01T00:00:00Z"},
              {"field": "created_at", "op": "lte", "val": "2026-05-31T00:00:00Z"}
          ],
          "limit": 100}, timeout=30), n=15)

print("\n--- 7g. Concurrency: 100 GETs (20 workers) ---")
workers = 20
total_reqs = 100
def single_get(_):
    id_here = random.choice(all_ids)
    resp = requests.get(f"{BASE_URL}/assets/{id_here}", headers=HEADERS, timeout=30)
    return resp.elapsed.total_seconds() * 1000

t0 = time.time()
with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as ex:
    results = list(ex.map(single_get, range(total_reqs)))
elapsed = time.time() - t0
st = sorted(results)
avg = sum(results) / len(results)
p50 = st[len(st)//2]
p99 = st[int(len(st)*0.99)]
mx = max(st)
mn = min(st)
tp = total_reqs / elapsed
print(f"  {'Concurrent GETs (20w, 100r)':45s}  avg={avg:8.1f}ms  p50={p50:8.1f}ms  "
      f"p99={p99:8.1f}ms  min={mn:.1f}ms  max={mx:.1f}ms  "
      f"total={elapsed:.2f}s  throughput={tp:.0f} req/s")

print("\n--- 7h. Concurrency: 50 queries/run (20 workers) ---")
def concurrent_query(_):
    resp = requests.post(
        f"{BASE_URL}/queries/run",
        headers=HEADERS,
        json={
            "schema_version": "v1", "scope": {"resource": "assets"},
            "fields": ["asset_id", "asset_type", "owner"],
            "facets": [{"field": "asset_type", "type": "terms", "size": 10}],
            "filters": [{"field": "owner", "op": "eq", "val": random.choice(OWNERS)}],
            "limit": 50
        }, timeout=30)
    return resp.elapsed.total_seconds() * 1000

t0 = time.time()
n_queries = 50
with concurrent.futures.ThreadPoolExecutor(max_workers=20) as ex:
    results = list(ex.map(concurrent_query, range(n_queries)))
elapsed = time.time() - t0
st = sorted(results)
avg = sum(results) / len(results)
p50 = st[len(st)//2]
p99 = st[int(len(st)*0.99)]
tp = n_queries / elapsed
print(f"  {'Concurrent queries/run + facets (20w, 50r)':45s}  avg={avg:8.1f}ms  p50={p50:8.1f}ms  "
      f"p99={p99:8.1f}ms  total={elapsed:.2f}s  throughput={tp:.0f} req/s")

print("\n--- 7i. ES direct query baseline ---")
def es_search():
    ow = random.choice(OWNERS)
    return requests.post(
        "http://localhost:9200/assets/_search",
        headers={"Content-Type": "application/json"},
        json={"query": {"term": {"owner": ow}}, "size": 100},
        timeout=10)
measure("ES _search by owner (direct)", es_search, n=20)

def es_agg():
    return requests.post(
        "http://localhost:9200/assets/_search",
        headers={"Content-Type": "application/json"},
        json={
            "query": {"match_all": {}},
            "aggs": {"by_owner": {"terms": {"field": "owner", "size": 10}}},
            "size": 0
        }, timeout=10)
measure("ES _search aggregation (direct)", es_agg, n=20)

print("\n=== DONE ===")
print(json.dumps({
    "total_pg": total_pg,
    "total_es": total_es,
    "sql_insert_time_s": round(t_sql, 2),
    "reindex_time_s": round(t_reindex, 2),
    "mcap_files_used": len(MCAP_FILE_IDS)
}, indent=2))
