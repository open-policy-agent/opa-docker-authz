// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package topdown

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tchap/go-patricia/v2/patricia"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/topdown/builtins"
)

func builtinAnyPrefixMatch(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	a, b := operands[0].Value, operands[1].Value

	var strs []string
	switch a := a.(type) {
	case ast.String:
		strs = []string{string(a)}
	case *ast.Array, ast.Set:
		var err error
		strs, err = builtins.StringSliceOperand(a, 1)
		if err != nil {
			return err
		}
	default:
		return builtins.NewOperandTypeErr(1, a, "string", "set", "array")
	}

	var prefixes []string
	switch b := b.(type) {
	case ast.String:
		prefixes = []string{string(b)}
	case *ast.Array, ast.Set:
		var err error
		prefixes, err = builtins.StringSliceOperand(b, 2)
		if err != nil {
			return err
		}
	default:
		return builtins.NewOperandTypeErr(2, b, "string", "set", "array")
	}

	return iter(ast.InternedTerm(anyStartsWithAny(strs, prefixes)))
}

func builtinAnySuffixMatch(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	a, b := operands[0].Value, operands[1].Value

	var strsReversed []string
	switch a := a.(type) {
	case ast.String:
		strsReversed = []string{reverseString(string(a))}
	case *ast.Array, ast.Set:
		strs, err := builtins.StringSliceOperand(a, 1)
		if err != nil {
			return err
		}
		strsReversed = make([]string, len(strs))
		for i := range strs {
			strsReversed[i] = reverseString(strs[i])
		}
	default:
		return builtins.NewOperandTypeErr(1, a, "string", "set", "array")
	}

	var suffixesReversed []string
	switch b := b.(type) {
	case ast.String:
		suffixesReversed = []string{reverseString(string(b))}
	case *ast.Array, ast.Set:
		suffixes, err := builtins.StringSliceOperand(b, 2)
		if err != nil {
			return err
		}
		suffixesReversed = make([]string, len(suffixes))
		for i := range suffixes {
			suffixesReversed[i] = reverseString(suffixes[i])
		}
	default:
		return builtins.NewOperandTypeErr(2, b, "string", "set", "array")
	}

	return iter(ast.InternedTerm(anyStartsWithAny(strsReversed, suffixesReversed)))
}

func anyStartsWithAny(strs []string, prefixes []string) bool {
	if len(strs) == 0 || len(prefixes) == 0 {
		return false
	}
	if len(strs) == 1 && len(prefixes) == 1 {
		return strings.HasPrefix(strs[0], prefixes[0])
	}

	trie := patricia.NewTrie()
	for i := range strs {
		trie.Insert([]byte(strs[i]), true)
	}

	for i := range prefixes {
		if trie.MatchSubtree([]byte(prefixes[i])) {
			return true
		}
	}

	return false
}

func builtinFormatInt(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {

	input, err := builtins.NumberOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	base, err := builtins.NumberOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}

	var format string
	switch base {
	case ast.Number("2"):
		format = "%b"
	case ast.Number("8"):
		format = "%o"
	case ast.Number("10"):
		if i, ok := input.Int(); ok {
			return iter(ast.InternedIntegerString(i))
		}
		format = "%d"
	case ast.Number("16"):
		format = "%x"
	default:
		return builtins.NewOperandEnumErr(2, "2", "8", "10", "16")
	}

	f := builtins.NumberToFloat(input)
	i, _ := f.Int(nil)

	return iter(ast.InternedTerm(fmt.Sprintf(format, i)))
}

