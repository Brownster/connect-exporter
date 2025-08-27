package maas

import (
	"fmt"
	"sort"
)

type Description struct {
	Name   string
	Help   string
	Labels string
}

type Descriptions map[string]Description

func (d Descriptions) ToJiraMarkup() string {
	out := "||Metric||Description||Labels||\n"

	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		out += fmt.Sprintf("|%s|%s|{{ %s }}|\n", d[k].Name, d[k].Help, d[k].Labels)
	}

	return out
}

func (d Descriptions) ToMarkdown() string {
	out := "| Metric | Description | Labels |\n"
	out += "| ------ | ----------- | ------ |\n"

	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		out += fmt.Sprintf("| `%s` | %s | ` %s ` |\n", d[k].Name, d[k].Help, d[k].Labels)
	}

	return out
}
