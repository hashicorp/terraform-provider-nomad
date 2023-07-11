// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helper

func ToMapStringString(m any) map[string]string {
	mss := map[string]string{}
	for k, v := range m.(map[string]any) {
		mss[k] = v.(string)
	}
	return mss
}