func builtinConcat(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	join, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	// fast path for empty or single string array/set, allocates no memory
	if term, ok := zeroOrOneStringTerm(operands[1].Value); ok {
		return iter(term)
	}

	// NOTE(anderseknert):
	// More or less Go's strings.Join implementation, but where we avoid
	// creating an intermediate []string slice to pass to that function,
	// as that's expensive (3.5x more space allocated). Instead we build
	// the string directly using a strings.Builder to concatenate the string
	// values from the array/set with the separator.
	n := 0
	switch b := operands[1].Value.(type) {
	case *ast.Array:
		l := b.Len()
		for i := range l {
			s, ok := b.Elem(i).Value.(ast.String)
			if !ok {
				return builtins.NewOperandElementErr(2, b, b.Elem(i).Value, "string")
			}
			n += len(s)
		}
		sep := string(join)
		n += len(sep) * (l - 1)
		var sb strings.Builder
		sb.Grow(n)
		sb.WriteString(string(b.Elem(0).Value.(ast.String)))
		if sep == "" {
			for i := 1; i < l; i++ {
				sb.WriteString(string(b.Elem(i).Value.(ast.String)))
			}
		} else if len(sep) == 1 {
			// when the separator is a single byte, sb.WriteByte is substantially faster
			bsep := sep[0]
			for i := 1; i < l; i++ {
				sb.WriteByte(bsep)
				sb.WriteString(string(b.Elem(i).Value.(ast.String)))
			}
		} else {
			// for longer separators, there is no such difference between WriteString and Write
			for i := 1; i < l; i++ {
				sb.WriteString(sep)
				sb.WriteString(string(b.Elem(i).Value.(ast.String)))
			}
		}
		return iter(ast.InternedTerm(sb.String()))
	case ast.Set:
		for _, v := range b.Slice() {
			s, ok := v.Value.(ast.String)
			if !ok {
				return builtins.NewOperandElementErr(2, b, v.Value, "string")
			}
			n += len(s)
		}
		sep := string(join)
		l := b.Len()
		n += len(sep) * (l - 1)
		var sb strings.Builder
		sb.Grow(n)
		for i, v := range b.Slice() {
			sb.WriteString(string(v.Value.(ast.String)))
			if i < l-1 {
				sb.WriteString(sep)
			}
		}
		return iter(ast.InternedTerm(sb.String()))
	}

	return builtins.NewOperandTypeErr(2, operands[1].Value, "set", "array")
}

func zeroOrOneStringTerm(a ast.Value) (*ast.Term, bool) {
	switch b := a.(type) {
	case *ast.Array:
		if b.Len() == 0 {
			return ast.InternedEmptyString, true
		}
		if b.Len() == 1 {
			e := b.Elem(0)
			if _, ok := e.Value.(ast.String); ok {
				return e, true
			}
		}
	case ast.Set:
		if b.Len() == 0 {
			return ast.InternedEmptyString, true
		}
		if b.Len() == 1 {
			e := b.Slice()[0]
			if _, ok := e.Value.(ast.String); ok {
				return e, true
			}
		}
	}
	return nil, false
}

func runesEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func builtinIndexOf(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	base, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	search, err := builtins.StringOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}
	if len(string(search)) == 0 {
		return errors.New("empty search character")
	}

	if isASCII(string(base)) && isASCII(string(search)) {
		// this is a false positive in the indexAlloc rule that thinks
		// we're converting byte arrays to strings
		//nolint:gocritic
		return iter(ast.InternedTerm(strings.Index(string(base), string(search))))
	}

	baseRunes := []rune(string(base))
	searchRunes := []rune(string(search))
	searchLen := len(searchRunes)

	for i, r := range baseRunes {
		if len(baseRunes) >= i+searchLen {
			if r == searchRunes[0] && runesEqual(baseRunes[i:i+searchLen], searchRunes) {
				return iter(ast.InternedTerm(i))
			}
		} else {
			break
		}
	}

	return iter(ast.InternedTerm(-1))
}

func builtinIndexOfN(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	base, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	search, err := builtins.StringOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}
	if len(string(search)) == 0 {
		return errors.New("empty search character")
	}

	baseRunes := []rune(string(base))
	searchRunes := []rune(string(search))
	searchLen := len(searchRunes)

	var arr []*ast.Term
	for i, r := range baseRunes {
		if len(baseRunes) >= i+searchLen {
			if r == searchRunes[0] && runesEqual(baseRunes[i:i+searchLen], searchRunes) {
				arr = append(arr, ast.InternedTerm(i))
			}
		} else {
			break
		}
	}

	return iter(ast.ArrayTerm(arr...))
}

func builtinSubstring(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {

	base, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	startIndex, err := builtins.IntOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}

	length, err := builtins.IntOperand(operands[2].Value, 3)
	if err != nil {
		return err
	}

	if startIndex < 0 {
		return errors.New("negative offset")
	}

	sbase := string(base)
	if sbase == "" {
		return iter(ast.InternedEmptyString)
	}

	// Optimized path for the likely common case of ASCII strings.
	// This allocates less memory and runs in about 1/3 the time.
	if isASCII(sbase) {
		if startIndex >= len(sbase) {
			return iter(ast.InternedEmptyString)
		}

		if length < 0 {
			return iter(ast.InternedTerm(sbase[startIndex:]))
		}

		if startIndex == 0 && length >= len(sbase) {
			return iter(operands[0])
		}

		upto := min(len(sbase), startIndex+length)
		return iter(ast.InternedTerm(sbase[startIndex:upto]))
	}

	if startIndex == 0 && length >= utf8.RuneCountInString(sbase) {
		return iter(operands[0])
	}

	runes := []rune(base)

	if startIndex >= len(runes) {
		return iter(ast.InternedEmptyString)
	}

	var s string
	if length < 0 {
		s = string(runes[startIndex:])
	} else {
		upto := min(len(runes), startIndex+length)
		s = string(runes[startIndex:upto])
	}

	return iter(ast.InternedTerm(s))
}

