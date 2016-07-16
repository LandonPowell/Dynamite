package main

import (
    "os"
    "fmt"
    "flag"
    "bufio"
    "regexp"
    "strconv"
    "strings"
    "io/ioutil"
)

func contains(x string, z string) bool {    // Checks if char is in string.
    for _, y := range z { if x == string(y) { return true } }
    return false
}

var tokenList = []string{}

// This is mostly self-explaining, but it's the tokenizer (obviously).
func lexer(plaintext string) []string {     // Returns a list of tokens.
    strings     := "'(\\\\'|[^'])+'|\"[^\n]+"       // Regex for strings.
    brackets    := "[\\[\\](){}:=]"                 // Regex for bracket chars.
    names       := "[\\w-@^-`~\\|*-/!-&;?]+"        // Regex for var names.

    tokens  := regexp.MustCompile( strings+"|"+brackets+"|"+names )
    return tokens.FindAllString(plaintext, -1)
}

type tree struct { // Tree of processes. It can also be a value.
    value   string // If it's a value.
    args    []tree // If it's a bunch of subprocesses.
}

var programTree = tree { // Default tree of the entire program.
    value: "run",
    args: []tree{},
}

// Instead of using 'tokenList' as an arg, we use a global token list. Recursion + Scope == Pain.
func parseNext() tree { // This is the actual meat of 'parser'.
    var currentTree = tree {    // Define the current token as a tree.
        value: tokenList[0],
        args: []tree{},
    }
    tokenList = tokenList[1:]   // Removes the first element in the slice.

    if len(tokenList) > 0 { // Everybody taking the chance. Safety dance.
        if contains(tokenList[0], "{[(f") { // If the next token is an opening bracket.
            tokenList = tokenList[1:]   // Remove it.
            currentTree.args = parser() // Make a nest of it.
        } else if tokenList[0] == ":" {     // If the next token is a monogomy symbol.
            tokenList = tokenList[1:]   // Remove it.
            currentTree.args = append(currentTree.args, parseNext())    // Nest it.
        } else if tokenList[0] == "=" {     // If the next token is a decleration.
            tokenList = tokenList[1:]   // Remove it.
            currentTree = tree {    // Make the tree into a 'set' function.
                value: "set",
                args: []tree { currentTree, parseNext() },  // Which sets currentTree as the next function.
            }
        }
    }

    return currentTree
}

func parser() []tree {
    var treeList = []tree{} // Define the empty tree list.
    
    for len(tokenList) > 0 && !contains(tokenList[0], "j)]}") { // So long as the current token isn't a closing character.
        treeList = append(treeList, parseNext())    // Append the next parsed tree to the tree list.
    }
    if len(tokenList) > 0 && contains(tokenList[0], "j)]}") {   // If the next token is a closing character,
        tokenList = tokenList[1:]                               // remove it.
    }

    return treeList // Return the tree list.
}

type atom struct {
    Type    string  // The type of variable the atom is.

    str     string  // If the type is 'str' (a string) this is the value.
    num     float64 // 'num' (a number)
    bit     bool    // 'bit' (a 1 or 0, True or False)
    fun     tree    // 'fun' (a function)
    list    []tree  // 'list'
    file    []string// 'file'
}

func atomize(preAtom tree) atom {
    var postAtom atom

    firstChar := string(preAtom.value[0]) 
    if firstChar == "\"" || firstChar == "'" {  // If the value is a string.

        postAtom.Type   = "str"
        if firstChar == "\"" {
            postAtom.str    = preAtom.value[1:] 
        } else {
            postAtom.str    = strings.Replace( 
                                strings.Replace(
                                    preAtom.value[1:len(preAtom.value)-1], 
                                    "\\'", "'", -1 ), 
                                "\\\\", "\\", -1)
        }

    } else if _, err := strconv.ParseFloat(preAtom.value, 64); err == nil {  // If the value is a number.

        postAtom.Type   = "num"
        postAtom.num, _ = strconv.ParseFloat(preAtom.value, 64)

    } else if preAtom.value == "on" || preAtom.value == "off" { // If the value is a bit/bool.

        postAtom.Type   = "bit"
        if preAtom.value == "on" {
            postAtom.bit = true
        } else {
            postAtom.bit = false 
        }

    } else if preAtom.value == "list" {

        postAtom.Type = "list"
        postAtom.list = preAtom.args

    } else { 
        postAtom.Type = "CAN NOT PARSE" 
    }

    return postAtom
}

