package tinynote

import (
	"strconv"
)

// ObjectToJSON converts an object to JSON
func ObjectToJSON(object map[string]interface{}) (objectJSON []byte, err error) {
	var objectJSONstr string
	objectJSONstr, err = walkMap(0, object)
	objectJSON = []byte(objectJSONstr)
	return
}

// Walk the map, separating fields with an underscore
func walkMap(level int, object map[string]interface{}) (out string, err error) {

	// Iterate over keys in object
	out += "{"

	for k, v := range object {

		// Output field
		if out != "{" {
			out += ","
		}
		out += "\""
		out += k
		out += "\":"

		// Only add the key if it's a basic data type
		value := "\"\""
		switch v.(type) {
		case nil:
			value = "null"
		case bool:
			value = strconv.FormatBool(v.(bool))
		case int:
			value = strconv.FormatInt(int64(v.(int)), 10)
		case uint:
			value = strconv.FormatInt(int64(v.(uint)), 10)
		case int32:
			value = strconv.FormatInt(int64(v.(int32)), 10)
		case uint32:
			value = strconv.FormatInt(int64(v.(uint32)), 10)
		case int64:
			value = strconv.FormatInt(int64(v.(int64)), 10)
		case uint64:
			value = strconv.FormatInt(int64(v.(uint64)), 10)
		case float32:
			value = strconv.FormatFloat(float64(v.(float32)), 'f', -1, 32)
		case float64:
			value = strconv.FormatFloat(v.(float64), 'f', -1, 64)
		case string:
			value = strconv.Quote(v.(string))
		case map[string]interface{}:
			value, err = walkMap(level+1, v.(map[string]interface{}))
			if err != nil {
				return
			}
		case []int:
			value = "["
			for i := 0; i < len(v.([]int)); i++ {
				if i != 0 {
					value += ","
				}
				value += strconv.FormatInt(int64(v.([]int)[i]), 10)
			}
			value += "]"
		case []uint:
			value = "["
			for i := 0; i < len(v.([]uint)); i++ {
				if i != 0 {
					value += ","
				}
				value += strconv.FormatInt(int64(v.([]uint)[i]), 10)
			}
			value += "]"
		case []int32:
			value = "["
			for i := 0; i < len(v.([]int32)); i++ {
				if i != 0 {
					value += ","
				}
				value += strconv.FormatInt(int64(v.([]int32)[i]), 10)
			}
			value += "]"
		case []uint32:
			value = "["
			for i := 0; i < len(v.([]uint32)); i++ {
				if i != 0 {
					value += ","
				}
				value += strconv.FormatInt(int64(v.([]uint32)[i]), 10)
			}
			value += "]"
		case []int64:
			value = "["
			for i := 0; i < len(v.([]int64)); i++ {
				if i != 0 {
					value += ","
				}
				value += strconv.FormatInt(v.([]int64)[i], 10)
			}
			value += "]"
		case []uint64:
			value = "["
			for i := 0; i < len(v.([]uint64)); i++ {
				if i != 0 {
					value += ","
				}
				value += strconv.FormatInt(int64(v.([]uint64)[i]), 10)
			}
			value += "]"
		case []float32:
			value = "["
			for i := 0; i < len(v.([]float32)); i++ {
				if i != 0 {
					value += ","
				}
				value += strconv.FormatFloat(float64(v.([]float32)[i]), 'f', -1, 32)
			}
			value += "]"
		case []float64:
			value = "["
			for i := 0; i < len(v.([]float64)); i++ {
				if i != 0 {
					value += ","
				}
				value += strconv.FormatFloat(v.([]float64)[i], 'f', -1, 64)
			}
			value += "]"
		case []string:
			value = "["
			for i := 0; i < len(v.([]string)); i++ {
				if i != 0 {
					value += ","
				}
				value += strconv.Quote(v.([]string)[i])
			}
			value += "]"
		case []map[string]interface{}:
			value = "["
			for i := 0; i < len(v.([]map[string]interface{})); i++ {
				if i != 0 {
					value += ","
				}
				var ovalue string
				ovalue, err = walkMap(level+1, v.([]map[string]interface{})[i])
				if err != nil {
					return
				}
				value += ovalue
			}
			value += "]"
		case []interface{}:
			value = "["
			for i := 0; i < len(v.([]interface{})); i++ {
				if i != 0 {
					value += ","
				}
				var ovalue string
				ovalue, err = walkMap(level+1, v.([]interface{})[i].(map[string]interface{}))
				if err != nil {
					return
				}
				value += ovalue
			}
			value += "]"
		}

		// Append the value
		out += value

	}

	// Done
	out += "}"
	return

}