func isASCII(s string) bool {
	for i := range len(s) {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func builtinContains(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	s, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	substr, err := builtins.StringOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}

	return iter(ast.InternedTerm(strings.Contains(string(s), string(substr))))
}

func builtinStringCount(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	s, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	substr, err := builtins.StringOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}

	baseTerm := string(s)
	searchTerm := string(substr)
	count := strings.Count(baseTerm, searchTerm)

	return iter(ast.InternedTerm(count))
}

func builtinStartsWith(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	s, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	prefix, err := builtins.StringOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}

	return iter(ast.InternedTerm(strings.HasPrefix(string(s), string(prefix))))
}

func builtinEndsWith(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	s, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	suffix, err := builtins.StringOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}

	return iter(ast.InternedTerm(strings.HasSuffix(string(s), string(suffix))))
}

func builtinLower(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	s, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	arg := string(s)
	low := strings.ToLower(arg)

	if arg == low {
		return iter(operands[0])
	}

	return iter(ast.InternedTerm(low))
}

func builtinUpper(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	s, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	arg := string(s)
	upp := strings.ToUpper(arg)

	if arg == upp {
		return iter(operands[0])
	}

	return iter(ast.InternedTerm(upp))
}

func builtinSplit(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	s, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	d, err := builtins.StringOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}

	if !strings.Contains(string(s), string(d)) {
		return iter(ast.ArrayTerm(operands[0]))
	}

	elems := strings.Split(string(s), string(d))
	arr := make([]*ast.Term, len(elems))

	for i := range elems {
		arr[i] = ast.InternedTerm(elems[i])
	}

	return iter(ast.ArrayTerm(arr...))
}

func builtinReplace(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	s, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	old, err := builtins.StringOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}

	n, err := builtins.StringOperand(operands[2].Value, 3)
	if err != nil {
		return err
	}

	replaced := strings.ReplaceAll(string(s), string(old), string(n))
	if replaced == string(s) {
		return iter(operands[0])
	}

	return iter(ast.InternedTerm(replaced))
}

func builtinReplaceN(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	patterns, err := builtins.ObjectOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}
	keys := patterns.Keys()
	sort.Slice(keys, func(i, j int) bool { return ast.Compare(keys[i].Value, keys[j].Value) < 0 })

	s, err := builtins.StringOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}

	oldnewArr := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		keyVal, ok := k.Value.(ast.String)
		if !ok {
			return builtins.NewOperandErr(1, "non-string key found in pattern object")
		}
		val := patterns.Get(k) // cannot be nil
		strVal, ok := val.Value.(ast.String)
		if !ok {
			return builtins.NewOperandErr(1, "non-string value found in pattern object")
		}
		oldnewArr = append(oldnewArr, string(keyVal), string(strVal))
	}

	return iter(ast.InternedTerm(strings.NewReplacer(oldnewArr...).Replace(string(s))))
}

func builtinTrim(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	s, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	c, err := builtins.StringOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}

	trimmed := strings.Trim(string(s), string(c))
	if trimmed == string(s) {
		return iter(operands[0])
	}

	return iter(ast.InternedTerm(strings.Trim(string(s), string(c))))
}

func builtinTrimLeft(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	s, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	c, err := builtins.StringOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}

	trimmed := strings.TrimLeft(string(s), string(c))
	if trimmed == string(s) {
		return iter(operands[0])
	}

	return iter(ast.InternedTerm(trimmed))
}

func builtinTrimPrefix(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	s, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	pre, err := builtins.StringOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}

	trimmed := strings.TrimPrefix(string(s), string(pre))
	if trimmed == string(s) {
		return iter(operands[0])
	}

	return iter(ast.InternedTerm(trimmed))
}

func builtinTrimRight(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	s, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	c, err := builtins.StringOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}

	trimmed := strings.TrimRight(string(s), string(c))
	if trimmed == string(s) {
		return iter(operands[0])
	}

	return iter(ast.InternedTerm(trimmed))
}

