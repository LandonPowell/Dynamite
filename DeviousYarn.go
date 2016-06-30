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

// This is mostly self-explaining, but it's the lexer (obviously).
func lexer(plaintext string) []string {
    strings     := "'(\\\\'|[^'])+'|\"[^\n]+"       // Regex for strings.
    brackets    := "[\\[\\](){}:]"                  // Regex for bracket chars.
    names       := "[\\w-@^-`~\\|*-/!-&;]+"         // Regex for var names.

    tokens  := regexp.MustCompile( strings+"|"+brackets+"|"+names )
    return tokens.FindAllString(plaintext, -1)
}


type tree struct { // Tree of processes. It can also be a value.
    value   string  // If it's a value.
    args    []tree  // If it's a bunch of subprocesses.
}

func parser(tokenList []string) []tree {

    var expression = []tree{}

    // Until the tokenList is empty or the next character is a closing bracket.
    for len(tokenList) > 0 && ! contains(tokenList[0], ")]j}") {
        token := tokenList[0]
        tokenList = tokenList[1:]

        subExpression := tree{
            value   : token,
            args    : []tree{},
        }

        // If the next character is an opening bracket.
        if len(tokenList) > 0 && contains(tokenList[0], "{f[(") {
            tokenList = tokenList[1:]               // Removes opening bracket.
            subExpression.args = parser(tokenList)  // Recurses through list.
        }

        expression = append(expression, subExpression)
    }
    return expression
}

func main() {

    reader := bufio.NewReader(os.Stdin)

    var input string
    for input != "kill\n" {

        fmt.Println(" -input- ")
        input, _ = reader.ReadString('\n')

        fmt.Println(" -output- ")
        var tokenList   = lexer ( input )
        var program     = parser( tokenList )
        fmt.Println( program )
    }

    fmt.Println(" Thanks for using DeviousYarn~! ")
}