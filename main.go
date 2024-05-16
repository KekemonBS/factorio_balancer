package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/KekemonBS/factorio_balancer/read"
)

func main() {
	var reader read.Interface
	if len(os.Args) < 2 {
		reader = read.NewPipeReader()
	} else {
		reader = read.NewFileReader(os.Args[1])
	}
	craft, err := reader.Read()
	if err != nil {
		fmt.Println(err)
		return
	}

	//craft := `  # Green circuit recipe
	//			(circuit, 0.5, 2,
	//				(metal_plate, 0.0, 1,)*1,
	//				(copper_wire, 0.5, 2,
	//					(copper_plate, 0.0, 1,)*1,
	//				)*3,
	//	    	)*2`

	//Break craft into tokens
	tokens := lex(craft)
	for _, v := range tokens {
		fmt.Printf("%-15s --- %s\n", v.name, v.what)
	}
	//Parse in tokens
	err = parse(&mleaf, tokens)
	if err != nil {
		fmt.Println(err)
		return
	}
	display(&mleaf, 0, true)

	//Transform AST
	transformTree(&mleaf)
	display(&mleaf, 0, true)

	flattenExpressions(&mleaf, nil, 0)
	display(&mleaf, 0, true)

	//Fill in comfy tree from AST
	err = fillElementTree(&mleaf, &el)
	if err != nil {
		fmt.Println(err)
		return
	}
	//Calculate equilibrium
	eq := make(map[string]result)
	calculateEquilibrium(&el, eq)
	for i := 0; i < 80; i++ {
		fmt.Print("-")
	}
	fmt.Print("\n")
	fmt.Print("\n")
	fmt.Print("\n")
	for k, v := range eq {
		fmt.Printf("%16s\t%v\n", k, v)
	}
	fmt.Print("\n")
	fmt.Print("\n")
	fmt.Print("\n")
}

//lexical analysis
//-----------------------------------------------------------------------------------

