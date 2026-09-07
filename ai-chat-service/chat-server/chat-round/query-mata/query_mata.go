package query_mata

type QueryMeta struct {
	Language string
	Version  string
	Format   string
	Intent   string
}

func ExtractQueryMeta(query string) QueryMeta {
	intent := ExtractIntent(query)
	meta := QueryMeta{
		Version:  ExtractVersion(query),
		Format:   ExtractFormat(query),
		Intent:   string(intent),
		Language: ExtractLanguage(query, intent),
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
