package helper

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

// MapStringStringSliceValueSet returns the set of values in a map[string][]string
func MapStringStringSliceValueSet(m map[string][]string) []string {
	set := make(map[string]struct{})
	for _, slice := range m {
		for _, v := range slice {
			set[v] = struct{}{}
		}
	}

	flat := make([]string, 0, len(set))
	for k := range set {
		flat = append(flat, k)
	}
	return flat
}

// BoolToPtr returns the pointer to a boolean
func BoolToPtr(b bool) *bool {
	return &b
}

// StringToPtr returns the pointer to a string
func StringToPtr(str string) *string {
	return &str
}

// TimeToPtr returns the pointer to a time stamp
func TimeToPtr(t time.Duration) *time.Duration {
	return &t
}

// IntToPtr returns the pointer to an int
func IntToPtr(i int) *int {
	return &i
}

func CheckHCLKeys(node ast.Node, valid []string) error {
	var list *ast.ObjectList
	switch n := node.(type) {
	case *ast.ObjectList:
		list = n
	case *ast.ObjectType:
		list = n.List
	default:
		return fmt.Errorf("cannot check HCL keys of type %T", n)
	}

	validMap := make(map[string]struct{}, len(valid))
	for _, v := range valid {
		validMap[v] = struct{}{}
	}

	var result error
	for _, item := range list.Items {
		key := item.Keys[0].Token.Value().(string)
		if _, ok := validMap[key]; !ok {
			result = multierror.Append(result, fmt.Errorf(
				"invalid key: %s", key))
		}
	}

	return result
}

// UnusedKeys returns a pretty-printed error if any `hcl:",unusedKeys"` is not empty
func UnusedKeys(obj interface{}) error {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}
	return unusedKeysImpl([]string{}, val)
}

func unusedKeysImpl(path []string, val reflect.Value) error {
	stype := val.Type()
	for i := 0; i < stype.NumField(); i++ {
		ftype := stype.Field(i)
		fval := val.Field(i)
		tags := strings.Split(ftype.Tag.Get("hcl"), ",")
		name := tags[0]
		tags = tags[1:]

		if fval.Kind() == reflect.Ptr {
			fval = reflect.Indirect(fval)
		}

		// struct? recurse. Add the struct's key to the path
		if fval.Kind() == reflect.Struct {
			err := unusedKeysImpl(append([]string{name}, path...), fval)
			if err != nil {
				return err
			}
			continue
		}

		// Search the hcl tags for "unusedKeys"
		unusedKeys := false
		for _, p := range tags {
			if p == "unusedKeys" {
				unusedKeys = true
				break
			}
		}

		if unusedKeys {
			ks, ok := fval.Interface().([]string)
			if ok && len(ks) != 0 {
				ps := ""
				if len(path) > 0 {
					ps = strings.Join(path, ".") + " "
				}
				return fmt.Errorf("%sunexpected keys %s",
					ps,
					strings.Join(ks, ", "))
			}
		}
	}
	return nil
}

type StateWriter struct {
	d      *schema.ResourceData
	errors []string
}

func NewStateWriter(d *schema.ResourceData) *StateWriter {
	return &StateWriter{d: d}
}

func (sw *StateWriter) Set(key string, value interface{}) {
	err := sw.d.Set(key, value)
	if err != nil {
		sw.errors = append(
			sw.errors,
			fmt.Sprintf(" - failed to set '%s': %v", key, err),
		)
	}
}

func (sw *StateWriter) Error() error {
	if sw.errors == nil {
		return nil
	}
	errors := strings.Join(sw.errors, "\n")
	return fmt.Errorf("Failed to write the state:\n%s", errors)
}
