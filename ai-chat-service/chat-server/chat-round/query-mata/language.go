package query_mata

import "strings"

type LanguageRule struct {
	Name string
	Aka  []string
}

var languageRules = []LanguageRule{
	{
		Name: "C++",
		Aka:  []string{"c++", "cpp", "cxx", "c plus plus"},
	},
	{
		Name: "Java",
		Aka:  []string{"java"},
	},
	{
		Name: "Python",
		Aka:  []string{"python", "py"},
	},
	{
		Name: "Go",
		Aka:  []string{"golang", "go"},
	},
	{
		Name: "Rust",
		Aka:  []string{"rust"},
	},
	{
		Name: "C",
		Aka:  []string{"c"},
	},
	{
		Name: "JavaScript",
		Aka:  []string{"javascript", "js"},
	},
	{
		Name: "TypeScript",
		Aka:  []string{"ts", "typescript"},
	},
}

func isLanguageConstraint(query, lang string) bool {

	patterns := []string{
		"用" + lang,
		"使用" + lang,

		lang + "版本",
		lang + "实现",
		lang + "代码",
		lang + "程序",
		lang + "写",
		lang + "编写",
		lang + "开发",

		"改成" + lang,
		"转换成" + lang,
		"转换为" + lang,
		"重写为" + lang,
	}

	for _, p := range patterns {
		if strings.Contains(query, p) {
			return true
		}
	}

	return false
}

func isLanguageTopic(query, lang string) bool {

	patterns := []string{
		"学习" + lang,
		"介绍" + lang,
		lang + "教程",
		lang + "是什么",
		lang + "原理",
		lang + "使用说明",
		lang + "特点",
	}

	for _, p := range patterns {
		if strings.Contains(query, p) {
			return true
		}
	}

	return false
}

func ExtractLanguage(query string, intent Intent) string {

	query = strings.ToLower(query)

	for _, lang := range languageRules {

		for _, alias := range lang.Aka {

			if !strings.Contains(query, alias) {
				continue
			}

			if isLanguageConstraint(query, alias) && (intent == IntentGenerate || intent == IntentModify ||
				intent == IntentUnknown) {
				return lang.Name
			}

			if isLanguagePrefix(query, alias) {

				if intent == IntentGenerate ||
					intent == IntentModify {

					return lang.Name
				}
			}

			if isLanguageSuffix(query, alias) {
				if intent == IntentGenerate ||
					intent == IntentModify {

					return lang.Name
				}
			}
			if isLanguageTopic(query, alias) {
				continue
			}

		}
	}

	return ""
}

func isLanguageSuffix(query, lang string) bool {

	return strings.HasSuffix(query, lang)
}

func isLanguagePrefix(query, lang string) bool {

	prefixPatterns := []string{

		"生成" + lang,
		"用" + lang,
		"使用" + lang,
		"基于" + lang,
		"采用" + lang,
	}

	for _, p := range prefixPatterns {

		if strings.Contains(query, p) {
			return true
		}
	}

	return false
}
