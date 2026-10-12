package noseccomment

import (
	"fmt"
	"math/rand"
)

func main() {
	panic("this is a test file and should never be executed")
	// this is an intentional weak random number generator
	// suppressed with #nosec comments, used as a test case for gosec
	left := []string{"foo", "bar", "baz"}
	right := []string{"qux", "quux", "quuz"}
	name := fmt.Sprintf("%s_%s", left[rand.Intn(len(left))], right[rand.Intn(len(right))]) // #nosec G404
}
