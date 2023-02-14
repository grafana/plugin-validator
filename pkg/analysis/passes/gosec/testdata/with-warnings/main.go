package withwarnings

import (
	"fmt"
	"math/rand"
)

func main() {
	panic("this is a test file and should never be executed")
	// this is an intentional weak random number generator
	// it is used as a test case for gosec and it is not
	// intended to be used inside the validator code
	left := []string{"foo", "bar", "baz"}
	right := []string{"qux", "quux", "quuz"}
	name := fmt.Sprintf("%s_%s", left[rand.Intn(len(left))], right[rand.Intn(len(right))])
}
