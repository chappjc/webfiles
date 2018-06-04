// Copyright (c) 2018, Jonathan Chappelow
// Copyright (c) 2017, The dcrdata developers
// See LICENSE for details.

package server

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
)

type PageTemplate struct {
	file     string
	template *template.Template
}

type SiteTemplates struct {
	pageTemplates map[string]PageTemplate
	helpers       template.FuncMap
}

func NewTemplates(folder string, names []string, helpers template.FuncMap) (*SiteTemplates, error) {
	templ := &SiteTemplates{
		pageTemplates: make(map[string]PageTemplate),
		helpers:       helpers,
	}

	for _, name := range names {
		fileName := filepath.Join(folder, name+".tmpl")
		temp, err := template.New(name).Funcs(templ.helpers).ParseFiles(fileName)
		if err != nil {
			return nil, err
		}
		templ.pageTemplates[name] = PageTemplate{
			file:     fileName,
			template: temp,
		}
	}

	return templ, nil
}

// ExecTemplateToString executes the specified template using the supplied data,
// and writes the result into a string. If the template fails to execute or
// isn't found, a non-nil error will be returned.
func (t *SiteTemplates) ExecTemplateToString(name string, data interface{}) (string, error) {
	temp, ok := t.pageTemplates[name]
	if !ok {
		return "", fmt.Errorf("unknown template %s", name)
	}

	var page bytes.Buffer
	err := temp.template.ExecuteTemplate(&page, name, data)
	return page.String(), err
}

// ExecTemplate executes the specified template using the supplied data, and
// writes the result directly to the ResponseWriter.
func (t *SiteTemplates) ExecTemplate(w http.ResponseWriter, name string, data interface{}) error {
	if temp, ok := t.pageTemplates[name]; ok {
		return temp.template.ExecuteTemplate(w, name, data)
	}
	return fmt.Errorf("unknown template %s", name)
}

func MakeTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		// "add": func(a int64, b int64) int64 {
		// 	return a + b
		// },
		// "subtract": func(a int64, b int64) int64 {
		// 	return a - b
		// },
	}
}
