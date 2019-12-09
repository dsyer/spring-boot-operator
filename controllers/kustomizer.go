/*

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

package controllers

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"sigs.k8s.io/kustomize/api/types"
)

// TypedObject Needs a comment
type TypedObject struct {
	types.TypeMeta `json:",inline"`
	ObjectMeta     v1.ObjectMeta `json:"metadata,omitempty"`
	Data           interface{}   `json:"data,omitempty"`
}

// Merge Merges two objects of the same type
func Merge(source interface{}, target interface{}) error {
	resource, err := json.Marshal(target)
	if err != nil {
		return err
	}
	patch, err := json.Marshal(source)
	if err != nil {
		return err
	}
	result, err := strategicMergePatch(resource, patch, target)
	if err != nil {
		return err
	}
	fmt.Println(string(result))
	err = json.Unmarshal(result, target)
	if err != nil {
		return err
	}
	return nil
}

func strategicMergePatch(original, patch []byte, dataStruct interface{}) ([]byte, error) {
	schema, err := strategicpatch.NewPatchMetaFromStruct(dataStruct)
	if err != nil {
		return nil, err
	}

	return strategicMergePatchUsingLookupPatchMeta(original, patch, schema)
}

func strategicMergePatchUsingLookupPatchMeta(original, patch []byte, schema strategicpatch.LookupPatchMeta) ([]byte, error) {
	originalMap, err := handleUnmarshal(original)
	if err != nil {
		return nil, err
	}
	patchMap, err := handleUnmarshal(patch)
	if err != nil {
		return nil, err
	}

	result, err := strategicpatch.MergeStrategicMergeMapPatchUsingLookupPatchMeta(schema, originalMap, patchMap)
	if err != nil {
		return nil, err
	}

	return json.Marshal(result)
}

func handleUnmarshal(j []byte) (map[string]interface{}, error) {
	if j == nil {
		j = []byte("{}")
	}

	m := map[string]interface{}{}
	err := json.Unmarshal(j, &m)
	if err != nil {
		return nil, mergepatch.ErrBadJSONDoc
	}
	return m, nil
}
