/*
Copyright 2018 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package parser

import (
	"fmt"
	"sort"
	"strings"
)

// Dict represents a PDF dict object.
type Dict map[string]Object

// NewDict returns a new PDFDict object.
func NewDict() Dict {
	return map[string]Object{}
}

// Clone returns a clone of d.
func (d Dict) Clone() Object {
	d1 := NewDict()
	for k, v := range d {
		if v != nil {
			v = v.Clone()
		}
		d1[k] = v
	}
	return d1
}

func (d Dict) indentedString(level int) string {

	logstr := []string{"<<\n"}
	tabstr := strings.Repeat("\t", level)

	var keys []string
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {

		v := d[k]

		if subdict, ok := v.(Dict); ok {
			dictStr := subdict.indentedString(level + 1)
			logstr = append(logstr, fmt.Sprintf("%s<%s, %s>\n", tabstr, k, dictStr))
			continue
		}

		if a, ok := v.(Array); ok {
			arrStr := a.indentedString(level + 1)
			logstr = append(logstr, fmt.Sprintf("%s<%s, %s>\n", tabstr, k, arrStr))
			continue
		}

		logstr = append(logstr, fmt.Sprintf("%s<%s, %v>\n", tabstr, k, v))

	}

	logstr = append(logstr, fmt.Sprintf("%s%s", strings.Repeat("\t", level-1), ">>"))

	return strings.Join(logstr, "")
}

// PDFString returns a string representation as found in and written to a PDF file.
func (d Dict) PDFString() string {
	logstr := []string{}
	logstr = append(logstr, "<<")

	var keys []string
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := d[k]
		if v == nil {
			logstr = append(logstr, fmt.Sprintf("/%s null", k))
			continue
		}
		logstr = append(logstr, fmt.Sprintf("/%s%s", k, v.PDFString()))
	}

	logstr = append(logstr, ">>")
	return strings.Join(logstr, "")
}

func (d Dict) String() string {
	return d.indentedString(1)
}
