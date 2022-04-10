package parsec

import (
	"container/list"
	"errors"
	"reflect"
	"strings"
	"testing"
	"unicode"
)

type TestInput struct {
	in []rune
}

func (i *TestInput) Car() rune {
	return (*i).in[0]
}

func (i *TestInput) Cdr() ParserInput {
	return &TestInput{
		in: (*i).in[1:],
	}
}

func (i *TestInput) Empty() bool {
	return len((*i).in) == 0
}

func (i *TestInput) String() string {
	var s strings.Builder

	for _, r := range i.in {
		s.WriteRune(r)
	}

	return s.String()
}

func TestTag(t *testing.T) {
	var inTable []*TestInput = []*TestInput{
		{[]rune{'a', 'b', 'c'}},
		{[]rune{'d', 'e', 'f'}},
	}

	expected := []rune{'a', 'd'}

	for i, r := range expected {
		res := Tag(r)(inTable[i])
		resR := res.Result.(rune)
		if resR != r {
			t.Errorf("Tag isn't popping the right rune \n")
		}
		inTable[i] = res.Rem.(*TestInput)
	}

	if !reflect.DeepEqual(*inTable[0], TestInput{[]rune{'b', 'c'}}) {
		t.Errorf("Remander is not correct: %s instead of: %s \n", inTable[0], &TestInput{[]rune{'b', 'c'}})
	}

	if !reflect.DeepEqual(*inTable[1], TestInput{[]rune{'e', 'f'}}) {
		t.Errorf("Remander is not correct: %s instead of %s \n", inTable[1], &TestInput{[]rune{'e', 'f'}})
	}

}

func TestIsNot(t *testing.T) {
	var inTable []*TestInput = []*TestInput{
		{[]rune{'a', 'b', 'c'}},
		{[]rune{'d', 'e', 'f'}},
	}

	runes := []rune{'b', 'e'}

	for i, r := range runes {
		res := IsNot(r)(inTable[i])
		resR := res.Result.(rune)
		if res.Err != nil {
			t.Errorf("Errored: %s", res.Err)
		}
		if resR == r {
			t.Errorf("IsNot matches the said rune: %v instead of not doing so %v\n", resR, r)
		}
	}
}

func TestCharUTF8(t *testing.T) {
	//these guys ar invalid utf-8s, they should fail
	nonUTF8s := &TestInput{
		in: []rune("pythön!"),
	}

	for !nonUTF8s.Empty() {
		res := CharUTF8()(nonUTF8s)
		if e, did := res.Errored(); did {
			if !errors.Is(e, UnmatchedErr()) {
				t.Errorf("Sould be unmatched error")
			}
		}
		nonUTF8s = res.Rem.(*TestInput)

	}

	nonUTF8s = &TestInput{
		in: []rune{0xe228a1, 0xe28228},
	}

	res := CharUTF8()(nonUTF8s)
	if res.Result != nil {
		t.Errorf("should be nill, but is: %v", res.Result)
	}

	if !reflect.DeepEqual(*nonUTF8s, *(res.Rem.(*TestInput))) {
		t.Errorf("Should both be equal")
	}
}

func TestOneOf(t *testing.T) {
	in := TestInput{
		in: []rune{'d', 'e', 'f'},
	}

	any := []rune{'d', 'a', 'b', 'c'}

	res := OneOf(any)(&in)

	if e, didErr := res.Errored(); didErr {
		t.Errorf("Errored when it shouldn't: %s", e)
	}

	if reflect.DeepEqual(*(res.Rem.(*TestInput)), in) {
		t.Errorf("should not be equal beacuse it got reduced")
	}

	if !reflect.DeepEqual(TestInput{in: []rune{'e', 'f'}}, *(res.Rem.(*TestInput))) {
		t.Errorf("should  be equal beacuse it got reduced")
	}
}

func TestDigit(t *testing.T) {
	in := &TestInput{
		in: []rune{'1', '2', '3', '4', '5', 'g'},
	}
	dig := Digit()
	expected := []int{1, 2, 3, 4, 5}

	for _, exp := range expected {
		res := dig(in)
		if res.Result.(int) != exp {
			t.Errorf("should be equal to the integers in expected. Should be: %d, but is %d", exp, res.Result.(int))
		}
		in = res.Rem.(*TestInput)
	}
}