var variables = make( map[string]tree )

func evalAll(treeList []tree) tree {
    for _, x := range(treeList) {
        currentRun := evaluator(x)

        if currentRun.value == "ERROR" {
            fmt.Println(" -error- ")
            fmt.Println(currentRun.args[0].value)
        } else if currentRun.value == "return" {
            return evaluator(currentRun.args[0])
        }
    }
    return tree { value: "False" }
}

var lastCondition bool = true; // This checks if the last conditional was true or not, for the sake of the elf function (else if)
func evaluator(subTree tree) tree {
    if val, ok := variables[subTree.value]; ok {    // This returns variable values.

        return evaluator(val)

    } else if subTree.value == "run" {  // This is a function similair to an anonymous function.

        return evalAll(subTree.args)

    } else if subTree.value == "set" {  // Sets variables.

        if len(subTree.args) >= 2 {
            variables[subTree.args[0].value] = subTree.args[1]
            return subTree.args[1]
        }

        return tree { value: "off" }

    // The following few values are in charge of conditionals.
    } else if subTree.value == "?" || subTree.value == "if" {   // Simple conditional. "If"

        if len(subTree.args) >= 2 && evaluator(subTree.args[0]).value == "on" {

            lastCondition = true;
            return evalAll(subTree.args[1:])

        } else {
            lastCondition = false;
        }

        return tree { value: "off" }

    } else if subTree.value == "-?" || subTree.value == "elf" { // Otherwise if conditional. "Else if"

        if !lastCondition && len(subTree.args) >= 2 && 
            evaluator(subTree.args[0]).value == "on" {

            lastCondition = true;
            return evalAll(subTree.args[1:])

        }

        return tree { value: "off" }

    } else if subTree.value == "&?" || subTree.value == "alf" { // Also if conditional.

        if lastCondition && len(subTree.args) >= 2 && 
            evaluator(subTree.args[0]).value == "on" {

            lastCondition = true;
            return evalAll(subTree.args[1:])

        }

        return tree { value: "off" }

    
    } else if subTree.value == "--" || subTree.value == "else" {// Otherwise conditional. "Else"

        if !lastCondition && len(subTree.args) >= 1 {
            return evalAll(subTree.args)
        }

        return tree { value: "off" }

    } else if subTree.value == "&&" || subTree.value == "also" {// Also conditional.

        if lastCondition && len(subTree.args) >= 1 {
            return evalAll(subTree.args)
        }

        return tree { value: "off" }

    } else if subTree.value == "o" || subTree.value == "out" {  // This is a formated output, or 'println' minus templating.

        for _, x := range(subTree.args) {
            
            printArg := atomize(evaluator(x))

            switch printArg.Type {  // Too bad I can't use printArg['str'] syntax.
            case "str"  : fmt.Println(printArg.str)
            case "num"  : fmt.Println(printArg.num)
            case "fun"  : fmt.Println(printArg.fun)
            case "file" : 
                if len(subTree.args) >= 2 { // To-do
                    fmt.Println(printArg.file)
                } else {
                    fmt.Println(printArg.file)
                }
            }

            return evaluator(subTree.args[0])
        }

        if len(subTree.args) == 0 {
            fmt.Println()
            return tree { value: "off" }
        }

    } else if subTree.value == "not" || subTree.value == "!" {  // Boolean 'not'.

        if len(subTree.args) == 1 && evaluator(subTree.args[0]).value == "off" {
            return tree { value: "on" }
        }
        return tree { value: "off" }

    } else if subTree.value == "or" {   // Boolean 'or'.

        for _, x := range(subTree.args) {
            if evaluator(x).value == "on" {
                return tree { value: "on" }
            }
        }
        return tree { value: "off" }

    } else if subTree.value == "and" {  // Boolean 'and'.

        for _, x := range(subTree.args) {
            if evaluator(x).value == "off" {
                return tree { value: "off" }
            }
        }
        return tree { value: "on" }

    } else if subTree.value == "print" || subTree.value == "p" {    // Print without a linebreak at the end.

        for _, x := range(subTree.args) {
            printArg := atomize(evaluator(x))

            switch printArg.Type {
            case "str"  : fmt.Print(printArg.str)
            case "num"  : fmt.Print(printArg.num)
            default     : return tree { value: "off" }
            }
        }
    
        return tree { value: "on" }

    } else if subTree.value == "rawOut" {   // This outputs the plaintext of a tree.

        if len(subTree.args) > 0 {
            fmt.Println(evaluator(subTree.args[0]))
            return subTree.args[0]
        }

        return tree { value: "off" }

    } else if atomize(subTree).Type != "CAN NOT PARSE" {    // Raw Data Types, such as 'str', 'num', etc. 
                                                            // It's placed here so that it'll be reached quickly
        return subTree

    } else if subTree.value == "each" || subTree.value == "e" {

        if len(subTree.args) >= 3 {
            for _, x := range(evaluator(subTree.args[1]).args) {
                variables[subTree.args[0].value] = x
                evalAll(subTree.args[2:])
            }
            return tree { value: "on" }
        }
        return tree { value: "off" }

    } else if subTree.value == "in" {   // Standard input.

        reader  := bufio.NewReader(os.Stdin)
        in, _   := reader.ReadString('\n')
        return tree {
            value: "\"" + in[:len(in)-1],
            args: []tree{},
        }
    
    } else if subTree.value == "equals" || subTree.value == "is" {  // Check for equality.

        firstTree := evaluator(subTree.args[0])
        for _, x := range(subTree.args[1:]) {
            x = evaluator(x)

            if len(x.args) != len(firstTree.args) ||
                firstTree.value != x.value {

                return tree { value: "off" }

            }

            for i, y := range(x.args) {
                if y.value != firstTree.args[i].value {
                    return tree { value: "off" }
                }
            }
        }
        return tree { value: "on" }

    // Mathmatical operators, such as adding numbers, checking for divisibility, etc.
    } else if subTree.value == "sum" {  // Sum all numerical args together.

        number := 0.0
        for _, x := range(subTree.args) {
            number += atomize(evaluator(x)).num
        }
        return tree { value: strconv.FormatFloat(number, 'E', -1, 64) }

    } else if subTree.value == "subtract" {  // Starting with the leftmost number, subtract all numbers after it.

        if len(subTree.args) >= 2 {
            number := atomize(evaluator(subTree.args[0])).num
            for _, x := range(subTree.args[1:]) {
                number -= atomize(evaluator(x)).num
            }
            return tree { value: strconv.FormatFloat(number, 'E', -1, 64) }
        }
        return tree { 
            value: "ERROR",
            args: []tree{
                tree {
                    value:  "The 'subtract' function takes two or more num args.",
                },
            },
        }

    } else if subTree.value == "multiply" {  // Multiply all numerical args together.

        number := 1.0
        for _, x := range(subTree.args) {
            number *= atomize(evaluator(x)).num
        }
        return tree { value: strconv.FormatFloat(number, 'E', -1, 64) }

    } else if subTree.value == "divide" {  // Starting with the leftmost number, divide it by all following numbers.

        if len(subTree.args) >= 2 {
            number := atomize(evaluator(subTree.args[0])).num
            for _, x := range(subTree.args[1:]) {
                number /= atomize(evaluator(x)).num
            }
            return tree { value: strconv.FormatFloat(number, 'E', -1, 64) }
        }
        return tree { 
            value: "ERROR",
            args: []tree{
                tree {
                    value:  "The 'divide' function takes two or more num args.",
                },
            },
        }
    
    } else if subTree.value == "mod" {

        if len(subTree.args) == 2 {

            arg1 := atomize(evaluator(subTree.args[0]))
            arg2 := atomize(evaluator(subTree.args[1]))
            return tree { value: strconv.Itoa( int(arg1.num) % int(arg2.num) ) }
        }

        return tree {   // Returns an error message for undefined names.
            value: "ERROR",
            args: []tree{
                tree { 
                    value: "The 'mod' function takes exactly two arguments.",
                },
            },
        }

    } else if subTree.value == "divisible" {    // Check for divisibility. 

        if len(subTree.args) == 2 {

            arg1 := atomize(evaluator(subTree.args[0]))
            arg2 := atomize(evaluator(subTree.args[1]))

            if arg1.Type == "num" && arg2.Type == "num" {

                if  arg1.num != 0 && arg2.num != 0 &&
                    int(arg1.num) % int(arg2.num) == 0 {

                    return tree { value: "on"}

                }
                return tree { value: "off" }
            }
            return tree {   // Returns an error message for undefined names.
                value: "ERROR",
                args: []tree{
                    tree {
                        value:  "The 'divisible' function only takes 'num' types.\n" +
                                "You've given '" + arg1.Type + "' and '" + arg2.Type + "'.",
                    },
                },
            }
        }

        return tree {   // Returns an error message for undefined names.
            value: "ERROR",
            args: []tree{
                tree { 
                    value: "The 'divisible' function takes exactly two arguments.",
                },
            },
        }

    } else if subTree.value == "concat" {   // Concatonate strings.

        newString := "\""

        for _, x := range(subTree.args) {
            subString := atomize(evaluator(x))
            newString += subString.str

            if subString.Type != "str" {
                fmt.Println(" -error- ")
                fmt.Println("You used " + x.value + ", a '" + subString.Type + "' as a str.")
            }
        }

        return tree { value: newString, args: []tree{} }

    } else if subTree.value == "range" {
        
        var generatedList = tree { 
            value: "list", 
            args: []tree{},
        }
        
        var start   = 0
        var end     = 100
        var iterate = 1

        switch len(subTree.args) {
        case 1:
            end     = int(atomize(evaluator(subTree.args[0])).num)
        case 2:
            start   = int(atomize(evaluator(subTree.args[0])).num)
            end     = int(atomize(evaluator(subTree.args[1])).num)
        case 3:
            start   = int(atomize(evaluator(subTree.args[0])).num)
            end     = int(atomize(evaluator(subTree.args[1])).num)
            iterate = int(atomize(evaluator(subTree.args[2])).num)
        }

        for x := start; x <= end; x += iterate {
            generatedList.args = append(generatedList.args, 
                tree { 
                    value: strconv.Itoa(x), 
                    args: []tree{},
                })
        }

        return generatedList

    } else if subTree.value == "kill" {

        // To-do
        return tree { value: "This is a built in function of DeviousYarn." }

    }

    return tree {   // Returns an error message for undefined names.
        value: "ERROR",
        args: []tree{
            tree { 
                value: "The word '" + subTree.value +  "' means nothing.",
            },
        },
    }
}

