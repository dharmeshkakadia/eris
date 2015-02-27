/*
* This file runs tests for all the modules.
*
*/
package types

import(
	// "fmt"
	"testing"
	// "reflect"
)

// An int and a string
func TestPrimitive(t *testing.T) {
	testInt := 55 
    intVal := ToJsValue(testInt)
    newInt, ok := intVal.(int)
    if !ok {
    	t.Error("Return value for integer is not of type 'int'")
    }
    // If value is messed up and becomes 0 or something...
    if testInt != newInt {
    	t.Errorf("Integer equality failed. Input: %d, Output: %d\n",testInt,newInt)
    }
    
    testStr := "test" 
    strVal := ToJsValue(testStr)
    newStr, ok := strVal.(string)
    if !ok {
    	t.Error("Return value for string is not of type 'string'")
    }
    if testStr != newStr {
    	t.Errorf("String equality failed. Input: %s, Output: %s\n",testStr,newStr)
    }
}

// An int, uint and a string.
func TestPtrToPrimitive(t *testing.T) {
	normInt := 55
	testInt := &normInt 
    intVal := ToJsValue(testInt)
    newInt, ok := intVal.(int64)
    if !ok {
    	t.Error("Return value for integer is not of type 'int64'")
    }
    // If value is messed up and becomes 0 or something...
    if int64(normInt) != newInt {
    	t.Errorf("Integer equality failed. Input: %d, Output: %d\n",testInt,newInt)
    }
    
    normUInt := uint(12)
	testUInt := &normUInt
    uintVal := ToJsValue(testUInt)
    newUInt, ok := uintVal.(uint64)
    if !ok {
    	t.Error("Return value for unsigned integer is not of type 'uint64'")
    }
    // If value is messed up and becomes 0 or something...
    if uint64(normUInt) != newUInt {
    	t.Errorf("Unsigned integer equality failed. Input: %d, Output: %d\n",testUInt,newUInt)
    }
    
    normStr := "test"
    testStr := &normStr
    strVal := ToJsValue(testStr)
    newStr, ok := strVal.(string)
    if !ok {
    	t.Error("Return value for string is not of type 'string'")
    }
    if normStr != newStr {
    	t.Errorf("String equality failed. Input: %s, Output: %s\n",testStr,newStr)
    }
}

// Test maps
func TestMap(t *testing.T) {
	testMap := make(map[string]interface{})
	testMap["Str0"] = "test0"
	val := ToJsValue(testMap)
	retMap, ok := val.(map[string]interface{})
	if !ok {
		t.Error("Returned value is not of type map[string]interface{}")
	}
	if len(testMap) != len(retMap) {
		t.Error("Returned map is not of the same size. Original size: %d, new size: %d\n",
			len(testMap),len(retMap))
	}
	
	if testMap["Str0"] != retMap["Str0"] {
		t.Error("Map elements does not match.")
	}
	
	// With pointers 
	testPMap := make(map[string]*string)
	testStr := "test0"
	testStr1 := "test1"
	testPMap["Str0"] = &testStr
	testPMap["Str1"] = &testStr1
	valP := ToJsValue(testPMap)
	retPMap, ok := valP.(map[string]interface{})
	if !ok {
		t.Error("Returned value is not of type map[string]interface{}")
	}
	if len(testPMap) != len(retPMap) {
		t.Error("Returned map is not of the same size. Original size: %d, new size: %d\n",
			len(testPMap),len(retPMap))
	}
	
	if *testPMap["Str0"] != retPMap["Str0"] {
		t.Error("Map elements does not match.")
	}
	if *testPMap["Str1"] != retPMap["Str1"] {
		t.Error("Map elements does not match.")
	}
	
	// Maps of maps:
	testMMap := make(map[string]map[string]int)
	testMapElem := make(map[string]int)
	testMapElem["Int0"] = 8686
	testMMap["Map0"] = testMapElem
	valM := ToJsValue(testMMap)
	retMMap, ok := valM.(map[string]interface{})
	if !ok {
		t.Error("Returned value is not of type map[string]interface{}")
	}
	
	if !IsJsCompat(retMMap) {
		t.Error("Returned map is not compatible with otto")
	}
	
	if len(testMMap) != len(retMMap) {
		t.Error("Returned map is not of the same size. Original size: %d, new size: %d\n",
			len(testMMap),len(retMMap))
	}
	
	retMT, ok := retMMap["Map0"].(map[string]interface{})
	
	if !ok {
		t.Error("Returned map element is not of type map[string]interface{}")
	}
	
	if testMMap["Map0"]["Int0"] != retMT["Int0"] {
		t.Error("Map elements does not match.")
	}
	
	// TODO add more
}

const aSize = 100

// Test slices and arrays
func TestSlice(t *testing.T) {
	
	var testArr [aSize]byte
	
	for i := 0; i < aSize; i++ {
		testArr[i] = byte(i % 256) 
	}
	
	retVal := ToJsValue(testArr)
	retSlice := retVal.([]interface{})
	
	if !IsJsCompat(retSlice) {
		t.Error("Returned slice is not compatible with otto")
	}
	
	if len(testArr) != len (retSlice){
		t.Error("Returned slice is not of the same size as the input array. Original size: %d, new size: %d\n",
			len(testArr),len(retSlice))
	}
	
	for idx , val := range testArr {
		rB := retSlice[idx].(byte)
		if rB != val {
			t.Error("Returned slice does not hold the same values as the original array")
		}	
	}
	
	// TODO add more
	
}

