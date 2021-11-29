package common

import "encoding/json"

func MarshalToMap(obj interface{}) (val map[string]interface{}, err error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &val)
	return
}

func UnmarshalMap(val map[string]interface{}, obj interface{}) error {
	j, err := json.Marshal(val)
	if err != nil {
		return err
	}

	return json.Unmarshal(j, obj)
}
