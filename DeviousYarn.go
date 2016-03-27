package main

import (
    "bufio"
    "fmt"
    "os"
)

func main() {

    reader := bufio.NewReader(os.Stdin)
    input  := ""

    for input != "KILL" {
        fmt.Println(" -input- ")
        input, _ := reader.ReadString('\n')

        fmt.Println(" -output- ")
        fmt.Println(input)
    }

    fmt.Println(" Thanks for using DeviousYarn~! ")
}
