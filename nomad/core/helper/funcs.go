package helper

import (
	"fmt"
	"time"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl/hcl/ast"
)

// boolToPtr returns the pointer to a boolean
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
