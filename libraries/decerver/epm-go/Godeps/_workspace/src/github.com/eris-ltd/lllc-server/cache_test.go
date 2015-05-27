package lllcserver

import (
	"bytes"
	"fmt"
	"github.com/eris-ltd/epm-go/utils"
	"os"
	"os/exec"
	"path"
	"testing"
)

func init() {
	DebugMode = 4
	ClearCaches()
}

func copyFile(src, dst string) error {
	cmd := exec.Command("cp", src, dst)
	return cmd.Run()
}

func testCache(t *testing.T) {
	ClearCaches()
	code, _, err := Compile("tests/test-inc1.lll")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%x\n", code)
	copyFile("tests/test-inc1.lll", path.Join(utils.Lllc, "test-inc1.lll"))
	copyFile("tests/test-inc2.lll", path.Join(utils.Lllc, "test-inc2.lll"))
	copyFile("tests/test-inc4.lll", path.Join(utils.Lllc, "test-inc3.lll"))
	cur, _ := os.Getwd()
	os.Chdir(utils.Lllc)
	code2, _, err := Compile(path.Join(utils.Lllc, "test-inc1.lll"))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%x\n", code2)
	if bytes.Compare(code, code2) == 0 {
		t.Fatal("failed to update cached file")
	}
	os.Chdir(cur)
}

func TestCacheLocal(t *testing.T) {
	SetLanguageNet("lll", false)
	testCache(t)
}

func TestCacheRemote(t *testing.T) {
	SetLanguageNet("lll", true)
	testCache(t)
}

func TestSimple(t *testing.T) {
	ClearCaches()
	SetLanguageNet("lll", false)
	code, _, err := Compile("tests/test-inc1.lll")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%x\n", code)
}
