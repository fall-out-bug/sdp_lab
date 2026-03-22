package control

import "encoding/json"

func jsonMarshal(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func jsonUnmarshal(data []byte) (*ExecutorResultPacket, error) {
	var result ExecutorResultPacket
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