func TestIsEmpty(t *testing.T) {
	in := &TestInput{
		in: []rune{},
	}

	res := IsNot('a')(in)

	if _, didErr := res.Errored(); !didErr {
		t.Errorf("Should have Errored but didn't")
	}

	// if e, did := res.Errored(); did {
	// 		if !errors.Is(e, IncompleteErr()) {
	// 			t.Logf("%v\n",e)
	// 			t.Logf("%v\n",IncompleteErr())
	// 			t.Fail()
	// 		}
	// }

}

func TestLetter(t *testing.T) {
	in := &TestInput{
		in: []rune{'a', 'b', 'c'},
	}

	let := Letter()

	res := let(in)
	if r := res.Result.(rune); !unicode.IsLetter(r) {
		t.Errorf("Wrong!")
	}
}

func TestTakeN(t *testing.T) {
	in := &TestInput{
		in: []rune{'a', 'b', 'c', 'd', 'e', 'f'},
	}

	take := TakeN(5)

	res := take(in)

	valRes := res.Result.(*list.List)

	if valRes.Len() != 5 {
		t.Errorf("Not all is taken, expected 5, got: %d", valRes.Len())
	}
	i := 0
	for e := valRes.Front(); e != nil; e = e.Next() {
		if val := e.Value.(rune); val != (*in).in[i] {
			t.Errorf("Expected: %d, got: %d", (*in).in[i], val)
		}
		i++
	}
}

func TestTakeTill(t *testing.T) {
	in := &TestInput{
		in: []rune{'a', 'b', 'c', 'd', 'e', 'f'},
	}

	var f Predicate = func(r rune) bool {
		return r == 'e'
	}

	take := TakeTill(f)

	res := take(in)

	resList := res.Result.(*list.List)

	if resList.Len() != 4 {
		t.Errorf("Should be 4")
	}
	i := 0
	for e := resList.Front(); e != nil; e = e.Next() {
		if v := e.Value.(rune); v != (*in).in[i] {
			t.Errorf("Not the runes we expected")
		}
		i++
	}
}

func TestTakeWhile(t *testing.T) {
	in := &TestInput{
		in: []rune{'a', 'b', 'c', 'd', 'e', 'f', 'h', 'k'},
	}

	var f Predicate = func(r rune) bool {
		return r <= 'e'
	}

	take := TakeWhile(f)

	res := take(in)

	resList := res.Result.(*list.List)

	if resList.Len() != 5 {
		t.Errorf("Should be 5")
	}

	i := 0

	for e := resList.Front(); e != nil; e = e.Next() {
		if v := e.Value.(rune); v != (*in).in[i] {
			t.Errorf("Not the runes we expected")
		}
		i++
	}

}


func TestTerminated(t *testing.T) {
	in := &TestInput{
		in: []rune{'c', 'a', 't', 'd', 'o', 'g', 'h', 'k'},
	}
	match := "cat"
	parser := Terminated(match, "dog")

	res := parser(in)
	if err, did := res.Errored(); did {
		t.Errorf("Errored when it shouldn't: %s", err)
	}

	if ret := res.Result.(string); ret != "cat" {
		t.Errorf("Should return: %s, but got: %s", match, ret)
	}


	in = &TestInput{
		in: []rune{'c', 'a', 't', 'd', 'o', 'g', 'h', 'k'},
	}

	match = "cats"
	parser = Terminated(match, "dog")
	res = parser(in)
	if _, did := res.Errored(); !did {
		t.Errorf("Should have errored")
	}

	if ret := res.Result; ret != nil  {
		t.Errorf("Should return: nil, but got: %s", ret)
	}

}


func TestPreceded(t *testing.T) {
	in := &TestInput{
		in: []rune{'c', 'a', 't', 'd', 'o', 'g', 'h', 'k'},
	}
	match := "dog"
	pre := "cat"
	parser := Preceded(match, pre)

	res := parser(in)
	if err, did := res.Errored(); did {
		t.Errorf("Errored when it shouldn't: %s", err)
	}

	if ret := res.Result.(string); ret != match {
		t.Errorf("Should return: %s, but got: %s", match, ret)
	}


	in = &TestInput{
		in: []rune{'c', 'a', 't', 'd', 'o', 'g', 'h', 'k'},
	}

	match = "dogs"
	pre = "cat"
	parser = Preceded(match, pre)
	res = parser(in)
	if _, did := res.Errored(); !did {
		t.Errorf("Should have errored")
	}

	if ret := res.Result; ret != nil  {
		t.Errorf("Should return: nil, but got: %s", ret)
	}
}

