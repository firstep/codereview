package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
)

func ExportCSV[T any](writer io.Writer, datas []T) {
	if len(datas) == 0 {
		return
	}

	w := csv.NewWriter(writer)

	t := reflect.TypeOf(datas[0])
	row := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		row = append(row, field.Tag.Get("col"))
	}
	w.Write(row)

	for _, data := range datas {
		row = row[:0]
		valType := reflect.ValueOf(data)
		for i := 0; i < valType.NumField(); i++ {
			valStr := fmt.Sprintf("%v", valType.Field(i).Interface())
			row = append(row, valStr)
		}

		w.Write(row)
	}

	w.Flush()
}
