package meta_data

type QueryMeta struct {
	Language string
	Version  string
	Format   string
	Intent   string
}

func ExtractMetaData(query string) QueryMeta {
	intent := extractIntent(query)
	meta := QueryMeta{
		Version:  extractVersion(query),
		Format:   extractFormat(query),
		Intent:   string(intent),
		Language: extractLanguage(query, intent),
	}
	return meta
}

func MatchTwoMeta(m1, m2 QueryMeta) bool {
	if m1.Format != m2.Format {
		return false
	}
	if m1.Language != m2.Language {
		return false
	}

	if m1.Version != m2.Version {
		return false
	}
	if m1.Intent != m2.Intent {
		return false
	}

	return true
}