func TestNumber(t *testing.T) {
	in := &TestInput{
		in: []rune{'2', '5', '6', 'd', 'o', 'g', 'h', 'k'},
	}
	expexted := 256
	res := Number()(in)
	if ans := res.Result.(int); ans != expexted {
		t.Errorf("Expected %d, found %d", expexted, ans)
	}

	rem := res.Rem.(*TestInput)
	expectedRem := &TestInput{
		in: in.in[3:],
	}
	if len(rem.in) != len(expectedRem.in) {
		t.Errorf("expected : %d, foundL %d", len(expectedRem.in), len(rem.in))
	}
} 


func TestChars(t *testing.T) {
	in := &TestInput{
		in: []rune{'2', '5', '6', 'd', 'o', 'g', 'h', 'k'},
	}

	chars := Chars([]rune{'2', '5', '6', 'd'})

	res := chars(in)

	expected := []rune{'2', '5', '6', 'd'}
	result := res.Result.([]rune)
	if e, did := res.Errored(); did {
		t.Errorf("Error: %s", e)
	}

	for i, r := range expected {
		if r != result[i] {
			t.Errorf("Should be: %v, but is %v", r, result[i])
		}
	}
}

func TestStr(t *testing.T) {
	str := "abeg"

	in := &TestInput{
		in: []rune{'a', 'b', 'e', 'g', 'o', 'g', 'h', 'k'},
	}

	strParsec := Str(str)

	res := strParsec(in);

	if err, did := res.Errored(); did {
		t.Errorf("Errored: %s", err)
	}

	if s, ok := res.Result.(string); ok {
		if s != str {
			t.Errorf("Expected: %s, found: %s", str, s)
		}
	} else {
		t.Errorf("Could not convert to string")
	}
}


func TestMany0(t *testing.T) {

	in := &TestInput{
		in: []rune{'a', 'a', 'a', 'a', 'o', 'g', 'h', 'k'},
	}

	isA := IsA('a')
	many0_IsA := isA.Many0()
	res := many0_IsA(in)

	if err, did := res.Errored(); did {
		t.Errorf("Error: %s", err)
	}

	lRes, ok := res.Result.(*list.List)

	if !ok {
		t.Errorf("SHould be a list but isn't")
	}

	if lRes.Len() != 4 {
		t.Errorf("list length should be 4")
	}

	for v := lRes.Front(); v != nil  ; v = v.Next() {
		r := v.Value.(rune)
		if r != 'a' {
			t.Errorf("Expected: %s, found: %s", "a", string(r))
		} 
	}

	in = &TestInput{
		in: []rune{'a', 'a', 'a', 'a', 'o', 'g', 'h', 'k'},
	}

	isA = IsA('b')
	many0_IsA = isA.Many0()
	res = many0_IsA(in)

	if err, did := res.Errored(); did {
		t.Errorf("Error: %s", err)
	}

	lRes, ok = res.Result.(*list.List)

	if !ok {
		t.Errorf("SHould be a list but isn't")
	}

	if lRes.Len() != 0 {
		t.Errorf("list length should be 0")
	}

}

func TestMany1 (t *testing.T) {
	in := &TestInput{
		in: []rune{'a', 'a', 'a', 'a', 'o', 'g', 'h', 'k'},
	}

	isA := IsA('a')
	many0_IsA := isA.Many0()
	res := many0_IsA(in)

	if err, did := res.Errored(); did {
		t.Errorf("Error: %s", err)
	}

	lRes, ok := res.Result.(*list.List)

	if !ok {
		t.Errorf("SHould be a list but isn't")
	}

	if lRes.Len() != 4 {
		t.Errorf("list length should be 4")
	}

	for v := lRes.Front(); v != nil  ; v = v.Next() {
		r := v.Value.(rune)
		if r != 'a' {
			t.Errorf("Expected: %s, found: %s", "a", string(r))
		} 
	}

}

func TestMany1Nil(t *testing.T) {
	in := &TestInput{
		in: []rune{'a', 'a', 'a', 'a', 'o', 'g', 'h', 'k'},
	}

	isA := IsA('b')
	many0_IsA := isA.Many0()
	res := many0_IsA(in)

	if _, did := res.Errored(); !did {
		t.Errorf(" Should have errored")
	}

	lRes, ok := res.Result.(*list.List)

	if !ok {
		t.Errorf("SHould be a list but isn't")
	}

	if lRes.Len() != 0 {
		t.Errorf("list length should be 0")
	}
}