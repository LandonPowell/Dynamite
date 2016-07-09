package main

import (
    "os"
    "fmt"
    "bufio"
    "regexp"
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

var variables map[string]tree

func evalAll(treeList []tree) {
    for _, x := range(treeList) {
        currentRun := evaluator(x)

        if currentRun.value == "ERROR" {
            fmt.Println(" -error- ")
            fmt.Println(currentRun.args[0].value)
        }
    }
}

func evaluator(subTree tree) tree {
    if val, ok := variables[subTree.value]; ok {    // This returns variable values.
        return evaluator(val)
    } else if subTree.value == "run" {
        evalAll(subTree.args)

        return tree {
            value: "true",
            args: []tree{},
        }
    } else if subTree.value == "?" {
        if len(subTree.args) > 1 && evaluator(subTree.args[0]).value == "true" {
            return evaluator(subTree.args[1])
            evalAll(subTree.args[2:])
        }
        return tree {
            value: "false",
            args: []tree{},
        }
    } else if subTree.value == "o" || subTree.value == "out" {
        if len(subTree.args) > 0 {
            fmt.Println(evaluator(subTree.args[0]))
            return tree {
                value: "true",
                args: []tree{},
            }
        }

        return tree {
            value: "false",
            args: []tree{},
        }
    }

    return tree {   // Returns an error message for undefined names.
        value: "ERROR",
        args: []tree{
            tree { 
                value: "Value '" + subTree.value +  "' not found.",
                args: []tree{},
            },
        },
    }
}

func main() {

    reader := bufio.NewReader(os.Stdin)

    var input string
    for input != "kill\n" {

        fmt.Println(" -input- ")

        input, _ = reader.ReadString('\n')

        fmt.Println(" -output- ")

        tokenList           = lexer     ( input )
        programTree.args    = parser    ( )

        evaluator ( programTree )
    }

    fmt.Println(" Thanks for using DeviousYarn~! ")
}