package main

import (
    "os"
    "fmt"
    "flag"
    "bufio"
    "regexp"
    "strconv"
    "strings"
    "net/http"
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
                args: []tree{ currentTree, parseNext() },  // Which sets currentTree as the next function.
            }
        }
    }

    return currentTree
}

func parser() []tree{
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
    file    []tree  // 'file'
    website []tree  // 'website'
}

func atomizer(preAtom tree) atom {
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

    } else if preAtom.value == "file" {

        postAtom.Type = "file"
        postAtom.file = preAtom.args

    } else if preAtom.value == "website" {

        postAtom.Type       = "website"
        postAtom.website    = preAtom.args

    } else { 
        postAtom.Type = "CAN NOT PARSE" 
    }

    return postAtom
}

var variables = make( map[string]tree )

func loadFile(fileName string) tree {
    file, err := ioutil.ReadFile( fileName )
    if err != nil {
        fmt.Println("The file '" + fileName + "' could not be opened.")

        return tree { 
            value: "ERROR",
            args: []tree{
                tree {
                    value:  "The file '" + fileName + "' could not be loaded.",
                },
            },
        }
    } else {
        fileArgs := []tree{ tree { value: "\"" + fileName } }

        for _, x := range(strings.Split(string(file), "\n")) {
            fileArgs = append(fileArgs, tree { value: "\"" + x })
        }

        return tree {
            value:  "file", 
            args:   fileArgs,
        }

    }
}

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

    } else if subTree.value == "set" {  // Sets variables.

        if len(subTree.args) >= 2 {
            variables[subTree.args[0].value] = evaluator(subTree.args[1])
            return variables[subTree.args[0].value]
        }

        return tree { value: "off" }

    } else if subTree.value == "lazySet" {  // Sets a tree to a variable without evaluating it.

        if len(subTree.args) >= 2 {
            variables[subTree.args[0].value] = subTree.args[1]
            return subTree.args[1]
        }

        return tree { value: "off" }

    } else if subTree.value == "run" {  // This is a function similair to an anonymous function.

        return evalAll(subTree.args)

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

    // The following are in charge of simple I/O.
    } else if subTree.value == "o" || subTree.value == "out" {  // This is a formated output, or 'println' minus templating.

        if len(subTree.args) > 0 {
            firstArg := evaluator(subTree.args[0])
            printArg := atomizer(firstArg)

            switch printArg.Type {  // Too bad I can't use printArg['str'] syntax.
            case "str"  : fmt.Println(printArg.str)
            case "num"  : fmt.Println(printArg.num)
            case "fun"  : fmt.Println(printArg.fun)
            case "file" :
                if len(subTree.args) >= 2 {
                    fmt.Println(atomizer(
                        printArg.file[int(atomizer(
                            evaluator(subTree.args[1]),
                        ).num)],
                    ).str)
                } else {
                    fmt.Println("fileName: " + atomizer(printArg.file[0]).str)
                    for i, x := range(printArg.file[1:]) {
                        fmt.Println(strconv.Itoa(i+1) + "│" + atomizer(x).str)
                    }
                }
            case "website":
                if len(subTree.args) >= 2 {
                    switch atomizer(evaluator(subTree.args[1])).str {
                    case "domain"   : fmt.Println(atomizer(printArg.website[0]).str)
                    case "header"   : fmt.Println(atomizer(printArg.website[1]).str)
                    case "content"  : fmt.Println(atomizer(printArg.website[2]).str)
                    }
                } else {
                    fmt.Println("domain:  " + atomizer(printArg.website[0]).str)
                    fmt.Println(" -header-\n" + atomizer(printArg.website[1]).str )
                    fmt.Println(" -content-\n" + atomizer(printArg.website[2]).str )
                }
            }

            return firstArg
        }

        if len(subTree.args) == 0 {
            fmt.Println()
            return tree { value: "off" }
        }

    } else if subTree.value == "print" || subTree.value == "p" {    // Print without a linebreak at the end.

        for _, x := range(subTree.args) {
            printArg := atomizer(evaluator(x))

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

    } else if subTree.value == "in" {   // Standard input.

        reader  := bufio.NewReader(os.Stdin)
        in, _   := reader.ReadString('\n')
        return tree {
            value: "\"" + in[:len(in)-1],
            args: []tree{},
        }

    // File i/o and editing.
    } else if subTree.value == "loadFile" || subTree.value == "open" {  // Open a file.

        fileName := atomizer( evaluator(subTree.args[0]) )
        if fileName.Type != "str" {
            return tree { 
                value: "ERROR",
                args: []tree{
                    tree {
                        value:  "The file loading function takes only strings.",
                    },
                },
            }
        }
        return loadFile(
            atomizer( evaluator(subTree.args[0]) ).str,
        )

    } else if subTree.value == "saveFile" || subTree.value == "save" {  // Save a file.

        fileArg := evaluator(subTree.args[0])
        if len(subTree.args) > 0 && fileArg.value == "file" {
            fileName := atomizer( evaluator(fileArg.args[0]) ).str

            fileContent := []string{}

            for _, x := range(subTree.args[0].args[1:]) {
                fileContent = append(fileContent, atomizer(evaluator(x)).str)
            }

            err := ioutil.WriteFile(fileName, 
                []byte(strings.Join(fileContent, "\n")), 
                0644)

            if err != nil {
                return tree { 
                    value: "ERROR",
                    args: []tree{
                        tree { value:  "The file '" + fileName + "' could not be opened for writing." },
                    },
                }
            }

            return subTree.args[0]

        }
        return tree { 
            value: "ERROR",
            args: []tree{
                tree {
                    value:  "The file saving function requires a file argument.",
                },
            },
        }

    } else if subTree.value == "get" {

        domain := atomizer(evaluator(subTree.args[0])).str
        response, err := http.Get(domain)

        if err != nil {
            return tree { 
                value: "ERROR",
                args: []tree{
                    tree { value:  "The webpage '" + domain + "' could not be opened." },
                },
            }
        }

        pageContent, _ := ioutil.ReadAll(response.Body)

        websiteArgs := []tree{ 
            tree { value: "\"" + domain },
            tree { value: "\"" + string(pageContent) },
            tree { value: "\"" + string(pageContent) },
        }

    	response.Body.Close()

        return tree {
            value: "website",
            args: websiteArgs,
        }

    // The following are boolean operators.
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

    // Simple comparison operators.
    } else if subTree.value == "equals" || subTree.value == "is" {  // Check for equality.

        if len(subTree.args) > 0 {
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
        }
        return tree { value: "off" }

    } else if subTree.value == "isMax" || subTree.value == ">" {

        if len(subTree.args) > 0 {

            firstAtom := atomizer(evaluator(subTree.args[0]))
            var secondAtom atom

            for _, x := range(subTree.args[1:]) {
                secondAtom = atomizer(evaluator(x))
                switch firstAtom.Type {
                case "num":
                    if firstAtom.num < secondAtom.num {
                        return tree { value: "off" }
                    }
                case "str":
                    if len(firstAtom.str) < len(secondAtom.str) {
                        return tree { value: "off" }
                    }
                case "list":
                    if len(firstAtom.list) < len(secondAtom.list) {
                        return tree { value: "off" }
                    }
                }
            }
            return tree { value: "on" }
        }
        return tree { value: "off" }

    } else if subTree.value == "isMin" || subTree.value == "<" {

        if len(subTree.args) > 0 {

            firstAtom := atomizer(evaluator(subTree.args[0]))
            var secondAtom atom

            for _, x := range(subTree.args[1:]) {
                secondAtom = atomizer(evaluator(x))
                switch firstAtom.Type {
                case "num":
                    if firstAtom.num > secondAtom.num {
                        return tree { value: "off" }
                    }
                case "str":
                    if len(firstAtom.str) > len(secondAtom.str) {
                        return tree { value: "off" }
                    }
                case "list":
                    if len(firstAtom.list) > len(secondAtom.list) {
                        return tree { value: "off" }
                    }
                }
            }
            return tree { value: "on" }
        }
        return tree { value: "off" }

    // Simple raw data type check using the atomizer function.
    } else if atomizer(subTree).Type != "CAN NOT PARSE" {    // Raw Data Types, such as 'str', 'num', etc. 
                                                            // It's placed here so that it'll be reached quickly
        return subTree

    // Loops.
    } else if subTree.value == "each" || subTree.value == "e" {     // For-each loop.

        if len(subTree.args) >= 3 {
            for _, x := range(evaluator(subTree.args[1]).args) {
                variables[subTree.args[0].value] = x
                evalAll(subTree.args[2:])
            }
            return tree { value: "on" }
        }
        return tree { value: "off" }

    } else if subTree.value == "while" || subTree.value == "w" {    // While-true loop.

        for evaluator(subTree.args[0]).value == "on" {
            evalAll(subTree.args[1:])
        }
        return tree { value: "off" }

    // List related functions and list generators.
    } else if subTree.value == "range" {    // Range list generator.
        
        var generatedList = tree { 
            value: "list",
            args: []tree{},
        }
        
        var start   = 0
        var end     = 100
        var iterate = 1

        switch len(subTree.args) {
        case 1:
            end     = int(atomizer(evaluator(subTree.args[0])).num)
        case 2:
            start   = int(atomizer(evaluator(subTree.args[0])).num)
            end     = int(atomizer(evaluator(subTree.args[1])).num)
        case 3:
            start   = int(atomizer(evaluator(subTree.args[0])).num)
            end     = int(atomizer(evaluator(subTree.args[1])).num)
            iterate = int(atomizer(evaluator(subTree.args[2])).num)
        }

        for x := start; x <= end; x += iterate {
            generatedList.args = append(generatedList.args, 
                tree { 
                    value: strconv.Itoa(x), 
                    args: []tree{},
                })
        }

        return generatedList

    // Mathmatical operators, such as adding numbers, checking for divisibility, etc.
    } else if subTree.value == "sum" {  // Sum all numerical args together.

        number := 0.0
        for _, x := range(subTree.args) {
            number += atomizer(evaluator(x)).num
        }
        return tree { value: strconv.FormatFloat(number, 'E', -1, 64) }

    } else if subTree.value == "subtract" {  // Starting with the leftmost number, subtract all numbers after it.

        if len(subTree.args) >= 2 {
            number := atomizer(evaluator(subTree.args[0])).num
            for _, x := range(subTree.args[1:]) {
                number -= atomizer(evaluator(x)).num
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
            number *= atomizer(evaluator(x)).num
        }
        return tree { value: strconv.FormatFloat(number, 'E', -1, 64) }

    } else if subTree.value == "divide" {  // Starting with the leftmost number, divide it by all following numbers.

        if len(subTree.args) >= 2 {
            number := atomizer(evaluator(subTree.args[0])).num
            for _, x := range(subTree.args[1:]) {
                number /= atomizer(evaluator(x)).num
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

            arg1 := atomizer(evaluator(subTree.args[0]))
            arg2 := atomizer(evaluator(subTree.args[1]))
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

            arg1 := atomizer(evaluator(subTree.args[0]))
            arg2 := atomizer(evaluator(subTree.args[1]))

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

    // String manipulation functions.
    } else if subTree.value == "concat" {   // Concatonate strings.

        newString := "\""

        for _, x := range(subTree.args) {
            subString := atomizer(evaluator(x))
            newString += subString.str

            if subString.Type != "str" {
                fmt.Println(" -error- ")
                fmt.Println("You used " + x.value + ", a '" + subString.Type + "' as a str.")
            }
        }

        return tree { value: newString, args: []tree{} }

    } else if subTree.value == "die" {

        os.Exit(0)

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
    for {

        fmt.Println(" -input- ")
        input, _ = reader.ReadString('\n')

        fmt.Println(" -output- ")
        execute( input )

    }
}

func runFile(fileName string) {
    file, err := ioutil.ReadFile( fileName )
    if err != nil {
        fmt.Println("The file '" + fileName + "' could not be opened.")
    } else {
        execute( string( file ) )
    }
}

func main() {
    flag.Parse()
    if len(flag.Args()) >= 2 {

        switch flag.Arg(0) {
            case "runFile": runFile(flag.Arg(1))
            case "run":     execute(flag.Arg(1))
            case "load":
                variables["load"] = loadFile(flag.Arg(0))
                prompt()

            default: fmt.Println(
                                    "The argument '" + 
                                    flag.Arg(0) + 
                                    "' is not recognized.")
        }

    } else if len( flag.Args() ) >= 1 {

        fileName    := strings.Split(flag.Arg(0), ".")
        extension   := fileName[len(fileName)-1]

        if extension == "die" || extension[:2] == "dy" {    // All files ending in '*.die' or '*.dy*' get executed.
            runFile(flag.Arg(0))
        } else { // Load a text file as a variable.
            variables["load"] = loadFile(flag.Arg(0))
            prompt()
        }

    } else {
        prompt()
    }
}