package main

import (
    "os"
    "fmt"
    "flag"
    "bufio"
    "regexp"
    "strconv"
    "os/exec"
    "strings"
    "net/http"
    "io/ioutil"
)

func contains(x string, z string) bool {
    // Checks if char is in string.
    for _, y := range z { if x == string(y) { return true } }
    return false 
}

func raiseError(error string) tree {
    fmt.Println(" -error- ")
    fmt.Println(error)
    return tree { value: "error" }
}

var tokenList = []string{}

// This is mostly self-explaining, but it's the tokenizer (obviously).
// It's basically just a wrapper for a big ass regex that'd be unreadable otherwise. 
func lexer(plaintext string) []string {
    // Returns a list of tokens.
    strings     := "'(\\\\\\\\|\\\\'|[^'])*'|\"[^\n]*"  // Regex for strings. http://www.xkcd.com/1638/
    comments    := "(#|;;)[^\n]*"                       // Regex for comments.
    key         := "[\\[\\](){}:=,]"                    // Regex for key chars.
    names       := "[^\\s\\[\\](){}:;#=',\"]+"          // Regex for var names.

    tokenRegex  := regexp.MustCompile(
        strings+"|"+comments+"|"+key+"|"+names,
    )

    tokens := tokenRegex.FindAllString(plaintext, -1)

    for i, x := range tokens {
        if len(x) >= 2 && ( x[:2] == ";;" || x[:1] == "#" ) {
            tokens = append(tokens[:i], tokens[i+1:]...)
        }
    }

    return tokens
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
func parseNext() tree { 
    // This is the actual meat of 'parser'.

    var currentTree = tree {    // Define the current token as a tree.
        value: tokenList[0],
        args: []tree{},
    }
    tokenList = tokenList[1:]   // Removes the first element in the slice.

    if currentTree.value == "," {
        currentTree = infixParser()
    }

    if len(tokenList) > 0 { // Everybody taking the chance... Safety dance.

        if contains(tokenList[0], "{[(") { 
            // If the next token is an opening bracket.
            tokenList = tokenList[1:]   // Remove it.
            currentTree.args = parser() // Make a nest of it.

            if len(tokenList) >= 2 && tokenList[0] == "=" { // If a "=" follows a closing symbol.
                tokenList = tokenList[1:]   // Remove it.
                currentTree = tree {    // Turn the current tree into a function definition. 
                    value: "defun",
                    args: []tree{ currentTree },
                }

                if contains(tokenList[0], "{[(") { // If it's a multi-line function definition.
                    tokenList = tokenList[1:]   // Remove it.
                    currentTree.args = append(
                        currentTree.args,
                        parser() ...,
                    )

                } else { // If it's a single-line function definition.
                    currentTree.args = append(currentTree.args, parseNext())
                }
            }

        } else if tokenList[0] == ":" {
            // If the next token is a monogomy symbol.
            tokenList = tokenList[1:]   // Remove it.
            currentTree.args = append(currentTree.args, parseNext())    // Nest it.

        } else if tokenList[0] == "=" {
            // If the next token is a decleration.
            tokenList = tokenList[1:]   // Remove it.
            currentTree = tree {    // Make the tree into a 'set' function.
                value: "set",
                args: []tree{ currentTree, parseNext() },  // Set currentTree as the first arg.
            }

        }
    }

    return currentTree
}

func infixParser() tree{
    // This is the parser for when infix gets switched on. Arg Func Arg Func Arg, etc.
    var currentTree     = parseNext()
    var infixFunction   = tree{}

    for len(tokenList) > 0 { // So long as we still have tokens, and we haven't turned off the infix parser.
        if tokenList[0] == "," {
            tokenList = tokenList[1:]
            return currentTree
        }

        infixFunction = parseNext()

        if len(tokenList) == 0 || tokenList[0] == "," {
            if len(tokenList) > 0 {
                tokenList = tokenList[1:]
            }
            raiseError("Insufficient calls in the infix function syntax.")
            return currentTree
        }

        if currentTree.value != infixFunction.value {
            infixFunction.args = append(infixFunction.args,
                currentTree,
                parseNext(),
            )
            currentTree = infixFunction
        } else {
            currentTree.args = append(currentTree.args, parseNext())
        }
    }
    return currentTree
}

func parser() []tree{
    // The token list is looped through and trees are created.
    var treeList = []tree{} // Define the empty tree list.

    for len(tokenList) > 0 && !contains(tokenList[0], ")]}") { 
        // So long as the current token isn't a closing character.
        treeList = append(treeList, parseNext())    // Append the next parsed tree to the tree list.
    }
    if len(tokenList) > 0 && contains(tokenList[0], ")]}") {   
        // If the next token is a closing character,
        tokenList = tokenList[1:]   // Remove it.
    }

    return treeList // Return the tree list.
}

type atom struct {
    Type    string  // The type of variable the atom is.

    str     string  // If the type is 'str' (a string) this is the value.
    num     float64 // 'num' (a number)
    bit     bool    // 'bit' (a 1 or 0, True or False)
    list    []tree  // 'list'
    file    []tree  // 'file'
    website []tree  // 'website'
    tree    tree    // 'tree'
}

func atomizer(preAtom tree) atom {
    var postAtom atom

    firstChar := string(preAtom.value[0]) 
    if firstChar == "\"" || firstChar == "'" {
        // If the value is a string (or str).

        postAtom.Type   = "str" // Firstly, declare the Type as 'str'

        if firstChar == "\"" {
            // If the first char is the string-line indicator (doublequote).
            postAtom.str    = preAtom.value[1:] 
        } else {
            // If the first char is a single quote.
            postAtom.str = preAtom.value[1:len(preAtom.value)-1] // Clip off the ends.

            replaceMap := map[string]string {
                "'": "'",
                "n": "\n",
                "t": "\t",
                "\\": "\\",
            }

            for x, y := range replaceMap {
                postAtom.str = strings.Replace(postAtom.str, "\\" + x, y, -1)
            }
        }

    } else if _, err := strconv.ParseFloat(preAtom.value, 64); err == nil {
        // If the tree is a number.
        postAtom.Type   = "num"
        postAtom.num, _ = strconv.ParseFloat(preAtom.value, 64)

    } else if preAtom.value == "on" || preAtom.value == "off" { 
        // If the tree is a bit/bool.
        postAtom.Type   = "bit"
        if preAtom.value == "on" {
            postAtom.bit = true
        } else {
            postAtom.bit = false 
        }

    } else if preAtom.value == "list" {
        // If the tree is a list.
        postAtom.Type = "list"
        postAtom.list = preAtom.args

    } else if preAtom.value == "file" {
        // If the tree is a file.
        postAtom.Type = "file"
        postAtom.file = preAtom.args

    } else if preAtom.value == "website" {
        // If the tree is a website.
        postAtom.Type       = "website"
        postAtom.website    = preAtom.args

    } else if preAtom.value == "lazyTree" {
        // If the tree is a lazy tree. The 'tree' function just generates this.
        postAtom.Type = "tree"
        postAtom.tree = preAtom

    } else { 
        postAtom.Type = "CAN NOT PARSE" 
    }

    return postAtom
}

func typeConverter(oldTree tree, newType string) tree {
    // Converts a tree into a tree of a different type.
    oldAtom := atomizer(oldTree)
    oldType := oldAtom.Type

    if oldType == newType {
        // If the tree is already the right type.
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
            for _, x := range oldAtom.str {
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

func loadFile(fileName string) tree {
    file, err := ioutil.ReadFile( fileName )

    if err != nil {
        return raiseError("The file '" + fileName + "' could not be loaded.")
    }

    fileArgs := []tree{ tree { value: "\"" + fileName } }

    for _, x := range strings.Split(string(file), "\n") {
        fileArgs = append(fileArgs, tree { value: "\"" + x })
    }

    return tree {
        value:  "file", 
        args:   fileArgs,
    }
}

func evalTrees(preTree []tree) []tree { // This is used for the 'tree' datatype, not for AST evaluations. See 'evaluator' function.
    // Effectively, this function takes a list of 'tree' datatypes, 
    // and compiles them recursively into a 'lazyTree' type.
    compiledTree := []tree{}

    for _, x := range preTree {

        if atomizer(x).Type == "CAN NOT PARSE" {
            compiledTree = append(compiledTree, evaluator(x))
        } else {
            compiledTree = append(compiledTree, tree {
                value: x.value,
                args: evalTrees(x.args),
            })
        }
    }

    return compiledTree
}

func evalAll(treeList []tree) tree {
    for _, x := range treeList {
        if x.value == "return" {
            if len(x.args) > 0 {
                return evaluator(x.args[0])
            } else { 
                return raiseError(
                    "You've attempted to call 'return' with no argument.\n" +
                    "It isn't 'return arg', it's 'return:arg'.",
                )
            }
        }
        evaluator(x)
    }
    return tree { value: "off" }
}

type function struct {
    args    []tree
    process []tree
}

var variables = make( map[string]tree )
var functions = make( map[string]function )

var lastCondition bool = true; // This checks the last conditional for the elf and alf functions.
func evaluator(subTree tree) tree {

    if variable, ok := variables[subTree.value]; ok { // Variable definition check.
        // This returns variable values.
        return evaluator(variable)
    } 

    if funk, ok := functions[subTree.value]; ok { // Function definition check.
        // This evaluates function values.
        oldVars := make( map[string]tree )

        oldVars["args"] = variables["args"]
        variables["args"] = tree {
            value:  "list",
            args:   subTree.args,
        }

        for i, x := range subTree.args {
            if i < len(funk.args) {
                thisVar := funk.args[i].value
                oldVars[thisVar]    = variables[thisVar]
                variables[thisVar]  = evaluator(x)
            }
        }

        returnMe := evalAll(funk.process)
    
        for key, value := range oldVars {
            variables[key] = value
        }
    
        return returnMe
    } 

    if atomizer(subTree).Type != "CAN NOT PARSE" {  
        // Raw Data Types, such as 'str', 'num', etc. 
        return subTree
    }

    switch subTree.value {
    case "set": // Sets variables.
        if len(subTree.args) != 2 {
            return raiseError("You must set variables using a single value and a name.")
        }

        variables[subTree.args[0].value] = evaluator(subTree.args[1])
        return variables[subTree.args[0].value]

    case "lazySet": // Sets a tree to a variable without evaluating it.
        if len(subTree.args) != 2 {
            return raiseError("You must set variables using a single value and a name.")
        }

        variables[subTree.args[0].value] = subTree.args[1]
        return subTree.args[1]

    case "defun":
        if len(subTree.args) < 2 {
            return raiseError("You can't set a function without both a name and a value.")
        }

        functions[subTree.args[0].value] = function {
            args:       subTree.args[0].args,
            process:    subTree.args[1:],
        }

        return tree { value: "on" }

    case "run": // This is a function similair to an anonymous function.
        return evalAll(subTree.args)

    // The following few values are in charge of conditionals.
    case "?", "if": // Simple conditional. "If"
        if len(subTree.args) < 2 {
            return raiseError("The 'if' conditional function requires at least two arguments.")
        }

        if typeConverter(evaluator(subTree.args[0]), "bit").value == "on" {
            lastCondition = true
            return evalAll(subTree.args[1:])
        }

        lastCondition = false;
        return tree { value: "off" }

    case "-?", "elf":   // Otherwise if conditional. "Else if"
        if len(subTree.args) < 2 {
            return raiseError("The 'elf' conditional function requires at least two arguments.")
        }

        if !lastCondition && 
            typeConverter(evaluator(subTree.args[0]), "bit").value == "on" {

            lastCondition = true
            return evalAll(subTree.args[1:])

        }

        lastCondition = false
        return tree { value: "off" }

    case "&?", "alf":   // Also if conditional.
        if len(subTree.args) < 2 {
            return raiseError("The 'alf' conditional function requires at least two arguments.")
        }

        if lastCondition && 
            typeConverter(evaluator(subTree.args[0]), "bit").value == "on" {

            lastCondition = true
            return evalAll(subTree.args[1:])

        }

        lastCondition = false
        return tree { value: "off" }
    
    case "--", "else":  // Otherwise conditional. "Else"
        if len(subTree.args) < 1 {
            return raiseError("The 'else' conditional function requires at least one argument.")
        }

        if !lastCondition {
            return evalAll(subTree.args)
        }

        return tree { value: "off" }

    case "&&", "also":  // Also conditional.
        if len(subTree.args) < 1 {
            return raiseError("The 'also' conditional function requires at least one argument.")
        }

        if lastCondition {
            return evalAll(subTree.args)
        }

        return tree { value: "off" }

    // The following are in charge of simple I/O.
    case "o", "out":    // This is a formated output, or 'println' minus templating.
        if len(subTree.args) < 1 {
            fmt.Println()
            return tree { value: "off" }
        }

        firstArg := evaluator(subTree.args[0])
        printArg := atomizer(firstArg)

        if printArg.Type == "file" {
            if len(subTree.args) == 2 {
                fmt.Println(atomizer(
                    printArg.file[int(atomizer(
                        evaluator(subTree.args[1]),
                    ).num)],
                ).str)
            } else {
                fmt.Println("fileName: " + atomizer(printArg.file[0]).str)
                for i, x := range printArg.file[1:] {
                    fmt.Println(strconv.Itoa(i+1) + "│" + atomizer(x).str)
                }
            }
        } else if printArg.Type == "website" {
            if len(subTree.args) == 2 {
                switch atomizer(evaluator(subTree.args[1])).str {
                case "domain":
                    fmt.Println(atomizer( printArg.website[0] ).str)
                case "header":
                    fmt.Println(atomizer( printArg.website[1] ).str)
                case "content":
                    fmt.Println(atomizer( printArg.website[2] ).str)
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

    case "print", "p":  // Print without a linebreak at the end.
        for _, x := range subTree.args {
            x = typeConverter(evaluator(x), "str")
            fmt.Print(atomizer(x).str)
        }

        return tree { value: "on" }

    case "rawOut":  // This outputs the plaintext of a tree.
        if len(subTree.args) != 1 {
            return raiseError("The 'rawOut' function requires exactly one argument.")
        }

        fmt.Println(evaluator(subTree.args[0]))
        return subTree.args[0]

    case "term", "cmd": // Disgusting useless command, yuck! (executes shell commands) (don't use it, you idiot)
        var command = []string{}
        for _, x := range subTree.args {
            command = append(command, strings.Fields( atomizer(evaluator(x)).str )... )
        }
        output, err := exec.Command(command[0], command[1:]...).Output()
        if err != nil {
            return tree { value: "off" }
        }
        return tree {
            value: "\"" + string(output),
            args: []tree{},
        }

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
            return raiseError("The file loading function takes only strings.")
        }

        return loadFile(
            atomizer( evaluator(subTree.args[0]) ).str,
        )

    case "saveFile", "save":    // Save a file. Heavily commented because it's kind of confusing.
        if len(subTree.args) < 1 { // Safety check.
            return raiseError("The file saving function requires at least one argument.")
        }

        fileArg := evaluator(subTree.args[0]) // Evaluate the file. 

        if fileArg.value != "file" { // If there isn't an argument of 'file' type,
            return raiseError("The file saving function requires a 'file' type argument.")  // raise an error.
        }

        fileName := atomizer( evaluator(fileArg.args[0]) ).str // Evaluate the first argument of the 'file' arg, aka the name of the file.

        fileContent := []string{} // Define an empty array of strings. 

        for _, x := range subTree.args[0].args[1:] { // Loop through all the lines in the file,
            fileContent = append(fileContent, atomizer(evaluator(x)).str) // and append them to the file content after evaluating them to strings.
        }

        err := ioutil.WriteFile(fileName,
            []byte(strings.Join(fileContent, "\n")),  // Join all the lines in the file.
            0644) // Arbitrary obscure number that no one understands.

        if err != nil { // If there was an error, aka if the error isn't not an error. Fucking '!= nil'? What in gods name is wrong with langdevs.
            return raiseError("The file '" + fileName + "' could not be opened for writing.")
        }

        return subTree.args[0]  // Return the file argument.

    // Network i/o. Doesn't include a web framework.
    case "get": // HTTP get request.
        domain := atomizer(evaluator(subTree.args[0])).str
        response, err := http.Get(domain)

        if err != nil {
            return raiseError("The webpage '" + domain + "' could not be opened.")
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
        for _, x := range subTree.args {
            if evaluator(x).value == "on" {
                return tree { value: "on" }
            }
        }
        return tree { value: "off" }

    case "and": // Boolean 'and'.
        for _, x := range subTree.args {
            if evaluator(x).value == "off" {
                return tree { value: "off" }
            }
        }
        return tree { value: "on" }

    // Simple comparison operators and comparison operations.
    case "equals", "is":    // Check for equality.
        if len(subTree.args) > 0 {
            firstTree := evaluator(subTree.args[0])
            for _, x := range subTree.args[1:] {
                x = evaluator(x)
    
                if len(x.args) != len(firstTree.args) ||
                    atomizer(firstTree).str != atomizer(x).str ||
                    atomizer(firstTree).num != atomizer(x).num {
    
                    return tree { value: "off" }
    
                }
    
                for i, y := range x.args {
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

            for _, x := range subTree.args[1:] {
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

            for _, x := range subTree.args[1:] {
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

    case "any": // Return the first thing that isn't blank or 0.
        for _, x := range subTree.args {
            x = evaluator(x)
            currentItem := atomizer(x)
            if currentItem.Type == "str" {
                if strings.TrimSpace(currentItem.str) != "" {
                    return x
                }
            } else if currentItem.Type == "num" {
                if currentItem.num != 0 {
                    return x
                }
            } else {
                raiseError("Can't use a " + currentItem.Type + " with the 'any' function.")
            }
        }

    // Loops.
    case "each", "e":   // For-each loop.
        if len(subTree.args) < 3 {
            return raiseError("The 'each' function requires at least 3 arguments.")
        }

        for _, x := range evaluator(subTree.args[1]).args {
            variables[subTree.args[0].value] = x
            evalAll(subTree.args[2:])
        }
        return tree { value: "on" }

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
                tree { value: strconv.Itoa(x) })
        }

        return generatedList

    case "append", "a": // Append to a list.
        if len(subTree.args) < 1 {
            return raiseError( "The 'append' function requires at least one argument.")
        }

        list := evaluator(subTree.args[0]).args

        for _, x := range subTree.args[1:] {
            list = append(list, evaluator(x))
        }

        return tree {
            value: "list", 
            args: list,
        }

    case "index", "i":  // Element at an index.
        if len(subTree.args) != 2 {
            return raiseError("The 'index' function requires two arguments.")
        }

        list    := evaluator(subTree.args[0])
        index   := int(atomizer(evaluator(subTree.args[1])).num)

        if index < len(list.args) && index >= 0 {
            return list.args[index]
        }

        if len(list.args) + index >= 0 && index < 0 {
            return list.args[len(list.args)+index]
        }

        return raiseError(
            "Index out of range. \n" + 
            strconv.Itoa(index) + " of " + subTree.args[0].value)

    case "length":  // Length of a list.
        if len(subTree.args) != 1 {
            return raiseError("The 'length' function requires one argument.")
        }
        return tree {
            value: strconv.Itoa(len(evaluator(subTree.args[0]).args)),
        }

    // Mathmatical operators, such as adding numbers, checking for divisibility, etc.
    case "sum", "plus": // Sum all numerical args together.
        number := 0.0
        for _, x := range subTree.args {
            number += atomizer(evaluator(x)).num
        }
        return tree { value: strconv.FormatFloat(number, 'f', -1, 64) }

    case "subtract", "minus":    // Starting with the leftmost number, subtract all numbers after it.
        if len(subTree.args) >= 2 {
            number := atomizer(evaluator(subTree.args[0])).num
            for _, x := range subTree.args[1:] {
                number -= atomizer(evaluator(x)).num
            }
            return tree { value: strconv.FormatFloat(number, 'f', -1, 64) }
        }
        return raiseError("The 'subtract' function takes two or more num args.")

    case "multiply", "times":    // Multiply all numerical args together.
        number := 1.0
        for _, x := range subTree.args {
            number *= atomizer(evaluator(x)).num
        }
        return tree { value: strconv.FormatFloat(number, 'f', -1, 64) }

    case "divide":  // Starting with the leftmost number, divide it by all following numbers.
        if len(subTree.args) >= 2 {
            number := atomizer(evaluator(subTree.args[0])).num
            for _, x := range subTree.args[1:] {
                number /= atomizer(evaluator(x)).num
            }
            return tree { value: strconv.FormatFloat(number, 'f', -1, 64) }
        }
        return raiseError("The 'divide' function takes two or more num args.")
    
    case "mod", "modulo": // Division with a reminader, modulo.
        if len(subTree.args) == 2 {

            arg1 := atomizer(evaluator(subTree.args[0]))
            arg2 := atomizer(evaluator(subTree.args[1]))
            return tree { value: strconv.Itoa( int(arg1.num) % int(arg2.num) ) }
        }

        return raiseError("The 'mod' function takes exactly two arguments.")

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
            return raiseError(
                "The 'divisible' function only takes 'num' types.\n" +
                "You've given '" + arg1.Type + "' and '" + arg2.Type + "'.")
        }

        return raiseError("The 'divisible' function takes exactly two arguments.")

    // String manipulation functions.
    case "concat", "&":  // Concatonate strings.
        newString := "\""

        for _, x := range subTree.args {
            subString := atomizer(evaluator(x))
            newString += subString.str

            if subString.Type != "str" {
                raiseError("You used " + x.value + ", a '" + subString.Type + "' as a str.")
            }
        }

        return tree { value: newString, args: []tree{} }

    case "replace", "$": // Replace things in a string with a key-value pair, or mutiple key-value pairs.
        if len(subTree.args) % 2 != 1 {
            return raiseError("The 'replace' function requires an odd number of args.")
        }

        originalString := atomizer(evaluator(subTree.args[0]))

        if originalString.Type != "str" {
            return raiseError(
                "You can only use 'replace' on strings. You used a " + 
                originalString.Type + ".")
        }

        for x := 1; x < len(subTree.args); x += 2 {
            previous    := atomizer(evaluator(subTree.args[x]))
            replacement := atomizer(evaluator(subTree.args[x + 1]))

            originalString.str = strings.Replace(
                originalString.str, 
                previous.str, 
                replacement.str, 
            -1)

            if previous.Type != "str" || replacement.Type != "str" {
                return raiseError("You can't replace a string using a non string key or value.")
            }
        }

        return tree { value: "\"" + originalString.str }


    case "split": // To-Do
        if len(subTree.args) != 2 {
            return raiseError("The 'split' function requires two args.")
        }

        preString := atomizer(evaluator(subTree.args[0]))

        if preString.Type != "str" {
            return raiseError("The 'split' function only takes strings.")
        }

        var newList = tree {
            value: "list",
            args: []tree{},
        }

        for _, x := range subTree.args[1:] {
            currentSplit := atomizer(evaluator(x))

            if currentSplit.Type != "str" {
                return raiseError("The 'split' function only takes strings.")
            }

            newList.args = append(newList.args, )
        }

        return newList

    // Tree manipulation functions and tree management. 
    case "tree": // Evaluate the tree and return a 'lazyTree'.
        return tree { 
            value: "lazyTree",
            args: evalTrees(subTree.args),
        }

    case "of": // Get list containing all values assigned to a key. Repeated key is the same as one key with more values.

        if len(subTree.args) == 2 {
            var listOfTrees = []tree{}

            for _, x := range evaluator(subTree.args[0]).args {
                if x.value == subTree.args[1].value {
                    listOfTrees = append(listOfTrees, evaluator(x).args ...)
                }
            }

            return tree {
                value: "list",
                args: listOfTrees,
            }
        }

        return raiseError("The 'of' function takes exactly two arguments.")

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

        return raiseError("The 'typeConvert' function takes exactly two arguments.")

    // Things you shouldn't actually use, but exist because sometimes you have to.
    case "return": // This isn't actually a thing, I just use it to catch bad returns.
        return raiseError("You can't call return outside of a function's root.")

    // Kill DeviousYarn.
    case "die":
        os.Exit(0)
    }

    return raiseError("The word '" + subTree.value +  "' means nothing.")
}

func execute(input string) {    
    /*  This goes through all three major functions,
        the lezer, parser, and evaluator, and then 
        executes all of them on the input string.
     */
    tokenList           = lexer     ( input )
    programTree.args    = parser    ( )
    evaluator ( programTree )
}

func prompt() {
    /*  This creates a simple entry prompt for the user,
        which allows them to give simple input at the CLI.
     */
    reader := bufio.NewReader(os.Stdin)
    for {
        fmt.Println(" -input- ")
        input, _ := reader.ReadString('\n')
        fmt.Println(" -output- ")
        execute( input )
    }
}

func runFile(fileName string) {
    file, err := ioutil.ReadFile( fileName )
    if err != nil {
        raiseError("The file '" + fileName + "' could not be opened.")
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

        if extension == "die" || extension[:2] == "dy" {
            // All files ending in '*.die' or '*.dy*' get executed.
            runFile(flag.Arg(0))
        } else { 
            // Load a text file as a variable.
            variables["load"] = loadFile(flag.Arg(0))
            prompt()
        }

    } else {
        prompt()
    }
}