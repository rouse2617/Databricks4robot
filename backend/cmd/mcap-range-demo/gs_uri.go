package main

import "strings"

// parseGSURI splits gs://bucket/object/key into bucket and object name.
func parseGSURI(uri string) (bucket, object string, ok bool) {
	const prefix = "gs://"
	if !strings.HasPrefix(uri, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(uri, prefix)
	i := strings.IndexByte(rest, '/')
	if i <= 0 || i >= len(rest)-1 {
		return "", "", false
	}
	return rest[:i], rest[i+1:], true
}
