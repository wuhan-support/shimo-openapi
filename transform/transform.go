package transform

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

var (
	suffix = "（"
)

type docResponse struct {
	Values [][]interface{} `json:"values,omitempty"`
}

func Transform(data []byte) ([]byte, error) {
	doc := &docResponse{}
	err := json.Unmarshal(data, doc)
	if err != nil {
		return nil, fmt.Errorf("unmarshal json failed: data:%s, err:%v", string(data), err)
	}
	if len(doc.Values) == 0 {
		return nil, nil
	}

	title := getTitle(doc.Values[0])

	rmap, err := obj2Map(doc, title)
	if err != nil {
		return nil, err
	}

	result := removeEmpty(rmap, title)

	b, _ := json.Marshal(result)
	return b, nil
}

func obj2Map(doc *docResponse, title []string) ([]map[string]string, error) {
	result := make([]map[string]string, 0, len(doc.Values)-1)

	for i := 1; i < len(doc.Values); i++ {
		r := make(map[string]string)
		for j := 0; j < len(doc.Values[i]); j++ {
			switch t := doc.Values[i][j].(type) {
			case float64:
				decnum, err := decimal.NewFromString(fmt.Sprintf("%f", t))
				if err != nil {
					return nil, err
				}
				r[title[j]] = decnum.String()
			case string:
				r[title[j]] = t
			case bool:
				r[title[j]] = fmt.Sprintf("%t", t)
			default:
				r[title[j]] = "null"
			}
		}
		result = append(result, r)
	}

	return result, nil
}

func getTitle(title []interface{}) []string {
	r := make([]string, len(title))
	for i, t := range title {
		r[i] = strings.Split(t.(string), suffix)[0]
	}
	return r
}

func removeEmpty(data []map[string]string, title []string) []map[string]string {
	var empty bool
	// delete empty row
	tmpRow := []map[string]string{}
	for _, row := range data {
		empty = true
		for _, v := range row {
			if v != "" && v != "null" {
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
			if row[k] != "" && row[k] != "null" {
				empty = false
				break
			}
		}
		if empty {
			delCol = append(delCol, k)
		}
	}

	// delete empty col
	result := make([]map[string]string, len(tmpRow))
	for i, row := range tmpRow {
		tmp := row
		for _, k := range delCol {
			delete(tmp, k)
		}
		result[i] = tmp
	}
	return result
}
