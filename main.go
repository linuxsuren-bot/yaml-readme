package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type option struct {
	pattern       string
	templateFile  string
	includeHeader bool
	sortBy        string
	groupBy       string
}

func (o *option) runE(cmd *cobra.Command, args []string) (err error) {
	var items []map[string]interface{}
	groupData := make(map[string][]map[string]interface{})

	// find YAML files
	var files []string
	var data []byte
	if files, err = filepath.Glob(o.pattern); err == nil {
		for _, metaFile := range files {
			if data, err = ioutil.ReadFile(metaFile); err != nil {
				cmd.PrintErrf("failed to read file [%s], error: %v\n", metaFile, err)
				continue
			}

			metaMap := make(map[string]interface{})
			if err = yaml.Unmarshal(data, metaMap); err != nil {
				cmd.PrintErrf("failed to parse file [%s] as a YAML, error: %v\n", metaFile, err)
				continue
			}

			// skip this item if there is a 'ignore' key is true
			if val, ok := metaMap["ignore"]; ok {
				if ignore, ok := val.(bool); ok && ignore {
					continue
				}
			}

			filename := strings.TrimSuffix(filepath.Base(metaFile), filepath.Ext(metaFile))
			parentname := filepath.Base(filepath.Dir(metaFile))

			metaMap["filename"] = filename
			metaMap["parentname"] = parentname
			metaMap["fullpath"] = metaFile

			if val, ok := metaMap[o.groupBy]; ok && val != "" {
				var strVal string
				switch val.(type) {
				case string:
					strVal = val.(string)
				case int:
					strVal = strconv.Itoa(val.(int))
				}

				if _, ok := groupData[strVal]; ok {
					groupData[strVal] = append(groupData[strVal], metaMap)
				} else {
					groupData[strVal] = []map[string]interface{}{
						metaMap,
					}
				}
			}

			items = append(items, metaMap)
		}
	}

	if o.sortBy != "" {
		descending := true
		if strings.HasPrefix(o.sortBy, "!") {
			o.sortBy = strings.TrimPrefix(o.sortBy, "!")
			descending = false
		}
		sortBy(items, o.sortBy, descending)
	}

	// load readme template
	var readmeTpl string
	if data, err = ioutil.ReadFile(o.templateFile); err != nil {
		fmt.Printf("failed to load README template, error: %v\n", err)
		readmeTpl = `
|中文名称|英文名称|JD|
|---|---|---|
{{- range $val := .}}
|{{$val.zh}}|{{$val.en}}|{{$val.jd}}|
{{end}}
`
	}
	if o.includeHeader {
		readmeTpl = fmt.Sprintf("> This file was generated by [%s](%s) via [yaml-readme](https://github.com/LinuxSuRen/yaml-readme), please don't edit it directly!\n\n",
			filepath.Base(o.templateFile), filepath.Base(o.templateFile))
	}
	readmeTpl = readmeTpl + string(data)

	// generate readme file
	var tpl *template.Template
	if tpl, err = template.New("readme").Parse(readmeTpl); err != nil {
		return
	}

	// render it with grouped data
	if o.groupBy != "" {
		err = tpl.Execute(os.Stdout, groupData)
	} else {
		err = tpl.Execute(os.Stdout, items)
	}
	return
}

func sortBy(items []map[string]interface{}, sortBy string, descending bool) {
	sort.SliceStable(items, func(i, j int) (compare bool) {
		left, ok := items[i][sortBy].(string)
		if !ok {
			return false
		}
		right, ok := items[j][sortBy].(string)
		if !ok {
			return false
		}

		compare = strings.Compare(left, right) < 0
		if !descending {
			compare = !compare
		}
		return
	})
}

func main() {
	opt := &option{}
	cmd := cobra.Command{
		Use:   "yaml-readme",
		Short: "A helper to generate a README file from Golang-based template",
		RunE:  opt.runE,
	}
	flags := cmd.Flags()
	flags.StringVarP(&opt.pattern, "pattern", "p", "items/*.yaml",
		"The glob pattern with Golang spec to find files")
	flags.StringVarP(&opt.templateFile, "template", "t", "README.tpl",
		"The template file which should follow Golang template spec")
	flags.BoolVarP(&opt.includeHeader, "include-header", "", true,
		"Indicate if include a notice header on the top of the README file")
	flags.StringVarP(&opt.sortBy, "sort-by", "", "",
		"Sort the array data descending by which field, or sort it ascending with the prefix '!'. For example: --sort-by !year")
	flags.StringVarP(&opt.groupBy, "group-by", "", "",
		"Group the array data by which field")

	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
