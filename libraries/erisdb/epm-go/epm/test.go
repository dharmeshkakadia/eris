package epm

import (
	"bufio"
	"fmt"
	"github.com/eris-ltd/epm-go/utils"
	"os"
	"strings"
)

// for parsing/running companion test files for an epm deploy

type TestResults struct {
	Tests  []string
	Errors []string // go can't marshal/unmarshal errors

	FailedTests []int
	Failed      int

	Err string // if we suffered a non-epm-test error

	PkgDefFile  string
	PkgTestFile string
}

// run through all tests in file
// execute each test
func (e *EPM) Test(filename string) (*TestResults, error) {
	lines := []string{}
	f, _ := os.Open(filename)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		t := scanner.Text()
		lines = append(lines, t)
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("No tests to run...")
	}

	results := TestResults{
		Tests:       lines,
		Errors:      []string{},
		FailedTests: []int{},
		Failed:      0,
		Err:         "",
		PkgDefFile:  e.pkgdef,
		PkgTestFile: filename,
	}

	for i, line := range lines {
		//fmt.Println("vars:", e.Vars())
		tt := strings.TrimSpace(line)
		if len(tt) == 0 || tt[0:1] == "#" {
			continue
		}
		line = strings.Split(line, "#")[0]

		err := e.ExecuteTest(line, i)
		if err != nil {
			results.Errors = append(results.Errors, err.Error())
		} else {
			results.Errors = append(results.Errors, "")
		}

		if err != nil {
			results.Failed += 1
			results.FailedTests = append(results.FailedTests, i)
			logger.Errorln(err)
		}
	}
	var err error
	if results.Failed == 0 {
		err = nil
		fmt.Println("passed all tests")
	} else {
		err = fmt.Errorf("failed %d/%d tests", results.Failed, len(lines))
	}
	return &results, err
}

// execute a single test line
func (e *EPM) ExecuteTest(line string, i int) error {
	args := parseLine(line)
	args = e.VarSub(args)
	fmt.Println("test!", i)
	s := "\t"
	for _, a := range args {
		s += a + "  "
	}
	fmt.Println(s)
	if len(args) < 3 || len(args) > 4 {
		return fmt.Errorf("invalid number of args for test on line %d", i)
	}
	margs := DoMath(args[:3])
	// a test is 'addr storage expected'
	// if there's a fourth, its the variable name to store the result under
	// expected can be `_` in which case it is not tested (but may be saved)
	addr := margs[0]
	storage := margs[1]
	//expected := args[2]

	// retrieve the value
	val := e.chain.StorageAt(utils.AddHex(addr), utils.AddHex(storage))
	val = utils.AddHex(val)

	if args[2] != "_" {
		expected := utils.Coerce2Hex(margs[2])
		if utils.StripHex(expected) != utils.StripHex(val) {
			return fmt.Errorf("\t!!!!!Test %d failed. Got: %s, expected %s", i, val, expected)
		} else {
			fmt.Println("\tTest Passed (with flying colors!)")
		}
	} else {
		fmt.Println("\tNo expected value specified. Skipping check")
	}

	// store the value
	if len(args) == 4 {
		e.StoreVar(args[3], val)
		fmt.Println("\tStored:", args[3], val)
	}
	return nil
}

// pretty print the test results for json (double escape \n!)
func (t *TestResults) String() string {
	result := ""

	result += fmt.Sprintf("PkgDefFile: %s\\n", t.PkgDefFile)
	result += fmt.Sprintf("PkgTestFile: %s\\n", t.PkgTestFile)

	if t.Err != "" {
		result += fmt.Sprintf("Fail due to error: %s", t.Err)
		return result
	}

	if t.Failed > 0 {
		for _, testN := range t.FailedTests {
			result += fmt.Sprintf("Test %d failed.\\n\\tQuery: %s\\n\\tError: %s", testN, t.Tests[testN], t.Errors[testN])
			if result[len(result)-1:] != "\n" {
				result += "\\n"
			}
		}
		return strings.Replace(result, `"`, `\"`, -1) // "
	}
	result += "\\nAll Tests Passed"
	return strings.Replace(result, `"`, `\"`, -1) // " // essential for color sanity
}
