package meta_data

import (
	"regexp"
	"strings"
)

type FormatRule struct {
	Name string

	Aka []string

	PositivePatterns []string

	NegativePatterns []string
}

var formatRules = []FormatRule{

	{
		Name: "json",

		Aka: []string{
			"json",
			"JSON",
		},

		PositivePatterns: []string{
			"json格式",
			"json输出",
			"返回json",
			"生成json",
			"转换成json",
			"转为json",
		},

		NegativePatterns: []string{
			"json是什么",
			"介绍json",
			"学习json",
			"json原理",
			"json教程",
		},
	},

	{
		Name: "markdown",

		Aka: []string{
			"markdown",
			"md",
		},

		PositivePatterns: []string{
			"markdown格式",
			"md格式",
			"输出markdown",
			"生成markdown",
			"markdown文档",
		},

		NegativePatterns: []string{
			"markdown是什么",
			"介绍markdown",
			"markdown语法",
			"学习markdown",
		},
	},

	{
		Name: "xml",

		Aka: []string{
			"xml",
			"XML",
		},

		PositivePatterns: []string{
			"xml格式",
			"xml输出",
			"生成xml",
			"转换成xml",
		},

		NegativePatterns: []string{
			"xml是什么",
			"介绍xml",
			"xml协议",
		},
	},

	{
		Name: "csv",

		Aka: []string{
			"csv",
			"CSV",
		},

		PositivePatterns: []string{
			"csv格式",
			"导出csv",
			"生成csv",
			"csv文件",
		},

		NegativePatterns: []string{
			"csv是什么",
			"介绍csv",
		},
	},

	{
		Name: "table",

		Aka: []string{
			"表格",
		},

		PositivePatterns: []string{
			"表格形式",
			"整理成表格",
			"生成表格",
			"表格输出",
			"以表格展示",
		},

		NegativePatterns: []string{
			"什么是表格",
			"介绍表格",
		},
	},

	{
		Name: "list",

		Aka: []string{
			"列表",
		},

		PositivePatterns: []string{
			"列表形式",
			"整理成列表",
			"生成列表",
			"列出",
		},

		NegativePatterns: []string{
			"什么是列表",
			"介绍列表",
		},
	},
}

func extractFormat(text string) string {

	text = strings.ToLower(text)

	for _, rule := range formatRules {

		for _, neg := range rule.NegativePatterns {

			if strings.Contains(text, strings.ToLower(neg)) {

				return ""
			}
		}
	}

	for _, rule := range formatRules {

		for _, pos := range rule.PositivePatterns {

			if strings.Contains(text, strings.ToLower(pos)) {

				return rule.Name
			}

		}

	}

	for _, rule := range formatRules {

		for _, alias := range rule.Aka {

			pattern := regexp.MustCompile(
				"(返回|输出|生成|导出|转换).*" + strings.ToLower(alias),
			)

			if pattern.MatchString(text) {

				return rule.Name
			}

		}

	}

	return ""
}
