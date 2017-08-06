(*| 
| |  Tokenization / lexical parser. 
|*)

type token = 
    | Name      of string   (* Used for function || variable names. *)
    | Number    of float    (* Used for all numbers, integer or float. *)
    | String    of string   (* String lits converted here. *)
    | Boolean   of bool     (* Boolean values. *)
    | Bracket   of bool     (* True if opening, False if closing. *)
    | ListBegin (* Special bracket type for square brackets. *)
    | Special   of char     (* Anything that doesn't fall into one of those goes here. *)
;;

let tokenize characters =
    let tokens = ref ( [] : token list )    in (* Stores the entire list of tokens. *)
    let buffer = Buffer.create 16           in (* Stores metadata for strings and names for identifiers. *)
    let accumulator = ref 0.0               in (* Stores metadata for Numbers *)
    let length = String.length characters   in
    let index  = ref 0                      in

    while !index < length do
        Buffer.clear buffer;
        accumulator := 0.0;

        match characters.[!index] with
            | (' ' | '\n' | '\r' | '\t') -> index := !index + 1 (* Whitespace? Ignore it. *)

            | ('#') -> ( (* Skip over comment lines. *)
                while !index < length && characters.[!index] != '\n' do
                    index := !index + 1
                done
            )

            (* Square Bracket. *)
            | ('[') -> (
                index := !index + 1;
                tokens := !tokens @ [ListBegin]
            )

            (* Brackets. *)
            | ('{' | '(') -> (
                index := !index + 1;
                tokens := !tokens @ [Bracket true]
            )
            | (')' | '}' | ']') -> (
                index := !index + 1;
                tokens := !tokens @ [Bracket false]
            )

            | ('\'') -> (
                (* Multi line string literals.*)
                index := !index + 1;
                while characters.[!index] != '\'' do
                    if characters.[!index] = '\\' then ( (* Escape handler. *)
                        index := !index + 1;
                    );
                    Buffer.add_char buffer characters.[!index];
                    index := !index + 1
                done;
                index := !index + 1;
                tokens := !tokens @ [String (Buffer.contents buffer)]
            )
            | ('"') -> (
                (* One line string literals. *)
                index := !index + 1;
                while !index < length && characters.[!index] != '\n' do
                    Buffer.add_char buffer characters.[!index];
                    index := !index + 1
                done;
                index := !index + 1;
                tokens := !tokens @ [String (Buffer.contents buffer)]
            )

            | ('=' | ':') -> (
                tokens := !tokens @ [Special characters.[!index]];
                index := !index + 1
            )
            | ('0' .. '9') -> (
                (* Numeric literals. *)
                while 
                    !index < length && 
                    match characters.[!index] with '0' .. '9' -> true | _ -> false
                do
                    accumulator := !accumulator *. 10.0 +. float_of_int (
                        int_of_char characters.[!index] - 48
                    );
                    index := !index + 1
                done;

                if !index < length && characters.[!index] = '.' then (
                    index := !index + 1;

                    let precision = ref 10.0 in
                    while 
                        !index < length && 
                        match characters.[!index] with '0' .. '9' -> true | _ -> false
                    do
                        accumulator := !accumulator +. float_of_int (
                            int_of_char characters.[!index] - 48
                        ) /. !precision;
                        precision := !precision *. 10.0;
                        index := !index + 1
                    done;
                );

                tokens := !tokens @ [Number !accumulator]
            )

            |  _  -> (
                while 
                    !index < length &&
                    not( String.contains " \n\r\t(){}[]'\"=:" characters.[!index] )
                do
                    Buffer.add_char buffer characters.[!index];
                    index := !index + 1
                done;

                tokens := !tokens @ [
                    if (Buffer.contents buffer) = "true" then Boolean true
                    else if Buffer.contents buffer = "false" then Boolean false
                    else Name (Buffer.contents buffer)
                ]
            )
        ; 
    done;

    !tokens
;;

(*| 
| |  Abstract syntax tree generation / central parser. 
|*)

type tree =
    | Call      of string * tree list (* Function calls are strings with a list of args. *)
    | List      of tree list
    | Number    of float
    | String    of string
    | Boolean   of bool
;;

let infixKeywords =
    let table = Hashtbl.create 15 in
    Hashtbl.add table (Name "is") ("is");
    Hashtbl.add table (Name "index") ("index");
    Hashtbl.add table (Special '=') ("function");
    table
;;

let rec parserLoop tokens =
    if tokens = [] || List.hd tokens = Bracket false then
        if tokens = [] then ([], []) (* SAFETY DANCE *)
        else ([], List.tl tokens)
    else
        let tree, leftover = parser tokens in
            let treelist, leftover = parserLoop leftover in
                (tree :: treelist, leftover)

and parser tokens = match tokens with
    | Name value::Special '='::leftover ->
        let tree, leftover = parser leftover in
        ( Call ("set", (String value)::[tree]), leftover )

    | Name value::Special ':'::leftover ->
        let tree, leftover = parser leftover in
        ( Call (value, [tree]), leftover )

    | Name value::Bracket true::leftover ->
        let trees, leftover = parserLoop leftover in
        infixParser (Call (value, trees)) leftover

    | ListBegin::leftover ->
        let trees, leftover = parserLoop leftover in
        infixParser (List trees) leftover

    | Name value::Name infix::leftover -> (
        if Hashtbl.mem infixKeywords (Name infix) then
            let newTree, leftover = parser leftover in
            ( Call (Hashtbl.find infixKeywords (Name infix),
                (Call (value, [])) :: [newTree]
            ),
            leftover)
        else ( Call (value, []), Name infix::leftover )
    )

    | Name value::leftover -> ( Call (value, []), leftover )
    | Number value::leftover -> ( Number value, leftover )
    | String value::leftover -> ( String value, leftover )
    | Boolean value::leftover -> ( Boolean value, leftover )

    | Bracket false::leftover ->
        ( print_endline "You used an unexpected closing bracket!";
        Number 0.0, [] )
    | other -> 
        ( print_endline "You mistyped something!"; 
        Number 0.0, [] )

and infixParser tree leftover =
    if (List.length leftover < 2) then
        (tree, leftover)

    else if Hashtbl.mem infixKeywords (List.hd leftover) then
        let operator = Hashtbl.find infixKeywords (List.hd leftover) in
        let newTree, leftover = parser (List.tl leftover) in
        (Call (operator, tree::[newTree]), leftover)

    else
        (tree, leftover)

;;

