// A prefix tree whose keys are path segments.
package walk

import (
	"io/fs"
	"strings"
)

// pathSegmenter segments string key paths by slash separators. For example,
// "a/b/c" -> ("a", 2), ("b", 4), ("c", -1) in successive calls. It does
// not allocate any heap memory.
func pathSegmenter(path string, start int) (string, int) {
	if len(path) == 0 || start < 0 || start >= len(path) {
		return "", -1
	}

	end := strings.IndexRune(path[start:], '/')
	if end == -1 {
		return path[start:], -1
	}

	next := start + end + 1
	if next >= len(path) {
		next = -1
	}

	return path[start : start+end], next
}

type pathTrie struct {
	children map[string]*pathTrie
	value    *fs.DirEntry
}

func (t *pathTrie) Get(key string) *pathTrie {
	node := t
	for part, i := pathSegmenter(key, 0); part != ""; part, i = pathSegmenter(key, i) {
		node = node.children[part]
		if node == nil {
			return nil
		}
	}
	return node
}

func (t *pathTrie) Put(key string, value *fs.DirEntry) bool {
	node := t
	for part, i := pathSegmenter(key, 0); part != ""; part, i = pathSegmenter(key, i) {
		child := node.children[part]
		if child == nil {
			if node.children == nil {
				node.children = map[string]*pathTrie{}
			}

			var child *pathTrie
			if i < 0 {
				child = &pathTrie{children: nil, value: value}
			} else {
				child = &pathTrie{}
			}

			node.children[part] = child
		}
		node = child
	}

	return true
}
