package lllcserver

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"path"
	"testing"
)

func init() {
	ClearCaches()
}

// test the result of compiling through the lllc pipeline vs giving it to the wrapper
func testContract(t *testing.T, file string) {
	our_code, our_abi, err := Compile(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(our_code) == 0 {
		t.Fatal(fmt.Errorf("Output is empty!"))
	}

	lang, _ := LangFromFile(file)
	truth_code, truth_abi, err := CompileWrapper(file, lang)
	if err != nil {
		t.Fatal(err)
	}
	if len(truth_code) == 0 {
		t.Fatal(fmt.Errorf("Output is empty!"))
	}
	N := 100
	printCodeTop("us", our_code, N)
	printCodeTop("them", truth_code, N)
	if bytes.Compare(our_code, truth_code) != 0 {
		t.Fatal(fmt.Errorf("Difference of %d", bytes.Compare(our_code, truth_code)))
	}
	if our_abi != truth_abi {
		t.Fatal(fmt.Errorf("ABI results don't match:", our_abi, truth_abi))
	}
}

func testLocalRemote(t *testing.T, lang, filename string) {
	ClearCaches()
	SetLanguageNet(lang, false)
	testContract(t, filename)
	ClearCaches()
	SetLanguageNet(lang, true)
	testContract(t, filename)
	ClearCaches()
}

func TestLLLClientLocal(t *testing.T) {
	ClearCaches()
	SetLanguageNet("lll", false)
	testContract(t, "tests/namereg.lll")
	// Note: can't test more complex ones against the native compiler
	// since it doesnt handle paths in the includes...
	//testContract(t, path.Join(utils.ErisLtd, "eris-std-lib", "DTT", "tests", "stdarraytest.lll"))
}

func TestLLLClientRemote(t *testing.T) {
	testLocalRemote(t, "lll", "tests/namereg.lll")
}

func TestSerpentClientLocal(t *testing.T) {
	ClearCaches()
	SetLanguageNet("se", false)
	testContract(t, "tests/test.se")
}

func TestSerpentClientRemote(t *testing.T) {
	testLocalRemote(t, "se", "tests/test.se")
	testLocalRemote(t, "se", path.Join(homeDir(), "serpent", "examples", "schellingcoin", "schellingcoin.se"))
}

func printCodeTop(s string, code []byte, n int) {
	fmt.Println("length:", len(code))
	if len(code) > n {
		code = code[:n]
	}
	fmt.Printf("%s\t %s\n", s, hex.EncodeToString(code))
}
