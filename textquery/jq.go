package textquery

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/itchyny/gojq"

	"arhat.dev/pkg/decodehelper"
)

func JQ(query, data string) (string, error) {
	return JQBytes(query, []byte(data))
}

func JQBytes(query string, input []byte) (string, error) {
	q, err := gojq.Parse(query)
	if err != nil {
		return "", fmt.Errorf("failed to parse query: %w", err)
	}

	strData, err := strconv.Unquote(string(input))
	if err == nil {
		input = []byte(strData)
	}

	var data interface{}
	data = make(map[string]interface{})
	err = decodehelper.UnmarshalJSON(input, &data)
	if err != nil {
		// maybe it's an array
		data = []interface{}{}
		err = decodehelper.UnmarshalJSON(input, &data)
	}

	if err != nil {
		// maybe it's plain text
		data = input
	}

	return runQuery(q, data)
}

func runQuery(query *gojq.Query, data interface{}) (string, error) {
	iter := query.Run(data)
	var result []interface{}
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}

		if err, ok := v.(error); ok {
			return "", err
		}

		result = append(result, v)
	}

	switch len(result) {
	case 0:
		return "", nil
	case 1:
		switch r := result[0].(type) {
		case string:
			return r, nil
		case []byte:
			return string(r), nil
		case []interface{}, map[string]interface{}:
			res, err := json.Marshal(r)
			return string(res), err
		case int64:
			return strconv.FormatInt(r, 10), nil
		case int32:
			return strconv.FormatInt(int64(r), 10), nil
		case int16:
			return strconv.FormatInt(int64(r), 10), nil
		case int8:
			return strconv.FormatInt(int64(r), 10), nil
		case int:
			return strconv.FormatInt(int64(r), 10), nil
		case uint64:
			return strconv.FormatUint(r, 10), nil
		case uint32:
			return strconv.FormatUint(uint64(r), 10), nil
		case uint16:
			return strconv.FormatUint(uint64(r), 10), nil
		case uint8:
			return strconv.FormatUint(uint64(r), 10), nil
		case uint:
			return strconv.FormatUint(uint64(r), 10), nil
		case float64:
			return strconv.FormatFloat(r, 'f', -1, 64), nil
		case float32:
			return strconv.FormatFloat(float64(r), 'f', -1, 64), nil
		case bool:
			if r {
				return "true", nil
			}
			return "false", nil
		case nil:
			return "null", nil
		default:
			return fmt.Sprintf("%v", r), nil
		}
	default:
		res, err := json.Marshal(result)
		return string(res), err
	}
}
