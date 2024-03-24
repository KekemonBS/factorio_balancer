package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

func main() {
	//if len(os.Args) < 2 {
	//	os.Exit(1)
	//}
	//craft := os.Args[1]
	craft := `  (circuit, 0.5, 2, 
					(metal_plate, 0.0, 1,)*1, 
					(copper_wire, 0.5, 2, 
						(copper_plate, 0.0, 1,)*1,
					)*3,
		    	)`
	tokens := lex(craft)
	for _, v := range tokens {
		fmt.Printf("%-15s --- %s\n", v.name, v.what)
	}

	parse(&mleaf, tokens)
	display(&mleaf, 0, true)

	transformTree(&mleaf)
	display(&mleaf, 0, true)

	flattenExpressions(&mleaf, nil, 0)
	display(&mleaf, 0, true)
}

//lexical analysis
//-----------------------------------------------------------------------------------

// Example
// (STR, VAL, VAL ,(STR, VAL, VAL, ...,)*MUL, (STR, VAL, VAL, ...,),)
// Tokens
// STR, VAL, SEP, MUL, OB, CB
// VAL can only be float
// MUL can only be int
type tok struct {
	what string
	name string
}

func lex(s string) []tok {
	var stream []tok
	empty := []rune{' ', '\t', '\n'}
	comment := '#'
	reader := strings.NewReader(s)

	inComment := false
	var temp string
	for {
		t, _, err := reader.ReadRune()
		if err == io.EOF {
			break
		}
		if in(t, empty) {
			continue
		}
		if inComment {
			if t == '\n' {
				inComment = false
			}
			continue
		}
		if t == comment {
			inComment = true
			continue
		}
		if t == '(' {
			stream = append(stream, tok{"OB", "("})
			continue
		}
		if t == ')' {
			stream = append(stream, tok{"CB", ")"})
			continue
		}
		if t == '*' {
			stream = append(stream, tok{"MUL", "*"})
			continue
		}
		if t == ',' {
			if temp != "" {
				stream = append(stream, tok{check(temp), temp})
				temp = ""
			}
			stream = append(stream, tok{"SEP", ","})
			continue
		}
		temp += string(t)
	}

	return stream
}

func in[T comparable](r T, arr []T) bool {
	for _, v := range arr {
		if r == v {
			return true
		}
	}
	return false
}

func check(s string) string {
	hasDot := false
	for _, v := range strings.Split(s, "") {
		if v == "." {
			hasDot = true
		}
	}
	_, err := strconv.ParseFloat(s, 32)
	if err == nil {
		if hasDot {
			return "FLOAT"
		} else {
			return "INT"
		}
	}
	return "STR"
}

//-----------------------------------------------------------------------------------

// parsing
// -----------------------------------------------------------------------------------
var mleaf leaf

type leaf struct {
	name     string
	value    string
	children []*leaf
}

// parse -> expression
func parse(parent *leaf, tokslice []tok) error {
	parent.name = "start"
	parent.value = "."

	var cleaf leaf
	cleaf.name = "braced expression"
	cleaf.value = "BREXPR"
	parent.children = append(parent.children, &cleaf)
	bracedexpr(&cleaf, tokslice)

	return nil
}

// bracedexpr -> ( expr ) | nothing
func bracedexpr(parent *leaf, tokslice []tok) error {
	tokslice = tokslice[1 : len(tokslice)-1]

	var cleaf leaf
	cleaf.name = "expression"
	cleaf.value = "EXPR"
	parent.children = append(parent.children, &cleaf)
	expr(&cleaf, tokslice)

	return nil
}

// expr -> arg, expr | nothing
func expr(parent *leaf, tokslice []tok) error {
	if len(tokslice) == 2 {
		var cleaf0 leaf
		cleaf0.name = "arguement"
		cleaf0.value = "ARG"
		parent.children = append(parent.children, &cleaf0)
		arg(&cleaf0, tokslice)

		return nil
	}

	i := 0
	braceDepth := 0
	for ; i < len(tokslice); i++ {
		if tokslice[i].what == "OB" {
			braceDepth++
		}
		if tokslice[i].what == "CB" {
			braceDepth--
		}

		if tokslice[i].what == "SEP" && braceDepth == 0 {
			break
		}
	}

	var cleaf1 leaf
	cleaf1.name = "arguement"
	cleaf1.value = "ARG"
	parent.children = append(parent.children, &cleaf1)
	arg(&cleaf1, tokslice[:i])

	//check for last item
	if i == len(tokslice)-1 {
		return nil
	}

	var cleaf2 leaf
	cleaf2.name = "expression"
	cleaf2.value = "EXPR"
	parent.children = append(parent.children, &cleaf2)
	expr(&cleaf2, tokslice[i+1:])

	return nil
}

