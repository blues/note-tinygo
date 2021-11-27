package tinynote

import (
	"fmt"

	"github.com/valyala/fastjson"
)

// Tracing during development
const j2oTrace = false

// JSONToObject unmarshals the specified JSON and returns it as a map[string]interface{}
func JSONToObject(objectJSON []byte) (object map[string]interface{}, err error) {

	// Parse the input JSON
	var p fastjson.Parser
	var v *fastjson.Value
	v, err = p.Parse(string(objectJSON))
	if err != nil {
		return
	}

	// Visit each of the values within the object
	var o *fastjson.Object
	o, err = v.Object()
	if err != nil {
		return
	}

	object = map[string]interface{}{}
	walkObjectInto(0, o, object)

	return

}

// Get a value
func getValue(level int, v *fastjson.Value) (result interface{}) {
	switch v.Type() {
	case fastjson.TypeTrue:
		if j2oTrace {
			fmt.Printf("BOOL %s\n", "true")
		}
		result = true
	case fastjson.TypeFalse:
		if j2oTrace {
			fmt.Printf("BOOL %s\n", "false")
		}
		result = false
	case fastjson.TypeNull:
		if j2oTrace {
			fmt.Printf("NULL\n")
		}
		result = nil
	case fastjson.TypeString:
		newStringBytes, _ := v.StringBytes()
		newString := string(newStringBytes)
		if j2oTrace {
			fmt.Printf("STRING %s\n", newString)
		}
		result = newString
	case fastjson.TypeNumber:
		f := v.GetFloat64()
		result = f
		if j2oTrace {
			fmt.Printf("FLOAT %f\n", result)
		}
	case fastjson.TypeObject:
		if j2oTrace {
			fmt.Printf("OBJECT\n")
		}
		o, _ := v.Object()
		newObject := map[string]interface{}{}
		walkObjectInto(level, o, newObject)
		result = newObject
	case fastjson.TypeArray:
		if j2oTrace {
			fmt.Printf("ARRAY\n")
		}
		a, _ := v.Array()
		result = walkArray(level, a)
	}
	return
}

// Walk an array into an object
func walkArray(level int, a []*fastjson.Value) (array interface{}) {

	array = []interface{}{}
	if a == nil {
		return
	}
	if len(a) == 0 {
		return
	}

	// We only support these array types
	switch a[0].Type() {
	case fastjson.TypeString:
		newArray := []string{}
		for i := 0; i < len(a); i++ {
			if j2oTrace {
				for i := 0; i < level; i++ {
					fmt.Printf("    ")
				}
			}
			value := getValue(level+1, a[i])
			newArray = append(newArray, value.(string))
		}
		array = newArray
	case fastjson.TypeNumber:
		newArray := []float64{}
		for i := 0; i < len(a); i++ {
			if j2oTrace {
				for i := 0; i < level; i++ {
					fmt.Printf("    ")
				}
			}
			value := getValue(level+1, a[i])
			newArray = append(newArray, value.(float64))
		}
		array = newArray
	case fastjson.TypeObject:
		newArray := []map[string]interface{}{}
		for i := 0; i < len(a); i++ {
			if j2oTrace {
				for i := 0; i < level; i++ {
					fmt.Printf("    ")
				}
			}
			value := getValue(level+1, a[i])
			newArray = append(newArray, value.(map[string]interface{}))
		}
		array = newArray
	}

	// Done
	return
}

// Decode an object
func walkObjectInto(level int, o *fastjson.Object, object map[string]interface{}) {
	o.Visit(func(k []byte, v *fastjson.Value) {
		if j2oTrace {
			for i := 0; i < level; i++ {
				fmt.Printf("    ")
			}
			fmt.Printf("%s ", k)
		}
		value := getValue(level+1, v)
		object[string(k)] = value
	})

}