type (
	// Testing basic fields.
	TestStruct0 struct {
		Field0 string
		Field1 int
	}
	
	
	TestStruct1 struct {
		Field0 string
		Field1 int
		Field2 TestStruct0
	}
	
	TestStruct2 struct {
		Field0 string
		Field1 int
		Field2 *TestStruct0
	}
	
	TestStruct3 struct {
		Field0 string
		Field1 int
		Field2 []TestStruct0
	}
	
	TestStruct4 struct {
		Field0 string
		Field1 int
		Field2 map[string]*TestStruct3
	}
	
	FuncStruct0 struct {
		Field0 int
	}
	
	WeirdStruct0 struct {}
	
)

func (f *FuncStruct0) failfunc(){
	// Just here to fail
}

// Test structs
func TestStruct(t *testing.T) {
	ts0 := TestStruct0{}
	ts0.Field0 = "test string"
	ts0.Field1 = -345
	retTs0 := ToJsValue(ts0)
	
	if !IsJsCompat(retTs0) {
		t.Errorf("Returned object is not js compatible %v\n",retTs0)
	}
	
	ts1 := TestStruct1{}
	ts1.Field0 = "test string 1"
	ts1.Field1 = 92
	ts1.Field2 = ts0
	retTs1 := ToJsValue(ts1)
	
	if !IsJsCompat(retTs1) {
		t.Errorf("Returned object is not js compatible %v\n",retTs0)
	}
	
	
	ts01 := &TestStruct0{}
	ts01.Field0 = "test string in pointer to struct"
	ts01.Field1 = 6424
	
	ts2 := TestStruct2{}
	ts2.Field0 = "test string 1"
	ts2.Field1 = 92
	ts2.Field2 = ts01
	retTs2 := ToJsValue(ts2)
	
	if !IsJsCompat(retTs2) {
		t.Errorf("Returned object is not js compatible %v\n",retTs2)
	}
	
	ts3 := &TestStruct3{}
	ts3.Field0 = "test string 3"
	ts3.Field1 = 155134
	ts3.Field2 = make([]TestStruct0,1)
	ts3.Field2[0] = ts0
	retTs3 := ToJsValue(ts3)
	
	if !IsJsCompat(retTs3) {
		t.Errorf("Returned object is not js compatible %v\n",retTs3)
	}
	
	ts31 := &TestStruct3{}
	ts31.Field0 = "test string 3"
	ts31.Field1 = 155134
	ts31.Field2 = make([]TestStruct0,1)
	ts31.Field2[0] = ts0
	
	ts4 := &TestStruct4{}
	ts4.Field0 = "test string 4"
	ts4.Field1 = -242424
	ts4.Field2 = make(map[string]*TestStruct3)
	ts4.Field2["Struct0"] = ts31
	retTs4 := ToJsValue(ts4)
	
	if !IsJsCompat(retTs4) {
		t.Errorf("Returned object is not js compatible %v\n",retTs4)
	}
	
	// fmt.Printf("Returned ts0 obj: %v\n",retTs0)
	// fmt.Printf("Returned ts1 obj: %v\n",retTs1)
	// fmt.Printf("Returned ts2 obj: %v\n",retTs2)
	// fmt.Printf("Returned ts3 obj: %v\n",retTs3)
	// fmt.Printf("Returned ts4 obj: %v\n",retTs4)
	
}


// Test if the function 'IsJsCompat' does proper filtering.
func TestIsJsCompat(t *testing.T) {
    
    if !IsJsCompat(546) {
    	t.Error("IsJsCompat does not recognize type as compatible: 'int'")
    }
    
    if !IsJsCompat("test string") {
    	t.Error("IsJsCompat does not recognize type as compatible: 'string'")
    }
    
    ts0 := TestStruct0{}
    if IsJsCompat(ts0) {
    	t.Error("IsJsCompat does not recognize type as incompatible: 'struct'")
    }
    
    ts0p := &TestStruct0{}
    if IsJsCompat(ts0p) {
    	t.Error("IsJsCompat does not recognize type as incompatible: '*struct'")
    }
    
    // Only allow strings as keys
    mp := make(map[int]interface{})
    mp[0] = 5
    if IsJsCompat(mp) {
    	t.Error("IsJsCompat does not recognize type as incompatible: map with int keys")
    }
    
    // A slice
    ts0sl := make([]interface{},1)
    ts0sl[0] = ts0
	if IsJsCompat(ts0sl) {
    	t.Error("IsJsCompat does not recognize type as incompatible: slice with 'struct' elements")
    }
	
	// Only allow strings as keys
    mpS := make(map[string]interface{})
    mpS["Struct0"] = ts0
    if IsJsCompat(mpS) {
    	t.Error("IsJsCompat does not recognize type as incompatible: map with 'struct' values")
    }
	
}



// An int and a string
func TestEdgeCases(t *testing.T) {
	
	nl := ToJsValue(nil)
   	if !IsJsCompat(nl) {
    	t.Error("IsJsCompat does not recognize type as incompatible: map with 'struct' values")
    }
   	
   	ws := WeirdStruct0{}
   	wsm := ToJsValue(ws) // Should panic
   	if !IsJsCompat(wsm) {
   		t.Errorf("Fail weirdstruct: %v\n",wsm)
   	}
   	
}