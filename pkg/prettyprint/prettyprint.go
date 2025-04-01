package prettyprint

import (
	"encoding/json"
	"fmt"
)

func Print(b any) {
	s, _ := json.MarshalIndent(b, "", "\t")
	fmt.Print(string(s))
}

func SPrint(b any) string {
	s, _ := json.MarshalIndent(b, "", "\t")
	return string(s)
}