func builtinTrimSuffix(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	s, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	suf, err := builtins.StringOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}

	trimmed := strings.TrimSuffix(string(s), string(suf))
	if trimmed == string(s) {
		return iter(operands[0])
	}

	return iter(ast.InternedTerm(trimmed))
}

func builtinTrimSpace(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	s, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	trimmed := strings.TrimSpace(string(s))
	if trimmed == string(s) {
		return iter(operands[0])
	}

	return iter(ast.InternedTerm(trimmed))
}

func builtinSprintf(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	s, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	astArr, ok := operands[1].Value.(*ast.Array)
	if !ok {
		return builtins.NewOperandTypeErr(2, operands[1].Value, "array")
	}

	// Optimized path for where sprintf is used as a "to_string" function for
	// a single integer, i.e. sprintf("%d", [x]) where x is an integer.
	if s == "%d" && astArr.Len() == 1 {
		if n, ok := astArr.Elem(0).Value.(ast.Number); ok {
			if i, ok := n.Int(); ok {
				if interned := ast.InternedIntegerString(i); interned != nil {
					return iter(interned)
				}
				return iter(ast.StringTerm(strconv.Itoa(i)))
			}
		}
	}

	args := make([]any, astArr.Len())

	for i := range args {
		switch v := astArr.Elem(i).Value.(type) {
		case ast.Number:
			if n, ok := v.Int(); ok {
				args[i] = n
			} else if b, ok := new(big.Int).SetString(v.String(), 10); ok {
				args[i] = b
			} else if f, ok := v.Float64(); ok {
				args[i] = f
			} else {
				args[i] = v.String()
			}
		case ast.String:
			args[i] = string(v)
		default:
			args[i] = astArr.Elem(i).String()
		}
	}

	return iter(ast.InternedTerm(fmt.Sprintf(string(s), args...)))
}

func builtinReverse(_ BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	s, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	return iter(ast.InternedTerm(reverseString(string(s))))
}

func reverseString(str string) string {
	var buf []byte
	var arr [255]byte
	size := len(str)

	if size < 255 {
		buf = arr[:size:size]
	} else {
		buf = make([]byte, size)
	}

	for start := 0; start < size; {
		r, n := utf8.DecodeRuneInString(str[start:])
		start += n
		utf8.EncodeRune(buf[size-start:], r)
	}

	return string(buf)
}

func init() {
	RegisterBuiltinFunc(ast.FormatInt.Name, builtinFormatInt)
	RegisterBuiltinFunc(ast.Concat.Name, builtinConcat)
	RegisterBuiltinFunc(ast.IndexOf.Name, builtinIndexOf)
	RegisterBuiltinFunc(ast.IndexOfN.Name, builtinIndexOfN)
	RegisterBuiltinFunc(ast.Substring.Name, builtinSubstring)
	RegisterBuiltinFunc(ast.Contains.Name, builtinContains)
	RegisterBuiltinFunc(ast.StringCount.Name, builtinStringCount)
	RegisterBuiltinFunc(ast.StartsWith.Name, builtinStartsWith)
	RegisterBuiltinFunc(ast.EndsWith.Name, builtinEndsWith)
	RegisterBuiltinFunc(ast.Upper.Name, builtinUpper)
	RegisterBuiltinFunc(ast.Lower.Name, builtinLower)
	RegisterBuiltinFunc(ast.Split.Name, builtinSplit)
	RegisterBuiltinFunc(ast.Replace.Name, builtinReplace)
	RegisterBuiltinFunc(ast.ReplaceN.Name, builtinReplaceN)
	RegisterBuiltinFunc(ast.Trim.Name, builtinTrim)
	RegisterBuiltinFunc(ast.TrimLeft.Name, builtinTrimLeft)
	RegisterBuiltinFunc(ast.TrimPrefix.Name, builtinTrimPrefix)
	RegisterBuiltinFunc(ast.TrimRight.Name, builtinTrimRight)
	RegisterBuiltinFunc(ast.TrimSuffix.Name, builtinTrimSuffix)
	RegisterBuiltinFunc(ast.TrimSpace.Name, builtinTrimSpace)
	RegisterBuiltinFunc(ast.Sprintf.Name, builtinSprintf)
	RegisterBuiltinFunc(ast.AnyPrefixMatch.Name, builtinAnyPrefixMatch)
	RegisterBuiltinFunc(ast.AnySuffixMatch.Name, builtinAnySuffixMatch)
	RegisterBuiltinFunc(ast.StringReverse.Name, builtinReverse)
}