// arg -> bracedexpr | bracedexpr mul int | float | string
func arg(parent *leaf, tokslice []tok) error {
	if tokslice[0].what == "INT" {
		var cleaf0 leaf
		cleaf0.name = "integer"
		cleaf0.value = tokslice[0].name
		parent.children = append(parent.children, &cleaf0)
	}

	if tokslice[0].what == "FLOAT" {
		var cleaf1 leaf
		cleaf1.name = "float"
		cleaf1.value = tokslice[0].name
		parent.children = append(parent.children, &cleaf1)
	}

	if tokslice[0].what == "STR" {
		var cleaf2 leaf
		cleaf2.name = "string"
		cleaf2.value = tokslice[0].name
		parent.children = append(parent.children, &cleaf2)

	}

	//Detect braced expression
	if tokslice[len(tokslice)-1].what == "CB" {
		var cleaf3 leaf
		cleaf3.name = "braced expression"
		cleaf3.value = "BREXPR"
		parent.children = append(parent.children, &cleaf3)
		bracedexpr(&cleaf3, tokslice)
	}

	if tokslice[0].what == "OB" && tokslice[len(tokslice)-1].what != "CB" {
		//find closing brace
		var i int
		braceDepth := 0
		for ; i < len(tokslice); i++ {
			if tokslice[i].what == "OB" {
				braceDepth++
			}
			if tokslice[i].what == "CB" {
				braceDepth--
			}
			if braceDepth == 0 {
				break
			}
		}
		var cleaf4 leaf
		cleaf4.name = "braced expression"
		cleaf4.value = "BREXPR"
		parent.children = append(parent.children, &cleaf4)
		bracedexpr(&cleaf4, tokslice[:i+1])

		var cleaf5 leaf
		cleaf5.name = "multiply"
		cleaf5.value = tokslice[len(tokslice)-2].name
		parent.children = append(parent.children, &cleaf5)

		var cleaf6 leaf
		cleaf6.name = "integer"
		cleaf6.value = tokslice[len(tokslice)-1].name
		parent.children = append(parent.children, &cleaf6)
	}
	return nil
}

// transformTree rewires node if it has <= 1 children and it is not in exeptions (shrinks tree)
func transformTree(node *leaf) {
	exceptions := []string{"."}

	for _, i := range node.children {
		if len(node.children) > 1 || in(node.value, exceptions) {
			transformTree(i)
		} else if len(node.children) <= 1 {
			node.value = i.value
			node.children = i.children
			transformTree(node)
		}
	}
}

// flattenExpressions places resolved expressions on one level
func flattenExpressions(node *leaf, parent *leaf, pos int) {
	for k, v := range node.children {
		flattenExpressions(v, node, k)
	}

	toflat := []string{"EXPR"}
	if in(node.value, toflat) {
		var nch []*leaf
		nch = append(nch, parent.children[:pos]...)
		nch = append(nch, node.children...)
		nch = append(nch, parent.children[pos+1:]...)
		parent.children = nch
	}

}

// display displays any tree
var idented map[int]bool = make(map[int]bool)

func display(n *leaf, ident int, last bool) {
	if len(n.children) > 1 {
		idented[ident] = true
	}
	ident++
	s := n.value
	if ident > 0 {
		for i := 0; i < ident-1; i++ {
			if i < ident-2 {
				if idented[i] {
					fmt.Print("\u2502 ")
				} else {
					fmt.Print("   ")
				}
			} else if !last {
				fmt.Print("\u251c\u2500\u2500 ")
			} else {
				fmt.Print("\u2514\u2500\u2500 ")
				idented[i] = false
			}
		}
		fmt.Print("")
	}
	fmt.Printf("%s\n", s)
	for i := 0; i < len(n.children); i++ {
		if i == len(n.children)-1 {
			display(n.children[i], ident, true)
		} else {
			display(n.children[i], ident, false)
		}
	}
}

//-----------------------------------------------------------------------------------
