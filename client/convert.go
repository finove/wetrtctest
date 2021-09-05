package client

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// String returns a pointer to the string value passed in.
func String(v string) *string {
	return &v
}

// StringValue returns the value of the string pointer passed in or
// "" if the pointer is nil.
func StringValue(v *string) string {
	if v != nil {
		return *v
	}
	return ""
}

// Bool returns a pointer to the bool value passed in.
func Bool(v bool) *bool {
	return &v
}

// BoolValue returns the value of the bool pointer passed in or
// false if the pointer is nil.
func BoolValue(v *bool) bool {
	if v != nil {
		return *v
	}
	return false
}

// Int returns a pointer to the int value passed in.
func Int(v int) *int {
	return &v
}

// IntValue returns the value of the int pointer passed in or
// 0 if the pointer is nil.
func IntValue(v *int) int {
	if v != nil {
		return *v
	}
	return 0
}

// Uint returns a pointer to the uint value passed in.
func Uint(v uint) *uint {
	return &v
}

// UintValue returns the value of the uint pointer passed in or
// 0 if the pointer is nil.
func UintValue(v *uint) uint {
	if v != nil {
		return *v
	}
	return 0
}

// Int64 returns a pointer to the int64 value passed in.
func Int64(v int64) *int64 {
	return &v
}

// Int64Value returns the value of the int64 pointer passed in or
// 0 if the pointer is nil.
func Int64Value(v *int64) int64 {
	if v != nil {
		return *v
	}
	return 0
}

// Uint16 returns a pointer to the uint16 value passed in.
func Uint16(v uint16) *uint16 {
	return &v
}

// Uint16Value returns the value of the uint16 pointer passed in or
// 0 if the pointer is nil.
func Uint16Value(v *uint16) uint16 {
	if v != nil {
		return *v
	}
	return 0
}

// ShowJSON 显示JSON格式字符串
func ShowJSON(v interface{}, indents ...bool) string {
	var vv []byte
	if len(indents) > 0 && indents[0] {
		vv, _ = json.MarshalIndent(v, "", "    ")
	} else {
		vv, _ = json.Marshal(v)
	}
	return string(vv)
}

func GetMapFieldString(m map[string]interface{}, field string) (value string, err error) {
	var ok bool
	var v interface{}
	if len(m) == 0 {
		err = fmt.Errorf("field not exists")
		return
	}
	if v, ok = m[field]; !ok {
		err = fmt.Errorf("field not exists")
		return
	}
	if value, ok = v.(string); ok {
		return
	} else if vv, ok2 := v.(map[string]interface{}); ok2 {
		vvv, _ := json.Marshal(vv)
		value = string(vvv)
	} else {
		value = fmt.Sprintf("%v", v)
	}
	return
}

func GetMapFieldInt64(m map[string]interface{}, field string) (value int64, err error) {
	var v interface{}
	var v64 float64
	var ok bool
	if len(m) == 0 {
		err = fmt.Errorf("field not exists")
		return
	}
	if v, ok = m[field]; !ok {
		err = fmt.Errorf("field not exists")
		return
	}
	if v64, ok = v.(float64); ok {
		value = int64(v64)
		return
	} else if value, ok = v.(int64); ok {
		return
	} else if _, ok = v.(map[string]interface{}); ok {
		err = fmt.Errorf("invalid int64 format")
		return
	} else {
		value, err = strconv.ParseInt(fmt.Sprintf("%v", v), 0, 64)
	}
	return
}

func GetMapFieldBool(m map[string]interface{}, field string) (value bool, err error) {
	var ok bool
	var v interface{}
	var vs string
	if len(m) == 0 {
		err = fmt.Errorf("field not exists")
		return
	}
	if v, ok = m[field]; !ok {
		err = fmt.Errorf("field not exists")
		return
	}
	if value, ok = v.(bool); ok {
		return
	}
	if vs, ok = v.(string); ok && vs == "true" {
		value = true
	} else if ok && vs == "false" {
		value = false
	} else {
		err = fmt.Errorf("invalid bool format")
	}
	return
}
