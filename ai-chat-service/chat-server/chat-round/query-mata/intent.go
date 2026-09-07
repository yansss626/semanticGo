package query_mata

import "strings"

type Intent string

const (
	IntentUnknown Intent = "unknown"

	IntentGenerate Intent = "generate"

	IntentExplain Intent = "explain"

	IntentCompare Intent = "compare"

	IntentModify Intent = "modify"
)

type IntentRule struct {
	Intent Intent

	Keywords []string
}

var intentRules = []IntentRule{

	{
		Intent: IntentGenerate,

		Keywords: []string{
			"生成",
			"写一个",
			"写一段",
			"编写",
			"实现",
			"创建",
			"开发",
			"设计",
			"给出代码",
			"提供代码",
		},
	},

	{
		Intent: IntentExplain,

		Keywords: []string{
			"介绍",
			"解释",
			"说明",
			"是什么",
			"原理",
			"教程",
			"学习",
			"讲解",
			"概述",
			"啥",
		},
	},

	{
		Intent: IntentCompare,

		Keywords: []string{
			"区别",
			"比较",
			"对比",
			"哪个好",
			"优缺点",
			"差异",
		},
	},

	{
		Intent: IntentModify,

		Keywords: []string{
			"修改",
			"改成",
			"转换",
			"迁移",
			"重构",
			"优化",
			"升级",
		},
	},
}

func ExtractIntent(query string) Intent {

	query = strings.ToLower(query)

	for _, rule := range intentRules {

		for _, keyword := range rule.Keywords {

			if strings.Contains(query, keyword) {

				return rule.Intent

			}

		}

	}

	return IntentUnknown
}
