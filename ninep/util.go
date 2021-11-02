package ninep

import (
	"log"
	"regexp"
	"strings"
)

func Split(path string) []string {
	if path == "" {
		return []string{}
	}
	slash := regexp.MustCompile(`//+`)
	path = strings.TrimRight(path, "/")
	path = slash.ReplaceAllString(path, "/")
	p := strings.Split(path, "/")
	return p
}

func Join(path []string) string {
	p := strings.Join(path, "/")
	return p
}

func Copy(path []string) []string {
	p := make([]string, len(path))
	copy(p, path)
	return p
}

func EndSlash(path string) bool {
	return path[len(path)-1] == '/'
}

func IsPathEq(p1, p2 []string) bool {
	if len(p1) != len(p2) {
		return false
	}
	for i := range p1 {
		if p1[i] != p2[i] {
			return false
		}
	}
	return true
}

func Dir(path []string) []string {
	if len(path) < 1 {
		log.Fatalf("Dir")
	}
	return path[0 : len(path)-1]
}
