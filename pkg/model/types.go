package model

var AllRules = []string{}

func newRule(name string) string {
	AllRules = append(AllRules, name)
	return name
}

var (
	RuleWorker = newRule("worker")
	RuleAdmin  = newRule("admin")
)
