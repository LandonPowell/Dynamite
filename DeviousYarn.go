package main

import (
    "os"
    "fmt"
    "bufio"
    "regexp"
)

func tokenizer(plaintext string) [][]string {
    strings     := "'(\\\\'|[^'])+'|\"[^\n]+"
    brackets    := "[\\[\\](){}]"
    names       := "[\\w:-@^-`~\\|*-/!-&]+"

    tokens  := regexp.MustCompile(strings+"|"+brackets+"|"+names)
    return tokens.FindAllStringSubmatch(plaintext, -1)
}

func nester(tokenList [][]string) {
}

func main() {

    reader := bufio.NewReader(os.Stdin)

    var input string
    for input != "KILL\n" {

        fmt.Println(" -input- ")
        input, _ := reader.ReadString('\n')

        fmt.Println(" -output- ")
        fmt.Println( tokenizer( input ) )

    }

    fmt.Println(" Thanks for using DeviousYarn~! ")
}

