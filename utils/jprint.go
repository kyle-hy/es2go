package utils

import (
	"encoding/json"
	"fmt"
)

// JPrint json序列化后终端打印
func JPrint(data any) {
	jdata, _ := json.MarshalIndent(data, "", "    ")
	fmt.Println("json:", string(jdata))
}
