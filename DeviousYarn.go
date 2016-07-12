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

// This is mostly self-explaining, but it's the tokenizer (obviously).
func lexer(plaintext string) []string {     // Returns a list of tokens.
    strings     := "'(\\\\'|[^'])+'|\"[^\n]+"       // Regex for strings.
    brackets    := "[\\[\\](){}:]"                  // Regex for bracket chars.
    names       := "[\\w-@^-`~\\|*-/!-&;?]+"        // Regex for var names.

    tokens  := regexp.MustCompile( strings+"|"+brackets+"|"+names )
    return tokens.FindAllString(plaintext, -1)
}

var tokenList = []string{}

type tree struct { // Tree of processes. It can also be a value.
    value   string // If it's a value.
    args    []tree // If it's a bunch of subprocesses.
}

var programTree = tree { // Default tree of the entire program.
    value: "run",
    args: []tree{},
}

// Instead of using 'tokenList' as an arg, we use a global token list. Recursion + Scope == Pain.
func parser() []tree {
    var treeList = []tree{}
    
    for len(tokenList) > 0 && !contains(tokenList[0], "j)]}") {

        var currentTree = tree {
            value: tokenList[0],
            args: []tree{},
        }
        tokenList = tokenList[1:] // Removes the first element in the slice.

        if len(tokenList) > 0 && contains(tokenList[0], "{[(f") {
            tokenList = tokenList[1:]
            currentTree.args = parser()
        }

        treeList = append(treeList, currentTree)
    }

    if len(tokenList) > 0 && contains(tokenList[0], "j)]}") {
        tokenList = tokenList[1:]
    }

    return treeList
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
        if (firstChar == "\"") {
            postAtom.str    = preAtom.value[1:] 
        } else {
            postAtom.str    = preAtom.value
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
        postAtom.type = "list"
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

func evaluator(subTree tree) tree {
    if val, ok := variables[subTree.value]; ok {    // This returns variable values.

        return evaluator(val)

    } else if subTree.value == "run" {  // This is a function similair to an anonymous function.

        return evalAll(subTree.args)

    } else if subTree.value == "set" {

        if len(subTree.args) > 1 {
            variables[subTree.args[0].value] = subTree.args[1]
            return subTree.args[1]
        }

        return tree { value: "off" }

    } else if subTree.value == "?" || subTree.value == "if" {   // Simple conditional.

        if len(subTree.args) > 1 && evaluator(subTree.args[0]).value == "on" {
            return evalAll(subTree.args[1:])
        }

        return tree { value: "off" }

    } else if subTree.value == "o" || subTree.value == "out" {  // This is a formated output, or 'println' minus templating.

        if len(subTree.args) > 0 {
            
            printArg := atomize(evaluator(subTree.args[0]))

            switch printArg.Type {  // Too bad I can't use printArg['str'] syntax.
            case "str"  : fmt.Println(printArg.str)
            case "num"  : fmt.Println(printArg.num)
            case "fun"  : fmt.Println(printArg.fun)
            case "file" : 
                if len(subTree.args) > 1 {
                    fmt.Println(printArg.file)
                } else {
                    fmt.Println(printArg.file)
                }
            }

            return evaluator(subTree.args[0])
        }

        return tree { value: "off" }
    
    } else if subTree.value == "print" || subTree.value == "p" {

            printArg := atomize(evaluator(subTree.args[0]))

            switch printArg.Type {
            case "str"  : fmt.Print(printArg.str)
            case "num"  : fmt.Print(printArg.num)
            default     : return tree { value: "on" }
            }

            return tree { value: "on" }

    } else if subTree.value == "rawOut" {   // This outputs the plaintext of a tree.

        if len(subTree.args) > 0 {
            fmt.Println(evaluator(subTree.args[0]))
            return subTree.args[0]
        }

        return tree { value: "off" }

    } else if subTree.value == "kill" {

        return tree { value: "This is a built in function of DeviousYarn." }

    } else if atomize(subTree).Type != "CAN NOT PARSE" {

        return subTree

    }

    return tree {   // Returns an error message for undefined names.
        value: "ERROR",
        args: []tree{
            tree { 
                value: "The word '" + subTree.value +  "' means nothing.",
                args: []tree{},
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

func runFile(input string) {
    file, err := ioutil.ReadFile( input )
    if err != nil {
        fmt.Println("The file '" + input + "' could not be opened.")
    } else {
        execute( string( file ) )
    }
}

func main() {
    flag.Parse()
    if len(flag.Args()) >= 2 {

        switch flag.Arg(0) {
            case "runFile": runFile(flag.Arg(1))
            case "load":    // To Do
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
            // To Do
        }

    } else {
        prompt()
    }
}