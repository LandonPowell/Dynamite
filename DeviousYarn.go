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
// It's basically just a wrapper for a big ass regex that'd be unreadable otherwise. 
func lexer(plaintext string) []string {     // Returns a list of tokens.
    strings     := "'(\\\\\\\\|\\\\'|[^'])+'|\"[^\n]+"  // Regex for strings. http://www.xkcd.com/1638/
    brackets    := "[\\[\\](){}:=]"                     // Regex for bracket chars.
    names       := "[^\\s\\[\\](){}:='\"]+"             // Regex for var names.

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
            postAtom.str = preAtom.value[1:len(preAtom.value)-1]

            replaceMap := map[string]string {
                "'": "'",
                "n": "\n",
                "t": "\t",
                "\\": "\\",
            }

            for x, y := range(replaceMap) {
                postAtom.str = strings.Replace(postAtom.str, "\\" + x, y, -1)
            }
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

func typeConverter(oldTree tree, newType string) tree {
    oldAtom := atomizer(oldTree)
    oldType := oldAtom.Type

    if oldType == newType {
        return oldTree
    }

    if newType == "fun" {
        return tree {
            value: "fun",
            args: []tree {
                tree {
                    value: "return",
                    args: []tree { oldTree },
                },
            },
        }
    }

    if newType == "str" {
        return tree {
            value: "\"" + oldTree.value,
            args: []tree{},
        }
    }

    if oldType == "num" && newType == "bit" {
        if oldAtom.num > 0 {
            return tree { value: "on" }
        }
        return tree { value: "off" }
    }

    if oldType == "bit" && newType == "num" {
        if oldAtom.bit {
            return tree { value: "1" }
        }
        return tree { value: "0" }
    }

    if oldType == "str" {
        switch newType {
        case "num":
            if _, err := strconv.ParseFloat(oldAtom.str, 64); err == nil {
                return tree { value: oldAtom.str }
            }
        case "bit":
            if len(oldAtom.str) <= 0 {
                return tree { value: "off" }
            }
            return tree { value: "on" }
        case "list":
            newArgs := []tree{}
            for _, x := range(oldAtom.str) {
                newArgs = append(newArgs, tree {
                    value: "\"" + string(x),
                    args: []tree{},
                })
            }
            return tree {
                value: "list",
                args: newArgs,
            }
        }
    }

    switch oldType { case "list", "file", "website":
        switch newType { case "list", "file", "website":
            return tree {
                value: newType,
                args: oldTree.args,
            }
        }
    }
    
    return tree { value: "off" }
}

var variables = make( map[string]tree )

func loadFile(fileName string) tree {
    file, err := ioutil.ReadFile( fileName )
    if err != nil {
        fmt.Println("The file '" + fileName + "' could not be opened.")

        return tree { 
            value: " -error- ",
            args: []tree{ tree {
                value:  "The file '" + fileName + "' could not be loaded.",
            } },
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

        if currentRun.value == " -error- " {
            fmt.Println(" -error- ")
            fmt.Println(currentRun.args[0].value)
        } else if currentRun.value == "return" {
            return evaluator(currentRun.args[0])
        }
    }
    return tree { value: "False" }
}

var lastCondition bool = true; // This checks the last conditional for the elf and alf functions.
func evaluator(subTree tree) tree {
    if val, ok := variables[subTree.value]; ok {    // This returns variable values.
        return evaluator(val)
    } 

    if atomizer(subTree).Type != "CAN NOT PARSE" {  // Raw Data Types, such as 'str', 'num', etc. 
        return subTree
    }

    switch subTree.value {
    case "set": // Sets variables.
        if len(subTree.args) == 2 {
            variables[subTree.args[0].value] = evaluator(subTree.args[1])
            return variables[subTree.args[0].value]
        }

        return tree { value: "off" }

    case "lazySet": // Sets a tree to a variable without evaluating it.
        if len(subTree.args) == 2 {
            variables[subTree.args[0].value] = subTree.args[1]
            return subTree.args[1]
        }

        return tree { value: "off" }

    case "run": // This is a function similair to an anonymous function.
        return evalAll(subTree.args)

    // The following few values are in charge of conditionals.
    case "?", "if": // Simple conditional. "If"
        if len(subTree.args) >= 2 && 
            typeConverter(evaluator(subTree.args[0]), "bit").value == "on" {

            lastCondition = true;
            return evalAll(subTree.args[1:])

        } else {
            lastCondition = false;
        }

        return tree { value: "off" }

    case "-?", "elf":   // Otherwise if conditional. "Else if"
        if !lastCondition && len(subTree.args) >= 2 && 
            typeConverter(evaluator(subTree.args[0]), "bit").value == "on" {

            lastCondition = true;
            return evalAll(subTree.args[1:])

        }

        return tree { value: "off" }

    case "&?", "alf":   // Also if conditional.
        if lastCondition && len(subTree.args) >= 2 && 
            typeConverter(evaluator(subTree.args[0]), "bit").value == "on" {

            lastCondition = true;
            return evalAll(subTree.args[1:])

        }

        return tree { value: "off" }
    
    case "--", "else":  // Otherwise conditional. "Else"
        if !lastCondition && len(subTree.args) >= 1 {
            return evalAll(subTree.args)
        }

        return tree { value: "off" }

    case "&&", "also":  // Also conditional.
        if lastCondition && len(subTree.args) >= 1 {
            return evalAll(subTree.args)
        }

        return tree { value: "off" }

    // The following are in charge of simple I/O.
    case "o", "out":    // This is a formated output, or 'println' minus templating.
        if len(subTree.args) > 0 {
            firstArg := evaluator(subTree.args[0])
            printArg := atomizer(firstArg)

            if printArg.Type == "file" {  // Too bad I can't use printArg['str'] syntax.
                if len(subTree.args) == 2 {
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
            } else if printArg.Type == "website" {
                if len(subTree.args) == 2 {
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
            } else {
                firstArg = typeConverter(firstArg, "str")
                fmt.Println(atomizer(firstArg).str)
            }

            return firstArg
        }

        fmt.Println()
        return tree { value: "off" }

    case "print", "p":  // Print without a linebreak at the end.
        for _, x := range(subTree.args) {
            x = typeConverter(evaluator(x), "str")
            fmt.Print(atomizer(x).str)
        }

        return tree { value: "on" }

    case "rawOut":  // This outputs the plaintext of a tree.
        if len(subTree.args) > 0 {
            fmt.Println(evaluator(subTree.args[0]))
            return subTree.args[0]
        }

        return tree { value: "off" }

    case "in":  // Standard input.
        reader  := bufio.NewReader(os.Stdin)
        in, _   := reader.ReadString('\n')
        return tree {
            value: "\"" + in[:len(in)-1],
            args: []tree{},
        }

    // File i/o and editing.
    case "loadFile", "open":    // Open a file.
        fileName := atomizer( evaluator(subTree.args[0]) )
        if fileName.Type != "str" {
            return tree { 
                value: " -error- ",
                args: []tree{ tree {
                    value:  "The file loading function takes only strings.",
                } },
            }
        }
        return loadFile(
            atomizer( evaluator(subTree.args[0]) ).str,
        )

    case "saveFile", "save":    // Save a file.
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
                    value: " -error- ",
                    args: []tree{ tree { 
                        value:  "The file '" + fileName + "' could not be opened for writing.",
                    } },
                }
            }

            return subTree.args[0]

        }
        return tree { 
            value: " -error- ",
            args: []tree{ tree {
                value:  "The file saving function requires a file argument.",
            } },
        }

    case "get": // HTTP get request.
        domain := atomizer(evaluator(subTree.args[0])).str
        response, err := http.Get(domain)

        if err != nil {
            return tree { 
                value: " -error- ",
                args: []tree{ tree { 
                    value:  "The webpage '" + domain + "' could not be opened.",
                } },
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
    case "not", "!":    // Boolean 'not'.
        if len(subTree.args) == 1 && evaluator(subTree.args[0]).value == "off" {
            return tree { value: "on" }
        }
        return tree { value: "off" }

    case "or":  // Boolean 'or'.
        for _, x := range(subTree.args) {
            if evaluator(x).value == "on" {
                return tree { value: "on" }
            }
        }
        return tree { value: "off" }

    case "and": // Boolean 'and'.
        for _, x := range(subTree.args) {
            if evaluator(x).value == "off" {
                return tree { value: "off" }
            }
        }
        return tree { value: "on" }

    // Simple comparison operators.
    case "equals", "is":    // Check for equality.
        if len(subTree.args) > 0 {
            firstTree := evaluator(subTree.args[0])
            for _, x := range(subTree.args[1:]) {
                x = evaluator(x)
    
                if len(x.args) != len(firstTree.args) ||
                    atomizer(firstTree).str != atomizer(x).str ||
                    atomizer(firstTree).num != atomizer(x).num {
    
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

    case "isMax", ">":  // Greater than / Check if the largest element in the list.
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

    case "isMin", "<":  // Less than / Check if smallest element in list.
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

    // Loops.
    case "each", "e":   // For-each loop.
        if len(subTree.args) >= 3 {
            for _, x := range(evaluator(subTree.args[1]).args) {
                variables[subTree.args[0].value] = x
                evalAll(subTree.args[2:])
            }
            return tree { value: "on" }
        }
        return tree { value: "off" }

    case "while", "w":  // While-true loop.
        for evaluator(subTree.args[0]).value == "on" {
            evalAll(subTree.args[1:])
        }
        return tree { value: "off" }

    // List related functions and list generators.
    case "range":   // Range list generator.
        
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

    case "append", "a": // Append to a list.
        if len(subTree.args) > 0 {
            list := evaluator(subTree.args[0]).args
            for _, x := range(subTree.args[1:]) {
                list = append(list, evaluator(x))
            }
            return tree {
                value: "list", 
                args: list,
            }
        }

        return tree { 
            value: " -error- ",
            args: []tree{ tree {
                value:  "The 'append' function requires at least one argument.",
            } },
        }

    case "index", "i":  // Element at an index.
        if len(subTree.args) == 2 {
            list    := evaluator(subTree.args[0])
            index   := int(atomizer(evaluator(subTree.args[1])).num)
            if index <= len(list.args) && index >= 0 {
                return list.args[index]
            }
            if len(list.args) + index >= 0 {
                return list.args[len(list.args)+index]
            }
            return tree {
                value: " -error- ",
                args: []tree{ tree {
                    value: "Index out of range.",
                } },
            }
        }

        return tree {
            value: " -error- ",
            args: []tree{ tree { 
                value: "The 'index' function requires two arguments.",
            } },
        }

    case "length":  // Length of a list.
        if len(subTree.args) == 1 {
            return tree {
                value: strconv.Itoa(len(evaluator(subTree.args[0]).args)),
            }
        }

        return tree {
            value: " -error- ",
            args: []tree{ tree { 
                value: "The 'length' function requires one argument.", 
            } },
        }

    // Mathmatical operators, such as adding numbers, checking for divisibility, etc.
    case "sum": // Sum all numerical args together.
        number := 0.0
        for _, x := range(subTree.args) {
            number += atomizer(evaluator(x)).num
        }
        return tree { value: strconv.FormatFloat(number, 'f', -1, 64) }

    case "subtract":    // Starting with the leftmost number, subtract all numbers after it.
        if len(subTree.args) >= 2 {
            number := atomizer(evaluator(subTree.args[0])).num
            for _, x := range(subTree.args[1:]) {
                number -= atomizer(evaluator(x)).num
            }
            return tree { value: strconv.FormatFloat(number, 'f', -1, 64) }
        }
        return tree { 
            value: " -error- ",
            args: []tree{ tree {
                value:  "The 'subtract' function takes two or more num args.",
            } },
        }

    case "multiply":    // Multiply all numerical args together.
        number := 1.0
        for _, x := range(subTree.args) {
            number *= atomizer(evaluator(x)).num
        }
        return tree { value: strconv.FormatFloat(number, 'f', -1, 64) }

    case "divide":  // Starting with the leftmost number, divide it by all following numbers.
        if len(subTree.args) >= 2 {
            number := atomizer(evaluator(subTree.args[0])).num
            for _, x := range(subTree.args[1:]) {
                number /= atomizer(evaluator(x)).num
            }
            return tree { value: strconv.FormatFloat(number, 'f', -1, 64) }
        }
        return tree { 
            value: " -error- ",
            args: []tree{ tree {
                value:  "The 'divide' function takes two or more num args.",
            } },
        }
    
    case "mod", "modulo": // Division with a reminader, modulo.
        if len(subTree.args) == 2 {

            arg1 := atomizer(evaluator(subTree.args[0]))
            arg2 := atomizer(evaluator(subTree.args[1]))
            return tree { value: strconv.Itoa( int(arg1.num) % int(arg2.num) ) }
        }

        return tree {   // Returns an error message for undefined names.
            value: " -error- ",
            args: []tree{ tree { 
                value: "The 'mod' function takes exactly two arguments.",
            } },
        }

    case "divisible":   // Check for divisibility. 
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
                value: " -error- ",
                args: []tree{ tree {
                    value:  "The 'divisible' function only takes 'num' types.\n" +
                            "You've given '" + arg1.Type + "' and '" + arg2.Type + "'.",
                } },
            }
        }

        return tree {   // Returns an error message for undefined names.
            value: " -error- ",
            args: []tree{ tree { 
                value: "The 'divisible' function takes exactly two arguments.",
            } },
        }

    // String manipulation functions.
    case "concat":  // Concatonate strings.
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

    // Type conversions.
    case "bit", "num", "str":
        return typeConverter(
            evaluator(subTree.args[0]), 
            subTree.value,
        )

    case "typeConvert": // Convert any type to any other type using a string.
        if len(subTree.args) == 2 {
            return typeConverter(
                evaluator(subTree.args[0]),
                atomizer(evaluator(subTree.args[1])).str,
            )
        }

        return tree {   // Returns an error message for undefined names.
            value: " -error- ",
            args: []tree{ tree { 
                value: "The 'typeConvert' function takes exactly two arguments.",
            } },
        }

    // Kill DeviousYarn.
    case "die":
        os.Exit(0)
    }

    return tree {   // Returns an error message for undefined names.
        value: " -error- ",
        args: []tree{ tree {
            value: "The word '" + subTree.value +  "' means nothing.",
        } },
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