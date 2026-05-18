package handlers

import "strconv"

// ParsePageParams normalises page and pageSize query-string values.
// Defaults: page=1, pageSize=20. Caps pageSize at 200.
func ParsePageParams(pageStr, pageSizeStr string) (int, int) {
	page := 1
	pageSize := 20
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if s, err := strconv.Atoi(pageSizeStr); err == nil && s > 0 && s <= 200 {
			pageSize = s
		}
	}
	return page, pageSize
}
