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
    names       := "[\\w-@^-`~\\|*-/!-&;]+"         // Regex for var names.

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
    var treeList = []tree{};
    
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

func main() {

    reader := bufio.NewReader(os.Stdin)

    var input string
    for input != "kill\n" {

        fmt.Println(" -input- ")

        input, _ = reader.ReadString('\n')

        fmt.Println(" -output- ")

        tokenList           = lexer ( input )
        programTree.args    = parser( )

        fmt.Println( tokenList )    // Tokenizer test.
        fmt.Println( programTree )  // Parser test.

    }

    fmt.Println(" Thanks for using DeviousYarn~! ")
}