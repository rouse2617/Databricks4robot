# CYB-1194: McapFilesPage filter pagination reset

## Summary

URL-sync `useEffect` overrides filter-initiated page reset, causing filter changes to land on wrong page.

## Fix

Remove the URL-sync effect (lines 64-69). Page is initialized from URL on mount; pagination onChange already writes URL. Back-sync is redundant and causes the bug.

## Verification

1. Load /mcap-files?page=3 → see page 3
2. Change filter → reset to page 1
3. Click page 2 → URL updates to ?page=2