func execute(input string) {
    tokenList           = lexer     ( input )
    programTree.args    = parser    ( )
    evaluator ( programTree )
}

func prompt() {
    reader := bufio.NewReader(os.Stdin)

    var input string
    for input != "kill\n" {

        fmt.Println(" -input- ")
        input, _ = reader.ReadString('\n')

        fmt.Println(" -output- ")
        execute( input )

    }
    fmt.Println(" -Thanks for using DeviousYarn-! ")
}

func runFile(filename string) {
    file, err := ioutil.ReadFile( filename )
    if err != nil {
        fmt.Println("The file '" + filename + "' could not be opened.")
    } else {
        execute( string( file ) )
    }
}

func main() {
    flag.Parse()
    if len(flag.Args()) >= 2 {

        switch flag.Arg(0) {
            case "runFile": runFile(flag.Arg(1))
            case "load":    // To-do
            case "run":     execute(flag.Arg(1))

            default: fmt.Println(
                                    "That argument '" + 
                                    flag.Arg(0) + 
                                    "' is not recognized.")
        }

    } else if len( flag.Args() ) >= 1 {

        fileName    := strings.Split(flag.Arg(0), ".")
        extension   := fileName[len(fileName)-1]

        if extension == "die" || extension[:2] == "dy" {    // All files ending in '*.die' or '*.dy*' get executed.
            runFile(flag.Arg(0))
        } else { // Load a text file as a variable.
            // To-do
        }

    } else {
        prompt()
    }
}