// Example
// (STR, VAL, VAL ,(STR, VAL, VAL, ...,)*MUL, (STR, VAL, VAL, ...,),)
// Tokens
// STR, VAL, SEP, MUL, OB, CB
// VAL can only be float or int
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
		if inComment {
			if t == '\n' {
				inComment = false
			}
			continue
		}
		if in(t, empty) {
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

	if temp != "" {
		stream = append(stream, tok{check(temp), temp})
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
		}
		return "INT"
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

// bracedexpr -> ( expr ) | ( expr ) mul int | nothing
func bracedexpr(parent *leaf, tokslice []tok) error {
	if tokslice[len(tokslice)-1].what != "INT" {
		tokslice = tokslice[1 : len(tokslice)-1]

		var cleaf leaf
		cleaf.name = "expression"
		cleaf.value = "EXPR"
		parent.children = append(parent.children, &cleaf)
		expr(&cleaf, tokslice)

	} else {

		var cleaf0 leaf
		cleaf0.name = "expression"
		cleaf0.value = "EXPR"
		parent.children = append(parent.children, &cleaf0)
		expr(&cleaf0, tokslice[1:len(tokslice)-3])

		var cleaf1 leaf
		cleaf1.name = "multiply"
		cleaf1.value = tokslice[len(tokslice)-2].name
		parent.children = append(parent.children, &cleaf1)

		var cleaf2 leaf
		cleaf2.name = "integer"
		cleaf2.value = tokslice[len(tokslice)-1].name
		parent.children = append(parent.children, &cleaf2)

	}

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
var idented = make(map[int]bool)

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

//solving
//-----------------------------------------------------------------------------------
//	(circuit, 0.5, 2,
//		(metal_plate, 0.0, 1,)*1,
//		(copper_wire, 0.5, 2,
//			(copper_plate, 0.0, 1,)*1,
//		)*3,
//	)

//	(NAME, TIME, QUANTITY, ARG...)

//	(what, crafting time, how many produced, (...ingredient...)*how many needed, ...)
//	EACH NAME SHOULD BE UNIQUE
//
// -----------------------------------------------------------------------------------
var el element

type element struct {
	//Element Specific
	name            string
	creationTime    float64
	createdQuantity int
	neededQuantity  int //for next craft
	//It's ingredients
	ingredients []*element
}

// fillElementTree fills in typed tree with data from AST tree
func fillElementTree(in *leaf, out *element) error {
	//Extract data if leaf has data inside (perhaps intermediary leaf)
	curr := 0
	if len(in.children) > 4 {
		out.name = in.children[0].value
		curr++

		f, err := strconv.ParseFloat(in.children[1].value, 64)
		if err != nil {
			return err
		}
		out.creationTime = f
		curr++

		i, err := strconv.Atoi(in.children[2].value)
		if err != nil {
			return err
		}
		out.createdQuantity = i
		curr++

		//Extract ingredients untill there is no more
		for in.children[curr].value == "ARG" {
			//Insert child node
			var e element
			err := fillElementTree(in.children[curr], &e)
			if err != nil {
				return err
			}
			out.ingredients = append(out.ingredients, &e)

			curr++
		}

		//curr + 1 to skip "*", last curr++ leaves next index pointing to "*"
		i, err = strconv.Atoi(in.children[curr+1].value)
		if err != nil {
			return err
		}
		out.neededQuantity = i
	}

	//If intermediary leaf, delve in
	for _, v := range in.children[curr:] {
		err := fillElementTree(v, out)
		if err != nil {
			return err
		}
	}

	return nil
}

// Calculations (make demand = surplus)

type result struct {
	demand      float64
	supply      float64
	blockScales []float64
}

func demandPerSecond(el *element) float64 {
	if el.creationTime == 0.0 {
		return -1.0
	}
	return float64(el.neededQuantity) / el.creationTime
}

func surplusPerSecond(el *element) float64 {
	if el.creationTime == 0.0 {
		return -1.0
	}
	return float64(el.createdQuantity) / el.creationTime
}

// Pass in tree root, and empty map to be filled in
func calculateEquilibrium(el *element, equilibrium map[string]result) {
	blockScale, supply := _calculateEquilibrium(el, equilibrium)
	equilibrium[el.name] = result{blockScale, supply, []float64{}}
}

// returns quantity of producers needed to satisfy parent (element of next iteration)
// demand , taking in bottom elements infinite
func _calculateEquilibrium(el *element, equilibrium map[string]result) (float64, float64) {
	var blockScales []float64
	for _, v := range el.ingredients {
		_, supply := _calculateEquilibrium(v, equilibrium)
		blockScales = append(blockScales, supply)
	}

	s := surplusPerSecond(el)
	d := demandPerSecond(el)
	if s == -1.0 || d == -1.0 {
		return 1, 1 // do not influence next blockScale
		//(infinite supply always satisfies demand)
	}
	//LCM to balance inner block
	res := floatLCM(s, d)
	sQ := res / s
	dQ := res / d //Pass as blockScale in next iteration
	equilibrium[el.name] = result{dQ, sQ, blockScales}

	return dQ, sQ
}

// least common multiple
func LCM(a, b int64) int64 {
	var lcm int64
	if a > b {
		lcm = a
	} else {
		lcm = b
	}
	for lcm%a != 0 || lcm%b != 0 {
		lcm++
	}
	return lcm
}

// only numbers with finite digits after decimal point
func floatLCM(a, b float64) float64 {
	powA := len(strings.Split(fmt.Sprintf("%.6f", a), ".")[1])
	powB := len(strings.Split(fmt.Sprintf("%.6f", b), ".")[1])
	var finPow int
	if powA > powB {
		finPow = powA
	} else {
		finPow = powB
	}
	lcm := LCM(int64(a*math.Pow(10, float64(finPow))),
		int64(b*math.Pow(10, float64(finPow))))
	return float64(lcm) * math.Pow(10, float64(-finPow))
}
