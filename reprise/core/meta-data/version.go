package meta_data

import (
	"regexp"
	"strings"
)

var versionEntities = []string{
	"python",
	"cuda",
	"ubuntu",
	"pytorch",
	"tensorflow",
	"docker",
	"redis",
	"mysql",
	"golang",
	"go",
	"java",
	"node",
	"nodejs",
	"gcc",
	"g++",
	"cmake",
	"linux",
}

func extractVersion(text string) string {

	text = strings.ToLower(text)
	// "python 3.10"
	versionPattern := regexp.MustCompile(
		`([a-zA-Z\+\-]+)\s*(v?\d+(\.\d+)+)`,
	)

	matches := versionPattern.FindStringSubmatch(text)

	if len(matches) > 0 {

		entity := matches[1]
		version := matches[2]

		if isVersionEntity(entity) {
			return version
		}
	}
	// "python 3"
	vPattern := regexp.MustCompile(
		`([a-zA-Z]+)\s*v(\d+)`,
	)

	matches = vPattern.FindStringSubmatch(text)

	if len(matches) > 0 {

		entity := matches[1]

		if isVersionEntity(entity) {
			return "v" + matches[2]
		}
	}

	return ""
}

func isVersionEntity(word string) bool {

	for _, item := range versionEntities {

		if word == item {
			return true
		}
	}

	return false
}
