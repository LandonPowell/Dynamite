package main

import (
    "os"
    "fmt"
    "bufio"
    "regexp"
)

func tokenizer(plaintext string) []string {
    strings     := "'(\\\\'|[^'])+'|\"[^\n]+"
    brackets    := "[\\[\\](){}]"
    names       := "[\\w:-@^-`~\\|*-/!-&]+"

    tokens  := regexp.MustCompile(strings+"|"+brackets+"|"+names)
    return tokens.FindAllString(plaintext, -1)
}


type tree struct { // Tree of processes. It can also be a value.
    value   string  // If it's a value.
    subtree []tree  // If it's a bunch of subprocesses.
}

func nester(tokenList []string) tree {
    for _, token := range tokenList {
        token
    }
}

func main() {

    reader := bufio.NewReader(os.Stdin)

    var input string
    for input != "kill\n" {

        fmt.Println(" -input- ")
        input, _ = reader.ReadString('\n')

        fmt.Println(" -output- ")
        fmt.Println( tokenizer( input ) )

    }

    fmt.Println(" Thanks for using DeviousYarn~! ")
}

