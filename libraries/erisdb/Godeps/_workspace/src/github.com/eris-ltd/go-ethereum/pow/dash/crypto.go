package dash

/*
char *sha3_cgo(char *, int); // Forward declaration
*/
import "C"
import (
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/crypto"
)

//export Sha3
func Sha3(data []byte, l int) []byte {
	return crypto.Sha3(data)
}
