package transform

import (
	"encoding/json"
	"fmt"
	"strings"
)

type docResponse struct {
	Values [][]interface{} `json:"values,omitempty"`
}

func Transform(data []byte, suffix string) ([]byte, error) {
	doc := &docResponse{}
	err := json.Unmarshal(data, doc)
	if err != nil {
		return nil, fmt.Errorf("unmarshal json failed: data:%s, err:%v", string(data), err)
	}
	if len(doc.Values) == 0 {
		return nil, nil
	}

	title := getTitle(doc.Values[0], suffix)

	rmap, err := obj2Map(doc, title)
	if err != nil {
		return nil, err
	}

	result := removeEmpty(rmap, title)

	b, _ := json.Marshal(result)
	return b, nil
}

func obj2Map(doc *docResponse, title []string) ([]map[string]interface{}, error) {
	result := make([]map[string]interface{}, 0, len(doc.Values)-1)

	for i := 1; i < len(doc.Values); i++ {
		r := make(map[string]interface{})
		for j := 0; j < len(doc.Values[i]); j++ {
			r[title[j]] = doc.Values[i][j]
		}
		result = append(result, r)
	}

	return result, nil
}

func getTitle(title []interface{}, suffix string) []string {
	r := make([]string, len(title))
	for i, t := range title {
		if t != nil {
			r[i] = strings.ToLower(strings.Split(t.(string), suffix)[0])
		} else {
			r[i] = ""
		}
	}
	return r
}

func removeEmpty(data []map[string]interface{}, title []string) []map[string]interface{} {
	var empty bool
	// delete empty row
	tmpRow := []map[string]interface{}{}
	for _, row := range data {
		empty = true
		for _, v := range row {
			if v != "" && v != nil {
				empty = false
				break
			}
		}
		if !empty {
			tmpRow = append(tmpRow, row)
		}
	}

	// find empty col
	delCol := []string{}
	for _, k := range title {
		empty = true
		for _, row := range tmpRow {
			if row[k] != "" && row[k] != nil {
				empty = false
				break
			}
		}
		if empty {
			delCol = append(delCol, k)
		}
	}

	// delete empty col
	result := make([]map[string]interface{}, len(tmpRow))
	for i, row := range tmpRow {
		tmp := row
		for _, k := range delCol {
			delete(tmp, k)
		}
		result[i] = tmp
	}
	return result
}
